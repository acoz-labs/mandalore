package api

import (
	"context"
	"testing"

	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func TestRealSynchronizationOperations(t *testing.T) {
	a := fixture(t)
	ctx := context.Background()
	for _, name := range []string{"memory_git_init", "memory_checkpoint"} {
		out := a.Call(ctx, name, []byte(`{}`))
		if !out.OK {
			t.Fatal(name, out.Error)
		}
		receipt := out.Result.(signetsync.Status)
		if receipt.State != "local-only" || !receipt.Checkpointed || receipt.Delivered {
			t.Fatal(receipt)
		}
	}
	a.ReadOnly = true
	for _, name := range []string{"memory_git_init", "memory_checkpoint"} {
		out := a.Call(ctx, name, []byte(`{}`))
		if out.OK || out.Error.Code != "operation.read_only" {
			t.Fatal(out)
		}
	}
	for _, name := range []string{"memory_sync", "memory_sync_status", "memory_remember_and_sync", "memory_journal_append_and_sync"} {
		if out := a.Call(ctx, name, []byte(`{}`)); out.OK || out.Error.Code != "operation.unknown" {
			t.Fatalf("%s is still in the catalog", name)
		}
	}
}
