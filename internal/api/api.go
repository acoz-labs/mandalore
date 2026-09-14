// Package api owns the shared CLI/MCP operation contract, not a second memory engine.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/migration"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
	"github.com/google/jsonschema-go/jsonschema"
)

const ProtocolVersion = 1
const MaxInputBytes = 32768
const MaxOutputBytes = 65536

// MemoryError is the bound memory protocol's error shape. Machine installation
// details must not inflate every memory tool's model-facing schema.
type MemoryError struct {
	SyncStatus           *signetsync.Status `json:"sync_status,omitempty"`
	Code                 string             `json:"code"`
	Message              string             `json:"message"`
	Retryable            bool               `json:"retryable"`
	WriteMayHaveOccurred bool               `json:"write_may_have_occurred"`
	InspectBeforeRetry   bool               `json:"inspect_before_retry"`
}
type Error struct {
	MemoryError
	FoundlingResult  *FoundlingMutationResult    `json:"foundling_result,omitempty"`
	ConnectionResult *install.Result             `json:"connection_result,omitempty"`
	ConnectionReport *install.Report             `json:"connection_report,omitempty"`
	MigrationResult  *migration.Result           `json:"migration_result,omitempty"`
	ReleaseResult    *distribution.InstallResult `json:"release_result,omitempty"`
}
type Envelope struct {
	ProtocolVersion int    `json:"protocol_version"`
	OK              bool   `json:"ok"`
	Result          any    `json:"result,omitempty"`
	Error           *Error `json:"error,omitempty"`
}

func Failure(code, message string, mayWrite bool) Envelope {
	return Envelope{ProtocolVersion: ProtocolVersion, Error: &Error{MemoryError: MemoryError{Code: code, Message: message, WriteMayHaveOccurred: mayWrite, InspectBeforeRetry: mayWrite}}}
}
func Success(result any) Envelope {
	return Envelope{ProtocolVersion: ProtocolVersion, OK: true, Result: result}
}
func ExitCode(v Envelope) int {
	if v.OK {
		return 0
	}
	switch v.Error.Code {
	case "input.invalid", "operation.unknown", "operation.read_only":
		return 2
	case "operation.cancelled":
		return 130
	case "store.busy":
		return 3
	default:
		return 1
	}
}

type Receipt struct {
	SignetID        string `json:"signet_id"`
	ID              string `json:"id"`
	RecordID        string `json:"record_id,omitempty"`
	DurableLocally  bool   `json:"durable_locally"`
	Synchronization string `json:"synchronization"`
}
type RecallInput struct {
	Query       string        `json:"query,omitempty" jsonschema:"Words or identifiers; empty lists current memory in the selected scope (maximum 2048 bytes)."`
	Scope       *memory.Scope `json:"scope,omitempty" jsonschema:"Explicit stored scope; omitted selects this signet-wide scope only."`
	Limit       *int          `json:"limit,omitempty" jsonschema:"Default 5; range 1–50."`
	BudgetBytes *int          `json:"budget_bytes,omitempty" jsonschema:"Default 8192; range 1024–32768 for the compact result payload."`
}
type PageInput struct {
	Offset int  `json:"offset,omitempty"`
	Limit  *int `json:"limit,omitempty" jsonschema:"Default 5; range 1–50."`
}
type HistoryInput struct {
	RecordID string `json:"record_id"`
	Offset   int    `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty" jsonschema:"Default 5; range 1–50."`
}
type JournalInput struct {
	Query string `json:"query,omitempty"`
	Limit *int   `json:"limit,omitempty"`
}
type JournalWrite struct {
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
}
type Inspection struct {
	SignetID string `json:"signet_id"`
	Root     string `json:"root"`
	Healthy  bool   `json:"healthy"`
	Notice   string `json:"notice"`
}
type Operation struct {
	CLIOnly         bool               `json:"cli_only"`
	Network         bool               `json:"network"`
	RequiresBinding bool               `json:"requires_binding"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	ReadOnly        bool               `json:"read_only"`
	Idempotent      bool               `json:"idempotent"`
	InputSchema     *jsonschema.Schema `json:"input_schema"`
	OutputSchema    *jsonschema.Schema `json:"output_schema"`
	invoke          func(context.Context, *memory.Service, []byte) (any, error)
}

func operation[I, O any](name, description string, readOnly bool, fn func(context.Context, *memory.Service, I) (O, error)) Operation {
	in, err := strictjson.Schema(new(I))
	if err != nil {
		panic("invalid built-in input schema")
	}
	out, err := strictjson.Schema(new(O))
	if err != nil {
		panic("invalid built-in output schema")
	}
	return Operation{Name: name, Description: description, ReadOnly: readOnly, Idempotent: readOnly, RequiresBinding: true, InputSchema: in, OutputSchema: out,
		invoke: func(ctx context.Context, s *memory.Service, data []byte) (any, error) {
			var input I
			if err := strictjson.Decode(data, &input, MaxInputBytes); err != nil {
				return nil, err
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return fn(ctx, s, input)
		}}
}
func number(n *int, fallback int) int {
	if n == nil {
		return fallback
	}
	return *n
}

var operations = []Operation{
	operation("memory_recall", "Recall scoped current evidence. Conflicting heads are not guidance; empty/truncated results do not prove absence.", true, func(_ context.Context, s *memory.Service, in RecallInput) (memory.RecallPacket, error) {
		return s.Recall(in.Query, in.Scope, number(in.Limit, 5), number(in.BudgetBytes, 8192))
	}),
	operation("memory_scopes", "List stable routing scopes without promoting their content into guidance.", true, func(_ context.Context, s *memory.Service, in PageInput) (memory.Page[memory.ScopeInfo], error) {
		return s.ScopePage(in.Offset, number(in.Limit, 5))
	}),
	operation("memory_history", "Inspect provenance, correction reasons and all competing revisions. Page offsets may shift after concurrent writes.", true, func(_ context.Context, s *memory.Service, in HistoryInput) (memory.Page[memory.Revision], error) {
		return s.HistoryPage(in.RecordID, in.Offset, number(in.Limit, 5))
	}),
	operation("memory_journal", "Search recent semantic journal entries, not authoritative current facts or raw transcripts.", true, func(_ context.Context, s *memory.Service, in JournalInput) (memory.Page[memory.JournalEntry], error) {
		return s.JournalPage(in.Query, number(in.Limit, 5))
	}),
	operation("memory_remember", "Store a confirmed fact, preference, decision, procedure, project-state, entity, commitment or research-claim. Basis: user-direction, observation, inference or import. Recall before adding duplicates; corrections name record_id and predecessor revision IDs. Never store secrets.", false, func(_ context.Context, s *memory.Service, in memory.Write) (Receipt, error) {
		r, err := s.Remember(in)
		if err != nil {
			return Receipt{}, err
		}
		return Receipt{SignetID: s.ID(), ID: r.ID, RecordID: r.RecordID, DurableLocally: true, Synchronization: "not-requested"}, nil
	}),
	operation("memory_journal_append", "Append a concise account of actual work and decisions, not transcripts or secrets. Do not journal a read-only task.", false, func(_ context.Context, s *memory.Service, in JournalWrite) (Receipt, error) {
		r, err := s.AppendJournal(in.Kind, in.Summary)
		if err != nil {
			return Receipt{}, err
		}
		return Receipt{SignetID: s.ID(), ID: r.ID, DurableLocally: true, Synchronization: "not-requested"}, nil
	}),
	operation("memory_inspect", "Read-only signet structure validation; no remote, credentials, native plugin or model-behavior checks.", true, func(_ context.Context, s *memory.Service, _ struct{}) (Inspection, error) {
		err := s.Validate()
		return Inspection{SignetID: s.ID(), Root: s.Root(), Healthy: err == nil, Notice: "Read-only structure checks only; no synchronization, authentication or native integration verification."}, err
	}),
}

func Catalog() []Operation {
	return append(append(append(append(append(append(append([]Operation(nil), operations...), administration...), synchronization...), connections...), migrations...), foundlingOperations...), releases...)
}

type API struct {
	service  *memory.Service
	ReadOnly bool
}

func New(service *memory.Service, readOnly bool) *API {
	return &API{service: service, ReadOnly: readOnly}
}

func (a *API) Call(ctx context.Context, name string, data []byte) Envelope {
	if err := ctx.Err(); err != nil {
		return Failure("operation.cancelled", "Operation cancelled before execution.", false)
	}
	for _, op := range Catalog() {
		if op.Name != name {
			continue
		}
		if a.ReadOnly && !op.ReadOnly {
			return Failure("operation.read_only", "Mutations are disabled for this connection.", false)
		}
		if op.RequiresBinding && a.service == nil {
			return Failure("binding.invalid", "An explicit valid signet binding is required.", false)
		}
		v, err := op.invoke(ctx, a.service, data)
		if err != nil {
			var release *releaseFailure
			if errors.As(err, &release) {
				code := "release.failed"
				if errors.Is(err, distribution.ErrNoRelease) {
					code = "release.unavailable"
				}
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					code = "operation.cancelled"
				}
				mayWrite := release.result != nil && release.result.DestinationChanged
				out := Failure(code, release.Error(), mayWrite)
				out.Error.ReleaseResult = release.result
				if release.result != nil && release.result.Pending != "" {
					out.Error.InspectBeforeRetry = true
				}
				return out
			}
			var reference *foundlingFailure
			if errors.As(err, &reference) {
				return foundlingFailureEnvelope(reference)
			}
			var migration *migrationFailure
			if errors.As(err, &migration) {
				code := "migration.failed"
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					code = "operation.cancelled"
				}
				mayWrite := migration.result != nil && migration.result.Phase != "preflight"
				out := Failure(code, migration.Error(), mayWrite)
				out.Error.MigrationResult = migration.result
				return out
			}
			var connection *connectionFailure
			if errors.As(err, &connection) {
				code := "connection.failed"
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					code = "operation.cancelled"
				}
				out := Failure(code, connection.Error(), connection.result != nil)
				out.Error.ConnectionResult, out.Error.ConnectionReport = connection.result, connection.report
				return out
			}
			var stopped *signetsync.Failure
			if errors.As(err, &stopped) && !errors.Is(err, memory.ErrWriterBusy) {
				code := "sync.failed"
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					code = "operation.cancelled"
				}
				out := Failure(code, stopped.Error(), true)
				out.Error.SyncStatus = &stopped.Status
				return out
			}
			switch {
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				return Failure("operation.cancelled", "Operation cancelled before execution.", false)
			case errors.Is(err, memory.ErrWriterBusy):
				out := Failure("store.busy", "Another writer holds the signet lock; retry after it finishes.", false)
				out.Error.Retryable = true
				return out
			case errors.Is(err, memory.ErrIdentityChanged):
				return Failure("binding.invalid", "The signet identity changed; inspect and explicitly rebind.", false)
			case errors.Is(err, strictjson.ErrInvalid):
				return Failure("input.invalid", strictjson.ErrInvalid.Error(), false)
			}
			var pathError *fs.PathError
			var linkError *os.LinkError
			var systemError syscall.Errno
			if errors.As(err, &pathError) || errors.As(err, &linkError) || errors.As(err, &systemError) {
				return Failure("operation.io", "Filesystem operation failed; inspect the selected signet before retrying a write.", !op.ReadOnly)
			}
			// Only recheck on failure: successful reads/writes already validate
			// their relevant records. Do not blame malformed stored data on the
			// caller or expose record bodies through validation diagnostics.
			if op.RequiresBinding && a.service.Validate() != nil {
				return Failure("store.invalid", "Selected signet failed structural validation; inspect its files and history before retrying.", !op.ReadOnly)
			}
			return Failure("input.invalid", "Memory validation failed; inspect the input schema and selected signet structure.", false)
		}
		// Reads can report cancellation after completing. Published writes return
		// their receipt; cancellation does not roll them back.
		if op.ReadOnly && ctx.Err() != nil {
			return Failure("operation.cancelled", "Read cancelled.", false)
		}
		envelope := Success(v)
		encoded, err := json.Marshal(envelope)
		if err != nil || len(encoded) > MaxOutputBytes {
			return Failure("output.invalid", "Response could not be represented within the output limit.", !op.ReadOnly)
		}
		return envelope
	}
	return Failure("operation.unknown", "Unknown operation; inspect the operation catalog.", false)
}
