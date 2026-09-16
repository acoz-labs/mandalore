package api

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

var piAdministration = []Operation{
	operation("pi_package_inspect", "Inspect the embedded native Pi package identity without changing the legacy runtime-version response. No binding, install, authentication or network access.", true, func(_ context.Context, _ *memory.Service, _ struct{}) (piplugin.Info, error) {
		return piplugin.Inspect()
	}),
	connectionOperation("pi_connection_plan", "Preview an explicit Pi connection. Reads local identities, ownership and bounded profile settings without executing binaries or writing files.", true, func(_ context.Context, in install.PiOptions) (install.PiPlan, error) {
		p, err := install.PreparePi(in)
		return p, piConnectionError(err, nil, nil)
	}),
	connectionOperation("pi_connection_apply", "Apply a reviewed Pi plan through native install/remove. Retains owned generations and phase evidence, preserves unrelated profile settings and never changes authentication or memory.", false, func(ctx context.Context, in install.PiPlan) (install.PiResult, error) {
		r, err := install.ApplyPi(ctx, in)
		return r, piConnectionError(err, &r, nil)
	}),
	connectionOperation("pi_connection_doctor", "Inspect Pi connection structure, native version and registration without repair, authentication, synchronization or claiming active-session loading.", true, func(ctx context.Context, in install.Profile) (install.PiReport, error) {
		r := install.DoctorPi(ctx, in)
		if !r.Healthy {
			return r, piConnectionError(errors.New("Pi connection needs attention; inspect checks and untested boundaries"), nil, &r)
		}
		return r, nil
	}),
	connectionOperation("pi_connection_repair_plan", "Preview Pi repair into a fresh owned generation. Preserves binding/read-only identity, refuses edited or foreign files and does not apply the plan.", true, func(_ context.Context, in install.RepairInput) (install.PiPlan, error) {
		p, err := install.PreparePiRepair(in)
		return p, piConnectionError(err, nil, nil)
	}),
}

func init() {
	for i := range piAdministration {
		piAdministration[i].RequiresBinding = false
		piAdministration[i].CLIOnly = true
	}
	piAdministration[4].Idempotent = false // Recovery allocates a new generation.
}

type piConnectionFailure struct {
	err    error
	result *install.PiResult
	report *install.PiReport
}

func (e *piConnectionFailure) Error() string { return e.err.Error() }
func (e *piConnectionFailure) Unwrap() error { return e.err }
func piConnectionError(err error, result *install.PiResult, report *install.PiReport) error {
	if err == nil {
		return nil
	}
	var path *fs.PathError
	var link *os.LinkError
	if errors.As(err, &path) || errors.As(err, &link) {
		err = errors.New("filesystem operation failed; inspect the selected Pi installation paths before retrying")
	}
	return &piConnectionFailure{err: err, result: result, report: report}
}
