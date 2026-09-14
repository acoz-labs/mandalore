package codex

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBridgeFailureAndRouting(t *testing.T) {
	bridge, err := filepath.Abs("../../plugins/codex/plugins/mandalore/scripts/run-memory.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, binary, body, mode, want string
		fail                           bool
	}{
		{"missing", "/missing-mandalore", "", "hook", "systemMessage", false},
		{"relative", "./mandalore", "", "hook", "absolute", false},
		{"old", "", "printf SECRET_CANARY; printf SECRET_CANARY >&2; exit 2", "hook", "systemMessage", false},
		{"mcp-relative", "./mandalore", "", "mcp", "absolute", true},
		{"hook-route", "", "test \"$1\" = codex-memory-hook || exit 3; test \"$MANDALORE_BINDING\" = /fixture/binding.json || exit 4; printf '{}\\n'", "hook", "{}", false},
		{"mcp-route", "", "test \"$1 $2 $3\" = 'mcp --harness codex' || exit 3; test \"$MANDALORE_BINDING\" = /fixture/binding.json || exit 4; printf mcp-routed", "mcp", "mcp-routed", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			binary := tc.binary
			if tc.body != "" {
				binary = filepath.Join(dir, "runtime with spaces")
				if err := os.WriteFile(binary, []byte("#!/bin/sh\n"+tc.body+"\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("/bin/sh", bridge, tc.mode)
			cmd.Dir = dir
			for _, e := range os.Environ() {
				if !strings.HasPrefix(e, "MANDALORE_BIN=") && !strings.HasPrefix(e, "MANDALORE_BINDING=") {
					cmd.Env = append(cmd.Env, e)
				}
			}
			cmd.Env = append(cmd.Env, "MANDALORE_BIN="+binary, "MANDALORE_BINDING=/fixture/binding.json")
			out, err := cmd.CombinedOutput()
			if (err != nil) != tc.fail || !strings.Contains(string(out), tc.want) || strings.Contains(string(out), "SECRET_CANARY") {
				t.Fatalf("%s: %v", out, err)
			}
		})
	}
}
