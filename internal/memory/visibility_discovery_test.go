package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWithheldDiscoveryIsScopedMetadataOnlyAndFresh(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-discovery")
	r.Summary, r.Body = "Fictional route", "BODY_CANARY"
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	store := visibilityStore(t, old)
	s, err := OpenService(store.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := s.WithheldRecords("", &r.Scope, 0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal(p, err)
	}
	h, err := s.VisibilityHistory(r.RecordID, 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.ChangeVisibility(context.Background(), "withdraw", VisibilityWrite{RecordID: r.RecordID, ContentHeads: h.State.ContentHeads, VisibilityHeads: h.State.VisibilityHeads, Reason: "REASON_CANARY"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.WithheldRecords("FICTIONAL route", &r.Scope, 0, 5)
	if err != nil || len(p.Items) != 1 || p.Items[0].RecordID != r.RecordID || p.Items[0].State != "withdrawn" {
		t.Fatal(p, err)
	}
	encoded, _ := json.Marshal(p)
	for _, hidden := range []string{"BODY_CANARY", "REASON_CANARY", "Fictional route", r.Authorship.Actor} {
		if hidden != "" && strings.Contains(string(encoded), hidden) {
			t.Fatal("discovery disclosed nonrouting data", hidden)
		}
	}
	if p, err := s.WithheldRecords("BODY_CANARY", &r.Scope, 0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal("body search not allowed", p, err)
	}
	other := Scope{Kind: "project", ID: "different-project"}
	if p, err := s.WithheldRecords("", &other, 0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal(p, err)
	}
	if _, err := s.WithheldRecords(strings.Repeat("x", 2049), &r.Scope, 0, 5); err == nil {
		t.Fatal("oversized query accepted")
	}
	if _, err := s.WithheldRecords("", &r.Scope, -1, 5); err == nil {
		t.Fatal("invalid page accepted")
	}
	if _, err := s.ChangeVisibility(context.Background(), "restore", VisibilityWrite{RecordID: r.RecordID, ContentHeads: w.State.ContentHeads, VisibilityHeads: w.State.VisibilityHeads, Reason: "Explicit restore"}); err != nil {
		t.Fatal(err)
	}
	if p, err := s.WithheldRecords("", &r.Scope, 0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal("stale discovery", p, err)
	}
}

func TestWithheldDiscoveryUsesStructuralHeadsAndListsVisibilityConflicts(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-discovery-old")
	r.Summary = "Obsolete name"
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	store := visibilityStore(t, old)
	storeVisibilityFixture(t, store, "visibility-first", "withdraw", []string{}, []string{r.ID})
	next := revision("revision-discovery-next", r.ID)
	next.Summary = "Replacement route"
	if err := store.Put(next); err != nil {
		t.Fatal(err)
	}
	s, err := OpenService(store.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := s.WithheldRecords("Obsolete", &r.Scope, 0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal("superseded summary searched", p, err)
	}
	if p, err := s.WithheldRecords("Replacement", &r.Scope, 0, 5); err != nil || len(p.Items) != 1 {
		t.Fatal(p, err)
	}
	storeVisibilityFixture(t, store, "visibility-concurrent", "restore", []string{}, []string{r.ID})
	if p, err := s.WithheldRecords("", &r.Scope, 0, 5); err != nil || len(p.Items) != 1 || p.Items[0].State != "visibility-conflict" {
		t.Fatal(p, err)
	}
}
