package api

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
	hermesplugin "github.com/acoz-labs/mandalore/plugins/hermes"
)

var hermesAdministration = []Operation{
	operation("hermes_package_inspect", "Inspect the embedded native Hermes package identity without changing the legacy runtime-version response. No binding, install, authentication or network access.", true, func(_ context.Context, _ *memory.Service, _ struct{}) (hermesplugin.Info, error) {
		return hermesplugin.Inspect()
	}),
	connectionOperation("hermes_connection_plan", "Preview an explicit Hermes connection. Reads local identities, ownership and bounded profile settings without executing binaries or writing files.", true, func(_ context.Context, in install.HermesOptions) (install.HermesPlan, error) {
		p, err := install.PrepareHermes(in)
		return p, hermesConnectionError(err, nil, nil)
	}),
	connectionOperation("hermes_connection_apply", "Apply a reviewed Hermes plan through native install/remove. Retains owned generations and phase evidence, preserves unrelated profile settings and never changes authentication or memory.", false, func(ctx context.Context, in install.HermesPlan) (install.HermesResult, error) {
		r, err := install.ApplyHermes(ctx, in)
		return r, hermesConnectionError(err, &r, nil)
	}),
	connectionOperation("hermes_connection_doctor", "Inspect Hermes connection structure, native version and registration without repair, authentication, synchronization or claiming active-session loading.", true, func(ctx context.Context, in install.Profile) (install.HermesReport, error) {
		r := install.DoctorHermes(ctx, in)
		if !r.Healthy {
			return r, hermesConnectionError(errors.New("Hermes connection needs attention; inspect checks and untested boundaries"), nil, &r)
		}
		return r, nil
	}),
	connectionOperation("hermes_connection_repair_plan", "Preview Hermes repair into a fresh owned generation. Preserves binding/read-only identity, refuses edited or foreign files and does not apply the plan.", true, func(_ context.Context, in install.RepairInput) (install.HermesPlan, error) {
		p, err := install.PrepareHermesRepair(in)
		return p, hermesConnectionError(err, nil, nil)
	}),
}

func init() {
	for i := range hermesAdministration {
		hermesAdministration[i].RequiresBinding = false
		hermesAdministration[i].CLIOnly = true
	}
	hermesAdministration[4].Idempotent = false // Recovery allocates a new generation.
}

type hermesConnectionFailure struct {
	err    error
	result *install.HermesResult
	report *install.HermesReport
}

func (e *hermesConnectionFailure) Error() string { return e.err.Error() }
func (e *hermesConnectionFailure) Unwrap() error { return e.err }
func hermesConnectionError(err error, result *install.HermesResult, report *install.HermesReport) error {
	if err == nil {
		return nil
	}
	var path *fs.PathError
	var link *os.LinkError
	if errors.As(err, &path) || errors.As(err, &link) {
		err = errors.New("filesystem operation failed; inspect the selected Hermes installation paths before retrying")
	}
	return &hermesConnectionFailure{err: err, result: result, report: report}
}
