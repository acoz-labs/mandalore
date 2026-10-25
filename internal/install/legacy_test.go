package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindLegacyConnectionsPerHarness(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(base, "pi-connection", "package")
	writeFiles(t, base, map[string]string{
		"claude/settings.json":                       `{"enabledPlugins":{"mandalore@mandalore":true,"other@market":true}}`,
		"claude/plugins/known_marketplaces.json":     `{"mandalore":{"source":{"source":"directory","path":"/fixture/connection"}}}`,
		"claude-off/settings.json":                   `{"enabledPlugins":{"mandalore@mandalore":false}}`,
		"claude-off/plugins/known_marketplaces.json": `{"mandalore":{"source":{"source":"directory","path":"/fixture/connection"}}}`,
		"pi/settings.json":                           `{"packages":["` + pkg + `","npm:unrelated"]}`,
		"pi-connection/package/package.json":         `{"name":"mandalore"}`,
		"hermes/config.yaml":                         "plugins:\n  enabled: [mandalore, other]\n",
		"codex/config.toml":                          "",
		"bin/codex":                                  "#!/bin/sh\n",
		"empty/settings.json":                        `{}`,
	})
	if err := os.MkdirAll(filepath.Join(base, "hermes", "plugins"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "hermes-connection", "package"), filepath.Join(base, "hermes", "plugins", "mandalore")); err != nil {
		t.Fatal(err)
	}
	codex := func(_ context.Context, _ Options, args ...string) ([]byte, error) {
		if strings.Join(args, " ") == "plugin marketplace list --json" {
			return []byte(`{"marketplaces":[]}`), nil
		}
		return []byte(`{"installed":[{"pluginId":"mandalore@mandalore-local","name":"mandalore","version":"1.6.1","enabled":true},{"pluginId":"my-friday@mandalore-local","name":"my-friday","version":"1.0.0","enabled":true},{"pluginId":"other@x","name":"other","enabled":true}]}`), nil
	}
	for _, test := range []struct{ harness, home, target string }{
		{"claude-code", "claude", "mandalore@mandalore"},
		{"claude-code", "claude-off", ""},
		{"claude-code", "empty", ""},
		{"pi", "pi", pkg},
		{"hermes", "hermes", "mandalore"},
		{"codex", "codex", "mandalore-local"},
	} {
		found, err := findLegacyConnections(context.Background(), test.harness, filepath.Join(base, test.home), filepath.Join(base, "bin", "codex"), codex)
		if err != nil {
			t.Fatalf("%s %s: %v", test.harness, test.home, err)
		}
		if test.target == "" && len(found) != 0 || test.target != "" && (len(found) != 1 || found[0].Target != test.target) {
			t.Errorf("%s %s: found %+v, want target %q", test.harness, test.home, found, test.target)
		}
	}
	if _, err := FindLegacyConnections(context.Background(), "copilot", base, base); err == nil {
		t.Fatal("unknown harness accepted")
	}
}

// A fake harness binary records how each disable command is invoked; no real
// harness or profile is touched.
func TestDisableLegacyConnectionUsesEachHarnessCommand(t *testing.T) {
	base := t.TempDir()
	log := filepath.Join(base, "calls.log")
	binary := filepath.Join(base, "harness")
	writeFiles(t, base, map[string]string{"harness": "#!/bin/sh\necho \"$CLAUDE_CONFIG_DIR$CODEX_HOME$PI_CODING_AGENT_DIR$HERMES_HOME|$*\" >> " + log + "\n"})
	home := filepath.Join(base, "home")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	for _, c := range []LegacyConnection{
		{"claude-code", home, binary, "mandalore@mandalore"},
		{"codex", home, binary, "mandalore-local"},
		{"pi", home, binary, "/fixture/pi/package"},
		{"hermes", home, binary, "mandalore"},
	} {
		if err := DisableLegacyConnection(context.Background(), c); err != nil {
			t.Fatalf("%s: %v", c.Harness, err)
		}
	}
	raw, _ := os.ReadFile(log)
	want := home + "|plugin disable mandalore@mandalore --scope user\n" +
		home + "|plugin marketplace remove mandalore-local --json\n" +
		home + "|remove /fixture/pi/package\n" +
		home + "|--profile default plugins disable mandalore\n"
	if string(raw) != want {
		t.Fatalf("disable commands:\n%s\nwant:\n%s", raw, want)
	}
}
