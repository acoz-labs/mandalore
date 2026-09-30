package launch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func fixture(t *testing.T) (string, Entry, func(install.ReceiptSelection) (install.LaunchConnection, error)) {
	t.Helper()
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	root := filepath.Join(dir, "signet")
	if _, e = memory.Create(root, "Synthetic", "device-bootstrap", "Fixture"); e != nil {
		t.Fatal(e)
	}
	bp := filepath.Join(dir, "binding.json")
	b, e := binding.Bind(root, bp, "Fixture", "Fixture")
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(bp)
	hash := sha256.Sum256(raw)
	entry := Entry{Binding: bp, DefaultAgent: "pi", Agents: map[string]Agent{}}
	for _, h := range []string{"pi", "codex", "claude-code"} {
		home := filepath.Join(dir, h+" profile")
		if e = os.Mkdir(home, 0700); e != nil {
			t.Fatal(e)
		}
		entry.Agents[h] = Agent{home, filepath.Join(dir, h+" binary"), filepath.Join(dir, "state"), filepath.Join(dir, "state", h)}
	}
	inspect := func(s install.ReceiptSelection) (install.LaunchConnection, error) {
		return install.LaunchConnection{Options: install.Options{Binding: bp, StateDir: s.StateDir, NativeHome: s.NativeHome, NativeBinary: s.NativeBinary}, Root: s.Root, SignetID: b.SignetID, BindingSHA256: hex.EncodeToString(hash[:]), Runtime: filepath.Join(dir, "runtime")}, nil
	}
	return filepath.Join(dir, "launch.json"), entry, inspect
}
func TestConfigureResolveProfilesAndLiteralArguments(t *testing.T) {
	path, entry, inspect := fixture(t)
	if _, e := configure(path, "work", entry, false, inspect); e == nil {
		t.Fatal("implicit ownership accepted")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("failed configure wrote file")
	}
	c, e := configure(path, "work", entry, true, inspect)
	if e != nil {
		t.Fatal(e)
	}
	rawBefore, _ := os.ReadFile(path)
	args := []string{"--print", "hello space", "$(touch never)", "`never`", "$HOME", "quote\"here"}
	for _, h := range []string{"", "codex", "pi", "claude-code"} {
		p, e := resolve(c, "work", h, args, inspect)
		if e != nil {
			t.Fatal(e)
		}
		expected := h
		if expected == "" {
			expected = "pi"
		}
		if p.Agent != expected || !reflect.DeepEqual(p.Arguments, args) {
			t.Fatal(p)
		}
		cwd, _ := os.Getwd()
		if p.WorkingDirectory != cwd {
			t.Fatal("cwd changed")
		}
		env := p.Environment([]string{"PATH=/bin", "MANDALORE_BINDING=/foreign", "MANDALORE_BIN=/old", "MANDALORE_SECRET=remove", "CODEX_HOME=/personal", "CLAUDE_CONFIG_DIR=/personal", "PI_CODING_AGENT_DIR=/personal", "CODEX_SQLITE_HOME=/personal", "CLAUDE_CODE_SAFE_MODE=1", "CLAUDE_CODE_PLUGIN_DIRS=/personal", "CLAUDE_CODE_PLUGIN_CACHE_DIR=/personal", "CLAUDE_CODE_PLUGIN_SEED_DIR=/personal", "CLAUDE_CODE_OAUTH_TOKEN=native-owned"})
		if strings.Contains(strings.Join(env, "\n"), "personal") || strings.Contains(strings.Join(env, "\n"), "foreign") {
			t.Fatal(env)
		}
		if strings.Contains(strings.Join(env, "\n"), "CLAUDE_CODE_SAFE_MODE=") {
			t.Fatal("inherited disabling flag retained", env)
		}
		if !strings.Contains(strings.Join(env, "\n"), "CLAUDE_CODE_OAUTH_TOKEN=native-owned") {
			t.Fatal("native auth removed")
		}
	}
	rawAfter, _ := os.ReadFile(path)
	if string(rawBefore) != string(rawAfter) {
		t.Fatal("preview mutated config")
	}
	if c.Entries["work"].DefaultAgent != "pi" {
		t.Fatal("override persisted")
	}
	loaded, e := Load(path)
	if e != nil || !reflect.DeepEqual(c, loaded) {
		t.Fatal("load", e)
	}
}
func TestProfileClaimsRejectRebindingAndInterruptedConfigurationCanRetry(t *testing.T) {
	path, entry, inspect := fixture(t)
	c, e := configure(path, "work", entry, true, inspect)
	if e != nil {
		t.Fatal(e)
	}
	a := entry.Agents["pi"]
	if e = checkOwner(a.NativeHome, "pi", "signet-another", false); e == nil {
		t.Fatal("foreign signet accepted")
	}
	if e = checkOwner(a.NativeHome, "codex", c.Entries["work"].SignetID, false); e == nil {
		t.Fatal("foreign harness accepted")
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if _, e = configure(path, "work", entry, true, inspect); e != nil {
		t.Fatal("idempotent profile claim", e)
	}
	changed := c.Entries["work"]
	changed.BindingSHA256 = strings.Repeat("0", 64)
	c.Entries["work"] = changed
	if _, e = resolve(c, "work", "", nil, inspect); e == nil {
		t.Fatal("changed binding accepted")
	}
}
func TestConfigurationRejectsAmbiguityAndPortableLocation(t *testing.T) {
	path, entry, inspect := fixture(t)
	raw := `{"schema_version":1,"entries":{},"entries":{}}`
	if e := os.WriteFile(path, []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(path); e == nil {
		t.Fatal("duplicate key accepted")
	}
	var b binding.Binding
	data, _ := os.ReadFile(entry.Binding)
	json.Unmarshal(data, &b)
	if _, e := configure(filepath.Join(b.Root, "launch.json"), "work", entry, true, inspect); e == nil {
		t.Fatal("portable configuration accepted")
	}
	entry.DefaultAgent = "unknown"
	if _, e := configure(path, "work", entry, true, inspect); e == nil {
		t.Fatal("unsupported default")
	}
}
func TestConcurrentSelectionsDoNotMutateGlobalEnvironment(t *testing.T) {
	path, entry, inspect := fixture(t)
	c, e := configure(path, "work", entry, true, inspect)
	if e != nil {
		t.Fatal(e)
	}
	t.Setenv("MANDALORE_BINDING", "untouched")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, e := resolve(c, "work", "pi", nil, inspect)
			if e != nil {
				t.Error(e)
				return
			}
			p.Environment(os.Environ())
		}()
	}
	wg.Wait()
	if os.Getenv("MANDALORE_BINDING") != "untouched" {
		t.Fatal("global selection mutated")
	}
}
func TestArgumentRoutingAndResumeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		h    string
		args []string
	}{
		{"codex", []string{"-c", "mcp_servers.foo={}"}}, {"codex", []string{"-cfoo=bar"}}, {"codex", []string{"--profile=personal"}},
		{"pi", []string{"--session=/personal/session.jsonl"}}, {"pi", []string{"--session-dir", "/personal"}}, {"pi", []string{"-eextension"}},
		{"pi", []string{"--fork", "/foreign/session"}}, {"pi", []string{"-ne"}}, {"pi", []string{"-ns"}}, {"pi", []string{"-np"}}, {"pi", []string{"--session-id", "../../foreign"}}, {"codex", []string{"--ignore-user-config"}}, {"codex", []string{"exec", "resume", "--ignore-user-config", "--last"}}, {"claude-code", []string{"--restricted"}}, {"claude-code", []string{"--safe-mode"}}, {"claude-code", []string{"--bare"}}, {"claude-code", []string{"--plugin-url=https://example.invalid/plugin"}}, {"claude-code", []string{"--settings=/personal/config"}}, {"claude-code", []string{"--plugin-dir", "/personal"}}, {"claude-code", []string{"--resume", "/foreign/session"}},
		{"codex", []string{"resume", "foreign-name"}},
	} {
		if e := ValidateArguments(tc.h, tc.args); e == nil {
			t.Errorf("accepted routing override %v", tc)
		}
	}
	for _, tc := range []struct {
		h    string
		args []string
	}{
		{"codex", []string{"resume", "00000000-0000-0000-0000-000000000001"}}, {"codex", []string{"resume", "--last"}},
		{"pi", []string{"--session-id", "00000000-0000-0000-0000-000000000001"}}, {"pi", []string{"--resume"}}, {"pi", []string{"--continue"}}, {"claude-code", []string{"--resume", "00000000-0000-0000-0000-000000000001"}},
		{"claude-code", []string{"--continue"}}, {"codex", []string{"--", "-c literal prompt"}},
	} {
		if e := ValidateArguments(tc.h, tc.args); e != nil {
			t.Errorf("rejected safe args %v: %v", tc, e)
		}
	}
}
