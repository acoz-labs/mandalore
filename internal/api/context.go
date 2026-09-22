package api

import (
	"context"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/memorycontext"
	"github.com/acoz-labs/mandalore/internal/sessionsync"
)

type NativeContextInput struct {
	Boundary *sessionsync.Boundary `json:"boundary,omitempty" jsonschema:"Native lifecycle identity for enabled session synchronization; ignored by legacy local-only context."`
	Prompt   *string               `json:"prompt,omitempty" jsonschema:"Omit for orientation only; supply a prompt (including empty) for local recall truncated to 2048 UTF-8 bytes. No transcript, writes or synchronization."`
}

var nativeContext = []Operation{
	operation("memory_context", "Read a bounded native lifecycle context packet: orientation, bank-wide evidence and scope routing. CLI-only; no writes, network, transcript reads or automatic foundling scan.", true, func(_ context.Context, s *memory.Service, in NativeContextInput) (memorycontext.Packet, error) {
		if in.Prompt == nil {
			return memorycontext.Build(s, "", false, memorycontext.Orientation), nil
		}
		return memorycontext.Build(s, *in.Prompt, true, memorycontext.Orientation), nil
	}),
}

func init() {
	nativeContext[0].CLIOnly = true
}
