package readiness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSelectionPreservesExplicitInputsWithoutCWDDiscovery(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "absent"))
	// Explicit paths must remain usable even with invalid environment defaults.
	t.Setenv("MANDALORE_BINDING", "relative")
	t.Setenv("CODEX_HOME", "relative")
	in := Input{Harness: "codex", StateDir: filepath.Join(root, "state"), NativeHome: filepath.Join(root, "profile"), NativeBinary: filepath.Join(root, "native"), Binding: filepath.Join(root, "binding")}
	s, err := resolveSelection(in)
	if err != nil || s.Binding.Path != in.Binding || s.Binding.Source != "explicit" || s.NativeHome.Path != in.NativeHome || s.ConnectionRoot.Path != "" || s.ConnectionRoot.Source != "unselected" {
		t.Fatal(s, err)
	}
	if _, err := os.Stat(in.StateDir); !os.IsNotExist(err) {
		t.Fatal("selection created state", err)
	}
}

func TestResolveSelectionDefaultsAndMissingExecutable(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "absent"))
	t.Setenv("MANDALORE_BINDING", filepath.Join(root, "binding"))
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(root, "pi-profile"))
	s, err := resolveSelection(Input{Harness: "pi", StateDir: filepath.Join(root, "state")})
	if err != nil || s.NativeBinary.Path != "" || s.NativeBinary.Source != "path-missing" || s.NativeHome.Source != "environment" || s.Binding.Source != "environment" {
		t.Fatal(s, err)
	}
}

func TestResolveSelectionRejectsInvalidInputAndDefaults(t *testing.T) {
	for _, in := range []Input{{Harness: "other"}, {Harness: "codex", Binding: "relative"}, {Harness: "pi", NativeHome: "/invalid\npath"}, {Harness: "pi", StateDir: "/" + strings.Repeat("x", 4096)}} {
		if _, err := resolveSelection(in); !errors.Is(err, ErrSelection) {
			t.Fatal("invalid selection accepted", err)
		}
	}
	t.Setenv("MANDALORE_BINDING", "relative")
	if _, err := resolveSelection(Input{Harness: "pi"}); !errors.Is(err, ErrSelection) {
		t.Fatal("invalid environment binding accepted", err)
	}
}

func TestExecutableDefaultDoesNotSilentlySkipRelativePATHPrecedence(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pi"), []byte("synthetic executable"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "relative"+string(os.PathListSeparator)+root)
	if got := executableOnPath("pi"); got.Source != "path-unresolved" || got.Path != "" {
		t.Fatal("ambiguous PATH selected a different default", got)
	}
}

func TestClaudeSelectionUsesOwnEnvironmentAndExecutable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", root)
	t.Setenv("MANDALORE_BINDING", filepath.Join(root, "binding"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(root, "claude-profile"))
	t.Setenv("CODEX_HOME", "relative")
	if err := os.WriteFile(filepath.Join(root, "claude"), []byte("never executed"), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSelection(Input{Harness: "claude-code", StateDir: filepath.Join(root, "state")})
	if err != nil || got.NativeBinary.Path != filepath.Join(root, "claude") || got.NativeHome.Path != filepath.Join(root, "claude-profile") {
		t.Fatal(got, err)
	}
	if _, err := os.Stat(got.NativeHome.Path); !os.IsNotExist(err) {
		t.Fatal("selection created profile")
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "relative")
	if _, err := resolveSelection(Input{Harness: "claude-code"}); !errors.Is(err, ErrSelection) {
		t.Fatal(err)
	}
}
