package api

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func TestRealSynchronizationOperations(t *testing.T) {
	a := fixture(t)
	ctx := context.Background()
	for _, name := range []string{"memory_git_init", "memory_checkpoint", "memory_sync"} {
		out := a.Call(ctx, name, []byte(`{}`))
		if !out.OK {
			t.Fatal(name, out.Error)
		}
		receipt := out.Result.(signetsync.Status)
		if receipt.State != "local-only" || !receipt.Checkpointed || receipt.Delivered {
			t.Fatal(receipt)
		}
	}
	out := a.Call(ctx, "memory_sync_status", []byte(`{}`))
	if !out.OK {
		t.Fatal(out.Error)
	}
	a.ReadOnly = true
	for _, name := range []string{"memory_git_init", "memory_checkpoint", "memory_sync"} {
		out := a.Call(ctx, name, []byte(`{}`))
		if out.OK || out.Error.Code != "operation.read_only" {
			t.Fatal(out)
		}
	}
	if out := a.Call(ctx, "memory_sync_status", []byte(`{}`)); !out.OK {
		t.Fatal(out.Error)
	}
}

func TestSyncTimeoutIsStrictInput(t *testing.T) {
	a := fixture(t)
	for _, raw := range []string{`{"timeout_seconds":0}`, `{"timeout_seconds":31}`, `{"timeout_seconds":"2"}`} {
		out := a.Call(context.Background(), "memory_sync", []byte(raw))
		if out.OK || out.Error.Code != "input.invalid" {
			t.Fatal(out)
		}
	}
}

func TestSyncCancellationExposesPartialWriteEvidence(t *testing.T) {
	a := fixture(t)
	if out := a.Call(context.Background(), "memory_git_init", []byte(`{}`)); !out.OK {
		t.Fatal(out.Error)
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(real, "-C", a.service.Root(), "remote", "add", "origin", "https://example.invalid/signet.git").CombinedOutput(); err != nil {
		t.Fatal(string(output), err)
	}
	bin := t.TempDir()
	quoted := "'" + strings.ReplaceAll(real, "'", "'\\''") + "'"
	script := "#!/bin/sh\nfor arg in \"$@\"; do if [ \"$arg\" = ls-remote ]; then sleep 30; exit 128; fi; done\nexec " + quoted + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	out := a.Call(context.Background(), "memory_sync", []byte(`{"timeout_seconds":1}`))
	if out.OK || out.Error.Code != "operation.cancelled" || !out.Error.WriteMayHaveOccurred || !out.Error.InspectBeforeRetry || out.Error.SyncStatus == nil || !out.Error.SyncStatus.Checkpointed || out.Error.SyncStatus.Phase != "fetch" {
		t.Fatal(out)
	}
}
