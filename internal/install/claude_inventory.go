package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// Native Claude owns these files. Preview reads only; apply uses its scoped CLI.
type claudeSettings struct {
	Exists  bool
	SHA256  string
	Other   map[string]any
	Root    string
	Cache   string
	Version string
	Enabled bool
}

func inspectClaudeSettings(home string) (claudeSettings, error) {
	r := claudeSettings{Other: map[string]any{}}
	identities := map[string]string{}
	for _, name := range []string{"settings.json", "plugins/known_marketplaces.json", "plugins/installed_plugins.json"} {
		raw, err := readRegular(filepath.Join(home, name), 256<<10)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return r, err
		}
		var fields map[string]any
		if strictjson.Decode(raw, &fields, 256<<10) != nil || fields == nil {
			return r, errors.New("invalid Claude native inventory")
		}
		identities[name] = hash(raw)
		r.Exists = true
		r.Other[name] = fields
	}
	raw, _ := json.Marshal(identities)
	r.SHA256 = hash(raw)
	settings, _ := r.Other["settings.json"].(map[string]any)
	marketplaces, _ := r.Other["plugins/known_marketplaces.json"].(map[string]any)
	installed, _ := r.Other["plugins/installed_plugins.json"].(map[string]any)
	getMap := func(fields map[string]any, key string) (map[string]any, error) {
		v, exists := fields[key]
		if !exists {
			return map[string]any{}, nil
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, errors.New("unsupported Claude inventory field")
		}
		return m, nil
	}
	enabled, err := getMap(settings, "enabledPlugins")
	if err != nil {
		return r, err
	}
	extra, err := getMap(settings, "extraKnownMarketplaces")
	if err != nil {
		return r, err
	}
	plugins, err := getMap(installed, "plugins")
	if err != nil {
		return r, err
	}
	if installed != nil && installed["version"] != float64(2) {
		return r, errors.New("unsupported Claude installed inventory version")
	}
	for _, fields := range []map[string]any{enabled, plugins} {
		for id := range fields {
			if (strings.HasPrefix(id, "mandalore@") || strings.HasPrefix(id, "my-friday")) && id != "mandalore@mandalore" {
				return r, errors.New("foreign Claude memory registration requires explicit review")
			}
		}
	}
	if value, ok := enabled["mandalore@mandalore"]; ok {
		var valid bool
		r.Enabled, valid = value.(bool)
		if !valid {
			return r, errors.New("invalid Claude enabled registration")
		}
	}
	sourcePath := func(value any) (string, error) {
		item, ok := value.(map[string]any)
		if !ok {
			return "", errors.New("invalid Claude marketplace entry")
		}
		source, ok := item["source"].(map[string]any)
		if !ok || source["source"] != "directory" {
			return "", errors.New("foreign Claude memory marketplace")
		}
		path, ok := source["path"].(string)
		if !ok || !filepath.IsAbs(path) {
			return "", errors.New("invalid Claude marketplace source")
		}
		if location, ok := item["installLocation"]; ok && location != path {
			return "", errors.New("redirected Claude marketplace")
		}
		return path, nil
	}
	if value, ok := marketplaces["mandalore"]; ok {
		r.Root, err = sourcePath(value)
		if err != nil {
			return r, err
		}
	}
	if value, ok := extra["mandalore"]; ok {
		path, e := sourcePath(value)
		if e != nil {
			return r, e
		}
		if r.Root != "" && path != r.Root {
			return r, errors.New("ambiguous Claude marketplace sources")
		}
		r.Root = path
	}
	if value, ok := plugins["mandalore@mandalore"]; ok {
		entries, ok := value.([]any)
		if !ok || len(entries) != 1 {
			return r, errors.New("ambiguous Claude memory installations")
		}
		item, ok := entries[0].(map[string]any)
		if !ok || item["scope"] != "user" {
			return r, errors.New("non-user Claude memory installation requires explicit review")
		}
		r.Cache, _ = item["installPath"].(string)
		r.Version, _ = item["version"].(string)
		if r.Cache == "" || r.Version == "" || r.Root == "" {
			return r, errors.New("incomplete Claude memory registration")
		}
	}
	if _, ok := enabled["mandalore@mandalore"]; ok && r.Root == "" {
		return r, errors.New("orphan Claude memory enablement")
	}
	return r, nil
}

func claudeSelectedRegistration(s claudeSettings, state string) (string, error) {
	if s.Root == "" {
		return "", nil
	}
	if filepath.Dir(s.Root) != filepath.Join(state, "claude-code", "connections") {
		return "", errors.New("Claude memory marketplace is not owned by this installation")
	}
	return s.Root, nil
}

func claudeSettingsPreserved(before, after claudeSettings, _, _ string) bool {
	encode := func(s claudeSettings) []byte {
		raw, _ := json.Marshal(s.Other)
		var fields map[string]map[string]any
		_ = json.Unmarshal(raw, &fields)
		for name, object := range fields {
			switch name {
			case "settings.json":
				for _, key := range []string{"enabledPlugins", "extraKnownMarketplaces"} {
					if values, ok := object[key].(map[string]any); ok {
						if key == "enabledPlugins" {
							delete(values, "mandalore@mandalore")
						} else {
							delete(values, "mandalore")
						}
						if len(values) == 0 {
							delete(object, key)
						}
					}
				}
			case "plugins/known_marketplaces.json":
				delete(object, "mandalore")
			case "plugins/installed_plugins.json":
				if values, ok := object["plugins"].(map[string]any); ok {
					delete(values, "mandalore@mandalore")
					if len(values) == 0 {
						delete(object, "plugins")
						delete(object, "version")
					}
				}
			}
			if len(object) == 0 {
				delete(fields, name)
			}
		}
		raw, _ = json.Marshal(fields)
		return raw
	}
	return bytes.Equal(encode(before), encode(after))
}

func verifyClaudeCache(s claudeSettings, p ClaudePlan, allowMissing bool) error {
	if s.Root != p.Root || s.Version != p.nativeVersion() || s.Cache != filepath.Join(p.NativeHome, "plugins", "cache", "mandalore", "mandalore", p.cacheVersion()) {
		return errors.New("Claude cache registration differs from owned connection")
	}
	r, err := loadClaudeReceipt(p.Root)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for name, value := range r.Files {
		if strings.HasPrefix(name, "package/") {
			files[strings.TrimPrefix(name, "package/")] = value
		}
	}
	if real, err := canonical(s.Cache); err != nil || real != s.Cache {
		return errors.New("Claude cache is redirected")
	}
	if _, err := os.Lstat(s.Cache); allowMissing && os.IsNotExist(err) {
		return nil
	}
	return verifyTree(s.Cache, files, allowMissing)
}
