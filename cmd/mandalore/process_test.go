package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// This exercises the compiled executable and the actual Streamable HTTP server,
// not just run() or an in-memory transport. No provider/model is involved.
func TestCompiledCLIAndServer(t *testing.T) {
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
	for _, name := range []string{"git-init", "checkpoint"} {
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
	before := treeDigest(t, root)
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

	serveCompiled(ctx, t, binary, base, root, binding, remote)
	t.Log("Compiled CLI and server: create, bind, correction/history, cwd and selection, read-only file hashes, malformed input, serve, backup push, status and clean stop passed")
}

// serveCompiled runs the compiled server on loopback over the CLI-made signet:
// MCP reads and writes over Streamable HTTP, backup reaches the bare remote,
// status reports it, and an interrupt stops the server cleanly.
func serveCompiled(ctx context.Context, t *testing.T, binary, base, root, bindingPath, remote string) {
	t.Helper()
	var bound binding.Binding
	raw, err := os.ReadFile(bindingPath)
	if err != nil || json.Unmarshal(raw, &bound) != nil {
		t.Fatal("binding unreadable", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	config := filepath.Join(base, "server.json")
	raw, _ = json.Marshal(map[string]any{"listen": address, "public_url": "http://" + address, "state_dir": filepath.Join(base, "state"), "owner": map[string]string{"name": "Example"}, "auth": "loopback-trust", "local_scope": "read-write",
		"signets": []map[string]string{{"slug": "primary", "root": root, "signet_id": bound.SignetID, "device_id": bound.DeviceID, "remote": "file://" + remote}}})
	if err := os.WriteFile(config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	serve := exec.CommandContext(ctx, binary, "serve", "--config", config)
	var stdout, logs bytes.Buffer
	serve.Stdout, serve.Stderr = &stdout, &logs
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serve.Process.Kill() })
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if r, err := http.Get("http://" + address + "/healthz"); err == nil {
			r.Body.Close()
			if r.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become healthy: %s", logs.String())
		}
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "compiled-test", Version: "1"}, nil).Connect(ctx, &sdk.StreamableClientTransport{Endpoint: "http://" + address + "/mcp", MaxRetries: -1, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "memory_recall", Arguments: map[string]any{}})
	if text, _ := json.Marshal(result.StructuredContent); err != nil || result.IsError || !bytes.Contains(text, []byte("Silver Heron")) || bytes.Contains(text, []byte("Copper Finch")) {
		t.Fatalf("server recall: %v %s", err, text)
	}
	result, err = session.CallTool(ctx, &sdk.CallToolParams{Name: "memory_remember", Arguments: json.RawMessage(`{"request_id":"compiled-request-0001","record":{"kind":"fact","summary":"Compiled route","body":"Golden Trail","basis":"user-direction","reason":"Synthetic compiled test"}}`)})
	if text, _ := json.Marshal(result.StructuredContent); err != nil || result.IsError || !bytes.Contains(text, []byte(`"durable":true`)) || !bytes.Contains(text, []byte(`"backup":"pending"`)) {
		t.Fatalf("server write: %v %s", err, text)
	}
	git := func(args ...string) string {
		out, err := exec.CommandContext(ctx, "git", args...).Output()
		if err != nil {
			t.Fatal(args, err)
		}
		return strings.TrimSpace(string(out))
	}
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if strings.Contains(git("--git-dir="+remote, "log", "--all", "-p", "--format="), "Golden Trail") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("backup did not reach the remote: %s", logs.String())
		}
	}
	status := exec.CommandContext(ctx, binary, "server", "status", "--config", config)
	out, err := status.Output()
	if err != nil || !bytes.Contains(out, []byte(`"remote":"current"`)) || !bytes.Contains(out, []byte(git("-C", root, "rev-parse", "HEAD"))) {
		t.Fatalf("server status: %v %s", err, out)
	}
	if err := serve.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := serve.Wait(); err != nil || stdout.Len() != 0 {
		t.Fatalf("server stop: %v %s %s", err, stdout.String(), logs.String())
	}
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
