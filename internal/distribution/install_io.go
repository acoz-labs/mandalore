package distribution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/acoz-labs/mandalore/internal/strictjson"
	"golang.org/x/sys/unix"
)

func syncInstallDirectory(path string) error {
	f, err := openInstallDirectory(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func readInstallFile(path string, limit int64) ([]byte, error) {
	d, err := openInstallDirectory(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return readCandidateFile(d, filepath.Base(path), limit)
}

func writeInstallFile(path string, b []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// Only newly created private staging roots are recursively cleaned. A changed
// inode is preserved instead of treating the original pathname as ownership.
func ownedInstallTemp(parent, pattern string) (string, func(), error) {
	path, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", nil, err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	return path, func() {
		if now, e := os.Lstat(path); e == nil && os.SameFile(st, now) {
			_ = os.RemoveAll(path)
		}
	}, nil
}

type installContextReader struct {
	context.Context
	io.Reader
}

func (r installContextReader) Read(b []byte) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(b)
}

func copyInstallPayload(ctx context.Context, from, to string, limit int64) error {
	d, err := openInstallDirectory(filepath.Dir(from))
	if err != nil {
		return err
	}
	defer d.Close()
	f, err := openCandidateFile(d, filepath.Base(from), limit)
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(installContextReader{ctx, f}, limit+1))
	if err == nil && n > limit {
		err = errors.New("installation payload exceeded its declared size")
	}
	if err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func stageInstallSource(ctx context.Context, p InstallPlan, client *ReleaseClient, root string) error {
	manifestPath := filepath.Join(root, "manifest.json")
	binaryPath := filepath.Join(root, "mandalore")
	if p.Source.Kind == "github-release" {
		for _, item := range []struct{ name, path string }{{"manifest.json", manifestPath}, {p.Binary.Name, binaryPath}} {
			f, err := os.OpenFile(item.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			err = client.DownloadAsset(ctx, *p.Source.Published, item.name, f)
			if err == nil {
				err = f.Sync()
			}
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
	} else {
		source, name := p.Candidate, p.Binary.Name
		if p.Source.Kind == "retained" {
			source, name = filepath.Dir(p.Runtime), "mandalore"
		}
		if err := copyInstallPayload(ctx, filepath.Join(source, "manifest.json"), manifestPath, MaxManifestBytes); err != nil {
			return err
		}
		if err := copyInstallPayload(ctx, filepath.Join(source, name), binaryPath, p.Binary.Size); err != nil {
			return err
		}
	}
	return verifyInstallPair(root, p)
}

func verifyInstallPair(root string, p InstallPlan) error {
	b, err := readInstallFile(filepath.Join(root, "manifest.json"), MaxManifestBytes)
	if err != nil {
		return err
	}
	m, err := ParseManifest(b)
	if err != nil || !reflect.DeepEqual(m, p.Source.Manifest) {
		return errors.New("staged manifest differs from the reviewed plan")
	}
	d, err := openInstallDirectory(root)
	if err != nil {
		return err
	}
	defer d.Close()
	f, err := openCandidateFile(d, "mandalore", p.Binary.Size)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, p.Binary.Size+1))
	if err != nil || n != p.Binary.Size || hex.EncodeToString(h.Sum(nil)) != p.Binary.SHA256 {
		return errors.New("staged runtime differs from the reviewed digest")
	}
	return nil
}

func copyInstallPair(ctx context.Context, from, to string, p InstallPlan) error {
	if err := copyInstallPayload(ctx, filepath.Join(from, "manifest.json"), filepath.Join(to, "manifest.json"), MaxManifestBytes); err != nil {
		return err
	}
	if err := copyInstallPayload(ctx, filepath.Join(from, "mandalore"), filepath.Join(to, "mandalore"), p.Binary.Size); err != nil {
		return err
	}
	if err := verifyInstallPair(to, p); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(to, "mandalore"), 0700); err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(to, "mandalore"))
	if err != nil {
		return err
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return syncInstallDirectory(to)
}

func verifyInstallRuntime(ctx context.Context, path string, p InstallPlan) error {
	if err := VerifyBuiltBinary(path, p.Source.Manifest.Manifest, p.Binary); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Version reporting needs no memory binding, provider credentials, native
	// account or home directory. Do not inherit those just to probe a CLI.
	raw, err := runBuildTool(ctx, filepath.Dir(path), []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}, 16<<10, path, "version")
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("selected runtime version probe failed; raw output suppressed")
	}
	return verifyRuntimeVersion(raw, p)
}

func verifyRuntimeVersion(raw []byte, p InstallPlan) error {
	var v struct {
		Protocol int  `json:"protocol_version"`
		OK       bool `json:"ok"`
		Result   struct {
			Name          string `json:"name"`
			Version       string `json:"version"`
			Source        string `json:"source_commit"`
			OS            string `json:"os"`
			Arch          string `json:"arch"`
			Go            string `json:"go_version"`
			Protocol      int    `json:"protocol_version"`
			Hook          int    `json:"codex_hook_protocol"`
			Read          []int  `json:"signet_read_versions"`
			Write         []int  `json:"signet_write_versions"`
			PluginVersion string `json:"plugin_version"`
			PluginSHA     string `json:"plugin_sha256"`
		} `json:"result"`
	}
	m := p.Source.Manifest.Manifest
	if err := strictjson.Decode(raw, &v, 16<<10); err != nil || !v.OK || v.Protocol != 1 || v.Result.Name != "mandalore" || v.Result.Version != m.Version || v.Result.Source != m.SourceCommit || v.Result.OS != p.OS || v.Result.Arch != p.Arch || v.Result.Go != m.GoVersion || v.Result.Protocol != m.ProtocolVersion || v.Result.Hook != 1 || v.Result.PluginVersion != m.Version || v.Result.PluginSHA != m.PluginSHA256 || !reflect.DeepEqual(v.Result.Read, m.SignetReadVersions) || !reflect.DeepEqual(v.Result.Write, m.SignetWriteVersions) {
		return errors.New("runtime version, source, target, protocol, schema or embedded plugin differs from its manifest")
	}
	return nil
}

func lockInstallState(state string) (func(), bool, error) {
	path := filepath.Join(state, "write.lock")
	flags := unix.O_RDWR | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC
	fd, err := unix.Open(path, flags|unix.O_CREAT|unix.O_EXCL, 0600)
	created := err == nil
	if errors.Is(err, unix.EEXIST) {
		fd, err = unix.Open(path, flags, 0)
	}
	if err != nil {
		return nil, created, errors.New("installation lock cannot be opened safely")
	}
	f := os.NewFile(uintptr(fd), "write.lock")
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 {
		f.Close()
		return nil, created, errors.New("installation lock is not a private regular file")
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, created, errors.New("another installer holds this prefix lock; retry after it finishes")
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = f.Close() }, created, nil
}

func atomicInstallJSON(path string, b []byte, replace bool) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".receipt-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if replace {
		err = os.Rename(tmp, path)
	} else {
		err = publishCandidate(tmp, path)
	}
	if err != nil {
		return err
	}
	return syncInstallDirectory(dir)
}

func nextInstallReceipt(p InstallPlan, key string) []byte {
	b, _ := json.Marshal(cliReceipt{FormatVersion: 1, Product: "mandalore", Prefix: p.Prefix, Current: p.Source.Manifest.SHA256, Previous: p.Observed.Current, LastPlan: key})
	return append(b, '\n')
}

func sameInstallBytes(path string, want []byte) bool {
	if len(want) == 0 {
		_, err := os.Lstat(path)
		return os.IsNotExist(err)
	}
	b, err := readInstallFile(path, MaxPendingInstallBytes)
	return err == nil && bytes.Equal(b, want)
}
