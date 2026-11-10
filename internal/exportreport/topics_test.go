package exportreport

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func topicReport(t *testing.T) (memory.ReportSnapshot, memory.ScopeLabel, memory.ScopeLabel, memory.JournalEntry, memory.JournalEntry) {
	t.Helper()
	store, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "Synthetic bank", "device-test", "Synthetic device")
	if err != nil {
		t.Fatal(err)
	}
	s, err := memory.OpenService(store.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	topic := memory.Scope{Kind: "project", ID: "atlas-launch"}
	if _, err := s.Remember(memory.Write{Kind: "fact", Scope: &topic, Summary: "Topic fact", Body: "RECORD_MARKER", Basis: "observation", Reason: "r"}); err != nil {
		t.Fatal(err)
	}
	old, err := s.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "OLD_NAME", Description: "Old description", Reason: "r"})
	if err != nil {
		t.Fatal(err)
	}
	current, err := s.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "NAME_MARKER", Description: "DESCRIPTION_MARKER", Reason: "Renamed", Supersedes: []string{old.ID}})
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := s.AppendJournal("outcome", "SCOPED_JOURNAL_MARKER", &topic)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendJournal("outcome", "OTHER_JOURNAL_MARKER", &memory.Scope{Kind: "task", ID: "journal-only"}); err != nil {
		t.Fatal(err)
	}
	plain, err := s.AppendJournal("outcome", "PLAIN_JOURNAL_MARKER", nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Labels) != 2 || len(snapshot.Journal) != 3 {
		t.Fatalf("snapshot read %d labels, %d journal entries", len(snapshot.Labels), len(snapshot.Journal))
	}
	return snapshot, old, current, scoped, plain
}

func TestSelectingAScopeExportsItsLabelsAndScopedJournal(t *testing.T) {
	s, old, current, scoped, plain := topicReport(t)
	p, raw, err := project(s, Selection{Scopes: []memory.Scope{{Kind: "project", ID: "atlas-launch"}}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{"RECORD_MARKER", "NAME_MARKER", "DESCRIPTION_MARKER", "SCOPED_JOURNAL_MARKER", `"scope_label"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s: %s", want, text)
		}
	}
	for _, unwanted := range []string{"OLD_NAME", "OTHER_JOURNAL_MARKER", "PLAIN_JOURNAL_MARKER"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("leaked %s", unwanted)
		}
	}
	if len(p.LabelIDs) != 1 || p.LabelIDs[0] != current.ID || len(p.JournalIDs) != 1 || p.JournalIDs[0] != scoped.ID {
		t.Fatalf("projection %+v", p)
	}
	// History adds the superseded label; content omission removes the text.
	p, raw, err = project(s, Selection{Scopes: []memory.Scope{{Kind: "project", ID: "atlas-launch"}}, IncludeHistory: true, IncludeDetails: true})
	if err != nil || len(p.LabelIDs) != 2 || !strings.Contains(string(raw), "OLD_NAME") || !strings.Contains(string(raw), old.ID) {
		t.Fatalf("history: %v %s", err, raw)
	}
	if _, raw, err = project(s, Selection{Scopes: []memory.Scope{{Kind: "project", ID: "atlas-launch"}}, OmitFields: []string{"content"}}); err != nil || strings.Contains(string(raw), "NAME_MARKER") || strings.Contains(string(raw), "DESCRIPTION_MARKER") {
		t.Fatalf("content omission: %v %s", err, raw)
	}
	// A scope with only journal entries is selectable; journal IDs and their
	// omissions reach both directories.
	if _, raw, err = project(s, Selection{Scopes: []memory.Scope{{Kind: "task", ID: "journal-only"}}}); err != nil || !strings.Contains(string(raw), "OTHER_JOURNAL_MARKER") {
		t.Fatalf("journal-only scope: %v %s", err, raw)
	}
	p, raw, err = project(s, Selection{JournalIDs: []string{scoped.ID, plain.ID}, OmitJournalIDs: []string{plain.ID}})
	if err != nil || len(p.JournalIDs) != 1 || p.OmittedJournals != 1 || !strings.Contains(string(raw), `"atlas-launch"`) {
		t.Fatalf("journal IDs: %v %+v %s", err, p, raw)
	}
}

func TestScopeSelectedJournalEntriesShareTheCap(t *testing.T) {
	store, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "Synthetic bank", "device-test", "Synthetic device")
	if err != nil {
		t.Fatal(err)
	}
	s, err := memory.OpenService(store.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	topic := memory.Scope{Kind: "project", ID: "atlas-launch"}
	for range 129 {
		if _, err := s.AppendJournal("outcome", "Entry", &topic); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := project(snapshot, Selection{Scopes: []memory.Scope{topic}}); err == nil {
		t.Fatal("129 scope-selected journal entries exported")
	}
}

func TestLegacyImportRefusesTopicData(t *testing.T) {
	s, _, _, _, _ := topicReport(t)
	snapshot := s.Snapshot
	snapshot.Visibility, snapshot.Upgrades = nil, nil
	if err := memory.ValidateSnapshot(snapshot); err == nil {
		t.Fatal("legacy import accepted labels")
	}
	snapshot.Labels = nil
	if err := memory.ValidateSnapshot(snapshot); err == nil {
		t.Fatal("legacy import accepted scoped journal entries")
	}
	unscoped := []memory.JournalEntry{}
	for _, e := range snapshot.Journal {
		if e.Scope == nil {
			unscoped = append(unscoped, e)
		}
	}
	snapshot.Journal = unscoped
	if err := memory.ValidateSnapshot(snapshot); err != nil {
		t.Fatal("the same snapshot without topic data is refused:", err)
	}
}
