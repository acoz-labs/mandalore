package api

import (
	"context"
	"errors"

	"github.com/acoz-labs/mandalore/internal/formatupgrade"
)

type upgradeFailure struct {
	err    error
	result *formatupgrade.Receipt
}

func (e *upgradeFailure) Error() string { return e.err.Error() }
func (e *upgradeFailure) Unwrap() error { return e.err }

func upgradeOperation[I, O any](name, description string, readOnly bool, fn func(context.Context, I) (O, error)) Operation {
	op := connectionOperation(name, description, readOnly, fn)
	op.CLIOnly = true
	return op
}

var upgradeOperations = []Operation{
	upgradeOperation("signet_upgrade_preview", "Read-only preview of an explicit binding's clean format1 checkpoint. Pins binding/source/HEAD and lists format2 effects. Does not checkpoint, fetch or upgrade. Older clients refuse upgraded banks.", true, func(ctx context.Context, in formatupgrade.Request) (formatupgrade.Plan, error) {
		p, err := formatupgrade.Preview(ctx, in)
		if err != nil {
			return p, &upgradeFailure{err: err}
		}
		return p, nil
	}),
	upgradeOperation("signet_upgrade_apply", "Apply the exact reviewed format upgrade with stopped_writers acknowledged. Preserves signet identity, evidence and Git history. Retains private recovery state. No automatic checkpoint, delivery, rollback or deletion. Inspect partial receipts before retrying.", false, func(ctx context.Context, in formatupgrade.ApplyRequest) (formatupgrade.Receipt, error) {
		r, err := formatupgrade.Apply(ctx, in)
		if err != nil {
			return r, &upgradeFailure{err: err, result: &r}
		}
		return r, nil
	}),
	upgradeOperation("signet_upgrade_recover", "Explicitly resume retained upgrade preparation using the original exact plan and stopped_writers acknowledgement. Revalidates pins; never starts an absent upgrade, rolls back, downgrades or synchronizes. Changed source/HEAD/binding requires diagnosis.", false, func(ctx context.Context, in formatupgrade.ApplyRequest) (formatupgrade.Receipt, error) {
		r, err := formatupgrade.Recover(ctx, in)
		if err != nil {
			return r, &upgradeFailure{err: err, result: &r}
		}
		return r, nil
	}),
}

func upgradeFailureEnvelope(e *upgradeFailure) Envelope {
	code := "upgrade.failed"
	if errors.Is(e.err, context.Canceled) || errors.Is(e.err, context.DeadlineExceeded) {
		code = "operation.cancelled"
	}
	mayWrite := e.result != nil && e.result.StagingDirectory != ""
	out := Failure(code, e.Error(), mayWrite)
	out.Error.UpgradeResult = e.result
	return out
}
