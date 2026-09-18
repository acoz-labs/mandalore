package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"runtime"

	"github.com/acoz-labs/mandalore/internal/api"
	codexplugin "github.com/acoz-labs/mandalore/plugins/codex"
)

func runtimeMetadata() (map[string]any, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(codexplugin.Files, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := codexplugin.Files.ReadFile(name)
		files[name] = b
		return err
	})
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	var plugin struct{ Name, Version string }
	if err := json.Unmarshal(files["plugins/mandalore/.codex-plugin/plugin.json"], &plugin); err != nil {
		return nil, err
	}
	return map[string]any{
		"name": "mandalore", "version": version, "source_commit": sourceCommit,
		"protocol_version": api.ProtocolVersion, "codex_hook_protocol": 1,
		"os": runtime.GOOS, "arch": runtime.GOARCH, "go_version": runtime.Version(),
		"signet_read_versions": []int{1, 2}, "signet_write_versions": []int{1, 2},
		"plugin_version": plugin.Version, "plugin_sha256": hex.EncodeToString(h[:]),
	}, nil
}
