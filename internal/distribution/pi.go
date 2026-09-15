package distribution

import (
	"encoding/json"
	"errors"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// preparePiManifest stamps only the builder's isolated source export, before
// compilation embeds it. Pi is not an extra format-1 release asset and does not
// extend the generic version response consumed by existing bootstrap clients.
func preparePiManifest(raw []byte, version string) ([]byte, error) {
	if !ValidVersion(version) {
		return nil, errors.New("invalid Pi release version")
	}
	var manifest map[string]any
	if err := strictjson.Decode(raw, &manifest, 32768); err != nil {
		return nil, err
	}
	name, _ := manifest["name"].(string)
	previous, _ := manifest["version"].(string)
	nativeRaw, err := json.Marshal(manifest["pi"])
	if err != nil {
		return nil, err
	}
	var native struct{ Extensions, Skills []string }
	if name != "mandalore" || previous == "" ||
		json.Unmarshal(nativeRaw, &native) != nil ||
		len(native.Extensions) != 1 || native.Extensions[0] != "./index.js" ||
		len(native.Skills) != 1 || native.Skills[0] != "./skills" {
		return nil, errors.New("invalid Pi package identity or native resources")
	}
	manifest["version"] = version
	stamped, err := json.MarshalIndent(manifest, "", "  ")
	return append(stamped, '\n'), err
}
