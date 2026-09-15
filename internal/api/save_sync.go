package api

import (
	"context"
	"errors"

	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

type RememberAndSyncInput struct {
	Record         memory.Write `json:"record" jsonschema:"Confirmed knowledge to save locally before delivery."`
	TimeoutSeconds *int         `json:"timeout_seconds,omitempty" jsonschema:"Delivery budget only: default 3 seconds; range 1–30."`
}

type JournalAndSyncInput struct {
	Entry          JournalWrite `json:"entry" jsonschema:"Useful semantic journal entry to save locally before delivery."`
	TimeoutSeconds *int         `json:"timeout_seconds,omitempty" jsonschema:"Delivery budget only: default 3 seconds; range 1–30."`
}

// DeliveryEnvelope is the typed memory_sync outcome, separate from local saving.
type DeliveryEnvelope struct {
	ProtocolVersion int                `json:"protocol_version"`
	OK              bool               `json:"ok"`
	Result          *signetsync.Status `json:"result,omitempty"`
	Error           *MemoryError       `json:"error,omitempty"`
}

type SaveAndSyncResult struct {
	Saved    Receipt          `json:"saved"`
	Delivery DeliveryEnvelope `json:"delivery"`
}

func remember(_ context.Context, s *memory.Service, in memory.Write) (Receipt, error) {
	r, err := s.Remember(in)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{SignetID: s.ID(), ID: r.ID, RecordID: r.RecordID, DurableLocally: true, Synchronization: "not-requested"}, nil
}

func appendJournal(_ context.Context, s *memory.Service, in JournalWrite) (Receipt, error) {
	r, err := s.AppendJournal(in.Kind, in.Summary)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{SignetID: s.ID(), ID: r.ID, DurableLocally: true, Synchronization: "not-requested"}, nil
}

func saveAndSync(ctx context.Context, s *memory.Service, timeout *int, save func() (Receipt, error)) (SaveAndSyncResult, error) {
	seconds := number(timeout, 3)
	if seconds < 1 || seconds > 30 {
		return SaveAndSyncResult{}, errors.New("sync timeout requires 1–30 seconds")
	}
	if err := ctx.Err(); err != nil {
		return SaveAndSyncResult{}, err
	}
	saved, err := save()
	if err != nil {
		return SaveAndSyncResult{}, err
	}
	// From here onward, delivery failure must not hide a published save or ask the
	// caller to repeat it. The same synchronizer reacquires the normal writer lock.
	out := SaveAndSyncResult{Saved: saved, Delivery: DeliveryEnvelope{ProtocolVersion: ProtocolVersion}}
	out.Saved.Synchronization = "pending"
	status, err := syncMemory(ctx, s, SyncInput{TimeoutSeconds: &seconds})
	if err != nil {
		failure := New(s, false).failure(Operation{RequiresBinding: true}, err)
		out.Delivery.Error = &failure.Error.MemoryError
		return out, nil
	}
	out.Delivery.OK, out.Delivery.Result = true, &status
	out.Saved.Synchronization = status.State
	return out, nil
}

var saveAndDelivery = []Operation{
	operation("memory_remember_and_sync", "Save confirmed knowledge locally, then attempt bounded delivery once. Use only when saving AND synchronization are allowed; otherwise use local-only memory_remember. Outer success confirms the save, not delivery: inspect saved and delivery separately. Never repeat the save to retry delivery.", false, func(ctx context.Context, s *memory.Service, in RememberAndSyncInput) (SaveAndSyncResult, error) {
		return saveAndSync(ctx, s, in.TimeoutSeconds, func() (Receipt, error) { return remember(ctx, s, in.Record) })
	}),
	operation("memory_journal_append_and_sync", "Save a useful semantic journal locally, then attempt bounded delivery once. Use only when journaling AND synchronization are allowed; otherwise use local-only memory_journal_append. Outer success confirms the save, not delivery: inspect saved and delivery separately. Never repeat the save to retry delivery.", false, func(ctx context.Context, s *memory.Service, in JournalAndSyncInput) (SaveAndSyncResult, error) {
		return saveAndSync(ctx, s, in.TimeoutSeconds, func() (Receipt, error) { return appendJournal(ctx, s, in.Entry) })
	}),
}

func init() {
	for i := range saveAndDelivery {
		saveAndDelivery[i].Network = true
	}
}
