package install

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/strictjson"
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

type PiReceipt struct {
	Plan  PiPlan            `json:"plan"`
	Files map[string]string `json:"files"`
}

func (p PiPlan) runtimePlan() Plan {
	return Plan{Options: p.Options, Runtime: p.Runtime, BinarySHA256: p.BinarySHA256}
}

func piConnection(p PiPlan) ([]byte, error) {
	return json.MarshalIndent(map[string]any{
		"schema_version": 1, "harness": "pi", "runtime": p.Runtime, "runtime_sha256": p.BinarySHA256,
		"binding": p.Binding, "binding_sha256": p.BindingSHA256, "signet_id": p.SignetID,
		"package_sha256": p.PackageSHA256, "package_version": p.PackageVersion, "read_only": p.ReadOnly,
		"native_home": p.NativeHome, "native_binary": p.NativeBinary, "state_dir": p.StateDir, "connection_root": p.Root,
	}, "", "  ")
}

func piBundle(p PiPlan) (map[string][]byte, PiReceipt, error) {
	public, err := piplugin.PackageFiles()
	if err != nil {
		return nil, PiReceipt{}, err
	}
	info, err := piplugin.Inspect()
	if err != nil || info.SHA256 != p.PackageSHA256 || info.Version != p.PackageVersion {
		return nil, PiReceipt{}, errors.New("selected Pi package differs from the reviewed plan")
	}
	files := map[string][]byte{}
	for name, raw := range public {
		files["package/"+name] = raw
	}
	files["package/connection.json"], err = piConnection(p)
	if err != nil {
		return nil, PiReceipt{}, err
	}
	r := PiReceipt{Plan: p, Files: map[string]string{}}
	for name, raw := range files {
		r.Files[name] = hash(raw)
	}
	return files, r, nil
}

func loadPiReceipt(root string) (PiReceipt, error) {
	var r PiReceipt
	raw, err := readRegular(filepath.Join(root, "receipt.json"), 65536)
	if err != nil {
		return r, err
	}
	if strictjson.Decode(raw, &r, 65536) != nil {
		return r, errors.New("invalid Pi ownership receipt")
	}
	p := r.Plan
	if p.SchemaVersion != 1 || p.Harness != "pi" || p.Root != root || root != filepath.Join(p.StateDir, "pi", "connections", piPlanKey(p)) || p.Runtime != filepath.Join(p.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore") || p.PackageVersion == "" {
		return PiReceipt{}, errors.New("Pi receipt identity or managed paths changed")
	}
	for _, value := range []string{p.BinarySHA256, p.NativeSHA256, p.BindingSHA256, p.PackageSHA256} {
		b, e := hex.DecodeString(value)
		if e != nil || len(b) != 32 || strings.ToLower(value) != value {
			return PiReceipt{}, errors.New("invalid Pi receipt digest")
		}
	}
	if len(r.Files) < 2 || len(r.Files) > 129 || r.Files["package/connection.json"] == "" || r.Files["package/package.json"] == "" {
		return PiReceipt{}, errors.New("invalid Pi receipt inventory")
	}
	for name, value := range r.Files {
		b, e := hex.DecodeString(value)
		if !fs.ValidPath(name) || !strings.HasPrefix(name, "package/") || strings.Contains(name, "\\") || e != nil || len(b) != 32 || strings.ToLower(value) != value {
			return PiReceipt{}, errors.New("invalid Pi receipt file")
		}
	}
	expected, err := piConnection(p)
	if err != nil || hash(expected) != r.Files["package/connection.json"] {
		return PiReceipt{}, errors.New("Pi administrative context differs from receipt")
	}
	return r, nil
}

func verifyPiTree(r PiReceipt, allowMissing bool) error {
	root := r.Plan.Root
	resolved, err := canonical(root)
	if err != nil || resolved != root {
		return errors.New("Pi generation is redirected")
	}
	seen := map[string]bool{}
	public := map[string][]byte{}
	entries, total := 0, 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries++
		if entries > 256 {
			return errors.New("Pi generation exceeds inventory limit")
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("Pi generation contains a symlink")
		}
		if d.IsDir() {
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if name == "receipt.json" {
			return nil
		}
		want, ok := r.Files[name]
		if !ok {
			return errors.New("Pi generation contains an unexpected file; preserve it")
		}
		raw, err := readRegular(path, 1<<20)
		if err != nil || hash(raw) != want {
			return errors.New("Pi package was edited or cannot be read; preserve it")
		}
		total += len(raw)
		if total > (1<<20)+16384 {
			return errors.New("Pi package exceeds byte limit")
		}
		seen[name] = true
		if name != "package/connection.json" {
			public[strings.TrimPrefix(name, "package/")] = raw
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != len(r.Files) {
		if allowMissing {
			return nil
		}
		return errors.New("Pi generation is incomplete; preview repair")
	}
	raw, err := json.Marshal(public)
	if err != nil || hash(raw) != r.Plan.PackageSHA256 {
		return errors.New("Pi public package differs from the pinned identity")
	}
	return nil
}

func ownedPi(root, state, home string, allowMissing bool) (PiReceipt, error) {
	r, err := loadPiReceipt(root)
	if err != nil {
		return r, err
	}
	if r.Plan.StateDir != state || r.Plan.NativeHome != home {
		return PiReceipt{}, errors.New("Pi connection belongs to a different installation or profile")
	}
	if err := verifyPiTree(r, allowMissing); err != nil {
		return PiReceipt{}, err
	}
	d, err := digest(r.Plan.Runtime)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return r, nil
		}
		return PiReceipt{}, err
	}
	if d != r.Plan.BinarySHA256 {
		return PiReceipt{}, errors.New("retained Pi runtime was edited; preserve it")
	}
	return r, nil
}

func publishPiBundle(p PiPlan) error {
	files, r, err := piBundle(p)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(p.Root); err == nil {
		old, e := loadPiReceipt(p.Root)
		if e != nil || old.Plan != p {
			return errors.New("existing Pi generation is incomplete or belongs to another plan")
		}
		return verifyPiTree(old, false)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := realDirectory(filepath.Dir(p.Root)); err != nil {
		return err
	}
	if err := os.Mkdir(p.Root, 0700); err != nil {
		return err
	}
	for name, raw := range files {
		if err := writeNew(filepath.Join(p.Root, filepath.FromSlash(name)), raw); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := writeNew(filepath.Join(p.Root, "receipt.json"), raw); err != nil {
		return err
	}
	return verifyPiTree(r, false)
}
