// Package hermes embeds a native general plugin, not a replacement memory provider.
package hermes

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
)

//go:embed package/*.py package/plugin.yaml package/skills
var files embed.FS

type Info struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	SHA256          string `json:"sha256"`
	HarnessProtocol int    `json:"harness_protocol_version"`
	FileCount       int    `json:"file_count"`
}

func PackageFiles() (map[string][]byte, error) {
	root, err := fs.Sub(files, "package")
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	total := 0
	err = fs.WalkDir(root, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if strings.Contains(path, "connection.json") || strings.Contains(path, "session-policy.json") {
			return errors.New("machine-local context cannot be embedded in the public Hermes package")
		}
		raw, err := fs.ReadFile(root, path)
		if err != nil {
			return err
		}
		total += len(raw)
		if total > 1<<20 || len(result) >= 128 {
			return errors.New("embedded Hermes package exceeded its file or byte limit")
		}
		result[path] = raw
		return nil
	})
	return result, err
}

func Inspect() (Info, error) {
	content, err := PackageFiles()
	if err != nil {
		return Info{}, err
	}
	var manifest struct{ Name, Version string }
	// JSON is a YAML subset accepted by Hermes, with deterministic release stamping.
	if err := json.Unmarshal(content["plugin.yaml"], &manifest); err != nil || manifest.Name != "mandalore" || manifest.Version == "" || len(manifest.Version) > 128 || len(content["__init__.py"]) == 0 {
		return Info{}, errors.New("invalid embedded Hermes manifest or entrypoint")
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return Info{}, err
	}
	digest := sha256.Sum256(raw)
	return Info{Name: manifest.Name, Version: manifest.Version, SHA256: hex.EncodeToString(digest[:]), HarnessProtocol: 1, FileCount: len(content)}, nil
}
