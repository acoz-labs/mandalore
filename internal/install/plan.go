// Package install owns explicit machine-local native memory connections.
// Adapted selectively from pinned public My Friday memoryinstall; see NOTICE.
package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/binding"
	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

const maxBinary = 128 << 20
const maxNativeBinary = 512 << 20 // The inspected native Codex executable exceeds 128 MiB.

type Options struct {
	SessionTransportVersion int    `json:"session_transport_version,omitempty"`
	StateDir                string `json:"state_dir"`
	NativeHome              string `json:"native_home"`
	NativeBinary            string `json:"native_binary"`
	Binary                  string `json:"source_binary"`
	Binding                 string `json:"binding"`
	Generation              string `json:"generation,omitempty"`
}

type Plan struct {
	Options
	SchemaVersion  int    `json:"schema_version"`
	SignetID       string `json:"signet_id"`
	BinarySHA256   string `json:"binary_sha256"`
	NativeSHA256   string `json:"native_sha256"`
	BindingSHA256  string `json:"binding_sha256"`
	PackageSHA256  string `json:"package_sha256"`
	PackageVersion string `json:"package_version"`
	Runtime        string `json:"runtime"`
	Root           string `json:"marketplace_root"`
	Marketplace    string `json:"marketplace"`
	PluginID       string `json:"plugin_id"`
	Version        string `json:"plugin_version"`
}

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func planKey(p Plan) string {
	p.Root, p.Version = "", ""
	b, _ := json.Marshal(p) // Plan contains only scalar values.
	return hash(b)
}

func inside(root, path string) bool {
	r, err := filepath.Rel(root, path)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

// canonical resolves an absent path through real existing ancestors without
// creating anything. Dangling symlinks cannot become installation destinations.
func canonical(path string) (string, error) {
	if !filepath.IsAbs(path) || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return "", errors.New("absolute paths without control characters required")
	}
	path = filepath.Clean(path)
	if _, err := os.Lstat(path); err == nil {
		return filepath.EvalSymlinks(path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if filepath.Dir(path) == path {
		return "", errors.New("no existing path ancestor")
	}
	parent, err := canonical(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func digest(path string) (string, error) {
	return digestLimit(path, maxBinary)
}

func digestLimit(path string, limit int64) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	// Refuse FIFOs/devices before open, which can block even without reading.
	if !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		return "", errors.New("executable must be a nonempty regular file within its size limit")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err = f.Stat()
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limit {
		return "", errors.New("executable must be a nonempty regular file within its size limit")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, limit+1))
	if err != nil || n > limit {
		return "", errors.New("runtime hashing failed or exceeded size limit")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func packageFiles() (map[string][]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(codexplugin.Files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := codexplugin.Files.ReadFile(path)
		if err == nil {
			files[path] = b
		}
		return err
	})
	return files, err
}

// Prepare only reads local inputs. Runtime/native execution and publication
// belong to explicit apply, never preview. A digest is not publisher trust.
func Prepare(o Options) (Plan, error) {
	var p Plan
	if o.SessionTransportVersion != 0 && o.SessionTransportVersion != 1 {
		return p, errors.New("unsupported session transport policy")
	}
	var err error
	for _, path := range []*string{&o.StateDir, &o.NativeHome, &o.NativeBinary, &o.Binary, &o.Binding} {
		*path, err = canonical(*path)
		if err != nil {
			return p, err
		}
	}
	for _, path := range []string{o.StateDir, o.NativeHome} {
		st, err := os.Lstat(path)
		if err == nil && !st.IsDir() {
			return p, errors.New("installation state and native home must be directories")
		}
		if err != nil && !os.IsNotExist(err) {
			return p, err
		}
	}
	if len(o.Generation) > 64 || strings.IndexFunc(o.Generation, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-')
	}) >= 0 {
		return p, errors.New("invalid connection generation")
	}
	s, err := binding.Open(o.Binding, "installation")
	if err != nil {
		return p, err
	}
	for _, pair := range [][2]string{{s.Root(), o.StateDir}, {s.Root(), o.NativeHome}, {o.StateDir, o.NativeHome}} {
		if inside(pair[0], pair[1]) || inside(pair[1], pair[0]) {
			return p, errors.New("signet, native home and installation state must not overlap")
		}
	}
	if inside(o.StateDir, o.Binding) || inside(o.NativeHome, o.Binding) {
		return p, errors.New("binding must be separate from managed installation and native home")
	}
	p = Plan{Options: o, SchemaVersion: 1, SignetID: s.ID(), Marketplace: "mandalore", PluginID: "mandalore@mandalore"}
	p.BinarySHA256, err = digest(o.Binary)
	if err != nil {
		return Plan{}, err
	}
	p.NativeSHA256, err = digestLimit(o.NativeBinary, maxNativeBinary)
	if err != nil {
		return Plan{}, err
	}
	// Binding.Open already enforces format/size; bound this second identity read too.
	f, err := os.Open(o.Binding)
	if err != nil {
		return Plan{}, err
	}
	b, err := io.ReadAll(io.LimitReader(f, 32769))
	closeErr := f.Close()
	if err != nil || closeErr != nil || len(b) > 32768 {
		return Plan{}, errors.New("cannot read bounded binding identity")
	}
	p.BindingSHA256 = hash(b)
	files, err := packageFiles()
	if err != nil {
		return Plan{}, err
	}
	var manifest struct{ Name, Version string }
	if err := json.Unmarshal(files["plugins/mandalore/.codex-plugin/plugin.json"], &manifest); err != nil || manifest.Name != "mandalore" || manifest.Version == "" {
		return Plan{}, errors.New("invalid embedded plugin manifest")
	}
	var marketplace struct{ Name string }
	if err := json.Unmarshal(files[".agents/plugins/marketplace.json"], &marketplace); err != nil || marketplace.Name != p.Marketplace {
		return Plan{}, errors.New("invalid embedded marketplace")
	}
	p.PackageVersion = manifest.Version
	b, err = json.Marshal(files)
	if err != nil {
		return Plan{}, err
	}
	p.PackageSHA256 = hash(b)
	p.Runtime = filepath.Join(o.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore")
	key := planKey(p)
	p.Root = filepath.Join(o.StateDir, "connections", key)
	base, _, _ := strings.Cut(manifest.Version, "+")
	p.Version = base + "+codex." + key
	for _, path := range []string{p.Root, p.Runtime} {
		resolved, err := canonical(path)
		if err != nil || resolved != path {
			return Plan{}, errors.New("managed destination is redirected; preserve it and choose a real state directory")
		}
	}
	return p, nil
}
