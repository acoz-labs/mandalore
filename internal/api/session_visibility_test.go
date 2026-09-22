package api

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/sessionsync"
)

func assertSessionVisibilityNotice(t *testing.T, r memory.VisibilityReceipt) {
	t.Helper()
	for _, wanted := range []string{"Inspect session_sync", "Evidence is preserved, not erased", "Previously read context and offline copies cannot be revoked"} {
		if !strings.Contains(r.Notice, wanted) {
			t.Fatal("missing receipt guidance", r)
		}
	}
	for _, forbidden := range []string{"not requested", "not-requested", "delivery requested"} {
		if strings.Contains(r.Notice, forbidden) {
			t.Fatal("contradictory transport notice", r)
		}
	}
}

func TestSessionVisibilityPendingReceipts(t *testing.T) {
	a, r := sessionVisibilityFixture(t)
	sessionGit(t, "-C", a.service.Root(), "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	for _, name := range []string{"memory_withdraw", "memory_restore"} {
		h := inlineCall(t, a, "memory_visibility_history", HistoryInput{RecordID: r.RecordID}).Result.(memory.VisibilityHistory)
		out := inlineCall(t, a, name, memory.VisibilityWrite{RecordID: r.RecordID, ContentHeads: h.State.ContentHeads, VisibilityHeads: h.State.VisibilityHeads, Reason: "Explicit decision"})
		receipt := out.Result.(memory.VisibilityReceipt)
		if out.SessionSync == nil || !out.SessionSync.Attempted || out.SessionSync.Status == nil || out.SessionSync.Status.Delivered || !receipt.DurableLocally || receipt.Synchronization != sessionState(*out.SessionSync) {
			t.Fatal(out, receipt)
		}
		assertSessionVisibilityNotice(t, receipt)
	}
}

func TestSessionVisibilityPartialPublicationPreservesEvidence(t *testing.T) {
	for _, scenario := range []struct {
		name                        string
		offline, cancelled, durable bool
	}{
		{name: "delivered after ambiguous publication"},
		{name: "pending after ambiguous publication", offline: true},
		{name: "cancelled after ambiguous publication", cancelled: true},
		{name: "cancelled after durable publication", cancelled: true, durable: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a, r := sessionVisibilityFixture(t)
			h := inlineCall(t, a, "memory_visibility_history", HistoryInput{RecordID: r.RecordID}).Result.(memory.VisibilityHistory)
			// Publish a real event, then inject the receipt/error that a failed final
			// durability check or cancellation returns to the session delivery layer.
			local, err := a.service.ChangeVisibility(context.Background(), "withdraw", memory.VisibilityWrite{RecordID: r.RecordID, ContentHeads: h.State.ContentHeads, VisibilityHeads: h.State.VisibilityHeads, Reason: "Explicit decision"})
			if err != nil {
				t.Fatal(err)
			}
			local.DurableLocally = scenario.durable
			if !scenario.durable {
				local.State = nil
				local.Notice = "Visibility publication attempted; inspect this event ID before retrying. Delivery was not requested."
			}
			original := local
			publicationErr := errors.New("injected publication durability failure")
			ctx := context.Background()
			if scenario.cancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				publicationErr = context.Canceled
			}
			if scenario.offline {
				sessionGit(t, "-C", a.service.Root(), "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
			}
			failed := visibilityFailureEnvelope(&visibilityFailure{err: publicationErr, result: local})
			originalError := failed.Error.MemoryError
			out := a.deliverSessionMutation(ctx, "memory_withdraw", sessionsync.Budget, failed)
			if out.OK || out.Error == nil || !reflect.DeepEqual(out.Error.MemoryError, originalError) || out.SessionSync == nil || out.Error.VisibilityResult == nil {
				t.Fatal(out)
			}
			projected := *out.Error.VisibilityResult
			assertSessionVisibilityNotice(t, projected)
			if projected.Synchronization != sessionState(*out.SessionSync) {
				t.Fatal("nested transport differs from attempt", out)
			}
			if !scenario.offline && !scenario.cancelled && (out.SessionSync.Status == nil || !out.SessionSync.Status.Delivered) {
				t.Fatal("fixture delivery failed", out)
			}
			if (scenario.offline || scenario.cancelled) && out.SessionSync.Status != nil && out.SessionSync.Status.Delivered {
				t.Fatal("claimed unavailable delivery", out)
			}
			if !scenario.durable && strings.Contains(projected.Notice, "saved locally") {
				t.Fatal("uncertain durability upgraded", projected)
			}
			projected.Synchronization, projected.Notice = original.Synchronization, original.Notice
			if !reflect.DeepEqual(projected, original) {
				t.Fatal("publication evidence changed", projected, original)
			}
			history, err := a.service.VisibilityHistory(r.RecordID, 0, 5)
			if err != nil || len(history.Events.Items) != 1 || history.Events.Items[0].ID != local.EventID {
				t.Fatal("publication lost or repeated", history, err)
			}
		})
	}
}
