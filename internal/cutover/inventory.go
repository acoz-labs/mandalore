package cutover

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/strictjson"
	"gopkg.in/yaml.v3"
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

// Hermes owns config.yaml; mutations use its native enable/disable commands.
// Preview observes bounded YAML and the exact managed symlink, without executing
// native code or rewriting the user's configuration.
type hermesSettings struct {
	Exists  bool
	SHA256  string
	Root    string
	Enabled bool
	Other   map[string]any
}

func inspectHermesSettings(home string) (hermesSettings, error) {
	s := hermesSettings{Other: map[string]any{}}
	if err := inspectHermesAliases(filepath.Join(home, "plugins")); err != nil {
		return s, err
	}
	path := filepath.Join(home, "plugins", "mandalore")
	st, err := os.Lstat(path)
	if err == nil {
		if st.Mode()&os.ModeSymlink == 0 {
			return s, errors.New("foreign Hermes Mandalore plugin; preserve it for explicit review")
		}
		target, err := os.Readlink(path)
		if err != nil || !filepath.IsAbs(target) || filepath.Base(target) != "package" {
			return s, errors.New("invalid Hermes registration target")
		}
		s.Root = filepath.Dir(target)
	} else if !os.IsNotExist(err) {
		return s, err
	}
	raw, err := readRegular(filepath.Join(home, "config.yaml"), 256<<10)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&s.Other); err != nil || s.Other == nil {
		return s, errors.New("invalid Hermes profile YAML")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return s, errors.New("Hermes configuration requires one YAML document")
	}
	s.Exists, s.SHA256 = true, hash(raw)
	plugins, exists := s.Other["plugins"]
	if !exists {
		return s, nil
	}
	fields, ok := plugins.(map[string]any)
	if !ok {
		return s, errors.New("invalid Hermes plugin configuration")
	}
	enabled, disabled := false, false
	for _, key := range []string{"enabled", "disabled"} {
		value, exists := fields[key]
		if !exists {
			continue
		}
		list, ok := value.([]any)
		if !ok || len(list) > 256 {
			return s, errors.New("invalid Hermes plugin selection")
		}
		other := []string{}
		for _, item := range list {
			name, ok := item.(string)
			if !ok {
				return s, errors.New("invalid Hermes plugin selection entry")
			}
			if name == "mandalore" {
				if key == "enabled" {
					enabled = true
				} else {
					disabled = true
				}
			} else {
				other = append(other, name)
			}
		}
		sort.Strings(other)
		delete(fields, key)
		if len(other) > 0 {
			fields[key] = other
		}
	}
	s.Enabled = enabled && !disabled
	if len(fields) == 0 {
		delete(s.Other, "plugins")
	}
	return s, nil
}

// Match the native flat/category discovery depth. A differently named directory
// can still claim Mandalore's manifest name and shadow its tool/skill identity.
func inspectHermesAliases(root string) error {
	entries, total := 0, 0
	var walk func(string, int) error
	walk = func(directory string, depth int) error {
		children, err := os.ReadDir(directory)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, child := range children {
			entries++
			if entries > 256 {
				return errors.New("Hermes plugin inventory exceeds its bound")
			}
			name := child.Name()
			if strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") || depth == 0 && name == "mandalore" {
				continue
			}
			path := filepath.Join(directory, name)
			st, err := os.Stat(path)
			if err != nil {
				return errors.New("unreadable Hermes plugin inventory; inspect before connecting")
			}
			if !st.IsDir() {
				continue
			}
			found := false
			for _, manifest := range []string{"plugin.yaml", "plugin.yml", "plugin.json"} {
				raw, err := readRegular(filepath.Join(path, manifest), 32768)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return errors.New("unreadable Hermes plugin manifest")
				}
				found = true
				total += len(raw)
				if total > 1<<20 {
					return errors.New("Hermes manifest inventory exceeds its bound")
				}
				var value map[string]any
				if yaml.Unmarshal(raw, &value) != nil {
					return errors.New("invalid Hermes plugin manifest")
				}
				if value["name"] == "mandalore" {
					return errors.New("another Hermes plugin claims Mandalore; preserve it for explicit review")
				}
				break
			}
			if !found && depth == 0 {
				if err := walk(path, 1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(root, 0)
}

type MemoryPlugin struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
}

type MemoryInventory struct {
	Status           string         `json:"status"`
	BinarySHA256     string         `json:"binary_sha256,omitempty"`
	Plugins          []MemoryPlugin `json:"plugins"`
	PotentialWriters int            `json:"potential_writers"`
	Notice           string         `json:"notice,omitempty"`
}

// InspectMemoryPlugins invokes only the selected native CLI's listing commands.
// Installation is not proof of an active session or the bank a plugin targets.
// Native logging/caching is owned by the selected CLI, not edited by Mandalore.
func InspectMemoryPlugins(ctx context.Context, home, binary string) (MemoryInventory, error) {
	return inspectMemoryPlugins(ctx, home, binary, native)
}

func inspectMemoryPlugins(ctx context.Context, home, binary string, run runner) (MemoryInventory, error) {
	result := MemoryInventory{Status: "not-tested", Plugins: []MemoryPlugin{}}
	var err error
	home, err = canonical(home)
	if err != nil {
		return result, err
	}
	binary, err = canonical(binary)
	if err != nil {
		return result, err
	}
	if st, err := os.Lstat(home); err != nil || !st.IsDir() {
		return result, errors.New("native inventory requires an existing explicit profile directory")
	}
	before, err := digestLimit(binary, maxNativeBinary)
	if err != nil {
		return result, err
	}
	_, plugins, err := inventory(ctx, Options{NativeHome: home, NativeBinary: binary, Binding: filepath.Join(home, "inventory-context")}, run)
	if err != nil {
		return result, err
	}
	for _, p := range plugins {
		if p.Name != "my-friday" && p.Name != "my-friday-memory" && p.Name != "mandalore" {
			continue
		}
		if len(result.Plugins) >= 32 || len(p.ID) > 512 || len(p.Version) > 256 {
			return result, errors.New("memory plugin inventory exceeds bounded report limits")
		}
		result.Plugins = append(result.Plugins, MemoryPlugin{ID: p.ID, Name: p.Name, Version: p.Version, Enabled: p.Enabled})
		if p.Enabled {
			result.PotentialWriters++
		}
	}
	sort.Slice(result.Plugins, func(i, j int) bool {
		a, b := result.Plugins[i], result.Plugins[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return !a.Enabled && b.Enabled
	})
	after, err := digestLimit(binary, maxNativeBinary)
	if err != nil || before != after {
		return result, errors.New("native binary changed during inventory")
	}
	result.Status, result.BinarySHA256 = "inspected", before
	result.Notice = "Enabled memory integrations are potential writers only. Active sessions, same-bank targeting, other machines and authentication are not tested. Listing may produce native-owned logs or caches."
	return result, nil
}

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
