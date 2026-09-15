// Package pi embeds the native extension, not a second memory engine. Local
// connection context is materialized only by explicit machine installation.
package pi

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
)

//go:embed all:package
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
		if path == "connection.json" || strings.HasSuffix(path, "/connection.json") {
			return errors.New("machine-local context cannot be embedded in the public Pi package")
		}
		raw, err := fs.ReadFile(root, path)
		if err != nil {
			return err
		}
		total += len(raw)
		if total > 1<<20 || len(result) >= 128 {
			return errors.New("embedded Pi package exceeded its file or byte limit")
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
	var manifest struct {
		Name, Version string
		Pi            struct{ Extensions []string }
	}
	if err := json.Unmarshal(content["package.json"], &manifest); err != nil || manifest.Name != "mandalore" || manifest.Version == "" || len(manifest.Version) > 128 || len(manifest.Pi.Extensions) != 1 || manifest.Pi.Extensions[0] != "./index.js" || len(content["index.js"]) == 0 {
		return Info{}, errors.New("invalid embedded Pi package manifest or entrypoint")
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return Info{}, err
	}
	digest := sha256.Sum256(raw)
	return Info{Name: manifest.Name, Version: manifest.Version, SHA256: hex.EncodeToString(digest[:]), HarnessProtocol: 1, FileCount: len(content)}, nil
}
