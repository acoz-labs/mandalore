package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func synchronizer(s *memory.Service) (*signetsync.Synchronizer, error) {
	return signetsync.Open(s.Root(), s.ID())
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
}
