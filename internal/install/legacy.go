package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// LegacyConnection is an enabled 1.x Mandalore registration in one native
// profile. Target is what the harness's own disable command names.
type LegacyConnection struct {
	Harness string `json:"harness"`
	Home    string `json:"native_home"`
	Binary  string `json:"native_binary"`
	Target  string `json:"target"`
}

// LegacyHarnesses are the 1.x harnesses retire can disable.
var LegacyHarnesses = []string{"claude-code", "codex", "pi", "hermes"}

// FindLegacyConnections detects enabled Mandalore registrations with the
// existing inventory code. Claude Code, Pi and Hermes are read from their
// settings files; Codex runs only its plugin listing commands.
func FindLegacyConnections(ctx context.Context, harness, home, binary string) ([]LegacyConnection, error) {
	return findLegacyConnections(ctx, harness, home, binary, native)
}

func findLegacyConnections(ctx context.Context, harness, home, binary string, codex runner) ([]LegacyConnection, error) {
	home, err := canonical(home)
	if err != nil {
		return nil, err
	}
	found := []LegacyConnection{}
	add := func(target string) { found = append(found, LegacyConnection{harness, home, binary, target}) }
	switch harness {
	case "claude-code":
		s, err := inspectClaudeSettings(home)
		if err != nil {
			return nil, err
		}
		if s.Root != "" && s.Enabled {
			add("mandalore@mandalore")
		}
	case "hermes":
		s, err := inspectHermesSettings(home)
		if err != nil {
			return nil, err
		}
		if s.Root != "" && s.Enabled {
			add("mandalore")
		}
	case "pi":
		s, err := inspectPiSettings(home)
		if err != nil {
			return nil, err
		}
		for _, p := range s.Packages {
			if p.Path == "" {
				continue
			}
			raw, err := readRegular(filepath.Join(p.Path, "package.json"), 32768)
			if err != nil {
				continue
			}
			var manifest map[string]any
			if strictjson.Decode(raw, &manifest, 32768) != nil {
				continue
			}
			if name, _ := manifest["name"].(string); name == "mandalore" || name == "my-friday" || name == "my-friday-memory" {
				add(p.Path)
			}
		}
	case "codex":
		inv, err := inspectMemoryPlugins(ctx, home, binary, codex)
		if err != nil {
			return nil, err
		}
		for _, p := range inv.Plugins {
			if _, marketplace, ok := strings.Cut(p.ID, "@"); ok && p.Enabled {
				add(marketplace)
			}
		}
	default:
		return nil, errors.New("unknown harness; use claude-code, codex, pi or hermes")
	}
	return found, nil
}

// DisableLegacyConnection runs the harness's own command to stop the
// registration. It never deletes the clone, the runtime or native data.
func DisableLegacyConnection(ctx context.Context, c LegacyConnection) error {
	o := Options{NativeHome: c.Home, NativeBinary: c.Binary, Binding: filepath.Join(c.Home, "retire")}
	var err error
	switch c.Harness {
	case "claude-code":
		_, err = nativeClaude(ctx, o, "plugin", "disable", c.Target, "--scope", "user")
	case "hermes":
		_, err = nativeHermes(ctx, o, "plugins", "disable", c.Target)
	case "pi":
		// Pi has no disable; remove unregisters the package and keeps its files.
		_, err = nativePi(ctx, o, "remove", c.Target)
	case "codex":
		// The same unregistration 1.x uses when it replaces a connection.
		_, err = native(ctx, o, "plugin", "marketplace", "remove", c.Target, "--json")
	default:
		err = errors.New("unknown harness")
	}
	return err
}
