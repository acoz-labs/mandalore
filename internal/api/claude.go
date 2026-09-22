package api

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
	claudeplugin "github.com/acoz-labs/mandalore/plugins/claude-code"
)

var claudeAdministration = []Operation{
	operation("claude_code_package_inspect", "Inspect the embedded native Claude package identity without changing the legacy runtime-version response. No binding, install, authentication or network access.", true, func(_ context.Context, _ *memory.Service, _ struct{}) (claudeplugin.Info, error) {
		return claudeplugin.Inspect()
	}),
	connectionOperation("claude_code_connection_plan", "Preview an explicit Claude connection. Reads local identities, ownership and bounded profile settings without executing binaries or writing files.", true, func(_ context.Context, in install.ClaudeOptions) (install.ClaudePlan, error) {
		p, err := install.PrepareClaude(in)
		return p, claudeConnectionError(err, nil, nil)
	}),
	connectionOperation("claude_code_connection_apply", "Apply a reviewed Claude plan through native install/remove. Retains owned generations and phase evidence, preserves unrelated profile settings and never changes authentication or memory.", false, func(ctx context.Context, in install.ClaudeApplyInput) (install.ClaudeResult, error) {
		r, err := install.ApplyClaude(ctx, in)
		return r, claudeConnectionError(err, &r, nil)
	}),
	connectionOperation("claude_code_connection_doctor", "Inspect Claude connection structure, native version and registration without repair, authentication, synchronization or claiming active-session loading.", true, func(ctx context.Context, in install.Profile) (install.ClaudeReport, error) {
		r := install.DoctorClaude(ctx, in)
		if !r.Healthy {
			return r, claudeConnectionError(errors.New("Claude connection needs attention; inspect checks and untested boundaries"), nil, &r)
		}
		return r, nil
	}),
	connectionOperation("claude_code_connection_repair_plan", "Preview Claude repair into a fresh owned generation. Preserves binding/read-only identity, refuses edited or foreign files and does not apply the plan.", true, func(_ context.Context, in install.RepairInput) (install.ClaudePlan, error) {
		p, err := install.PrepareClaudeRepair(in)
		return p, claudeConnectionError(err, nil, nil)
	}),
}

func init() {
	for i := range claudeAdministration {
		claudeAdministration[i].RequiresBinding = false
		claudeAdministration[i].CLIOnly = true
	}
	claudeAdministration[4].Idempotent = false // Recovery allocates a new generation.
}

type claudeConnectionFailure struct {
	err    error
	result *install.ClaudeResult
	report *install.ClaudeReport
}

func (e *claudeConnectionFailure) Error() string { return e.err.Error() }
func (e *claudeConnectionFailure) Unwrap() error { return e.err }
func claudeConnectionError(err error, result *install.ClaudeResult, report *install.ClaudeReport) error {
	if err == nil {
		return nil
	}
	var path *fs.PathError
	var link *os.LinkError
	if errors.As(err, &path) || errors.As(err, &link) {
		err = errors.New("filesystem operation failed; inspect the selected Claude installation paths before retrying")
	}
	return &claudeConnectionFailure{err: err, result: result, report: report}
}
