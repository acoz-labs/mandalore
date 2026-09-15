package pi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestEmbeddedPackageIdentity(t *testing.T) {
	files, err := PackageFiles()
	if err != nil {
		t.Fatal(err)
	}
	info, err := Inspect()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if info.Name != "mandalore" || info.Version == "" || info.HarnessProtocol != 1 || info.SHA256 != hex.EncodeToString(digest[:]) || info.FileCount != len(files) {
		t.Fatal("unbound package identity", info)
	}
	for _, name := range []string{"index.js", "extension.js", "transport.js", "connection.js", "package.json"} {
		if len(files[name]) == 0 {
			t.Fatal("missing native package entry", name)
		}
	}
	if _, exists := files["connection.json"]; exists {
		t.Fatal("machine-local connection embedded in public runtime")
	}
}
