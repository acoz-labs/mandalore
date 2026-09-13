package api

import (
	"context"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

type CreateInput struct {
	Repository  string `json:"repository"`
	Name        string `json:"name"`
	DeviceLabel string `json:"device_label"`
}
type BindInput struct {
	Repository  string `json:"repository"`
	Binding     string `json:"binding"`
	DeviceLabel string `json:"device_label"`
	Actor       string `json:"actor"`
}
type CreateReceipt struct {
	SignetID        string `json:"signet_id"`
	Root            string `json:"root"`
	DeviceID        string `json:"device_id"`
	Created         bool   `json:"created"`
	GitInitialized  bool   `json:"git_initialized"`
	Synchronization string `json:"synchronization"`
}

func admin[I, O any](name, description string, fn func(context.Context, *memory.Service, I) (O, error)) Operation {
	op := operation(name, description, false, fn)
	op.RequiresBinding = false
	return op
}

var administration = []Operation{
	admin("signet_create", "Create a new local signet without overwriting an existing target. Does not initialize Git or configure a remote; sync setup is separate.", func(_ context.Context, _ *memory.Service, in CreateInput) (CreateReceipt, error) {
		id := memory.NewID("device")
		s, err := memory.Create(in.Repository, in.Name, id, in.DeviceLabel)
		if err != nil {
			return CreateReceipt{}, err
		}
		return CreateReceipt{SignetID: s.Signet.ID, Root: s.Root, DeviceID: id, Created: true, Synchronization: "not-configured"}, nil
	}),
	admin("signet_bind", "Create a machine-local binding outside the signet and enroll a new explicitly labeled device. Never overwrite an existing binding; partial I/O can leave an unused enrollment.", func(_ context.Context, _ *memory.Service, in BindInput) (binding.Binding, error) {
		return binding.Bind(in.Repository, in.Binding, in.DeviceLabel, in.Actor)
	}),
}
