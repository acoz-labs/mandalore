package exportreport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func writeVisibilityJSON(t *testing.T, file string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}
}

// Synthetic format fixture, not the production preview/apply migration path.
func upgradeReportFixture(t *testing.T, root string) memory.ReportSnapshot {
	t.Helper()
	s, err := ReadReportSnapshot(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := s.Revisions[0]
	u := memory.UpgradeRecord{Version: 1, ID: "upgrade-test", SignetID: s.Signet.ID, From: 1, To: 2, OriginalManifestSHA256: digest(manifest), PortableSHA256: s.Digest, BaseHead: strings.Repeat("a", 40), RecordedAt: r.RecordedAt, Authorship: r.Authorship}
	writeVisibilityJSON(t, filepath.Join(root, "provenance/upgrades", u.ID+".json"), u)
	s.Signet.Version = 2
	writeVisibilityJSON(t, filepath.Join(root, "signet.json"), s.Signet)
	s, err = ReadReportSnapshot(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestVisibilitySnapshotPinsPreviewAndPublication(t *testing.T) {
	in, service := previewFixture(t)
	root := service.Root()
	before := upgradeReportFixture(t, root)
	p, err := Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	r := before.Revisions[0]
	e := memory.VisibilityEvent{Version: 1, ID: "visibility-withdraw", RecordID: r.RecordID, Action: "withdraw", Parents: []string{}, Observed: []string{r.ID}, Reason: "PRIVATE_WITHDRAWAL_REASON", RecordedAt: r.RecordedAt, Authorship: r.Authorship}
	writeVisibilityJSON(t, filepath.Join(root, "memory/visibility", r.RecordID, e.ID+".json"), e)
	stale, err := Apply(context.Background(), p)
	if err == nil || stale.StagingDirectory != "" || stale.Published {
		t.Fatal("stale visible preview wrote output", stale, err)
	}
	after, err := ReadReportSnapshot(context.Background(), root)
	if err != nil || before.Digest == after.Digest || len(after.Visibility) != 1 || len(after.Upgrades) != 1 {
		t.Fatal("snapshot omitted new portable evidence", err)
	}
	p, err = Preview(context.Background(), in)
	if err != nil || len(p.Projection.Withheld) != 1 {
		t.Fatal(p, err)
	}
	encoded, err := json.Marshal(p)
	if err != nil || strings.Contains(string(encoded), "PRIVATE_") {
		t.Fatal("preview leaked withheld content or decision reasons", err)
	}
	result, err := Apply(context.Background(), p)
	if err != nil || !result.Published || !result.Durable {
		t.Fatal(result, err)
	}
	raw, err := os.ReadFile(filepath.Join(in.Destination, "report.json"))
	if err != nil || strings.Contains(string(raw), "PRIVATE_") {
		t.Fatal("publication leaked withheld material", err)
	}
	unchanged, err := ReadReportSnapshot(context.Background(), root)
	if err != nil || unchanged.Digest != after.Digest {
		t.Fatal("export changed source", err)
	}
	in.Destination += "-history"
	in.Selection.IncludeHistory, in.Selection.IncludeWithdrawn = true, true
	p, err = Preview(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	result, err = Apply(context.Background(), p)
	if err != nil || !result.Published {
		t.Fatal(result, err)
	}
	raw, err = os.ReadFile(filepath.Join(in.Destination, "report.json"))
	if err != nil || !strings.Contains(string(raw), "PRIVATE_BODY") || !strings.Contains(string(raw), "withheld-history") || strings.Contains(string(raw), "PRIVATE_WITHDRAWAL_REASON") {
		t.Fatal("explicit historical publication contract changed", err)
	}
}

func withdrawnReportFixture(t *testing.T) (memory.ReportSnapshot, memory.Revision, memory.JournalEntry) {
	t.Helper()
	s, _, current, journal := reportFixture(t)
	s.Signet.Version = 2
	s.Upgrades = []memory.UpgradeRecord{{Version: 1, ID: "upgrade-test", SignetID: s.Signet.ID, From: 1, To: 2, OriginalManifestSHA256: strings.Repeat("a", 64), PortableSHA256: strings.Repeat("b", 64), BaseHead: strings.Repeat("c", 40), RecordedAt: current.RecordedAt, Authorship: current.Authorship}}
	s.Visibility = []memory.VisibilityEvent{{Version: 1, ID: "visibility-withdraw", RecordID: current.RecordID, Action: "withdraw", Parents: []string{}, Observed: []string{current.ID}, Reason: "WITHDRAWAL_REASON_CANARY", RecordedAt: current.RecordedAt, Authorship: current.Authorship}}
	return s, current, journal
}

func TestWithdrawnReportRequiresBothHistoryOptIns(t *testing.T) {
	s, current, journal := withdrawnReportFixture(t)
	for _, history := range []bool{false, true} {
		p, raw, err := project(s, Selection{RecordIDs: []string{current.RecordID}, IncludeHistory: history, IncludeDetails: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.RevisionIDs) != 0 || p.OmittedRecords != 1 || len(p.Withheld) != 1 || p.Withheld[0].RecordID != current.RecordID || p.Withheld[0].State != "withdrawn" {
			t.Fatal(p)
		}
		for _, canary := range []string{"Silver Heron", "Copper Finch", "WITHDRAWAL_REASON_CANARY", "Synthetic actor", current.ID} {
			if strings.Contains(string(raw), canary) {
				t.Fatal("default/history-only export leaked", canary)
			}
		}
	}
	if _, _, err := project(s, Selection{RecordIDs: []string{current.RecordID}, IncludeWithdrawn: true}); err == nil {
		t.Fatal("withdrawn opt-in did not require history")
	}
	p, raw, err := project(s, Selection{RecordIDs: []string{current.RecordID}, JournalIDs: []string{journal.ID}, IncludeHistory: true, IncludeWithdrawn: true})
	if err != nil || len(p.RevisionIDs) != 2 || len(p.JournalIDs) != 1 || !strings.Contains(string(raw), "Silver Heron") || !strings.Contains(string(raw), "withheld-history") {
		t.Fatal(p, string(raw), err)
	}
	if strings.Contains(string(raw), `"status": "current"`) || strings.Contains(string(raw), "WITHDRAWAL_REASON_CANARY") {
		t.Fatal("withheld history labeled guidance or leaked event reason")
	}
	p, raw, err = project(s, Selection{RecordIDs: []string{current.RecordID}, JournalIDs: []string{journal.ID}})
	if err != nil || len(p.RevisionIDs) != 0 || len(p.JournalIDs) != 1 || !strings.Contains(string(raw), "JOURNAL_MARKER") {
		t.Fatal("record withdrawal suppressed an independently selected journal", p, err)
	}
}

func TestVisibilityConflictsAndRedactionInReports(t *testing.T) {
	s, current, _ := withdrawnReportFixture(t)
	restore := s.Visibility[0]
	restore.ID, restore.Action, restore.Parents = "visibility-restore", "restore", []string{"visibility-withdraw"}
	s.Visibility = append(s.Visibility, restore)
	other := restore
	other.ID = "visibility-other"
	s.Visibility = append(s.Visibility, other)
	p, _, err := project(s, Selection{RecordIDs: []string{current.RecordID}})
	if err != nil || len(p.RevisionIDs) != 0 || len(p.Withheld) != 1 || p.Withheld[0].State != "visibility-conflict" {
		t.Fatal(p, err)
	}
	_, raw, err := project(s, Selection{RecordIDs: []string{current.RecordID}, IncludeHistory: true, IncludeWithdrawn: true, IncludeDetails: true, OmitFields: append([]string{}, fieldCategories...)})
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{current.RecordID, current.ID, "Silver Heron", "Copper Finch", "Synthetic actor", "device-test", "WITHDRAWAL_REASON_CANARY", "visibility-restore"} {
		if strings.Contains(string(raw), canary) {
			t.Fatal("omission leaked withheld history", canary)
		}
	}
}
