package install

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type Receipt struct {
	Plan  Plan              `json:"plan"`
	Files map[string]string `json:"files"`
}

func realDirectory(path string) error {
	resolved, err := canonical(path)
	if err != nil || resolved != path {
		return errors.New("managed directory is redirected")
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	resolved, err = canonical(path)
	if err != nil || resolved != path {
		return errors.New("managed directory changed during preparation")
	}
	return nil
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func stageRuntime(p Plan) error {
	got, err := digest(p.Binary)
	if err != nil || got != p.BinarySHA256 {
		return errors.New("runtime changed after preview; preview again")
	}
	dir := filepath.Dir(p.Runtime)
	if err := realDirectory(dir); err != nil {
		return err
	}
	if st, err := os.Lstat(p.Runtime); err == nil {
		got, readErr := digest(p.Runtime)
		if readErr != nil || got != p.BinarySHA256 || !st.Mode().IsRegular() || st.Mode().Perm() != 0700 {
			return errors.New("existing runtime copy was edited; preserve it before recovery")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	in, err := os.Open(p.Binary)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(dir, ".stage-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // Only this operation's newly created scratch file.
	defer tmp.Close()
	n, err := io.Copy(tmp, io.LimitReader(in, maxBinary+1))
	if err != nil || n > maxBinary {
		return errors.New("runtime copy failed or exceeded size limit")
	}
	if err := tmp.Chmod(0700); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	got, err = digest(tmp.Name())
	if err != nil || got != p.BinarySHA256 {
		return errors.New("runtime changed during staging")
	}
	if err := os.Link(tmp.Name(), p.Runtime); err != nil {
		return err
	}
	return syncDirectory(dir)
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

func bundle(p Plan) (map[string][]byte, Receipt, error) {
	files, err := packageFiles()
	if err != nil {
		return nil, Receipt{}, err
	}
	prefix := "plugins/mandalore/"
	files[prefix+"scripts/connection.sh"] = []byte("#!/bin/sh\n" +
		"if [ -z \"${MANDALORE_BIN:-}\" ]; then MANDALORE_BIN=" + quote(p.Runtime) + "; fi\n" +
		"if [ -z \"${MANDALORE_BINDING:-}\" ]; then MANDALORE_BINDING=" + quote(p.Binding) + "; fi\n" +
		"export MANDALORE_BIN MANDALORE_BINDING\n" +
		"exec /bin/sh \"$(dirname \"$0\")/run-memory.sh\" \"$@\"\n")
	for _, name := range []string{prefix + ".mcp.json", prefix + "hooks/hooks.json"} {
		if !bytes.Contains(files[name], []byte("scripts/run-memory.sh")) {
			return nil, Receipt{}, errors.New("embedded bridge contract changed")
		}
		files[name] = bytes.ReplaceAll(files[name], []byte("scripts/run-memory.sh"), []byte("scripts/connection.sh"))
	}
	var manifest map[string]any
	if err := json.Unmarshal(files[prefix+".codex-plugin/plugin.json"], &manifest); err != nil {
		return nil, Receipt{}, err
	}
	manifest["version"] = p.Version
	files[prefix+".codex-plugin/plugin.json"], err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, Receipt{}, err
	}
	r := Receipt{Plan: p, Files: map[string]string{}}
	for name, b := range files {
		r.Files[name] = hash(b)
	}
	files["connection.json"], err = json.MarshalIndent(r, "", "  ")
	return files, r, err
}

func readRegular(path string, limit int64) ([]byte, error) {
	resolved, err := canonical(path)
	if err != nil || resolved != path {
		return nil, errors.New("managed file is redirected")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > limit {
		return nil, errors.New("managed file type or size is invalid")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, errors.New("managed file read exceeded its limit")
	}
	return b, nil
}

func loadReceipt(root string) (Receipt, error) {
	var r Receipt
	b, err := readRegular(filepath.Join(root, "connection.json"), 32768)
	if err != nil {
		return r, err
	}
	if err := strictjson.Decode(b, &r, 32768); err != nil {
		return r, err
	}
	if r.Plan.SchemaVersion != 1 || r.Plan.Root != root || len(r.Files) == 0 || len(r.Files) > 64 {
		return Receipt{}, errors.New("invalid connection receipt")
	}
	p := r.Plan
	if p.Marketplace != "mandalore" || p.PluginID != "mandalore@mandalore" ||
		p.Root != filepath.Join(p.StateDir, "connections", planKey(p)) ||
		p.Runtime != filepath.Join(p.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore") {
		return Receipt{}, errors.New("connection receipt identity changed")
	}
	for _, value := range []string{p.BinarySHA256, p.NativeSHA256, p.BindingSHA256, p.PackageSHA256} {
		if b, err := hex.DecodeString(value); err != nil || len(b) != 32 {
			return Receipt{}, errors.New("invalid connection digest")
		}
	}
	for name, digest := range r.Files {
		if !fs.ValidPath(name) || name == "." || name == "connection.json" || strings.Contains(name, "\\") || len(digest) != 64 {
			return Receipt{}, errors.New("invalid connection receipt file inventory")
		}
		if b, err := hex.DecodeString(digest); err != nil || len(b) != 32 {
			return Receipt{}, errors.New("invalid file digest")
		}
	}
	return r, nil
}

// verifyTree permits absent resources only for an explicit recovery preflight.
// Unexpected files, symlinks and edited bytes always require ownership review.
func verifyTree(root string, files map[string]string, allowMissing bool) error {
	resolved, err := canonical(root)
	if err != nil || resolved != root {
		return errors.New("managed tree is redirected")
	}
	seen := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("managed tree contains a symlink")
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if name == "connection.json" && files[".agents/plugins/marketplace.json"] != "" {
			return nil
		} // Validated separately by loadReceipt.
		want, ok := files[name]
		if !ok {
			return errors.New("managed tree contains an unexpected file; preserve edits")
		}
		b, err := readRegular(path, 128<<10)
		if err != nil || hash(b) != want {
			return errors.New("managed file changed; preserve edits before recovery")
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	if !allowMissing && len(seen) != len(files) {
		return errors.New("managed tree is incomplete; preview repair")
	}
	return nil
}

func writeNew(path string, b []byte) error {
	if err := realDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func publishBundle(p Plan) error {
	files, expected, err := bundle(p)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(p.Root); err == nil {
		r, err := loadReceipt(p.Root)
		if err != nil || r.Plan != p {
			return errors.New("existing generation is incomplete or belongs to another plan; preserve it")
		}
		b, _ := json.Marshal(r.Files)
		want, _ := json.Marshal(expected.Files)
		if !bytes.Equal(b, want) {
			return errors.New("existing generation receipt was edited; preserve it")
		}
		return verifyTree(p.Root, r.Files, false)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := realDirectory(filepath.Dir(p.Root)); err != nil {
		return err
	}
	// Claim the directory exclusively. A partial write stays visibly incomplete;
	// no existing generation is renamed over, erased or silently repaired.
	if err := os.Mkdir(p.Root, 0700); err != nil {
		return err
	}
	for name, b := range files {
		if name == "connection.json" {
			continue
		}
		if err := writeNew(filepath.Join(p.Root, filepath.FromSlash(name)), b); err != nil {
			return err
		}
	}
	if err := writeNew(filepath.Join(p.Root, "connection.json"), files["connection.json"]); err != nil {
		return err
	}
	return verifyTree(p.Root, expected.Files, false)
}
