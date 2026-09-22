// Package claudecode is the read-only native lifecycle adapter. It does not write
// memory, synchronize, read transcripts, or invoke other processes.
// Adapted from the pinned public My Friday memory adapter; see NOTICE.
package claudecode

import (
	"encoding/json"
	"io"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memorycontext"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const orientation = memorycontext.Orientation + " Claude native memory is separate. Do not copy Mandalore signet knowledge into native memory, import native notes automatically, or recreate withdrawn facts from native notes. Do not disable native memory implicitly. Resolve disagreements against current user direction and current Mandalore evidence."

type contextOutput struct {
	Event   string `json:"hookEventName"`
	Context string `json:"additionalContext"`
}
type output struct {
	Context *contextOutput `json:"hookSpecificOutput,omitempty"`
	Warning string         `json:"systemMessage,omitempty"`
}

func Run(path string, guard binding.Guard, input io.Reader, out io.Writer) error {
	emit := func(v output) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if len(b) > 9500 {
			b, _ = json.Marshal(output{Warning: "Mandalore hook context exceeded its budget. Use memory tools directly; no memory was changed."})
		}
		_, err = out.Write(append(b, '\n'))
		return err
	}
	warn := func() error {
		return emit(output{Warning: "Mandalore could not read the native hook event. Use memory tools directly; no memory was changed."})
	}
	b, err := io.ReadAll(io.LimitReader(input, 65537))
	if err != nil {
		return warn()
	}
	// Native fields evolve. Validate the whole object without rejecting unknown
	// fields, then interpret only exact-case fields with explicit types.
	var event map[string]any
	if err := strictjson.Decode(b, &event, 65536); err != nil {
		return warn()
	}
	name, ok := event["hook_event_name"].(string)
	if !ok || name == "" {
		return warn()
	}
	if name != "SessionStart" && name != "UserPromptSubmit" {
		return emit(output{})
	}
	var prompt string
	if name == "UserPromptSubmit" {
		prompt, ok = event["prompt"].(string)
		if !ok {
			return warn()
		}
	}
	if path == "" || guard.SHA256 == "" || guard.SignetID == "" {
		return emit(output{Warning: "Mandalore binding selection is invalid. Check MANDALORE_BINDING; no memory was changed."})
	}
	s, err := binding.OpenGuarded(path, "claude-code", guard)
	if err != nil {
		return emit(output{Warning: "Mandalore memory is unavailable. Check the selected local binding and runtime; no memory was changed."})
	}
	packet := memorycontext.Build(s, prompt, name == "UserPromptSubmit", orientation)
	result := output{Warning: packet.Warning}
	if packet.Context != "" {
		result.Context = &contextOutput{name, packet.Context}
	}
	return emit(result)
}
