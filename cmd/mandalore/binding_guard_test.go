package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func cliGuardFixture(t *testing.T) (string, string, binding.Binding) {
	t.Helper()
	s, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Example", "device-example", "Test device")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "binding.json")
	b, err := binding.Bind(s.Root, path, "Bound test device", "Test actor")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	return path, hex.EncodeToString(digest[:]), b
}

func TestCLIExplicitBindingGuards(t *testing.T) {
	path, digest, b := cliGuardFixture(t)
	t.Chdir(t.TempDir())
	out, code := cli(t, []string{"call", "memory_inspect", "--binding", path, "--binding-sha256", digest, "--signet-id", b.SignetID, "--harness", "pi"}, `{}`)
	if code != 0 || !out.OK || out.Result.(map[string]any)["signet_id"] != b.SignetID {
		t.Fatal("valid guarded call failed", out, code)
	}
	before := treeDigest(t, b.Root)
	for _, guards := range [][]string{
		{"--binding-sha256", "", "--signet-id", ""},
		{"--binding-sha256", digest},
		{"--signet-id", b.SignetID},
		{"--binding-sha256", "invalid", "--signet-id", b.SignetID},
		{"--binding-sha256", strings.Repeat("0", 64), "--signet-id", b.SignetID},
		{"--binding-sha256", digest, "--signet-id", "signet-other"},
	} {
		var stdout, stderr bytes.Buffer
		args := append([]string{"call", "memory_remember", "--binding", path}, guards...)
		code := run(context.Background(), args, forbiddenRead{t}, &stdout, &stderr)
		var envelope api.Envelope
		if json.Unmarshal(stdout.Bytes(), &envelope) != nil || code == 0 || envelope.OK || envelope.Error == nil || envelope.Error.WriteMayHaveOccurred {
			t.Fatalf("guard refusal ambiguous: %d %s", code, stdout.String())
		}
	}
	if after := treeDigest(t, b.Root); !reflect.DeepEqual(after, before) {
		t.Fatal("refused guard changed bank")
	}
	out, code = cli(t, []string{"call", "signet_create", "--binding-sha256", digest, "--signet-id", b.SignetID}, `{}`)
	if code != 2 || out.Error == nil || out.Error.Code != "input.invalid" {
		t.Fatal("unbound setup accepted an inapplicable guard", out, code)
	}
}

func TestCompiledGuardedCallsDoNotFollowBindingReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "mandalore")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	path, digest, original := cliGuardFixture(t)
	otherPath, _, other := cliGuardFixture(t)
	cwd := t.TempDir()
	invoke := func(operation, input string) (api.Envelope, int) {
		t.Helper()
		cmd := exec.CommandContext(ctx, binary, "call", operation, "--binding", path, "--binding-sha256", digest, "--signet-id", original.SignetID, "--harness", "pi")
		cmd.Dir = cwd
		cmd.Stdin = strings.NewReader(input)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		_ = cmd.Run()
		var envelope api.Envelope
		if cmd.ProcessState == nil || stderr.Len() != 0 || json.Unmarshal(stdout.Bytes(), &envelope) != nil {
			t.Fatalf("invalid compiled output: %s / %s", stdout.String(), stderr.String())
		}
		return envelope, cmd.ProcessState.ExitCode()
	}
	if out, code := invoke("memory_inspect", `{}`); code != 0 || !out.OK {
		t.Fatal("guarded process could not read", out, code)
	}
	beforeOriginal, beforeOther := treeDigest(t, original.Root), treeDigest(t, other.Root)
	replacement, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, replacement, 0600); err != nil {
		t.Fatal(err)
	}
	out, code := invoke("memory_remember", `{"kind":"fact","summary":"Must not save","body":"Wrong bank","basis":"observation","reason":"Probe"}`)
	if code == 0 || out.OK || out.Error == nil || out.Error.Code != "binding.invalid" || out.Error.WriteMayHaveOccurred {
		t.Fatal("fresh process followed replaced binding", out, code)
	}
	if !reflect.DeepEqual(beforeOriginal, treeDigest(t, original.Root)) || !reflect.DeepEqual(beforeOther, treeDigest(t, other.Root)) {
		t.Fatal("guarded refusal modified a bank")
	}
}
