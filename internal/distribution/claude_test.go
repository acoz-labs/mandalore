package distribution

import (
	"bytes"
	"encoding/json"
	"testing"

	claudeplugin "github.com/acoz-labs/mandalore/plugins/claude-code"
)

func TestPrepareClaudeManifestPreservesPackageAndSource(t *testing.T) {
	files, err := claudeplugin.PackageFiles()
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), files[".claude-plugin/plugin.json"]...)
	stamped, err := prepareClaudeManifest(before, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	var original, result map[string]any
	if err := json.Unmarshal(before, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stamped, &result); err != nil {
		t.Fatal(err)
	}
	if result["version"] != "1.2.3" {
		t.Fatal("release version not stamped")
	}
	result["version"] = original["version"]
	a, _ := json.Marshal(original)
	b, _ := json.Marshal(result)
	if !bytes.Equal(a, b) || !bytes.Equal(before, files[".claude-plugin/plugin.json"]) {
		t.Fatal("stamping changed package contract or source")
	}
	if _, err := prepareClaudeManifest(before, "not a release"); err == nil {
		t.Fatal("invalid version accepted")
	}
}

func TestPrepareClaudeManifestRejectsInvalidSource(t *testing.T) {
	for _, raw := range []string{`{}`, `{"name":"foreign","version":"dev"}`, `{"name":"mandalore","name":"mandalore","version":"dev"}`, `{"name":"mandalore","version":4}`} {
		if _, err := prepareClaudeManifest([]byte(raw), "1.2.3"); err == nil {
			t.Fatal("invalid package accepted", raw)
		}
	}
}
