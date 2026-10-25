package distribution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// legacyManifestFixture has the published 1.6.1 manifest's exact shape, with
// plugin_sha256 and a codex-plugin asset, over synthetic payload digests.
func legacyManifestFixture(binary []byte) legacyManifest {
	l := legacyManifest{FormatVersion: 1, Product: "mandalore", Version: "1.6.1", Tag: "v1.6.1", SourceCommit: strings.Repeat("1", 40), GoVersion: "go1.26.4",
		ProtocolVersion: 1, SignetReadVersions: []int{1, 2}, SignetWriteVersions: []int{1, 2}, PluginSHA256: strings.Repeat("f", 64)}
	for _, target := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}} {
		a := Asset{Kind: "cli", OS: target[0], Arch: target[1], Name: "mandalore_1.6.1_" + target[0] + "_" + target[1], Size: 20, SHA256: strings.Repeat("c", 64)}
		if target[0] == runtime.GOOS && target[1] == runtime.GOARCH {
			a.Size, a.SHA256 = int64(len(binary)), Digest(binary)
		}
		l.Assets = append(l.Assets, a)
	}
	l.Assets = append(l.Assets,
		Asset{Kind: "codex-plugin", Name: "mandalore_1.6.1_codex.zip", Size: 50804, SHA256: strings.Repeat("d", 64)},
		Asset{Kind: "bootstrap", Name: "install.sh", Size: 4673, SHA256: strings.Repeat("e", 64)})
	return l
}

func encodeLegacy(t *testing.T, l legacyManifest) []byte {
	t.Helper()
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

// fixtureLegacyInstall lays out a prefix the way 1.6.1's installer leaves it:
// a retained manifest/runtime pair, an ownership receipt and the launcher.
func fixtureLegacyInstall(t *testing.T, prefix string, manifest, binary []byte) string {
	t.Helper()
	identity := Digest(manifest)
	root := retainedRoot(prefix, identity, runtime.GOOS, runtime.GOARCH)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mandalore"), binary, 0700); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cliReceipt{FormatVersion: 1, Product: "mandalore", Prefix: prefix, Current: identity})
	if err := os.WriteFile(filepath.Join(cliState(prefix), "receipt.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(prefix, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "mandalore"), filepath.Join(prefix, "bin", "mandalore")); err != nil {
		t.Fatal(err)
	}
	return identity
}

func legacyInstallOptions(t *testing.T) (InstallOptions, string) {
	t.Helper()
	o := localInstallOptions(t)
	if err := os.MkdirAll(o.Prefix, 0700); err != nil {
		t.Fatal(err)
	}
	prefix, err := canonicalPrefix(o.Prefix)
	if err != nil {
		t.Fatal(err)
	}
	o.Prefix = prefix
	binary := []byte("synthetic 1.6.1 runtime")
	return o, fixtureLegacyInstall(t, prefix, encodeLegacy(t, legacyManifestFixture(binary)), binary)
}

func TestInstallUpgradesAndRollsBackRetained1xPrefix(t *testing.T) {
	o, legacy := legacyInstallOptions(t)
	p, err := PlanInstall(context.Background(), o)
	if err != nil || p.Observed.Current != legacy || p.Source.Manifest.Legacy != nil {
		t.Fatal("2.0 cannot plan over a 1.x install", err)
	}
	r, err := installForTest(t, p, nil)
	if err != nil || !r.Installed || r.PreviousRuntime != filepath.Join(retainedRoot(o.Prefix, legacy, p.OS, p.Arch), "mandalore") {
		t.Fatal("update over 1.x failed", r, err)
	}
	raw, _ := os.ReadFile(filepath.Join(cliState(o.Prefix), "receipt.json"))
	var receipt cliReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil || receipt.Current != p.Source.Manifest.SHA256 || receipt.Previous != legacy {
		t.Fatal("receipt does not keep the 1.x runtime as previous", string(raw))
	}

	back, err := PlanInstall(context.Background(), InstallOptions{Prefix: o.Prefix, Retained: legacy})
	if err != nil || back.Source.Manifest.Legacy == nil || back.Source.Manifest.Legacy.PluginSHA256 != strings.Repeat("f", 64) || back.Binary.Name != "mandalore_1.6.1_"+runtime.GOOS+"_"+runtime.GOARCH {
		t.Fatal("rollback to the retained 1.x runtime not planned", err)
	}
	encoded, _ := json.Marshal(back)
	if parsed, err := ParseInstallPlan(encoded); err != nil || !reflect.DeepEqual(parsed, back) {
		t.Fatal("1.x rollback plan does not round trip", err)
	}
	r, err = installForTest(t, back, nil)
	if err != nil || r.Runtime != back.Runtime || r.PreviousRuntime != p.Runtime {
		t.Fatal("rollback to 1.x failed", r, err)
	}
	if target, err := os.Readlink(back.Launcher); err != nil || target != back.Runtime {
		t.Fatal("launcher not back on the 1.x runtime", err)
	}
}

func TestInstallRefusesTampered1xRetainedRuntime(t *testing.T) {
	for kind, want := range map[string]string{
		"manifest":    "retained manifest no longer matches its receipt identity",
		"binary":      "retained runtime does not match its manifest",
		"codex-claim": "retained manifest is not a readable release manifest: retained 1.x Codex package claim is invalid",
	} {
		t.Run(kind, func(t *testing.T) {
			o, legacy := legacyInstallOptions(t)
			root := retainedRoot(o.Prefix, legacy, runtime.GOOS, runtime.GOARCH)
			path := filepath.Join(root, "manifest.json")
			switch kind {
			case "manifest":
				b, _ := os.ReadFile(path)
				if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			case "binary":
				if err := os.WriteFile(filepath.Join(root, "mandalore"), []byte("synthetic 1.6.1 runtimE"), 0700); err != nil {
					t.Fatal(err)
				}
			case "codex-claim":
				// Re-point the receipt at a manifest whose digest matches but whose
				// retired claim is malformed: identity alone must not be enough.
				binary := []byte("synthetic 1.6.1 runtime")
				l := legacyManifestFixture(binary)
				l.Assets[4].Name = "mandalore_1.6.0_codex.zip"
				b := encodeLegacy(t, l)
				if err := os.RemoveAll(filepath.Join(o.Prefix, "bin")); err != nil {
					t.Fatal(err)
				}
				if err := os.RemoveAll(cliState(o.Prefix)); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(cliState(o.Prefix), 0700); err != nil {
					t.Fatal(err)
				}
				fixtureLegacyInstall(t, o.Prefix, b, binary)
			}
			if _, err := PlanInstall(context.Background(), o); err == nil || err.Error() != want {
				t.Fatalf("tampered 1.x install: got %v, want %q", err, want)
			}
		})
	}
}

func TestRetained1xManifestToleratesOnlyRetiredClaims(t *testing.T) {
	binary := []byte("synthetic 1.6.1 runtime")
	b := encodeLegacy(t, legacyManifestFixture(binary))
	p, err := parseRetainedManifest(b)
	if err != nil || p.SHA256 != Digest(b) || len(p.Manifest.Assets) != 5 || p.Legacy == nil || p.Legacy.CodexPlugin.Name != "mandalore_1.6.1_codex.zip" {
		t.Fatal("1.6.1-shaped manifest refused", err)
	}
	if _, err := p.Manifest.Binary(runtime.GOOS, runtime.GOARCH); err != nil {
		t.Fatal(err)
	}
	// New candidates and published releases keep the strict 2.0 parser.
	if _, err := ParseManifest(b); err == nil {
		t.Fatal("strict parser accepted a 1.x manifest")
	}
	current, err := EncodeManifest(fixtureManifest())
	if err != nil {
		t.Fatal(err)
	}
	if p, err := parseRetainedManifest(current); err != nil || p.Legacy != nil {
		t.Fatal("current manifest not read as current", err)
	}
	for name, change := range map[string]func(*legacyManifest){
		"2.x with retired claims": func(l *legacyManifest) {
			l.Version, l.Tag = "2.0.0", "v2.0.0"
			for i := range l.Assets {
				l.Assets[i].Name = strings.Replace(l.Assets[i].Name, "1.6.1", "2.0.0", 1)
			}
		},
		"plugin digest":    func(l *legacyManifest) { l.PluginSHA256 = "not-a-digest" },
		"no codex package": func(l *legacyManifest) { l.Assets = append(l.Assets[:4], l.Assets[5]) },
		"two codex packages": func(l *legacyManifest) {
			l.Assets = append(l.Assets, l.Assets[4])
		},
		"codex target":   func(l *legacyManifest) { l.Assets[4].OS = "linux" },
		"codex size":     func(l *legacyManifest) { l.Assets[4].Size = maxLegacyPluginBytes + 1 },
		"codex digest":   func(l *legacyManifest) { l.Assets[4].SHA256 = "x" },
		"other rule":     func(l *legacyManifest) { l.Assets[0].Name = "renamed" },
		"unknown kind":   func(l *legacyManifest) { l.Assets[4].Kind = "pi-plugin" },
		"bad go version": func(l *legacyManifest) { l.GoVersion = "go1" },
	} {
		l := legacyManifestFixture(binary)
		change(&l)
		if _, err := parseRetainedManifest(encodeLegacy(t, l)); err == nil {
			t.Error("accepted", name)
		}
	}
	for name, raw := range map[string]string{
		"unknown field": strings.Replace(string(b), `"plugin_sha256"`, `"extra": 1, "plugin_sha256"`, 1),
		"garbled":       string(b[:len(b)/2]),
		"duplicate":     strings.Replace(string(b), `"plugin_sha256"`, `"product": "mandalore", "plugin_sha256"`, 1),
	} {
		if _, err := parseRetainedManifest([]byte(raw)); err == nil {
			t.Error("accepted", name)
		}
	}
}

func TestInstallRefuses1xManifestAsNewCandidate(t *testing.T) {
	o := localInstallOptions(t)
	binary := []byte("synthetic 1.6.1 runtime")
	b := encodeLegacy(t, legacyManifestFixture(binary))
	if err := os.WriteFile(filepath.Join(o.Candidate, "manifest.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanInstall(context.Background(), o); err == nil {
		t.Fatal("1.x candidate accepted by the 2.0 installer")
	}
}

func TestRuntimeVersionBindsRetired1xClaimsExactly(t *testing.T) {
	o, legacy := legacyInstallOptions(t)
	p, err := PlanInstall(context.Background(), InstallOptions{Prefix: o.Prefix, Retained: legacy})
	if err != nil {
		t.Fatal(err)
	}
	m := p.Source.Manifest.Manifest
	// The shape 1.6.1's `mandalore version` prints.
	fields := map[string]any{"name": "mandalore", "version": m.Version, "source_commit": m.SourceCommit, "os": p.OS, "arch": p.Arch, "go_version": m.GoVersion, "protocol_version": 1,
		"signet_read_versions": []int{1, 2}, "signet_write_versions": []int{1, 2}, "codex_hook_protocol": 1, "plugin_version": m.Version, "plugin_sha256": p.Source.Manifest.Legacy.PluginSHA256}
	probe := func() []byte {
		raw, _ := json.Marshal(map[string]any{"protocol_version": 1, "ok": true, "result": fields})
		return raw
	}
	if err := verifyRuntimeVersion(probe(), p); err != nil {
		t.Fatal("1.x runtime metadata refused", err)
	}
	for key, wrong := range map[string]any{"codex_hook_protocol": 2, "plugin_version": "1.6.0", "plugin_sha256": strings.Repeat("0", 64)} {
		old := fields[key]
		fields[key] = wrong
		if err := verifyRuntimeVersion(probe(), p); err == nil {
			t.Error("wrong 1.x claim accepted", key)
		}
		delete(fields, key)
		if err := verifyRuntimeVersion(probe(), p); err == nil {
			t.Error("missing 1.x claim accepted", key)
		}
		fields[key] = old
	}

	current, err := PlanInstall(context.Background(), localInstallOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	cm := current.Source.Manifest.Manifest
	raw, _ := json.Marshal(map[string]any{"protocol_version": 1, "ok": true, "result": map[string]any{"name": "mandalore", "version": cm.Version, "source_commit": cm.SourceCommit, "os": current.OS, "arch": current.Arch,
		"go_version": cm.GoVersion, "protocol_version": 1, "signet_read_versions": []int{1}, "signet_write_versions": []int{1}, "plugin_sha256": strings.Repeat("f", 64)}})
	if err := verifyRuntimeVersion(raw, current); err == nil {
		t.Fatal("current runtime accepted retired plugin claims")
	}
}
