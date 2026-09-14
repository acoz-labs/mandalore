package distribution

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/sys/unix"
)

// Checksums covers exact manifest bytes and the complete payload inventory. The
// checksum file cannot hash itself. Sort independently of manifest asset order.
func Checksums(manifest []byte) ([]byte, error) {
	p, err := ParseManifest(manifest)
	if err != nil {
		return nil, err
	}
	sums := map[string]string{"manifest.json": p.SHA256}
	for _, a := range p.Manifest.Assets {
		sums[a.Name] = a.SHA256
	}
	names := make([]string, 0, len(sums))
	for name := range sums {
		names = append(names, name)
	}
	sort.Strings(names)
	var b bytes.Buffer
	for _, name := range names {
		b.WriteString(sums[name] + "  " + name + "\n")
	}
	return b.Bytes(), nil
}

// openCandidateFile accepts only a direct regular child of an already-open
// directory. NOFOLLOW/NONBLOCK avoid symlink escapes and FIFO races between
// metadata inspection and open. The descriptor pins the selected file.
func openCandidateFile(dir *os.File, name string, limit int64) (*os.File, error) {
	if filepath.Base(name) != name || name == "." || name == ".." {
		return nil, errors.New("invalid candidate filename")
	}
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errors.New("candidate file is missing, redirected or unreadable")
	}
	f := os.NewFile(uintptr(fd), name)
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		f.Close()
		return nil, errors.New("candidate file type or size is invalid")
	}
	return f, nil
}

func readCandidateFile(dir *os.File, name string, limit int64) ([]byte, error) {
	f, err := openCandidateFile(dir, name, limit)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, errors.New("candidate file exceeded read limit")
	}
	return b, nil
}

// VerifyDirectory checks a complete local payload without executing code or
// modifying files. It is byte/content verification, NOT publisher authentication,
// source-build attestation or native runtime acceptance. Callers must establish
// transport provenance separately and reverify when applying an installation.
func VerifyDirectory(path string) (ParsedManifest, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return ParsedManifest{}, errors.New("candidate directory is unavailable")
	}
	dir := os.NewFile(uintptr(fd), path)
	defer dir.Close()
	st, err := dir.Stat()
	if err != nil || !st.IsDir() {
		return ParsedManifest{}, errors.New("candidate path is not a directory")
	}
	b, err := readCandidateFile(dir, "manifest.json", MaxManifestBytes)
	if err != nil {
		return ParsedManifest{}, err
	}
	p, err := ParseManifest(b)
	if err != nil {
		return ParsedManifest{}, err
	}
	wantSums, err := Checksums(b)
	if err != nil {
		return ParsedManifest{}, err
	}
	gotSums, err := readCandidateFile(dir, "SHA256SUMS", MaxManifestBytes)
	if err != nil || !bytes.Equal(gotSums, wantSums) {
		return ParsedManifest{}, errors.New("candidate checksums do not match its complete manifest")
	}
	want := map[string]bool{"manifest.json": true, "SHA256SUMS": true}
	for _, a := range p.Manifest.Assets {
		want[a.Name] = true
		f, err := openCandidateFile(dir, a.Name, a.Size)
		if err != nil {
			return ParsedManifest{}, err
		}
		h := sha256.New()
		n, readErr := io.Copy(h, io.LimitReader(f, a.Size+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
			return ParsedManifest{}, errors.New("candidate payload size or digest mismatch")
		}
		if a.Kind == "codex-plugin" {
			archive, err := readCandidateFile(dir, a.Name, MaxPluginBytes)
			if err != nil || Digest(archive) != a.SHA256 {
				return ParsedManifest{}, errors.New("candidate plugin changed during verification")
			}
			if err := VerifyPlugin(archive, p.Manifest.Version, p.Manifest.PluginSHA256); err != nil {
				return ParsedManifest{}, err
			}
		}
	}
	// Bound enumeration too; an untrusted directory need not be small just because
	// its manifest is. Transport receipts must remain outside the payload directory.
	entries, err := dir.ReadDir(len(want) + 1)
	if err != nil && err != io.EOF {
		return ParsedManifest{}, errors.New("cannot inspect candidate inventory")
	}
	if len(entries) != len(want) {
		return ParsedManifest{}, errors.New("candidate must contain exactly its declared payload and checksums")
	}
	for _, entry := range entries {
		if !want[entry.Name()] || !entry.Type().IsRegular() {
			return ParsedManifest{}, errors.New("candidate contains undeclared or nonregular entries")
		}
	}
	return p, nil
}
