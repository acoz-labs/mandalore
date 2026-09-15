package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/memory"
	piplugin "github.com/acoz-labs/mandalore/plugins/pi"
)

var piAdministration = []Operation{
	operation("pi_package_inspect", "Inspect the embedded native Pi package identity without changing the legacy runtime-version response. No binding, install, authentication or network access.", true, func(_ context.Context, _ *memory.Service, _ struct{}) (piplugin.Info, error) {
		return piplugin.Inspect()
	}),
}

func init() {
	piAdministration[0].RequiresBinding = false
	piAdministration[0].CLIOnly = true
}
