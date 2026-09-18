package memory

import (
	"context"
	"errors"
	"testing"
)

func TestVisibilityServiceWithdrawRestoreAndStaleDecisions(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-first")
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	s := visibilityStore(t, old)
	service, err := OpenService(s.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	h, err := service.VisibilityHistory(r.RecordID, 0, 10)
	if err != nil || h.State.State != "visible" || len(h.Events.Items) != 0 {
		t.Fatal(h, err)
	}
	in := VisibilityWrite{RecordID: r.RecordID, ContentHeads: h.State.ContentHeads, VisibilityHeads: h.State.VisibilityHeads, Reason: "Stop recalling this evidence"}
	w, err := service.ChangeVisibility(context.Background(), "withdraw", in)
	if err != nil || !w.DurableLocally || w.State.State != "withdrawn" || w.Synchronization != "not-requested" {
		t.Fatal(w, err)
	}
	if p, err := service.Recall("", &r.Scope, 5, 8192); err != nil || len(p.Current) != 0 {
		t.Fatal(p, err)
	}
	if stale, err := service.ChangeVisibility(context.Background(), "restore", in); !errors.Is(err, ErrStaleHeads) || stale.WriteMayHaveOccurred {
		t.Fatal("stale restore accepted", stale, err)
	}
	h, err = service.VisibilityHistory(r.RecordID, 0, 10)
	if err != nil || len(h.Events.Items) != 1 || h.Events.Items[0].ID != w.EventID || h.Events.Items[0].Authorship != r.Authorship {
		t.Fatal(h, err)
	}
	in.ContentHeads, in.VisibilityHeads = h.State.ContentHeads, h.State.VisibilityHeads
	in.Reason = "Explicitly restore current evidence"
	restored, err := service.ChangeVisibility(context.Background(), "restore", in)
	if err != nil || !restored.DurableLocally || restored.State.State != "visible" {
		t.Fatal(restored, err)
	}
	if p, err := service.Recall("", &r.Scope, 5, 8192); err != nil || len(p.Current) != 1 {
		t.Fatal(p, err)
	}
	if history, err := service.History(r.RecordID); err != nil || len(history) != 1 {
		t.Fatal("visibility rewrote content", history, err)
	}
}

func TestVisibilityServiceRefusesLegacyInvalidAndCanceledWrites(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-first")
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	service, err := OpenService(old.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	in := VisibilityWrite{RecordID: r.RecordID, ContentHeads: []string{r.ID}, VisibilityHeads: []string{}, Reason: "Explicit decision"}
	if got, err := service.ChangeVisibility(context.Background(), "withdraw", in); !errors.Is(err, ErrVisibilityUpgradeRequired) || got.WriteMayHaveOccurred {
		t.Fatal(got, err)
	}
	s := visibilityStore(t, old)
	service, err = OpenService(s.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := service.ChangeVisibility(ctx, "withdraw", in); !errors.Is(err, context.Canceled) || got.WriteMayHaveOccurred {
		t.Fatal(got, err)
	}
	for _, bad := range []VisibilityWrite{
		{RecordID: r.RecordID, ContentHeads: []string{r.ID}, Reason: "Missing explicit visibility heads"},
		{RecordID: r.RecordID, ContentHeads: []string{r.ID, r.ID}, VisibilityHeads: []string{}, Reason: "Duplicate heads"},
		{RecordID: "record-missing", ContentHeads: []string{r.ID}, VisibilityHeads: []string{}, Reason: "Wrong record"},
	} {
		if got, err := service.ChangeVisibility(context.Background(), "withdraw", bad); err == nil || got.WriteMayHaveOccurred {
			t.Fatal(got, err)
		}
	}
	h, err := service.VisibilityHistory(r.RecordID, 0, 10)
	if err != nil || len(h.Events.Items) != 0 {
		t.Fatal("rejected operation wrote event", h, err)
	}
}

func TestVisibilityPublicationFailureAndLateCancellationRemainInspectable(t *testing.T) {
	for _, mode := range []string{"ambiguous-publication", "late-cancellation"} {
		t.Run(mode, func(t *testing.T) {
			old := fixtureStore(t)
			r := revision("revision-first")
			if err := old.Put(r); err != nil {
				t.Fatal(err)
			}
			s := visibilityStore(t, old)
			service, err := OpenService(s.Root, r.Authorship)
			if err != nil {
				t.Fatal(err)
			}
			in := VisibilityWrite{RecordID: r.RecordID, ContentHeads: []string{r.ID}, VisibilityHeads: []string{}, Reason: "Explicit decision"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			got, err := service.changeVisibility(ctx, "withdraw", in, func(path string, value any) error {
				if err := writeNewJSON(path, value); err != nil {
					return err
				}
				if mode == "late-cancellation" {
					cancel()
					return nil
				}
				return errors.New("synthetic interruption after publication")
			})
			if err == nil || !got.WriteMayHaveOccurred || got.EventID == "" || got.DurableLocally != (mode == "late-cancellation") {
				t.Fatal(got, err)
			}
			h, err := service.VisibilityHistory(r.RecordID, 0, 10)
			if err != nil || len(h.Events.Items) != 1 || h.Events.Items[0].ID != got.EventID || h.State.State != "withdrawn" {
				t.Fatal(h, err)
			}
			if retried, err := service.ChangeVisibility(context.Background(), "withdraw", in); !errors.Is(err, ErrStaleHeads) || retried.WriteMayHaveOccurred {
				t.Fatal("blind retry wrote again", retried, err)
			}
		})
	}
}

func TestVisibilityRestoreCannotIgnoreNewContent(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-first")
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	s := visibilityStore(t, old)
	service, err := OpenService(s.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	in := VisibilityWrite{RecordID: r.RecordID, ContentHeads: []string{r.ID}, VisibilityHeads: []string{}, Reason: "Explicit decision"}
	w, err := service.ChangeVisibility(context.Background(), "withdraw", in)
	if err != nil {
		t.Fatal(err)
	}
	in.VisibilityHeads = w.State.VisibilityHeads
	next := revision("revision-correction", r.ID)
	if err := s.Put(next); err != nil {
		t.Fatal(err)
	}
	if got, err := service.ChangeVisibility(context.Background(), "restore", in); !errors.Is(err, ErrStaleHeads) || got.WriteMayHaveOccurred {
		t.Fatal("restored unseen correction", got, err)
	}
	if p, err := service.Recall("", &r.Scope, 5, 8192); err != nil || len(p.Current) != 0 {
		t.Fatal("correction restored withdrawn record", p, err)
	}
	h, err := service.VisibilityHistory(r.RecordID, 0, 10)
	if err != nil || len(h.Events.Items) != 1 || len(h.State.ContentHeads) != 1 || h.State.ContentHeads[0] != next.ID {
		t.Fatal(h, err)
	}
	in.ContentHeads = h.State.ContentHeads
	if got, err := service.ChangeVisibility(context.Background(), "restore", in); err != nil || !got.DurableLocally || got.State.State != "visible" {
		t.Fatal(got, err)
	}
}
