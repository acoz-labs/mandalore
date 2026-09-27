package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"runtime"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/strictjson"
	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

func TestVersionRemainsReadableByReleased13Installer(t *testing.T) {
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"version"}, strings.NewReader(""), &out, &errout); code != 0 {
		t.Fatalf("version exit %d: %s", code, errout.String())
	}
	// Frozen v1.3 installer wire contract. Do not replace this with the current
	// runtime's metadata type: released updaters cannot accept additive fields.
	var released13 struct {
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
	if err := strictjson.Decode(out.Bytes(), &released13, 16<<10); err != nil {
		t.Fatalf("released v1.3 installer cannot decode actual version output: %v", err)
	}
	if !released13.OK || released13.Protocol != 1 || released13.Result.Name != "mandalore" {
		t.Fatalf("invalid version envelope: %+v", released13)
	}
}

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
		if !ok || len(values) != 2 || values[0] != float64(1) || values[1] != float64(2) {
			t.Fatal("missing schema compatibility", r)
		}
	}
}
