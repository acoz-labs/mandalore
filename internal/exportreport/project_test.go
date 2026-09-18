package exportreport

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
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

func TestOmissionsCloseOverNestedCitationMetadata(t *testing.T) {
	s, a, _, _ := reportFixture(t)
	for _, category := range []string{"authorship", "timestamps", "classification"} {
		p, raw, err := project(s, Selection{RecordIDs: []string{a.RecordID}, IncludeDetails: true, IncludeHistory: true, OmitFields: []string{category}})
		if err != nil {
			t.Fatal(err)
		}
		// Citations contain device IDs, source kind/recorded time and imported
		// author/time metadata. Omitting a parent category must not leave those
		// alternate disclosure routes behind.
		if !slices.Contains(p.OmittedFields, "citations") || strings.Contains(string(raw), `"citations":`) {
			t.Fatal("dependent citation surface retained", category)
		}
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

func TestEveryFieldRedactionWithHistoricalOriginsAndOpaqueExtensions(t *testing.T) {
	s, a, b, j := reportFixture(t)
	model, session := "MODEL_CANARY", "SESSION_CANARY"
	for i := range s.Revisions {
		s.Revisions[i].Authorship.Model = &model
		s.Revisions[i].Authorship.SessionID = &session
		s.Revisions[i].Extensions = map[string]any{"test.opaque": map[string]any{"secret": "EXTENSION_CANARY"}}
	}
	r := memory.FoundlingRegistration{Version: 1, ID: "registration-original", FoundlingID: "foundling-original", Name: "REGISTRATION_CANARY", Description: "Reference", Source: memory.FoundlingSource{Kind: "local", Locator: "source-original"}, Pin: memory.SourcePin{Algorithm: "sha256", Value: strings.Repeat("a", 64)}, State: "active", RecordedAt: a.RecordedAt, Authorship: a.Authorship, Supersedes: []string{}, ChangeReason: "Registered"}
	s.Registrations = []memory.FoundlingRegistration{r}
	for i := range s.Sources {
		s.Sources[i].ExternalOrigin = &memory.ExternalOrigin{FoundlingID: r.FoundlingID, RegistrationRevisionID: r.ID, SourceIdentity: r.Source, SourcePin: r.Pin, RelativeLocator: "ORIGIN_CANARY.md", ContentSHA256: strings.Repeat("b", 64), OriginalAuthor: "AUTHOR_CANARY", OriginalRecordedAt: a.RecordedAt}
	}
	in := Selection{RecordIDs: []string{a.RecordID}, JournalIDs: []string{j.ID}, IncludeDetails: true, IncludeHistory: true}
	if err := memory.ValidateReportSnapshot(s.Snapshot, s.Registrations); err != nil {
		t.Fatal("synthetic fixture", err)
	}
	_, visible, err := project(s, in)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"MODEL_CANARY", "SESSION_CANARY", "EXTENSION_CANARY", "ORIGIN_CANARY", "AUTHOR_CANARY", a.ID, b.ID, j.ID, a.RecordedAt} {
		if !strings.Contains(string(visible), marker) {
			t.Fatal("fixture omitted intended marker", marker)
		}
	}
	if strings.Contains(string(visible), "REGISTRATION_CANARY") {
		t.Fatal("copied unrelated registration content")
	}
	in.OmitFields = append([]string{}, fieldCategories...)
	_, raw, err := project(s, in)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	for _, value := range report["items"].([]any) {
		for key := range value.(map[string]any) {
			if key != "ordinal" && key != "type" && key != "status" {
				t.Fatal("unexpected disclosure surface", key)
			}
		}
	}
	if _, ok := report["identity"]; ok {
		t.Fatal("report identity survived omission")
	}
	for _, category := range fieldCategories {
		in.OmitFields = []string{category}
		p, raw, err := project(s, in)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		for _, item := range v.Items {
			for _, omitted := range p.OmittedFields {
				if _, exists := item[omitted]; exists {
					t.Fatal("effective omission retained", category, omitted)
				}
			}
		}
		if strings.Contains(string(raw), "EXTENSION_CANARY") {
			t.Fatal("opaque extension survived omission", category)
		}
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
