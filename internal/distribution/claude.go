package distribution

import (
	"encoding/json"
	"errors"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// prepareClaudeManifest stamps the isolated tracked export before compilation.
// Claude Code is embedded in the runtime; format-1's separate plugin remains Codex.
func prepareClaudeManifest(raw []byte, version string) ([]byte, error) {
	if !ValidVersion(version) {
		return nil, errors.New("invalid Claude Code release version")
	}
	var manifest map[string]any
	if err := strictjson.Decode(raw, &manifest, 32768); err != nil {
		return nil, err
	}
	name, _ := manifest["name"].(string)
	previous, _ := manifest["version"].(string)
	if name != "mandalore" || previous == "" {
		return nil, errors.New("invalid Claude Code plugin identity")
	}
	manifest["version"] = version
	stamped, err := json.MarshalIndent(manifest, "", "  ")
	return append(stamped, '\n'), err
}
