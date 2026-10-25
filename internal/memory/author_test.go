package memory

import (
	"path/filepath"
	"testing"
)

func TestPerCallAuthorshipRecordsEnrolledClientDevice(t *testing.T) {
	s := fixture(t)
	device, err := s.EnrollClientDevice("Synthetic client (test), approved 2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	var d Device
	if err := readJSON(filepath.Join(s.Root(), "provenance/devices", device+".json"), &d); err != nil || d.Label != "Synthetic client (test), approved 2026-01-01" {
		t.Fatalf("device record %+v: %v", d, err)
	}
	client := Authorship{DeviceID: device, Actor: "Example user", Harness: "claude-code"}
	view, err := s.As(client)
	if err != nil {
		t.Fatal(err)
	}
	r, err := view.RememberIdempotent("client-c1\x00request", Write{Kind: "fact", Summary: "Route", Body: "Birch Loop", Basis: "user-direction", Reason: "Confirmed"})
	if err != nil || r.Authorship != client {
		t.Fatalf("revision authorship %+v: %v", r.Authorship, err)
	}
	var source Source
	if err := readJSON(filepath.Join(s.Root(), "memory/sources", r.Evidence.SourceRefs[0]+".json"), &source); err != nil || source.DeviceID != device {
		t.Fatalf("source device %q: %v", source.DeviceID, err)
	}
	j, err := view.AppendJournalIdempotent("client-c1\x00journal", "outcome", "Synthetic outcome.")
	if err != nil || j.Authorship != client {
		t.Fatalf("journal authorship %+v: %v", j.Authorship, err)
	}
	// The base service keeps its own author; views do not mutate it.
	base, err := s.Remember(Write{Kind: "fact", Summary: "Other", Body: "Cedar Path", Basis: "user-direction", Reason: "Confirmed"})
	if err != nil || base.Authorship.DeviceID != "device-test" {
		t.Fatalf("base authorship %+v: %v", base.Authorship, err)
	}
	if _, err := s.As(Authorship{DeviceID: "device-missing", Actor: "Example user", Harness: "test"}); err == nil {
		t.Fatal("unenrolled device accepted")
	}
	if _, err := s.As(Authorship{DeviceID: device, Actor: "Example user"}); err == nil {
		t.Fatal("missing harness accepted")
	}
	if _, err := s.EnrollClientDevice(""); err == nil {
		t.Fatal("empty label accepted")
	}
}

func TestIdempotentReplayIsBoundToAuthor(t *testing.T) {
	s := fixture(t)
	first, _ := s.EnrollClientDevice("Client one")
	second, _ := s.EnrollClientDevice("Client two")
	a, _ := s.As(Authorship{DeviceID: first, Actor: "Example user", Harness: "test"})
	b, _ := s.As(Authorship{DeviceID: second, Actor: "Example user", Harness: "test"})
	in := Write{Kind: "fact", Summary: "Route", Body: "Birch Loop", Basis: "user-direction", Reason: "Confirmed"}
	if _, err := a.RememberIdempotent("shared-key", in); err != nil {
		t.Fatal(err)
	}
	if _, err := b.RememberIdempotent("shared-key", in); err == nil {
		t.Fatal("another author's replay accepted")
	}
}

func TestRecallAllSpansScopesAndFiltersConflicts(t *testing.T) {
	s := fixture(t)
	project := &Scope{Kind: "project", ID: "project-amber"}
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Deploy target", Body: "Staging uses the heron cluster", Basis: "user-direction", Reason: "Confirmed", Scope: project}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Editor", Body: "Prefers the heron theme", Basis: "user-direction", Reason: "Confirmed"}); err != nil {
		t.Fatal(err)
	}
	first, err := s.Remember(Write{Kind: "decision", Summary: "Codename", Body: "Copper Finch", Basis: "user-direction", Reason: "Selected", Scope: &Scope{Kind: "account", ID: "account-example"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"Silver Lark", "Amber Lark"} {
		if _, err := s.Remember(Write{Kind: "decision", Summary: "Codename", Body: body, Basis: "user-direction", Reason: "Changed", Scope: &first.Scope, RecordID: first.RecordID, Supersedes: []string{first.ID}}); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.RecallAll("heron", 10)
	if err != nil || len(p.Current) != 2 || len(p.Conflicts) != 0 {
		t.Fatalf("cross-scope recall %+v: %v", p, err)
	}
	p, err = s.RecallAll("lark", 10)
	if err != nil || len(p.Current) != 0 || len(p.Conflicts) != 1 {
		t.Fatalf("relevant conflict %+v: %v", p, err)
	}
	if _, err := s.RecallAll("heron", 0); err == nil {
		t.Fatal("invalid limit accepted")
	}
}

func TestCachedReadsSeeEveryWrite(t *testing.T) {
	s := fixture(t)
	s.CacheReads()
	if p, err := s.RecallAll("heron", 5); err != nil || len(p.Current) != 0 {
		t.Fatalf("empty recall %+v: %v", p, err)
	}
	device, _ := s.EnrollClientDevice("Client")
	view, _ := s.As(Authorship{DeviceID: device, Actor: "Example user", Harness: "test"})
	first, err := view.Remember(Write{Kind: "fact", Summary: "Theme", Body: "Prefers the heron theme", Basis: "user-direction", Reason: "Confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	if p, err := s.RecallAll("heron", 5); err != nil || len(p.Current) != 1 {
		t.Fatalf("write through a view not seen %+v: %v", p, err)
	}
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Theme", Body: "Prefers the lark theme", Basis: "user-direction", Reason: "Changed", RecordID: first.RecordID, Supersedes: []string{first.ID}}); err != nil {
		t.Fatal(err)
	}
	if p, err := s.RecallAll("heron", 5); err != nil || len(p.Current) != 0 {
		t.Fatalf("superseded revision still recalled %+v: %v", p, err)
	}
	scopes, err := s.Scopes()
	if err != nil || len(scopes) != 1 || scopes[0].RecordCount != 1 {
		t.Fatalf("scopes %+v: %v", scopes, err)
	}
	if p, err := s.RecallAll("!!!", 5); err != nil || len(p.Current) != 0 {
		t.Fatalf("punctuation-only query matched %+v: %v", p, err)
	}
}
