package exportreport

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func reportFixture(t *testing.T) (memory.ReportSnapshot, memory.Revision, memory.Revision, memory.JournalEntry) {
	t.Helper()
	store, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "Synthetic bank", "device-test", "Synthetic device")
	if err != nil {
		t.Fatal(err)
	}
	s, err := memory.OpenService(store.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	scope := memory.Scope{Kind: "project", ID: "copper-finch"}
	a, err := s.Remember(memory.Write{Kind: "fact", Scope: &scope, Summary: "Current project", Body: "Copper Finch", Basis: "observation", Reason: "Old reason"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Remember(memory.Write{Kind: "fact", Scope: &scope, Summary: "Current project", Body: "Silver Heron", Basis: "user-direction", Reason: "Renamed", RecordID: a.RecordID, Supersedes: []string{a.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Remember(memory.Write{Kind: "fact", Summary: "Unrelated", Body: "UNSELECTED_MARKER", Basis: "observation", Reason: "Unrelated reason"}); err != nil {
		t.Fatal(err)
	}
	j, err := s.AppendJournal("test", "JOURNAL_MARKER")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil {
		t.Fatal(err)
	}
	return snapshot, a, b, j
}

func TestProjectionSelectionDefaultsAndOptIns(t *testing.T) {
	s, a, b, j := reportFixture(t)
	selection := Selection{RecordIDs: []string{a.RecordID}}
	p, raw, err := project(s, selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.RevisionIDs) != 1 || p.RevisionIDs[0] != b.ID || !strings.Contains(string(raw), "Silver Heron") {
		t.Fatal("wrong current selection")
	}
	for _, excluded := range []string{"UNSELECTED_MARKER", "JOURNAL_MARKER", "Old reason", "Synthetic actor", a.ID} {
		if strings.Contains(string(raw), excluded) {
			t.Fatalf("leaked default surface %q", excluded)
		}
	}
	selection.IncludeHistory = true
	selection.IncludeDetails = true
	selection.JournalIDs = []string{j.ID}
	p, raw, err = project(s, selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.RevisionIDs) != 2 || len(p.JournalIDs) != 1 || !strings.Contains(string(raw), "Old reason") || !strings.Contains(string(raw), "JOURNAL_MARKER") {
		t.Fatal("opt-ins missing")
	}
	if strings.Contains(string(raw), "UNSELECTED_MARKER") {
		t.Fatal("expanded selection")
	}
}

func TestProjectionOmissionsCoverEverySurface(t *testing.T) {
	s, a, _, j := reportFixture(t)
	selection := Selection{RecordIDs: []string{a.RecordID}, JournalIDs: []string{j.ID}, IncludeHistory: true, IncludeDetails: true, OmitFields: append([]string{}, fieldCategories...)}
	p, raw, err := project(s, selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.OmittedFields) != len(fieldCategories) {
		t.Fatal("missing omission policy")
	}
	for _, secret := range []string{"Silver Heron", "Copper Finch", a.RecordID, a.ID, j.ID, s.Signet.ID, "device-test", "Synthetic actor", "JOURNAL_MARKER", "user-direction", "copper-finch"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("omission leaked %q", secret)
		}
	}
	if !strings.Contains(string(raw), "not a restorable signet") {
		t.Fatal("boundary missing")
	}
}

func TestProjectionRequiresExplicitSelection(t *testing.T) {
	s, _, _, _ := reportFixture(t)
	for _, input := range []Selection{{}, {RecordIDs: []string{"record-missing"}}, {Scopes: []memory.Scope{{Kind: "project", ID: "missing"}}}, {JournalIDs: []string{"event-missing"}}, {RecordIDs: []string{s.Revisions[0].RecordID}, OmitFields: []string{"typo"}}} {
		if _, _, err := project(s, input); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
}

func TestProjectionConflictFutureAndItemOmission(t *testing.T) {
	s, a, b, j := reportFixture(t)
	c := b
	c.ID = "revision-concurrent"
	c.Body = "CONFLICT_MARKER"
	s.Revisions = append(s.Revisions, c)
	selection := Selection{Scopes: []memory.Scope{a.Scope}}
	p, raw, err := project(s, selection)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Conflicts) != 1 || len(p.Conflicts[0].HeadIDs) != 2 || len(p.RevisionIDs) != 0 || strings.Contains(string(raw), "CONFLICT_MARKER") {
		t.Fatal("conflict promoted")
	}
	selection.IncludeHistory = true
	p, raw, err = project(s, selection)
	if err != nil || len(p.RevisionIDs) != 3 || !strings.Contains(string(raw), `"status": "conflicted"`) {
		t.Fatal("explicit conflict history missing", err)
	}
	selection.OmitRecordIDs = []string{a.RecordID}
	selection.JournalIDs = []string{j.ID}
	selection.OmitJournalIDs = []string{j.ID}
	p, raw, err = project(s, selection)
	if err != nil || len(p.RevisionIDs) != 0 || len(p.JournalIDs) != 0 || p.OmittedRecords != 1 || p.OmittedJournals != 1 || strings.Contains(string(raw), "CONFLICT_MARKER") {
		t.Fatal("item omissions failed", err)
	}
	s.Revisions = s.Revisions[:len(s.Revisions)-1]
	for i := range s.Revisions {
		if s.Revisions[i].ID == b.ID {
			s.Revisions[i].EffectiveFrom = time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
		}
	}
	p, raw, err = project(s, Selection{RecordIDs: []string{a.RecordID}})
	if err != nil || len(p.RevisionIDs) != 1 || p.RevisionIDs[0] != a.ID || strings.Contains(string(raw), "Silver Heron") {
		t.Fatal("future head treated as current", err)
	}
	_, raw, err = project(s, Selection{RecordIDs: []string{a.RecordID}, IncludeHistory: true})
	if err != nil || !strings.Contains(string(raw), `"status": "future"`) {
		t.Fatal("future revision mislabeled as history", err)
	}
}

func TestProjectionIdentityAndContentOmitDependentProvenance(t *testing.T) {
	s, a, _, _ := reportFixture(t)
	for _, field := range []string{"identity", "content"} {
		p, raw, err := project(s, Selection{RecordIDs: []string{a.RecordID}, IncludeHistory: true, IncludeDetails: true, OmitFields: []string{field}})
		if err != nil {
			t.Fatal(err)
		}
		for _, dependent := range []string{"citations", "change_history", "extensions"} {
			found := false
			for _, f := range p.OmittedFields {
				found = found || f == dependent
			}
			if !found {
				t.Fatal("incomplete provenance closure", p)
			}
		}
		for _, hidden := range []string{"Old reason", "Renamed", `"supersedes"`, `"source_refs"`} {
			if strings.Contains(string(raw), hidden) {
				t.Fatal("dependent provenance leaked", field, hidden)
			}
		}
	}
}
