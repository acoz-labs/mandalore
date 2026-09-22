package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/readiness"
)

func TestAssessmentCLIAndSharedCallHaveIdenticalReports(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "absent"))
	for _, harness := range []string{"codex", "pi", "claude-code"} {
		in := readiness.Input{Harness: harness, StateDir: filepath.Join(root, "state"), NativeHome: filepath.Join(root, "profile"), NativeBinary: filepath.Join(root, "native"), Binding: filepath.Join(root, "binding"), IncludePrompt: true}
		raw, _ := json.Marshal(in)
		human, hcode := cli(t, []string{"connection", "assess", "--harness", harness, "--state-dir", in.StateDir, "--native-home", in.NativeHome, "--native-binary", in.NativeBinary, "--binding", in.Binding, "--prompt", "--read-only"}, "")
		shared, scode := cli(t, []string{"call", "connection_assess", "--read-only"}, string(raw))
		if hcode != 0 || scode != 0 || !human.OK || !shared.OK || !reflect.DeepEqual(human.Result, shared.Result) {
			t.Fatal("assessment transport drift", hcode, scode, human, shared)
		}
	}
}

func TestAssessmentCLIRejectsInvalidFlagsAndCancellation(t *testing.T) {
	for _, args := range [][]string{{"connection", "assess"}, {"connection", "assess", "--harness", "other"}, {"connection", "assess", "--harness", "pi", "--binding", "relative"}, {"connection", "assess", "--harness", "pi", "--apply"}} {
		v, code := cli(t, args, "")
		if code != 2 || v.OK || v.Error.Code != "input.invalid" {
			t.Fatal(args, v, code)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	code := run(ctx, []string{"connection", "assess", "--harness", "pi"}, strings.NewReader(""), &out, &out)
	var v api.Envelope
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if code != 130 || v.OK || v.Result != nil || v.Error.Code != "operation.cancelled" {
		t.Fatal("cancelled CLI returned a report", v, code)
	}
}

func TestCompiledAssessmentNeverRunsNativeOrDependencyTraps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "mandalore")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	traps := filepath.Join(root, "traps")
	if err := os.Mkdir(traps, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"codex", "pi", "claude", "git", "node"} {
		if err := os.WriteFile(filepath.Join(traps, name), []byte("#!/bin/sh\nprintf executed > \"${0%/*}/execution-canary\"\nexit 97\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// Count connections, not only successful requests, to catch failed probes.
	// This observes the configured proxy/provider endpoints; it is not a
	// system-wide network sandbox or proof against arbitrary direct sockets.
	var connections atomic.Int64
	endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	endpoint.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	endpoint.Start()
	defer endpoint.Close()
	if err := os.Mkdir(filepath.Join(root, "empty-unrelated-directory"), 0700); err != nil {
		t.Fatal(err)
	}
	before := assessmentFixtureInventory(t, root)
	for _, harness := range []string{"codex", "pi", "claude-code"} {
		cmd := exec.CommandContext(ctx, binary, "connection", "assess", "--harness", harness, "--state-dir", filepath.Join(root, "state"), "--native-home", filepath.Join(root, "profile"), "--binding", filepath.Join(root, "binding"), "--prompt")
		cmd.Env = []string{"PATH=" + traps, "HTTP_PROXY=" + endpoint.URL, "HTTPS_PROXY=" + endpoint.URL, "ALL_PROXY=" + endpoint.URL, "OPENAI_BASE_URL=" + endpoint.URL, "ANTHROPIC_BASE_URL=" + endpoint.URL}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("assessment: %v %s", err, out)
		}
		var v api.Envelope
		if err := json.Unmarshal(out, &v); err != nil || !v.OK {
			t.Fatal("invalid compiled report", string(out), err)
		}
		if after := assessmentFixtureInventory(t, root); !reflect.DeepEqual(before, after) {
			t.Fatal("assessment changed fixture bytes, modes, timestamps or directories", harness)
		}
		if connections.Load() != 0 {
			t.Fatal("assessment contacted instrumented network endpoint", connections.Load())
		}
	}
	entries, err := os.ReadDir(traps)
	if err != nil || len(entries) != 5 {
		t.Fatal("dependency/native trap executed", entries, err)
	}
	for _, name := range []string{"state", "profile", "binding"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatal("assessment created fixture state", name, err)
		}
	}
}

// Include empty directories, modes and mtimes: unchanged file counts alone
// cannot establish the no-product-write boundary.
func assessmentFixtureInventory(t *testing.T, root string) map[string]any {
	t.Helper()
	result := map[string]any{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		var digest [sha256.Size]byte
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			digest = sha256.Sum256(raw)
		}
		result[path] = struct {
			Mode     fs.FileMode
			Size     int64
			Modified int64
			Digest   [sha256.Size]byte
		}{info.Mode(), info.Size(), info.ModTime().UnixNano(), digest}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
