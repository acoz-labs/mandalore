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
	hermesplugin "github.com/acoz-labs/mandalore/plugins/hermes"
)

type HermesReceipt struct {
	Plan  HermesPlan        `json:"plan"`
	Files map[string]string `json:"files"`
}

func (p HermesPlan) runtimePlan() Plan {
	return Plan{Options: p.Options, Runtime: p.Runtime, BinarySHA256: p.BinarySHA256}
}

func hermesConnection(p HermesPlan) ([]byte, error) {
	value := map[string]any{
		"schema_version": 1, "harness": "hermes", "runtime": p.Runtime, "runtime_sha256": p.BinarySHA256,
		"binding": p.Binding, "binding_sha256": p.BindingSHA256, "signet_id": p.SignetID,
		"package_sha256": p.PackageSHA256, "package_version": p.PackageVersion, "read_only": p.ReadOnly,
		"native_home": p.NativeHome, "native_binary": p.NativeBinary, "state_dir": p.StateDir, "connection_root": p.Root,
	}
	if p.SessionTransportVersion == 1 {
		raw, err := sessionPolicy(p.Options, p.Runtime, p.BinarySHA256, p.BindingSHA256, p.SignetID)
		if err != nil {
			return nil, err
		}
		value["session_policy"] = filepath.Join(p.Root, "package", "session-policy.json")
		value["session_policy_sha256"] = hash(raw)
	}
	return json.MarshalIndent(value, "", "  ")
}

func hermesBundle(p HermesPlan) (map[string][]byte, HermesReceipt, error) {
	public, err := hermesplugin.PackageFiles()
	if err != nil {
		return nil, HermesReceipt{}, err
	}
	info, err := hermesplugin.Inspect()
	if err != nil || info.SHA256 != p.PackageSHA256 || info.Version != p.PackageVersion {
		return nil, HermesReceipt{}, errors.New("selected Hermes package differs from the reviewed plan")
	}
	files := map[string][]byte{}
	for name, raw := range public {
		files["package/"+name] = raw
	}
	files["package/connection.json"], err = hermesConnection(p)
	if err == nil && p.SessionTransportVersion == 1 {
		files["package/session-policy.json"], err = sessionPolicy(p.Options, p.Runtime, p.BinarySHA256, p.BindingSHA256, p.SignetID)
	}
	if err != nil {
		return nil, HermesReceipt{}, err
	}
	r := HermesReceipt{Plan: p, Files: map[string]string{}}
	for name, raw := range files {
		r.Files[name] = hash(raw)
	}
	return files, r, nil
}

func loadHermesReceipt(root string) (HermesReceipt, error) {
	raw, err := readRegular(filepath.Join(root, "receipt.json"), 65536)
	if err != nil {
		return HermesReceipt{}, err
	}
	return decodeHermesReceipt(root, raw)
}

func decodeHermesReceipt(root string, raw []byte) (HermesReceipt, error) {
	var r HermesReceipt
	if strictjson.Decode(raw, &r, 65536) != nil {
		return r, errors.New("invalid Hermes ownership receipt")
	}
	p := r.Plan
	if p.SchemaVersion != 1 || p.Harness != "hermes" || p.Root != root || root != filepath.Join(p.StateDir, "hermes", "connections", hermesPlanKey(p)) || p.Runtime != filepath.Join(p.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore") || p.PackageVersion == "" {
		return HermesReceipt{}, errors.New("Hermes receipt identity or managed paths changed")
	}
	for _, value := range []string{p.BinarySHA256, p.NativeSHA256, p.BindingSHA256, p.PackageSHA256} {
		b, e := hex.DecodeString(value)
		if e != nil || len(b) != 32 || strings.ToLower(value) != value {
			return HermesReceipt{}, errors.New("invalid Hermes receipt digest")
		}
	}
	if len(r.Files) < 2 || len(r.Files) > 129 || r.Files["package/connection.json"] == "" || r.Files["package/plugin.yaml"] == "" {
		return HermesReceipt{}, errors.New("invalid Hermes receipt inventory")
	}
	for name, value := range r.Files {
		b, e := hex.DecodeString(value)
		if !fs.ValidPath(name) || !strings.HasPrefix(name, "package/") || strings.Contains(name, "\\") || e != nil || len(b) != 32 || strings.ToLower(value) != value {
			return HermesReceipt{}, errors.New("invalid Hermes receipt file")
		}
	}
	expected, err := hermesConnection(p)
	if err != nil || hash(expected) != r.Files["package/connection.json"] {
		return HermesReceipt{}, errors.New("Hermes administrative context differs from receipt")
	}
	if err := validateSessionReceipt(p.Options, p.Runtime, p.BinarySHA256, p.BindingSHA256, p.SignetID, r.Files, "package/session-policy.json"); err != nil {
		return HermesReceipt{}, err
	}
	if p.ReadOnly && p.SessionTransportVersion != 0 {
		return HermesReceipt{}, errors.New("read-only receipt cannot authorize synchronization")
	}

	return r, nil
}

func verifyHermesTree(r HermesReceipt, allowMissing bool) error {
	root := r.Plan.Root
	resolved, err := canonical(root)
	if err != nil || resolved != root {
		return errors.New("Hermes generation is redirected")
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
			return errors.New("Hermes generation exceeds inventory limit")
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("Hermes generation contains a symlink")
		}
		if d.IsDir() {
			if d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
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
			return errors.New("Hermes generation contains an unexpected file; preserve it")
		}
		raw, err := readRegular(path, 1<<20)
		if err != nil || hash(raw) != want {
			return errors.New("Hermes package was edited or cannot be read; preserve it")
		}
		total += len(raw)
		if total > (1<<20)+16384 {
			return errors.New("Hermes package exceeds byte limit")
		}
		seen[name] = true
		if name != "package/session-policy.json" && name != "package/connection.json" {
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
		return errors.New("Hermes generation is incomplete; preview repair")
	}
	raw, err := json.Marshal(public)
	if err != nil || hash(raw) != r.Plan.PackageSHA256 {
		return errors.New("Hermes public package differs from the pinned identity")
	}
	return nil
}

func ownedHermes(root, state, home string, allowMissing bool) (HermesReceipt, error) {
	r, err := loadHermesReceipt(root)
	if err != nil {
		return r, err
	}
	if r.Plan.StateDir != state || r.Plan.NativeHome != home {
		return HermesReceipt{}, errors.New("Hermes connection belongs to a different installation or profile")
	}
	if err := verifyHermesTree(r, allowMissing); err != nil {
		return HermesReceipt{}, err
	}
	d, err := digest(r.Plan.Runtime)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return r, nil
		}
		return HermesReceipt{}, err
	}
	if d != r.Plan.BinarySHA256 {
		return HermesReceipt{}, errors.New("retained Hermes runtime was edited; preserve it")
	}
	return r, nil
}

func publishHermesBundle(p HermesPlan) error {
	files, r, err := hermesBundle(p)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(p.Root); err == nil {
		old, e := loadHermesReceipt(p.Root)
		if e != nil || old.Plan != p {
			return errors.New("existing Hermes generation is incomplete or belongs to another plan")
		}
		return verifyHermesTree(old, false)
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
	return verifyHermesTree(r, false)
}
