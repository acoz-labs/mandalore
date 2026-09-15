package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// Pi owns settings.json. Mandalore observes bounded structured data, but all
// registration changes go through native install/remove, never a JSON rewrite.
type piRegistration struct {
	Source   string
	Path     string // Empty for remote packages; inspection never downloads them.
	Filtered bool
	Value    any
}

type piSettings struct {
	Exists   bool
	SHA256   string
	Other    map[string]any
	Packages []piRegistration
}

func piSourcePath(home, source string) (string, error) {
	value := strings.TrimSpace(source)
	if value == "" || len(value) > 4096 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", errors.New("invalid Pi package source")
	}
	for _, prefix := range []string{"npm:", "git:", "github:", "http:", "https:", "ssh:"} {
		if strings.HasPrefix(value, prefix) {
			return "", nil
		}
	}
	if strings.HasPrefix(value, "file://") {
		u, err := url.Parse(value)
		if err != nil || (u.Host != "" && u.Host != "localhost") || !filepath.IsAbs(u.Path) {
			return "", errors.New("unsupported Pi local file URL")
		}
		value = u.Path
	}
	if value == "~" || strings.HasPrefix(value, "~/") {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		value = filepath.Join(userHome, strings.TrimPrefix(value, "~"))
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(home, value)
	}
	return filepath.Clean(value), nil
}

func inspectPiSettings(home string) (piSettings, error) {
	r := piSettings{Other: map[string]any{}, Packages: []piRegistration{}}
	raw, err := readRegular(filepath.Join(home, "settings.json"), 256<<10)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return r, err
	}
	var fields map[string]any
	if err := strictjson.Decode(raw, &fields, 256<<10); err != nil {
		return r, errors.New("invalid Pi profile settings; no native changes attempted")
	}
	r.Exists, r.SHA256 = true, hash(raw)
	list := []any{}
	if value, present := fields["packages"]; present {
		var ok bool
		list, ok = value.([]any)
		if !ok || len(list) > 256 {
			return r, errors.New("unsupported Pi package inventory")
		}
	}
	delete(fields, "packages")
	r.Other = fields
	seen := map[string]bool{}
	for _, value := range list {
		item := piRegistration{Value: value}
		switch v := value.(type) {
		case string:
			item.Source = v
		case map[string]any:
			var ok bool
			item.Source, ok = v["source"].(string)
			if !ok {
				return r, errors.New("Pi package source is missing")
			}
			item.Filtered = len(v) > 1
			for key, filter := range v {
				if key == "source" {
					continue
				}
				if key != "extensions" && key != "skills" && key != "prompts" && key != "themes" {
					return r, errors.New("unsupported Pi resource filter field")
				}
				patterns, ok := filter.([]any)
				if !ok || len(patterns) > 256 {
					return r, errors.New("invalid Pi resource filter")
				}
				for _, pattern := range patterns {
					if _, ok := pattern.(string); !ok {
						return r, errors.New("invalid Pi resource filter pattern")
					}
				}
			}
		default:
			return r, errors.New("unsupported Pi package entry")
		}
		item.Path, err = piSourcePath(home, item.Source)
		if err != nil {
			return r, err
		}
		key := "remote:" + item.Source
		if item.Path != "" {
			key = "local:" + item.Path
		}
		if seen[key] {
			return r, errors.New("ambiguous duplicate Pi package registration")
		}
		seen[key] = true
		r.Packages = append(r.Packages, item)
	}
	return r, nil
}

// Compare semantic settings because Pi may reformat JSON. Preserve every
// unrelated package, its resource filters and order, and all unrelated keys.
func piSettingsPreserved(before, after piSettings, removed, added string) bool {
	encode := func(s piSettings, selected string) []byte {
		other := make(map[string]any, len(s.Other)+1)
		for k, v := range s.Other {
			other[k] = v
		}
		packages := []any{}
		for _, p := range s.Packages {
			if selected != "" && p.Path == selected {
				continue
			}
			packages = append(packages, p.Value)
		}
		other["packages"] = packages
		raw, _ := json.Marshal(other)
		return raw
	}
	return bytes.Equal(encode(before, removed), encode(after, added))
}
