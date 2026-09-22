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
	claudeplugin "github.com/acoz-labs/mandalore/plugins/claude-code"
)

type ClaudeReceipt struct {
	Plan  ClaudePlan        `json:"plan"`
	Files map[string]string `json:"files"`
}

func (p ClaudePlan) runtimePlan() Plan {
	return Plan{Options: p.Options, Runtime: p.Runtime, BinarySHA256: p.BinarySHA256}
}

func claudeConnection(p ClaudePlan) ([]byte, error) {
	return json.MarshalIndent(map[string]any{
		"schema_version": 1, "harness": "claude-code", "runtime": p.Runtime, "runtime_sha256": p.BinarySHA256,
		"binding": p.Binding, "binding_sha256": p.BindingSHA256, "signet_id": p.SignetID,
		"package_sha256": p.PackageSHA256, "package_version": p.PackageVersion, "read_only": p.ReadOnly,
		"native_home": p.NativeHome, "native_binary": p.NativeBinary, "state_dir": p.StateDir, "connection_root": p.Root,
	}, "", "  ")
}

func claudeBundle(p ClaudePlan) (map[string][]byte, ClaudeReceipt, error) {
	public, err := claudeplugin.PackageFiles()
	if err != nil {
		return nil, ClaudeReceipt{}, err
	}
	info, err := claudeplugin.Inspect()
	if err != nil || info.SHA256 != p.PackageSHA256 || info.Version != p.PackageVersion {
		return nil, ClaudeReceipt{}, errors.New("selected Claude package differs from the reviewed plan")
	}
	files := map[string][]byte{}
	for name, raw := range public {
		files["package/"+name] = raw
	}
	files["package/connection.json"], err = claudeConnection(p)
	if err == nil && p.SessionTransportVersion == 1 {
		files["package/session-policy.json"], err = sessionPolicy(p.Options, p.Runtime, p.BinarySHA256, p.BindingSHA256, p.SignetID)
	}
	if err == nil {
		files["package/scripts/connection.sh"] = claudeBridge(p)
		files[".claude-plugin/marketplace.json"], err = claudeMarketplace(p)
	}
	if err == nil {
		var manifest map[string]any
		_ = json.Unmarshal(files["package/.claude-plugin/plugin.json"], &manifest)
		manifest["version"] = p.nativeVersion()
		files["package/.claude-plugin/plugin.json"], err = json.Marshal(manifest)
	}
	if err != nil {
		return nil, ClaudeReceipt{}, err
	}
	r := ClaudeReceipt{Plan: p, Files: map[string]string{}}
	for name, raw := range files {
		r.Files[name] = hash(raw)
	}
	return files, r, nil
}

func loadClaudeReceipt(root string) (ClaudeReceipt, error) {
	raw, err := readRegular(filepath.Join(root, "receipt.json"), 65536)
	if err != nil {
		return ClaudeReceipt{}, err
	}
	return decodeClaudeReceipt(root, raw)
}

func decodeClaudeReceipt(root string, raw []byte) (ClaudeReceipt, error) {
	var r ClaudeReceipt
	if strictjson.Decode(raw, &r, 65536) != nil {
		return r, errors.New("invalid Claude ownership receipt")
	}
	p := r.Plan
	if p.SchemaVersion != 1 || p.Harness != "claude-code" || p.Root != root || root != filepath.Join(p.StateDir, "claude-code", "connections", claudePlanKey(p)) || p.Runtime != filepath.Join(p.StateDir, "runtimes", "sha256-"+p.BinarySHA256, "mandalore") || p.PackageVersion == "" {
		return ClaudeReceipt{}, errors.New("Claude receipt identity or managed paths changed")
	}
	for _, value := range []string{p.BinarySHA256, p.NativeSHA256, p.BindingSHA256, p.PackageSHA256} {
		b, e := hex.DecodeString(value)
		if e != nil || len(b) != 32 || strings.ToLower(value) != value {
			return ClaudeReceipt{}, errors.New("invalid Claude receipt digest")
		}
	}
	if len(r.Files) < 2 || len(r.Files) > 129 || r.Files["package/connection.json"] == "" || r.Files["package/.claude-plugin/plugin.json"] == "" {
		return ClaudeReceipt{}, errors.New("invalid Claude receipt inventory")
	}
	for name, value := range r.Files {
		b, e := hex.DecodeString(value)
		if !fs.ValidPath(name) || (!strings.HasPrefix(name, "package/") && name != ".claude-plugin/marketplace.json") || strings.Contains(name, "\\") || e != nil || len(b) != 32 || strings.ToLower(value) != value {
			return ClaudeReceipt{}, errors.New("invalid Claude receipt file")
		}
	}
	marketplace, err := claudeMarketplace(p)
	if err != nil || r.Files[".claude-plugin/marketplace.json"] != hash(marketplace) {
		return ClaudeReceipt{}, errors.New("Claude marketplace differs from receipt")
	}
	if r.Files["package/scripts/connection.sh"] != hash(claudeBridge(p)) {
		return ClaudeReceipt{}, errors.New("Claude bridge differs from receipt")
	}
	expected, err := claudeConnection(p)
	if err != nil || hash(expected) != r.Files["package/connection.json"] {
		return ClaudeReceipt{}, errors.New("Claude administrative context differs from receipt")
	}
	if err := validateSessionReceipt(p.Options, p.Runtime, p.BinarySHA256, p.BindingSHA256, p.SignetID, r.Files, "package/session-policy.json"); err != nil {
		return ClaudeReceipt{}, err
	}
	if p.ReadOnly && p.SessionTransportVersion != 0 {
		return ClaudeReceipt{}, errors.New("read-only receipt cannot authorize synchronization")
	}

	return r, nil
}

func verifyClaudeTree(r ClaudeReceipt, allowMissing bool) error {
	root := r.Plan.Root
	resolved, err := canonical(root)
	if err != nil || resolved != root {
		return errors.New("Claude generation is redirected")
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
			return errors.New("Claude generation exceeds inventory limit")
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("Claude generation contains a symlink")
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
			return errors.New("Claude generation contains an unexpected file; preserve it")
		}
		raw, err := readRegular(path, 1<<20)
		if err != nil || hash(raw) != want {
			return errors.New("Claude package was edited or cannot be read; preserve it")
		}
		total += len(raw)
		if total > (1<<20)+16384 {
			return errors.New("Claude package exceeds byte limit")
		}
		seen[name] = true
		if name != "package/session-policy.json" && name != "package/connection.json" && name != "package/scripts/connection.sh" && name != ".claude-plugin/marketplace.json" {
			if name == "package/.claude-plugin/plugin.json" {
				var manifest map[string]any
				if json.Unmarshal(raw, &manifest) != nil || manifest["version"] != r.Plan.nativeVersion() {
					return errors.New("Claude native version changed")
				}
				manifest["version"] = r.Plan.PackageVersion
				raw, _ = json.Marshal(manifest)
			}
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
		return errors.New("Claude generation is incomplete; preview repair")
	}
	raw, err := json.Marshal(public)
	if err != nil || hash(raw) != r.Plan.PackageSHA256 {
		return errors.New("Claude public package differs from the pinned identity")
	}
	return nil
}

func ownedClaude(root, state, home string, allowMissing bool) (ClaudeReceipt, error) {
	r, err := loadClaudeReceipt(root)
	if err != nil {
		return r, err
	}
	if r.Plan.StateDir != state || r.Plan.NativeHome != home {
		return ClaudeReceipt{}, errors.New("Claude connection belongs to a different installation or profile")
	}
	if err := verifyClaudeTree(r, allowMissing); err != nil {
		return ClaudeReceipt{}, err
	}
	d, err := digest(r.Plan.Runtime)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return r, nil
		}
		return ClaudeReceipt{}, err
	}
	if d != r.Plan.BinarySHA256 {
		return ClaudeReceipt{}, errors.New("retained Claude runtime was edited; preserve it")
	}
	return r, nil
}

func publishClaudeBundle(p ClaudePlan) error {
	files, r, err := claudeBundle(p)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(p.Root); err == nil {
		old, e := loadClaudeReceipt(p.Root)
		if e != nil || old.Plan != p {
			return errors.New("existing Claude generation is incomplete or belongs to another plan")
		}
		return verifyClaudeTree(old, false)
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
	return verifyClaudeTree(r, false)
}

func (p ClaudePlan) nativeVersion() string {
	base, _, _ := strings.Cut(p.PackageVersion, "+")
	return base + "+claude." + claudePlanKey(p)
}
func claudeBridge(p ClaudePlan) []byte {
	args := " --binding " + quote(p.Binding) + " --binding-sha256 " + quote(p.BindingSHA256) + " --signet-id " + quote(p.SignetID)
	transport, _ := sessionArgs(p.Options, filepath.Join(p.Root, "package"), p.Runtime, p.BinarySHA256, p.BindingSHA256, p.SignetID)
	args += transport
	readOnly := ""
	if p.ReadOnly {
		readOnly = " --read-only"
	}
	guard := "#!/bin/sh\nmandalore_runtime=" + quote(p.Runtime) + "\n" +
		"if [ -x /usr/bin/sha256sum ]; then mandalore_digest=$(/usr/bin/sha256sum \"$mandalore_runtime\" 2>/dev/null); elif [ -x /usr/bin/shasum ]; then mandalore_digest=$(/usr/bin/shasum -a 256 \"$mandalore_runtime\" 2>/dev/null); else mandalore_digest=unavailable; fi\n" +
		"mandalore_digest=${mandalore_digest%% *}\nif [ \"$mandalore_digest\" != " + quote(p.BinarySHA256) + " ]; then\n if [ \"${1:-}\" = hook ]; then printf '%s\\n' '{\"systemMessage\":\"Mandalore retained runtime integrity check failed; no memory was changed.\"}'; exit 0; fi\n printf '%s\\n' 'Mandalore retained runtime integrity check failed.' >&2; exit 1\nfi\n"
	fallback := "Mandalore hook unavailable; no memory was changed."
	if p.SessionTransportVersion == 1 {
		fallback = "Mandalore hook unavailable after a possible refresh attempt; freshness is unconfirmed."
	}
	return []byte(guard + "case ${1:-} in\nhook)\n if value=$(" + quote(p.Runtime) + " claude-code-memory-hook" + args + " 2>/dev/null); then printf '%s\\n' \"$value\"; else printf '%s\\n' '{\"systemMessage\":\"" + fallback + "\"}'; fi\n ;;\nmcp) exec " + quote(p.Runtime) + " mcp --harness claude-code" + args + readOnly + " ;;\n*) exit 1 ;;\nesac\n")
}

func (p ClaudePlan) cacheVersion() string { return strings.ReplaceAll(p.nativeVersion(), "+", "-") }

func claudeMarketplace(p ClaudePlan) ([]byte, error) {
	return json.Marshal(map[string]any{"name": "mandalore", "owner": map[string]string{"name": "Mandalore"}, "plugins": []any{map[string]any{"name": "mandalore", "source": "./package", "version": p.nativeVersion()}}})
}
