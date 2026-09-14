package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGeneratedWrapperPinsDefaultsButAllowsExplicitOverrides(t *testing.T) {
	o := fixture(t)
	renamed := filepath.Join(filepath.Dir(o.Binding), "binding's file.json")
	if err := os.Rename(o.Binding, renamed); err != nil {
		t.Fatal(err)
	}
	o.Binding = renamed
	if err := os.WriteFile(o.Binary, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" \"$MANDALORE_BINDING\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := Prepare(o)
	if err != nil {
		t.Fatal(err)
	}
	if err := stageRuntime(p); err != nil {
		t.Fatal(err)
	}
	if err := publishBundle(p); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(p.Root, "plugins/mandalore/scripts/connection.sh")
	env := environment(map[string]string{"MANDALORE_BIN": "", "MANDALORE_BINDING": ""})
	out, err := execute(context.Background(), "/bin/sh", t.TempDir(), env, nil, wrapper, "mcp")
	if err != nil || string(out) != "mcp --harness codex\n"+p.Binding+"\n" {
		t.Fatal("default routing", string(out), err)
	}
	env = environment(map[string]string{"MANDALORE_BIN": o.Binary, "MANDALORE_BINDING": "/synthetic/other-binding.json"})
	out, err = execute(context.Background(), "/bin/sh", t.TempDir(), env, nil, wrapper, "hook")
	if err != nil || string(out) != "codex-memory-hook\n/synthetic/other-binding.json\n" {
		t.Fatal("override routing", string(out), err)
	}
}

func TestProcessFailureOutputSuppressedAndCancellationBounded(t *testing.T) {
	out, err := execute(context.Background(), "/bin/sh", t.TempDir(), os.Environ(), nil, "-c", "printf SECRET_CANARY; printf SECRET_CANARY >&2; exit 2")
	if err == nil || len(out) != 0 || strings.Contains(err.Error(), "SECRET_CANARY") {
		t.Fatal("raw failure leaked", string(out), err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := execute(ctx, "/bin/sh", t.TempDir(), os.Environ(), nil, "-c", "sleep 10"); err == nil {
		t.Fatal("cancellation ignored")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("process cancellation exceeded bounded wait")
	}
	var b boundedOutput
	if _, err := b.Write(make([]byte, 1<<20)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("x")); err == nil {
		t.Fatal("output bound ignored")
	}
}

func TestRealProcessCannotBypassOutputLimitThroughReadFrom(t *testing.T) {
	out, err := execute(context.Background(), "/bin/sh", t.TempDir(), os.Environ(), nil,
		"-c", "dd if=/dev/zero bs=1048576 count=2 2>/dev/null")
	if err == nil || len(out) != 0 {
		t.Fatalf("real subprocess bypassed output limit: returned %d bytes, error %v", len(out), err)
	}
}
