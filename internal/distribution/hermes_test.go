package distribution

import (
	"encoding/json"
	"testing"

	hermesplugin "github.com/acoz-labs/mandalore/plugins/hermes"
)

func TestHermesReleaseStampPreservesNativeContract(t *testing.T) {
	files, err := hermesplugin.PackageFiles()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := prepareHermesManifest(files["plugin.yaml"], "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]any
	if json.Unmarshal(files["plugin.yaml"], &before) != nil || json.Unmarshal(raw, &after) != nil {
		t.Fatal("invalid manifest")
	}
	if after["version"] != "1.2.3" || before["version"] == "1.2.3" {
		t.Fatal("release identity or source changed")
	}
	delete(before, "version")
	delete(after, "version")
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatal("native contract changed during stamping")
	}
}
