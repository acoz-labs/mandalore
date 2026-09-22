package readiness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var ErrSelection = errors.New("invalid readiness selection; use codex, pi or claude-code and absolute paths without control characters, at most 4096 bytes")

type Input struct {
	Harness        string `json:"harness"`
	StateDir       string `json:"state_dir,omitempty"`
	NativeHome     string `json:"native_home,omitempty"`
	NativeBinary   string `json:"native_binary,omitempty"`
	Binding        string `json:"binding,omitempty"`
	ConnectionRoot string `json:"connection_root,omitempty"`
	IncludePrompt  bool   `json:"include_prompt,omitempty"`
}

type SelectedPath struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

type Selection struct {
	StateDir       SelectedPath `json:"state_dir"`
	NativeHome     SelectedPath `json:"native_home"`
	NativeBinary   SelectedPath `json:"native_binary"`
	Binding        SelectedPath `json:"binding"`
	ConnectionRoot SelectedPath `json:"connection_root"`
}

func validPath(path string) bool {
	return filepath.IsAbs(path) && len(path) <= 4096 && strings.IndexFunc(path, unicode.IsControl) < 0
}

func resolveSelection(in Input) (Selection, error) {
	if in.Harness != "codex" && in.Harness != "pi" && in.Harness != "claude-code" {
		return Selection{}, ErrSelection
	}
	s := Selection{StateDir: SelectedPath{in.StateDir, "explicit"}, NativeHome: SelectedPath{in.NativeHome, "explicit"}, NativeBinary: SelectedPath{in.NativeBinary, "explicit"}, Binding: SelectedPath{in.Binding, "explicit"}, ConnectionRoot: SelectedPath{in.ConnectionRoot, "explicit"}}
	if in.StateDir == "" {
		base, err := os.UserConfigDir()
		if err != nil || !validPath(base) {
			return Selection{}, ErrSelection
		}
		s.StateDir = SelectedPath{filepath.Join(base, "mandalore", "installation"), "platform-default"}
	}
	if in.Binding == "" {
		s.Binding = SelectedPath{os.Getenv("MANDALORE_BINDING"), "environment"}
		if s.Binding.Path == "" {
			base, err := os.UserConfigDir()
			if err != nil || !validPath(base) {
				return Selection{}, ErrSelection
			}
			s.Binding = SelectedPath{filepath.Join(base, "mandalore", "binding.json"), "platform-default"}
		}
	}
	if in.NativeHome == "" {
		key := "CODEX_HOME"
		if in.Harness == "pi" {
			key = "PI_CODING_AGENT_DIR"
		} else if in.Harness == "claude-code" {
			key = "CLAUDE_CONFIG_DIR"
		}
		s.NativeHome = SelectedPath{os.Getenv(key), "environment"}
		if s.NativeHome.Path == "" {
			home, err := os.UserHomeDir()
			if err != nil || !validPath(home) {
				return Selection{}, ErrSelection
			}
			path := filepath.Join(home, ".codex")
			if in.Harness == "pi" {
				path = filepath.Join(home, ".pi", "agent")
			} else if in.Harness == "claude-code" {
				path = filepath.Join(home, ".claude")
			}
			s.NativeHome = SelectedPath{path, "home-default"}
		}
	}
	if in.NativeBinary == "" {
		executable := in.Harness
		if executable == "claude-code" {
			executable = "claude"
		}
		s.NativeBinary = executableOnPath(executable)
	}
	if in.ConnectionRoot == "" {
		s.ConnectionRoot.Source = "unselected"
	}
	for _, p := range []*SelectedPath{&s.StateDir, &s.NativeHome, &s.NativeBinary, &s.Binding, &s.ConnectionRoot} {
		if p.Path == "" && (p == &s.NativeBinary || p == &s.ConnectionRoot) {
			continue
		}
		if !validPath(p.Path) {
			return Selection{}, ErrSelection
		}
		p.Path = filepath.Clean(p.Path)
	}
	return s, nil
}

// ResolveSelection previews the same bounded defaults used by Assess without
// reading binding, receipt or memory metadata. It never creates selected paths.
func ResolveSelection(in Input) (Selection, error) { return resolveSelection(in) }

// Non-executing, bounded PATH lookup. Relative entries (including cwd) are not
// trusted installation defaults. Excessive PATH input yields unknown, not a
// false claim that a dependency is absent.
func executableOnPath(name string) SelectedPath {
	value := os.Getenv("PATH")
	if len(value) > 64<<10 {
		return SelectedPath{Source: "path-unresolved"}
	}
	entries := filepath.SplitList(value)
	if len(entries) > 256 {
		return SelectedPath{Source: "path-unresolved"}
	}
	for _, dir := range entries {
		if !validPath(dir) {
			return SelectedPath{Source: "path-unresolved"}
		}
		path := filepath.Join(dir, name)
		if !validPath(path) {
			continue
		}
		st, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return SelectedPath{Source: "path-unresolved"}
		}
		if err == nil && st.Mode().IsRegular() && st.Mode()&0111 != 0 {
			return SelectedPath{Path: path, Source: "path"}
		}
	}
	return SelectedPath{Source: "path-missing"}
}
