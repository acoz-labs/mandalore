package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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
	for _, harness := range []string{"codex", "pi"} {
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
	for _, name := range []string{"codex", "pi", "git", "node"} {
		if err := os.WriteFile(filepath.Join(traps, name), []byte("#!/bin/sh\nprintf executed > \"${0%/*}/execution-canary\"\nexit 97\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, harness := range []string{"codex", "pi"} {
		cmd := exec.CommandContext(ctx, binary, "connection", "assess", "--harness", harness, "--state-dir", filepath.Join(root, "state"), "--native-home", filepath.Join(root, "profile"), "--binding", filepath.Join(root, "binding"), "--prompt")
		cmd.Env = []string{"PATH=" + traps, "HOME=" + root, "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1"}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("assessment: %v %s", err, out)
		}
		var v api.Envelope
		if err := json.Unmarshal(out, &v); err != nil || !v.OK {
			t.Fatal("invalid compiled report", string(out), err)
		}
	}
	entries, err := os.ReadDir(traps)
	if err != nil || len(entries) != 4 {
		t.Fatal("dependency/native trap executed", entries, err)
	}
	for _, name := range []string{"state", "profile", "binding"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatal("assessment created fixture state", name, err)
		}
	}
}
