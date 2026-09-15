package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// This exercises the compiled executable and the actual newline stdio transport,
// not just run() or the SDK's in-memory transport. No provider/model is involved.
func TestCompiledCLIAndStdio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	base := t.TempDir()
	binary := filepath.Join(base, "mandalore")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, output)
	}
	root, binding := filepath.Join(base, "signet"), filepath.Join(base, "binding.json")
	cwd := filepath.Join(base, "unrelated-project")
	if err := os.Mkdir(cwd, 0700); err != nil {
		t.Fatal(err)
	}
	invoke := func(input string, wantCode int, args ...string) api.Envelope {
		t.Helper()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = cwd
		cmd.Env = append(os.Environ(), "MANDALORE_BINDING="+filepath.Join(base, "absent.json"))
		cmd.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != wantCode || stderr.Len() != 0 {
			t.Fatalf("exit: %v, stderr: %s", err, stderr.String())
		}
		var out api.Envelope
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatalf("JSON: %v: %s", err, stdout.String())
		}
		return out
	}
	invoke("", 0, "signet", "create", "--repository", root, "--name", "Example", "--device-label", "Test")
	invoke("", 0, "signet", "bind", "--repository", root, "--binding", binding, "--device-label", "Bound test", "--actor", "Example")
	for _, name := range []string{"git-init", "checkpoint", "sync", "sync-status"} {
		if out := invoke("", 0, "memory", name, "--binding", binding); !out.OK {
			t.Fatal(name, out.Error)
		}
	}
	remote := filepath.Join(base, "remote.git")
	for _, args := range [][]string{{"init", "--bare", "--initial-branch=main", remote}, {"-C", root, "remote", "add", "origin", remote}} {
		command := exec.CommandContext(ctx, "git", args...)
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "GIT_") {
				command.Env = append(command.Env, entry)
			}
		}
		command.Env = append(command.Env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatal(string(output), err)
		}
	}
	first := invoke(`{"kind":"fact","summary":"Project name","body":"Copper Finch","basis":"user-direction","reason":"Confirmed"}`, 0, "memory", "remember", "--binding", binding)
	data, _ := json.Marshal(first.Result)
	var receipt api.Receipt
	if err := json.Unmarshal(data, &receipt); err != nil || !receipt.DurableLocally {
		t.Fatal(receipt, err)
	}
	correction, _ := json.Marshal(map[string]any{"kind": "fact", "summary": "Project name", "body": "Silver Heron", "basis": "user-direction", "reason": "Renamed", "record_id": receipt.RecordID, "supersedes": []string{receipt.ID}})
	invoke(string(correction), 0, "call", "memory_remember", "--binding", binding)
	delivery := invoke("", 0, "memory", "sync", "--binding", binding)
	if !delivery.OK || delivery.Result.(map[string]any)["delivered"] != true {
		t.Fatal("compiled sync did not deliver", delivery)
	}
	checkCombined := func(encoded []byte) {
		t.Helper()
		var out struct {
			OK     bool                  `json:"ok"`
			Result api.SaveAndSyncResult `json:"result"`
		}
		if err := json.Unmarshal(encoded, &out); err != nil || !out.OK || !out.Result.Saved.DurableLocally || out.Result.Saved.ID == "" || !out.Result.Delivery.OK || out.Result.Delivery.Result == nil || !out.Result.Delivery.Result.Delivered {
			t.Fatalf("compiled combined receipt: %s %v", encoded, err)
		}
		for _, args := range [][]string{{"--git-dir=" + remote, "rev-parse", "main"}, {"-C", root, "rev-parse", "HEAD"}} {
			head, err := exec.CommandContext(ctx, "git", args...).Output()
			if err != nil || strings.TrimSpace(string(head)) != out.Result.Delivery.Result.Head {
				t.Fatalf("compiled combined delivery mismatch: %s %v", head, err)
			}
		}
	}
	combined := invoke(`{"entry":{"kind":"test","summary":"Compiled combined journal delivery."}}`, 0, "memory", "journal-append-and-sync", "--binding", binding)
	encoded, _ := json.Marshal(combined)
	checkCombined(encoded)
	before := treeDigest(t, root)
	// Exercise the actual native bridge against this compiled binary, from a
	// different project, with an explicit binding shared by MCP and hooks.
	bridge, err := filepath.Abs("../../plugins/codex/plugins/mandalore/scripts/run-memory.sh")
	if err != nil {
		t.Fatal(err)
	}
	hook := exec.CommandContext(ctx, "/bin/sh", bridge, "hook")
	hook.Dir = cwd
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "MANDALORE_BIN=") && !strings.HasPrefix(e, "MANDALORE_BINDING=") {
			hook.Env = append(hook.Env, e)
		}
	}
	hook.Env = append(hook.Env, "MANDALORE_BIN="+binary, "MANDALORE_BINDING="+binding)
	hook.Stdin = strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"project"}`)
	hookBytes, err := hook.CombinedOutput()
	if err != nil || !bytes.Contains(hookBytes, []byte("Silver Heron")) || bytes.Contains(hookBytes, []byte("Copper Finch")) {
		t.Fatalf("native bridge: %s: %v", hookBytes, err)
	}
	var hookOutput map[string]any
	if err := json.Unmarshal(hookBytes, &hookOutput); err != nil || hookOutput["hookSpecificOutput"] == nil {
		t.Fatal("invalid native hook output", err)
	}
	recall := invoke("", 0, "memory", "recall", "--binding", binding)
	data, _ = json.Marshal(recall.Result)
	if !bytes.Contains(data, []byte("Silver Heron")) || bytes.Contains(data, []byte("Copper Finch")) {
		t.Fatal(string(data))
	}
	invoke("", 0, "memory", "history", "--binding", binding, "--record-id", receipt.RecordID)
	refused := invoke(`{}`, 2, "call", "memory_remember", "--binding", binding, "--read-only")
	if refused.Error.Code != "operation.read_only" {
		t.Fatal(refused)
	}
	invalid := invoke(`{"query":"PRIVATE-CANARY","query":"other"}`, 2, "call", "memory_recall", "--binding", binding)
	if invalid.Error.Code != "input.invalid" || strings.Contains(invalid.Error.Message, "PRIVATE-CANARY") {
		t.Fatal(invalid)
	}
	// Explicit flag wins over an intentionally unusable environment selection.
	invoke("", 1, "memory", "recall")
	if !reflect.DeepEqual(before, treeDigest(t, root)) {
		t.Fatal("read-only CLI changed the signet")
	}

	cmd := exec.CommandContext(ctx, binary, "mcp", "--binding", binding, "--read-only", "--harness", "synthetic-mcp")
	cmd.Dir = cwd
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	client, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic", Version: "1"}, nil).Connect(ctx, &sdk.CommandTransport{Command: cmd, TerminateDuration: time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_recall", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	data, _ = json.Marshal(result.StructuredContent)
	var mcpRecall api.Envelope
	if err := json.Unmarshal(data, &mcpRecall); err != nil || !reflect.DeepEqual(recall, mcpRecall) {
		t.Fatal("CLI/MCP result mismatch", err)
	}
	result, err = client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_remember", Arguments: map[string]any{}})
	if err != nil || !result.IsError {
		t.Fatal("MCP readonly", result, err)
	}
	if err := client.Close(); err != nil {
		t.Fatal("clean EOF shutdown", err)
	}
	if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 0 || diagnostics.Len() != 0 {
		t.Fatalf("stdio shutdown: %v: %s", cmd.ProcessState, diagnostics.String())
	}
	if !reflect.DeepEqual(before, treeDigest(t, root)) {
		t.Fatal("read-only MCP changed the signet")
	}
	// A completed initialization handshake proves signal handlers are active;
	// no timing sleep or racing a newly spawned process is needed.
	interrupted := exec.CommandContext(ctx, binary, "mcp", "--binding", binding, "--read-only")
	var interruptedDiagnostics bytes.Buffer
	interrupted.Stderr = &interruptedDiagnostics
	session, err := sdk.NewClient(&sdk.Implementation{Name: "signal-test", Version: "1"}, nil).Connect(ctx, &sdk.CommandTransport{Command: interrupted, TerminateDuration: time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := interrupted.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	// Do not close stdin immediately: EOF can legitimately win the signal
	// race and exit cleanly. Observe server shutdown before client cleanup.
	_ = session.Wait()
	_ = session.Close() // The expected nonzero process exit is checked below.
	if interrupted.ProcessState == nil || interrupted.ProcessState.ExitCode() != 130 {
		t.Fatal("signal exit", interrupted.ProcessState)
	}
	var cancelled api.Envelope
	if err := json.Unmarshal(interruptedDiagnostics.Bytes(), &cancelled); err != nil || cancelled.Error == nil || cancelled.Error.Code != "operation.cancelled" {
		t.Fatal("signal diagnostic", cancelled, err)
	}
	if !reflect.DeepEqual(before, treeDigest(t, root)) {
		t.Fatal("interrupted MCP changed the signet")
	}
	// A fresh writable stdio connection must retain both saved and delivered
	// identities; this happens after the read-only inventory assertions above.
	writable := exec.CommandContext(ctx, binary, "mcp", "--binding", binding)
	writer, err := sdk.NewClient(&sdk.Implementation{Name: "combined-test", Version: "1"}, nil).Connect(ctx, &sdk.CommandTransport{Command: writable, TerminateDuration: time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	wire, err := writer.CallTool(ctx, &sdk.CallToolParams{Name: "memory_remember_and_sync", Arguments: json.RawMessage(`{"record":{"kind":"fact","summary":"Compiled route","body":"Golden Trail","basis":"user-direction","reason":"Synthetic compiled test"}}`)})
	if err != nil || wire.IsError {
		t.Fatalf("compiled combined stdio: %v %+v", err, wire)
	}
	encoded, _ = json.Marshal(wire.StructuredContent)
	checkCombined(encoded)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	t.Log("Compiled CLI and stdio: create, bind, correction/history, cwd and selection, shared results, read-only file hashes, malformed input, clean EOF passed")
}

func treeDigest(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	result := make(map[string][32]byte)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[path] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
