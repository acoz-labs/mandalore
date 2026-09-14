package distribution

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// StageCandidateArchive verifies before creating a fresh private directory. It
// preserves exact payload bytes with non-executable permissions, never follows
// member paths or overwrites entries, and returns a partial directory on failure
// for inspection. A receipt/expected identity is not acceptance or provenance:
// the caller must independently refresh trusted transport and authorize release.
// reader must refer to the caller's retained, stable archive descriptor.
func StageCandidateArchive(ctx context.Context, reader io.ReaderAt, size int64, t CandidateTransport, identity, parent string) (directory string, manifest ParsedManifest, err error) {
	if err := ctx.Err(); err != nil {
		return "", manifest, err
	}
	verified, err := VerifyCandidateArchive(ctx, reader, size, t)
	if err != nil {
		return "", manifest, err
	}
	if verified.Identity() != identity {
		return "", manifest, errors.New("archive does not match the selected candidate identity")
	}
	parentDir, err := openInstallDirectory(parent)
	if err != nil {
		return "", manifest, errors.New("staging parent must be an existing non-redirected directory")
	}
	defer parentDir.Close()
	if err := candidateZipLayout(reader, size); err != nil {
		return "", manifest, err
	}
	z, err := zip.NewReader(reader, size)
	if err != nil || len(z.File) != 8 {
		return "", manifest, errors.New("candidate archive changed before staging")
	}
	want := map[string]Asset{}
	for _, a := range verified.Manifest.Assets {
		want[a.Name] = a
	}
	// Exact manifest/checksum bytes were already verified. The final directory
	// check below binds them to the selected manifest identity again.
	want["manifest.json"] = Asset{Name: "manifest.json", SHA256: verified.SHA256}
	want["SHA256SUMS"] = Asset{Name: "SHA256SUMS"}
	seen := map[string]bool{}
	for _, f := range z.File {
		a, ok := want[f.Name]
		if !ok || seen[f.Name] || !f.Mode().IsRegular() || f.UncompressedSize64 == 0 || f.UncompressedSize64 > MaxBinaryBytes || a.Size > 0 && f.UncompressedSize64 != uint64(a.Size) || a.Size == 0 && f.UncompressedSize64 > MaxManifestBytes {
			return "", manifest, errors.New("candidate member inventory changed before staging")
		}
		seen[f.Name] = true
	}
	if err := ctx.Err(); err != nil {
		return "", manifest, err
	}
	directory, err = os.MkdirTemp(parent, "mandalore-payload-")
	if err != nil {
		return "", manifest, errors.New("cannot create a fresh private staging directory")
	}
	dir, err := openInstallDirectory(directory)
	if err != nil {
		return directory, manifest, err
	}
	defer dir.Close()
	for _, member := range z.File {
		if err := ctx.Err(); err != nil {
			return directory, manifest, err
		}
		r, err := member.Open()
		if err != nil {
			return directory, manifest, errors.New("candidate member cannot be read during staging")
		}
		fd, err := unix.Openat(int(dir.Fd()), member.Name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
		if err != nil {
			r.Close()
			return directory, manifest, errors.New("cannot create a new private candidate member")
		}
		file := os.NewFile(uintptr(fd), member.Name)
		h := sha256.New()
		n, readErr := io.Copy(io.MultiWriter(file, h), io.LimitReader(installContextReader{ctx, r}, int64(member.UncompressedSize64)+1))
		closeRead, closeWrite := r.Close(), file.Close()
		a := want[member.Name]
		if readErr != nil || closeRead != nil || closeWrite != nil || n != int64(member.UncompressedSize64) || a.SHA256 != "" && hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
			return directory, manifest, errors.New("candidate member failed exact-byte staging; partial directory retained")
		}
	}
	staged, err := VerifyDirectory(directory)
	if err != nil {
		return directory, manifest, err
	}
	if staged.Identity() != identity {
		return directory, manifest, errors.New("staged candidate identity changed")
	}
	// Detect archive changes during extraction before returning a usable payload.
	if _, err := VerifyCandidateArchive(ctx, reader, size, t); err != nil {
		return directory, manifest, err
	}
	if err := ctx.Err(); err != nil {
		return directory, manifest, err
	}
	return directory, staged, nil
}
