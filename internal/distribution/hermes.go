package distribution

import (
	"encoding/json"
	"errors"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// Hermes is embedded in each CLI, so the format-1 asset contract stays intact.
func prepareHermesManifest(raw []byte, version string) ([]byte, error) {
	if !ValidVersion(version) {
		return nil, errors.New("invalid Hermes release version")
	}
	var manifest map[string]any
	if err := strictjson.Decode(raw, &manifest, 32768); err != nil {
		return nil, err
	}
	if manifest["name"] != "mandalore" || manifest["version"] == "" || manifest["version"] == nil {
		return nil, errors.New("invalid Hermes package identity")
	}
	manifest["version"] = version
	stamped, err := json.MarshalIndent(manifest, "", "  ")
	return append(stamped, '\n'), err
}
