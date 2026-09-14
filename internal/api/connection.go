package api

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
)

type connectionFailure struct {
	err    error
	result *install.Result
	report *install.Report
}

func (e *connectionFailure) Error() string { return e.err.Error() }
func (e *connectionFailure) Unwrap() error { return e.err }

func connectionError(err error, result *install.Result, report *install.Report) error {
	if err == nil {
		return nil
	}
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	if errors.As(err, &pathErr) || errors.As(err, &linkErr) {
		err = errors.New("filesystem operation failed; inspect the selected installation paths before retrying")
	}
	return &connectionFailure{err: err, result: result, report: report}
}

func connectionOperation[I, O any](name, description string, readOnly bool, fn func(context.Context, I) (O, error)) Operation {
	op := operation(name, description, readOnly, func(ctx context.Context, _ *memory.Service, input I) (O, error) { return fn(ctx, input) })
	op.RequiresBinding = false
	return op
}

var connections = []Operation{
	connectionOperation("connection_plan", "Preview a machine-local Codex connection without executing binaries or writing files. All paths are explicit. Does not establish publisher trust.", true, func(_ context.Context, in install.Options) (install.Plan, error) {
		p, err := install.Prepare(in)
		return p, connectionError(err, nil, nil)
	}),
	connectionOperation("connection_apply", "Apply an explicitly approved connection plan. Executes selected trusted binaries, retains runtime/plugin copies and manages only a proven native registration. No signet writes or native authentication changes.", false, func(ctx context.Context, in install.Plan) (install.Result, error) {
		r, err := install.Apply(ctx, in)
		return r, connectionError(err, &r, nil)
	}),
	connectionOperation("connection_doctor", "Inspect structure and native plugin inventory. Does not test login, hook trust, live MCP, remote freshness or active context. No repair or synchronization.", true, func(ctx context.Context, in install.Profile) (install.Report, error) {
		r := install.Doctor(ctx, in)
		if !r.Healthy {
			return r, connectionError(errors.New("connection needs attention; inspect the structural checks and untested boundaries"), nil, &r)
		}
		return r, nil
	}),
	connectionOperation("connection_repair_plan", "Preview recovery into a fresh retained generation. Requires intact ownership evidence; refuses edited files or changed bindings. Does not apply the plan.", true, func(_ context.Context, in install.RepairInput) (install.Plan, error) {
		p, err := install.PrepareRepair(in)
		return p, connectionError(err, nil, nil)
	}),
}

func init() {
	// Repair previews intentionally allocate a new generation identifier.
	connections[3].Idempotent = false
}
