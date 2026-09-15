package api

import (
	"context"
	"errors"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

type SyncInput struct {
	TimeoutSeconds *int `json:"timeout_seconds,omitempty" jsonschema:"Default 10; range 1–30. Use a short budget for authorized passive sync; offline changes remain local."`
}

func synchronizer(s *memory.Service) (*signetsync.Synchronizer, error) {
	return signetsync.Open(s.Root(), s.ID())
}

func syncMemory(ctx context.Context, s *memory.Service, in SyncInput) (signetsync.Status, error) {
	seconds := number(in.TimeoutSeconds, 10)
	if seconds < 1 || seconds > 30 {
		return signetsync.Status{}, errors.New("sync timeout requires 1–30 seconds")
	}
	g, err := synchronizer(s)
	if err != nil {
		return signetsync.Status{}, err
	}
	return g.Sync(ctx, time.Duration(seconds)*time.Second)
}

var synchronization = []Operation{
	operation("memory_git_init", "Explicitly initialize the bound signet as its own Git repository on main and checkpoint it. Does not configure a remote or credentials.", false, func(ctx context.Context, s *memory.Service, _ struct{}) (signetsync.Status, error) {
		g, err := synchronizer(s)
		if err != nil {
			return signetsync.Status{}, err
		}
		return g.Initialize(ctx)
	}),
	operation("memory_checkpoint", "Commit valid local signet changes without network access. Requires explicit Git initialization. Preserves unrelated/partially staged work by refusing unsafe checkpoints.", false, func(ctx context.Context, s *memory.Service, _ struct{}) (signetsync.Status, error) {
		g, err := synchronizer(s)
		if err != nil {
			return signetsync.Status{}, err
		}
		return g.Checkpoint(ctx)
	}),
	operation("memory_sync", "Checkpoint, fetch, validate, integrate and push the bound signet through its native Git origin. Never force-push. Pending/offline or semantic-conflict status is not complete delivery/agreement. Do not sync a no-save/read-only task.", false, syncMemory),
	operation("memory_sync_status", "Read local Git/receipt state without network or writes. Last delivered head/time does not establish current remote freshness. Does not initialize, checkpoint or repair.", true, func(ctx context.Context, s *memory.Service, _ struct{}) (signetsync.Inspection, error) {
		g, err := synchronizer(s)
		if err != nil {
			return signetsync.Inspection{}, err
		}
		return g.Inspect(ctx)
	}),
}

func init() {
	for i := range synchronization {
		if synchronization[i].Name == "memory_sync" {
			synchronization[i].Network = true
		}
	}
}
