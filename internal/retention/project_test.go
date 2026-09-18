package retention

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/exportreport"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestSharedSourcesAndFoundlingAttributionRemainMetadataOnly(t *testing.T) {
	in, service, r, _ := fixture(t)
	s, err := exportreport.ReadReportSnapshot(context.Background(), service.Root())
	if err != nil {
		t.Fatal(err)
	}
	other := r
	other.ID = "revision-other"
	other.RecordID = "record-other"
	s.Revisions = append(s.Revisions, other)
	registration := memory.FoundlingRegistration{Version: 1, ID: "registration-test", FoundlingID: "foundling-test", Name: "PRIVATE_NAME", Description: "PRIVATE_DESCRIPTION", Source: memory.FoundlingSource{Kind: "git", Locator: "https://example.invalid/PRIVATE_REF.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("a", 40)}, State: "active", RecordedAt: r.RecordedAt, Authorship: r.Authorship, Supersedes: []string{}, ChangeReason: "PRIVATE_REASON"}
	s.Registrations = append(s.Registrations, registration)
	for i := range s.Sources {
		if s.Sources[i].ID == r.Evidence.SourceRefs[0] {
			s.Sources[i].ExternalOrigin = &memory.ExternalOrigin{FoundlingID: registration.FoundlingID, RegistrationRevisionID: registration.ID, SourceIdentity: registration.Source, SourcePin: registration.Pin, RelativeLocator: "PRIVATE_FILE.md", ContentSHA256: strings.Repeat("b", 64)}
		}
	}
	records, journals, sources, err := project(context.Background(), s, in.Selection, in.Policy)
	if err != nil || len(records) != 1 || len(journals) != 0 || len(sources) != 1 || len(sources[0].RecordIDs) != 2 || sources[0].FoundlingID != registration.FoundlingID || sources[0].RegistrationID != registration.ID {
		t.Fatal(records, sources, err)
	}
	encoded, _ := json.Marshal([]any{records, journals, sources})
	if strings.Contains(string(encoded), "PRIVATE_") {
		t.Fatal("retention projection exposed source content/path/author")
	}
}

func TestAgeRequiresEveryCurrentHeadAndIncludesFutureHeads(t *testing.T) {
	in, service, r, _ := fixture(t)
	s, err := exportreport.ReadReportSnapshot(context.Background(), service.Root())
	if err != nil {
		t.Fatal(err)
	}
	at, _ := time.Parse(time.RFC3339Nano, r.RecordedAt)
	other := r
	other.ID = "revision-future"
	other.Supersedes = []string{r.ID}
	other.RecordedAt = at.Add(24 * time.Hour).Format(time.RFC3339Nano)
	other.EffectiveFrom = other.RecordedAt
	older := r
	older.ID = "revision-older"
	older.Supersedes = []string{r.ID}
	s.Revisions = append(s.Revisions, older, other)
	in.Policy.Age = &AgePolicy{Timestamp: "recorded_at", Before: at.Add(time.Hour).Format(time.RFC3339Nano)}
	if err := memory.ValidateReportSnapshot(s.Snapshot, s.Registrations); err != nil {
		t.Fatal(err)
	}
	records, _, _, err := project(context.Background(), s, in.Selection, in.Policy)
	if err != nil || len(records) != 1 || records[0].Matched || records[0].Visibility.State != "content-conflict" || records[0].Reason != "not-before-cutoff" {
		t.Fatal(records, err)
	}
}

func TestWithheldPolicyDoesNotInferJournalRelationships(t *testing.T) {
	in, service, r, j := fixture(t)
	s, err := exportreport.ReadReportSnapshot(context.Background(), service.Root())
	if err != nil {
		t.Fatal(err)
	}
	// Pure projection fixture, not evidence of a real Git-backed upgrade.
	s.Signet.Version = 2
	s.Upgrades = []memory.UpgradeRecord{{Version: 1, ID: "upgrade-test", SignetID: s.Signet.ID, From: 1, To: 2, BaseHead: strings.Repeat("a", 40), OriginalManifestSHA256: strings.Repeat("b", 64), PortableSHA256: strings.Repeat("c", 64), RecordedAt: r.RecordedAt, Authorship: r.Authorship}}
	s.Visibility = []memory.VisibilityEvent{{Version: 1, ID: "visibility-withdraw", RecordID: r.RecordID, Action: "withdraw", Observed: []string{r.ID}, Parents: []string{}, Reason: "PRIVATE_REASON", RecordedAt: r.RecordedAt, Authorship: r.Authorship}}
	in.Policy.Visibility = "withheld"
	records, journals, _, err := project(context.Background(), s, in.Selection, in.Policy)
	if err != nil || !records[0].Matched || records[0].Visibility.State != "withdrawn" || len(journals) != 0 {
		t.Fatal(records, journals, err)
	}
	in.Selection = Selection{Scopes: []memory.Scope{r.Scope}, JournalIDs: []string{j.ID}}
	records, journals, _, err = project(context.Background(), s, in.Selection, in.Policy)
	if err != nil || len(records) != 1 || len(journals) != 1 || !journals[0].Matched {
		t.Fatal(records, journals, err)
	}
}

func TestValidZeroTimestampIsNotMissing(t *testing.T) {
	r := memory.Revision{ID: "revision-test", RecordedAt: "0001-01-01T00:00:00Z"}
	matched, reason, value := matchAge([]memory.Revision{r}, []string{r.ID}, &AgePolicy{Timestamp: "recorded_at", Before: "0002-01-01T00:00:00Z"})
	if !matched || reason != "before-cutoff" || value == nil {
		t.Fatal(matched, reason, value)
	}
}
