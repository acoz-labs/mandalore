package memory

import (
	"strings"
	"testing"
)

func validSnapshot() Snapshot {
	a := revision("revision-root")
	a.Supersedes = []string{}
	b := revision("revision-left", a.ID)
	c := revision("revision-right", a.ID)
	return Snapshot{
		Signet:    Signet{Version: 1, ID: "bank-example", Name: "Example"},
		Devices:   []Device{{Version: 1, ID: "device-laptop", Label: "Original laptop"}},
		Sources:   []Source{{Version: 1, ID: "source-orphan", Kind: "observation", Summary: "Retain unreferenced evidence", DeviceID: "device-laptop", RecordedAt: a.RecordedAt}},
		Revisions: []Revision{a, b, c},
		Journal:   []JournalEntry{{Version: 1, ID: "event-original", Kind: "setup", Summary: "Original setup", RecordedAt: a.RecordedAt, Authorship: a.Authorship}},
	}
}

func TestSnapshotValidationPreservesBranchesAndOrphanEvidence(t *testing.T) {
	snapshot := validSnapshot()
	if err := ValidateSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Revisions) != 3 || len(snapshot.Sources) != 1 {
		t.Fatal("validation rewrote history")
	}
}

func TestSnapshotValidationHasNoFilesystemFallback(t *testing.T) {
	for name, change := range map[string]func(*Snapshot){
		"unknown device":     func(s *Snapshot) { s.Devices = nil },
		"bad orphan source":  func(s *Snapshot) { s.Sources[0].DeviceID = "device-unknown" },
		"missing source":     func(s *Snapshot) { s.Revisions[0].Evidence.SourceRefs = []string{"source-missing"} },
		"duplicate device":   func(s *Snapshot) { s.Devices = append(s.Devices, s.Devices[0]) },
		"duplicate source":   func(s *Snapshot) { s.Sources = append(s.Sources, s.Sources[0]) },
		"duplicate journal":  func(s *Snapshot) { s.Journal = append(s.Journal, s.Journal[0]) },
		"bad journal author": func(s *Snapshot) { s.Journal[0].Authorship.Actor = "" },
		"bad journal device": func(s *Snapshot) { s.Journal[0].Authorship.DeviceID = "device-missing" },
		"missing parent":     func(s *Snapshot) { s.Revisions[1].Supersedes = []string{"revision-missing"} },
		"scope mismatch":     func(s *Snapshot) { s.Revisions[0].Scope = Scope{Kind: "signet", ID: "bank-wrong"} },
		"bad manifest":       func(s *Snapshot) { s.Signet.Version = 2 },
		"unsupported origin": func(s *Snapshot) { s.Sources[0].ExternalOrigin = &ExternalOrigin{} },
	} {
		t.Run(name, func(t *testing.T) {
			s := validSnapshot()
			change(&s)
			if err := ValidateSnapshot(s); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
}

func TestReportSnapshotValidatesHistoricalOriginWithoutDisk(t *testing.T) {
	s := validSnapshot()
	r := FoundlingRegistration{Version: 1, ID: "registration-original", FoundlingID: "foundling-original", Name: "Archive", Description: "Synthetic reference", Source: FoundlingSource{Kind: "local", Locator: "source-original"}, Pin: SourcePin{Algorithm: "sha256", Value: strings.Repeat("a", 64)}, State: "active", RecordedAt: s.Revisions[0].RecordedAt, Authorship: s.Revisions[0].Authorship, Supersedes: []string{}, ChangeReason: "Registered"}
	s.Sources[0].ExternalOrigin = &ExternalOrigin{FoundlingID: r.FoundlingID, RegistrationRevisionID: r.ID, SourceIdentity: r.Source, SourcePin: r.Pin, RelativeLocator: "notes.md", ContentSHA256: strings.Repeat("b", 64)}
	if err := ValidateReportSnapshot(s, []FoundlingRegistration{r}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSnapshot(s); err == nil {
		t.Fatal("legacy importer unexpectedly accepted origins")
	}
	r.Pin.Value = strings.Repeat("c", 64)
	if err := ValidateReportSnapshot(s, []FoundlingRegistration{r}); err == nil {
		t.Fatal("mismatched origin accepted")
	}
	if err := ValidateReportSnapshot(s, nil); err == nil {
		t.Fatal("missing registration accepted")
	}
}
