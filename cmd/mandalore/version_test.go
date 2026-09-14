package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"runtime"
	"testing"

	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

func TestVersionReportsActualEmbeddedPackageAndBuildSource(t *testing.T) {
	v, code := cli(t, []string{"version"}, "")
	if code != 0 || !v.OK {
		t.Fatal(v, code)
	}
	r := v.Result.(map[string]any)
	files := map[string][]byte{}
	if err := fs.WalkDir(codexplugin.Files, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := codexplugin.Files.ReadFile(name)
		files[name] = b
		return err
	}); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	if r["plugin_sha256"] != hex.EncodeToString(h[:]) || r["go_version"] != runtime.Version() {
		t.Fatal("runtime does not report its actual embedded plugin/toolchain", r)
	}
	if r["source_commit"] != "" || r["version"] != "0.0.0-dev" {
		t.Fatal("unstamped development build claims release provenance", r)
	}
	for _, key := range []string{"signet_read_versions", "signet_write_versions"} {
		values, ok := r[key].([]any)
		if !ok || len(values) != 1 || values[0] != float64(1) {
			t.Fatal("missing schema compatibility", r)
		}
	}
}
