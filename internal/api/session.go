package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/memorycontext"
	"github.com/acoz-labs/mandalore/internal/sessionsync"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	"github.com/google/jsonschema-go/jsonschema"
)

// NewSession is a separate opt-in constructor. New(...,readOnly) retains all
// existing local-only and read-only behavior and never upgrades permissions.
func NewSession(s *memory.Service, c *sessionsync.Coordinator) (*API, error) {
	if !c.Matches(s) {
		return nil, errors.New("session coordinator does not match selected service")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &API{service: s, session: c}, nil
}
func (a *API) SessionEnabled() bool { return a != nil && a.session != nil }
func SessionSyncSchema() *jsonschema.Schema {
	s, err := strictjson.Schema(new(sessionsync.Attempt))
	if err != nil {
		panic(err)
	}
	return s
}

// Explicit portable mutator set: local mappings, Git administration, migration
// and installation remain outside automatic semantic delivery.
func sessionMutation(name string) bool {
	switch name {
	case "memory_remember", "memory_journal_append", "memory_remember_and_sync", "memory_journal_append_and_sync", "memory_withdraw", "memory_restore", "foundling_promote", "foundling_register", "foundling_disconnect":
		return true
	}
	return false
}

var sessionCatalog = operation("memory_session_catalog", "Inspect the selected session's active operation schemas and effects. No synchronization or semantic writes.", true, func(context.Context, *memory.Service, struct{}) (map[string]any, error) {
	return nil, errors.New("session catalog requires an enabled-session connection")
})

func (a *API) Catalog() []Operation {
	ops := Catalog()
	if !a.SessionEnabled() {
		return ops
	}
	for i := range ops {
		op := &ops[i]
		if sessionMutation(op.Name) {
			op.Network = true
			op.Description = strings.ReplaceAll(op.Description, "Do not journal a read-only task.", "Honor requests not to journal particular content.")
			op.Description = strings.ReplaceAll(op.Description, "Saves locally only; sync separately when allowed.", "Software attempts bounded delivery after the local save.")
			switch op.Name {
			case "memory_remember":
				op.Description = "Save confirmed knowledge locally; enabled-session software then attempts bounded delivery automatically. Honor do-not-remember content requests. Inspect result for the saved identity and session_sync separately for delivery; never repeat the save to retry delivery." + correctionGuidance
			case "memory_journal_append":
				op.Description = "Save a concise useful semantic journal, never transcripts or secrets; enabled-session software attempts bounded delivery automatically. Honor no-journal content requests. Inspect local receipt and session_sync separately; never repeat a journal to retry delivery."
			case "memory_remember_and_sync":
				op.Description = "Compatibility combined save: save confirmed knowledge once and make exactly one software-controlled bounded delivery attempt. Ordinary memory_remember already delivers in this enabled session. Inspect saved, delivery and session_sync separately; never repeat the save to retry delivery." + correctionGuidance
			case "memory_journal_append_and_sync":
				op.Description = "Compatibility combined journal: save once and make exactly one software-controlled bounded delivery attempt. Ordinary memory_journal_append already delivers in this enabled session. Honor no-journal content requests and inspect receipts separately."
			}
		}
		if op.Name == "memory_context" {
			op.ReadOnly = false
			op.Idempotent = false
			op.Network = true
			op.Description = "Refresh the enabled session with one bounded foreground synchronization attempt, then assemble native memory context. No transcript reads or semantic saves. Report pending/stale state separately."
		}
		if op.Name == "memory_recall" {
			op.Description = "Recall local evidence in the selected scope. Enabled-session lifecycle software already attempts bounded refresh before context; inspect that synchronization receipt for current or pending evidence. Discover memory_scopes after an empty unscoped result. Do not add an automatic model-selected sync retry loop."
		}
		if op.Name == "memory_sync" {
			op.Description = "Explicit bounded retry of this enabled session's synchronization. Ordinary lifecycle and semantic saves already trigger software attempts. Inspect status; never force-push or automatically resolve semantic conflicts."
		}
	}
	return ops
}

func (a *API) callSession(ctx context.Context, name string, data []byte) Envelope {
	if a.ReadOnly {
		return Failure("operation.read_only", "Session synchronization cannot override a read-only connection.", false)
	}
	if err := ctx.Err(); err != nil {
		return Failure("operation.cancelled", "Operation cancelled before execution.", false)
	}
	if err := a.session.Validate(); err != nil {
		return Failure("session.policy-invalid", "Enabled-session selection changed; reconnect explicitly before continuing.", false)
	}
	legacy := *a
	legacy.session = nil
	if name == "memory_session_catalog" {
		var in struct{}
		if strictjson.Decode(data, &in, MaxInputBytes) != nil {
			return Failure("input.invalid", "Session catalog expects an empty object.", false)
		}
		return Success(map[string]any{"operations": a.Catalog()})
	}
	if name == "memory_context" {
		var in NativeContextInput
		if strictjson.Decode(data, &in, MaxInputBytes) != nil {
			return Failure("input.invalid", "Invalid native context request.", false)
		}
		boundary := sessionsync.Boundary{Kind: "turn"}
		if in.Boundary != nil {
			boundary = *in.Boundary
		}
		attempt := a.session.Attempt(ctx, boundary)
		if attempt.Error != nil && attempt.Error.Code == "session.policy-invalid" {
			out := Failure("session.policy-invalid", "Enabled-session selection changed; no context assembled.", false)
			out.SessionSync = &attempt
			return out
		}
		prompt := ""
		if in.Prompt != nil {
			prompt = *in.Prompt
		}
		packet := memorycontext.Build(a.service, prompt, in.Prompt != nil, memorycontext.SessionOrientation+" "+attempt.Summary())
		packet.Warning = strings.ReplaceAll(packet.Warning, "no memory was changed", "no semantic content was saved; synchronization status is separate")
		packet.Synchronization = &attempt
		return Success(packet)
	}
	if name == "memory_sync" {
		var in SyncInput
		if strictjson.Decode(data, &in, MaxInputBytes) != nil {
			return Failure("input.invalid", "Invalid synchronization request.", false)
		}
		if seconds := number(in.TimeoutSeconds, 3); seconds < 1 || seconds > 30 {
			return Failure("input.invalid", "Synchronization timeout requires 1–30 seconds.", false)
		}
		deliveryCtx, cancel := context.WithTimeout(ctx, time.Duration(number(in.TimeoutSeconds, 3))*time.Second)
		defer cancel()
		attempt := a.session.Attempt(deliveryCtx, sessionsync.Boundary{Kind: "manual"})
		out := Success(attempt.Status)
		if attempt.Error != nil {
			out = Failure(attempt.Error.Code, attempt.Error.Message, attempt.Attempted)
			out.Error.SyncStatus = attempt.Status
		}
		out.SessionSync = &attempt
		return out
	}
	originalName := name
	deliveryBudget := sessionsync.Budget
	if name == "memory_remember_and_sync" {
		var in RememberAndSyncInput
		if strictjson.Decode(data, &in, MaxInputBytes) != nil || number(in.TimeoutSeconds, 3) < 1 || number(in.TimeoutSeconds, 3) > 30 {
			return Failure("input.invalid", "Invalid combined save request.", false)
		}
		deliveryBudget = min(deliveryBudget, time.Duration(number(in.TimeoutSeconds, 3))*time.Second)
		name = "memory_remember"
		data, _ = json.Marshal(in.Record)
	} else if name == "memory_journal_append_and_sync" {
		var in JournalAndSyncInput
		if strictjson.Decode(data, &in, MaxInputBytes) != nil || number(in.TimeoutSeconds, 3) < 1 || number(in.TimeoutSeconds, 3) > 30 {
			return Failure("input.invalid", "Invalid combined journal request.", false)
		}
		deliveryBudget = min(deliveryBudget, time.Duration(number(in.TimeoutSeconds, 3))*time.Second)
		name = "memory_journal_append"
		data, _ = json.Marshal(in.Entry)
	}
	out := legacy.Call(ctx, name, data)
	if !sessionMutation(originalName) || (!out.OK && (out.Error == nil || !out.Error.WriteMayHaveOccurred)) {
		return out
	}
	// This runs after a successful or possibly partial publication. Cancellation
	// suppresses further work while preserving the actual published receipt.
	deliveryCtx, cancel := context.WithTimeout(ctx, deliveryBudget)
	defer cancel()
	attempt := a.session.Attempt(deliveryCtx, sessionsync.Boundary{Kind: "write"})
	out.SessionSync = &attempt
	if out.OK {
		if saved, ok := out.Result.(Receipt); ok {
			saved.Synchronization = sessionState(attempt)
			out.Result = saved
			if originalName == "memory_remember_and_sync" || originalName == "memory_journal_append_and_sync" {
				delivery := DeliveryEnvelope{ProtocolVersion: ProtocolVersion, OK: attempt.Error == nil, Result: attempt.Status}
				if attempt.Error != nil {
					delivery.Result = nil
					delivery.Error = &MemoryError{Code: attempt.Error.Code, Message: attempt.Error.Message, SyncStatus: attempt.Status, WriteMayHaveOccurred: true, InspectBeforeRetry: true}
				}
				out.Result = SaveAndSyncResult{Saved: saved, Delivery: delivery}
			}
		}
		if saved, ok := out.Result.(memory.VisibilityReceipt); ok {
			saved.Synchronization = sessionState(attempt)
			out.Result = saved
		}
	}
	return out
}
func sessionState(a sessionsync.Attempt) string {
	if a.Status != nil && a.Status.State != "" {
		return a.Status.State
	}
	return "pending"
}

func init() { sessionCatalog.CLIOnly = true }
