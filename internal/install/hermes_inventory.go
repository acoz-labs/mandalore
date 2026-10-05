package install

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

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

func hermesSelectedRegistration(s hermesSettings, state string) (string, error) {
	if s.Root != "" && filepath.Dir(s.Root) != filepath.Join(state, "hermes", "connections") {
		return "", errors.New("Hermes registration belongs to another installation")
	}
	return s.Root, nil
}

func hermesSettingsPreserved(before, after hermesSettings) bool {
	return reflect.DeepEqual(before.Other, after.Other)
}
