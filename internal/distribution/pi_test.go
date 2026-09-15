package distribution

import (
	"bytes"
	"encoding/json"
	"testing"

	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

func TestPreparePiManifestPreservesPackageAndSource(t *testing.T) {
	files, err := piplugin.PackageFiles()
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), files["package.json"]...)
	stamped, err := preparePiManifest(before, "1.2.3")
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
	if !bytes.Equal(a, b) || !bytes.Equal(before, files["package.json"]) {
		t.Fatal("stamping changed package contract or source")
	}
	if _, err := preparePiManifest(before, "not a release"); err == nil {
		t.Fatal("invalid version accepted")
	}
}

func TestPreparePiManifestRejectsInvalidSource(t *testing.T) {
	for _, raw := range []string{
		`{}`, `{"name":"foreign","version":"dev","pi":{"extensions":["./index.js"],"skills":["./skills"]}}`,
		`{"name":"mandalore","name":"mandalore","version":"dev","pi":{"extensions":["./index.js"],"skills":["./skills"]}}`,
		`{"name":"mandalore","version":"dev","pi":{"extensions":["./wrong.js"],"skills":["./skills"]}}`,
		`{"name":"mandalore","version":"dev","pi":{"extensions":["./index.js"]}}`,
	} {
		if _, err := preparePiManifest([]byte(raw), "1.2.3"); err == nil {
			t.Fatal("invalid package accepted", raw)
		}
	}
}
