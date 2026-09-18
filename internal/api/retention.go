package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/retention"
)

func retentionPreviewOperation() Operation {
	op := connectionOperation("retention_preview", "Read-only metadata review of explicitly selected record IDs/scopes and separately selected journal IDs under a named policy. Optional age requires a named timestamp and absolute cutoff. Pins binding/source/policy, reports shared source relationships, and refuses oversized results instead of truncating. No automatic expiry, withdrawal, deletion, checkpoint, sync or retention apply. CLI-only.", true, func(ctx context.Context, in retention.Request) (retention.Plan, error) {
		return retention.Preview(ctx, in)
	})
	op.CLIOnly = true
	return op
}

var retentionOperations = []Operation{retentionPreviewOperation()}
