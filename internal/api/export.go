package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/exportreport"
)

type exportFailure struct {
	err    error
	result *exportreport.Receipt
}

func (e *exportFailure) Error() string { return e.err.Error() }
func (e *exportFailure) Unwrap() error { return e.err }

func exportOperation[I, O any](name, description string, readOnly bool, fn func(context.Context, I) (O, error)) Operation {
	op := connectionOperation(name, description, readOnly, fn)
	op.CLIOnly = true
	return op
}

var exports = []Operation{
	exportOperation("export_preview", "Preview an explicitly selected derived report without writing. Withheld records require both history and withdrawn opt-ins. Review metadata is sensitive. No default whole-bank export, synchronization or source changes.", true, func(ctx context.Context, in exportreport.Request) (exportreport.Plan, error) {
		p, err := exportreport.Preview(ctx, in)
		if err != nil {
			return p, &exportFailure{err: err}
		}
		return p, nil
	}),
	exportOperation("export_apply", "Reconstruct the complete reviewed export preview and write a private report into a fresh destination. Never overwrite. Inspect retained partial-output receipts before retrying. Not a backup or safe-to-publish certification.", false, func(ctx context.Context, in exportreport.Plan) (exportreport.Receipt, error) {
		r, err := exportreport.Apply(ctx, in)
		if err != nil {
			return r, &exportFailure{err: err, result: &r}
		}
		return r, nil
	}),
}
