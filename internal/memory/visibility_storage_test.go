package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatOneCannotIgnoreVisibilityEvidence(t *testing.T) {
	s := fixtureStore(t)
	r := revision("revision-first")
	if err := s.Put(r); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.Root, "memory/visibility", r.RecordID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	e := VisibilityEvent{Version: 1, ID: "visibility-first", RecordID: r.RecordID, Action: "withdraw", Observed: []string{r.ID}, Parents: []string{}, Reason: "Test withdrawal", RecordedAt: r.RecordedAt, Authorship: r.Authorship}
	if err := writeNewJSON(filepath.Join(dir, e.ID+".json"), e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recall(Query{Scope: r.Scope}, time.Now()); err == nil {
		t.Fatal("old format ignored visibility evidence")
	}
	if _, err := s.Scopes(); err == nil {
		t.Fatal("scope inventory ignored visibility evidence")
	}
	if _, err := s.History(r.RecordID); err == nil {
		t.Fatal("history ignored incompatible format")
	}
	if err := s.Validate(); err == nil {
		t.Fatal("accepted mixed format")
	}
	if err := s.Put(revision("revision-second", r.ID)); err == nil {
		t.Fatal("wrote through mixed format")
	}
}
