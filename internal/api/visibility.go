package api

import (
	"context"
	"errors"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// Only visibility tools advertise this receipt-bearing error schema; unrelated
// memory tools retain their compact common error contract.
type VisibilityError struct {
	MemoryError
	VisibilityResult *memory.VisibilityReceipt `json:"visibility_result,omitempty"`
}

type visibilityFailure struct {
	err    error
	result memory.VisibilityReceipt
}

func (e *visibilityFailure) Error() string {
	return "Visibility decision stopped; inspect its receipt and current history before retrying."
}
func (e *visibilityFailure) Unwrap() error { return e.err }

func visibilityDecision(action string) func(context.Context, *memory.Service, memory.VisibilityWrite) (memory.VisibilityReceipt, error) {
	return func(ctx context.Context, s *memory.Service, in memory.VisibilityWrite) (memory.VisibilityReceipt, error) {
		r, err := s.ChangeVisibility(ctx, action, in)
		if err != nil {
			return r, &visibilityFailure{err: err, result: r}
		}
		return r, nil
	}
}

var visibilityOperations = []Operation{
	operation("memory_visibility_history", "Inspect a record's current content/visibility heads and paged visibility decisions. Metadata includes reasons and authorship; content history is separate. Read before withdrawing or restoring. Pages may shift after writes.", true, func(_ context.Context, s *memory.Service, in HistoryInput) (memory.VisibilityHistory, error) {
		return s.VisibilityHistory(in.RecordID, in.Offset, number(in.Limit, 5))
	}),
	operation("memory_withdraw", "Explicitly withhold a record from ordinary recall without deleting evidence. Requires format2, current expected content and visibility heads, and reason. Does not erase history, journals, foundlings, offline copies or existing model context. Saves locally only; sync separately when allowed.", false, visibilityDecision("withdraw")),
	operation("memory_restore", "Explicitly restore reviewed record content using current expected content and visibility heads and a reason. Resolves visibility heads, not conflicting content revisions. Requires format2. Saves locally only; sync separately when allowed.", false, visibilityDecision("restore")),
}

func visibilityFailureEnvelope(e *visibilityFailure) Envelope {
	code, message := "visibility.failed", e.Error()
	switch {
	case errors.Is(e.err, context.Canceled), errors.Is(e.err, context.DeadlineExceeded):
		code = "operation.cancelled"
	case errors.Is(e.err, memory.ErrVisibilityUpgradeRequired):
		code, message = "memory.upgrade_required", memory.ErrVisibilityUpgradeRequired.Error()
	case errors.Is(e.err, memory.ErrStaleHeads):
		code, message = "memory.stale_heads", memory.ErrStaleHeads.Error()
	case errors.Is(e.err, memory.ErrWriterBusy):
		code, message = "store.busy", "Another writer holds the signet lock; inspect current heads after it finishes."
	}
	out := Failure(code, message, e.result.WriteMayHaveOccurred)
	out.Error.VisibilityResult = &e.result
	return out
}
