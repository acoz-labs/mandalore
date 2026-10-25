package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/acoz-labs/mandalore/internal/api"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func cli(t *testing.T, args []string, input string) (api.Envelope, int) {
	t.Helper()
	var out, errout bytes.Buffer
	code := run(context.Background(), args, strings.NewReader(input), &out, &errout)
	var v api.Envelope
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("invalid output: %s / %s: %v", out.String(), errout.String(), err)
	}
	return v, code
}

func TestCLICreateBindRememberRecallAndDiscovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "signet")
	binding := filepath.Join(t.TempDir(), "binding.json")
	for _, args := range [][]string{
		{"signet", "create", "--repository", root, "--name", "Example", "--device-label", "Test"},
		{"signet", "bind", "--repository", root, "--binding", binding, "--device-label", "Test bound", "--actor", "Example"},
	} {
		if out, code := cli(t, args, ""); code != 0 || !out.OK {
			t.Fatal(out, code)
		}
	}
	t.Chdir(t.TempDir())
	out, code := cli(t, []string{"memory", "remember", "--binding", binding}, `{"kind":"fact","summary":"Project","body":"Copper Finch","basis":"user-direction","reason":"Confirmed"}`)
	if code != 0 || !out.OK {
		t.Fatal(out, code)
	}
	out, code = cli(t, []string{"memory", "recall", "--binding", binding, "--query", "Copper"}, "")
	if code != 0 || !out.OK {
		t.Fatal(out, code)
	}
	data, _ := json.Marshal(out.Result)
	if !strings.Contains(string(data), "Copper Finch") {
		t.Fatal(out)
	}
	if out, code := cli(t, []string{"operations"}, ""); code != 0 || !out.OK {
		t.Fatal(out, code)
	}
	out, code = cli(t, []string{"call", "memory_remember", "--binding", binding, "--read-only"}, `{}`)
	if code != 2 || out.Error.Code != "operation.read_only" {
		t.Fatal(out, code)
	}
}

func TestCLIUsageDoesNotFallThrough(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"call", "mcp"}, {"memory", "recall", "extra"}, {"memory", "recall", "--scope-id", "alone"}, {"call", "memory_recall", "--unknown"}} {
		out, code := cli(t, args, `{}`)
		if code != 2 || out.OK {
			t.Fatal(args, out, code)
		}
	}
}

func TestCLIVisibilityAndUpgradeReadOnlyBeforeInput(t *testing.T) {
	for _, args := range [][]string{
		{"memory", "withdraw", "--read-only"},
		{"memory", "restore", "--read-only"},
		{"call", "signet_upgrade_apply", "--read-only"},
		{"call", "signet_upgrade_recover", "--read-only"},
	} {
		out, code := cli(t, args, "invalid")
		if code != 2 || out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
			t.Fatal(args, out, code)
		}
	}
	out, code := cli(t, []string{"call", "signet_upgrade_apply", "--binding", "/synthetic/ignored.json"}, "{}")
	if code != 2 || out.OK || out.Error.Code != "input.invalid" {
		t.Fatal("silently ignored binding flag", out, code)
	}
}

func TestVersionDeclaresMachineCompatibility(t *testing.T) {
	v, code := cli(t, []string{"version"}, "")
	if code != 0 || !v.OK {
		t.Fatal(v, code)
	}
	r, ok := v.Result.(map[string]any)
	if !ok || r["os"] != runtime.GOOS || r["arch"] != runtime.GOARCH {
		t.Fatal("missing machine compatibility", r)
	}
}

// 2.0 serves agents only through `mandalore serve`; the 1.x stdio server,
// lifecycle hooks and Razor Crest are gone from the build.
func TestOneXAgentCommandsAreRemoved(t *testing.T) {
	for _, command := range []string{"mcp", "codex-memory-hook", "claude-code-memory-hook", "razor-crest"} {
		var out, errout bytes.Buffer
		code := run(context.Background(), []string{command, "--binding", filepath.Join(t.TempDir(), "missing.json")}, strings.NewReader(""), &out, &errout)
		var failure api.Envelope
		if code != 2 || json.Unmarshal(out.Bytes(), &failure) != nil || failure.Error == nil || failure.Error.Code != "operation.unknown" {
			t.Fatalf("%s: %d %s %s", command, code, out.String(), errout.String())
		}
	}
}

type forbiddenRead struct{ t *testing.T }

func (r forbiddenRead) Read([]byte) (int, error) { r.t.Fatal("unexpected stdin read"); return 0, nil }
func TestReadOnlyMutationDoesNotWaitForInput(t *testing.T) {
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"call", "signet_create", "--read-only"}, forbiddenRead{t}, &out, &errout); code != 2 {
		t.Fatal(code)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestBrokenOutputIsNotSuccess(t *testing.T) {
	if code := run(context.Background(), []string{"version"}, strings.NewReader(""), brokenWriter{}, io.Discard); code == 0 {
		t.Fatal("broken stdout reported success")
	}
}
