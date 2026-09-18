package signetsync

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// The real activation/recovery transaction is tested in formatupgrade. This
// fixture constructs its portable output to isolate synchronization behavior.
func upgradedFixture(t *testing.T, s *Synchronizer, actor memory.Authorship) *Synchronizer {
	t.Helper()
	if _, err := s.Checkpoint(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := s.UpgradeSource(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	u := memory.UpgradeRecord{Version: 1, ID: memory.NewID("upgrade"), SignetID: p.SignetID, From: 1, To: 2, OriginalManifestSHA256: p.ManifestSHA256, BaseHead: p.Head, PortableSHA256: p.PortableSHA256, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: actor}
	dir := filepath.Join(s.store.Root, "provenance/upgrades")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(u, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, u.ID+".json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := s.store.Signet
	manifest.Version = 2
	b, _ = json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(s.store.Root, "signet.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	next, err := Open(s.store.Root, s.store.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

func TestUpgradedBankCanCheckpointWithoutEditingOldEvidence(t *testing.T) {
	s, a := fixture(t)
	r := remember(t, a, "Preserved evidence")
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	s = upgradedFixture(t, s, r.Authorship)
	out, err := s.Checkpoint(context.Background())
	if err != nil || !out.Checkpointed {
		t.Fatal(out, err)
	}
	if err := s.validateCandidate(context.Background(), out.Head); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteUpgradeNeedsLocalOptInAndIndependentUpgradesConverge(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	for _, s := range []*Synchronizer{b, a} {
		if out, err := s.Sync(context.Background(), 30*time.Second); err != nil || !out.Delivered {
			t.Fatal(out, err)
		}
	}
	old := gitTest(t, second.Root(), "rev-parse", "HEAD")
	a = upgradedFixture(t, a, memory.Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"})
	if out, err := a.Sync(context.Background(), 30*time.Second); err != nil || !out.Delivered {
		t.Fatal(out, err)
	}
	out, err := b.Sync(context.Background(), 30*time.Second)
	if err != nil || out.State != "upgrade-required" || out.Delivered || gitTest(t, second.Root(), "rev-parse", "HEAD") != old {
		t.Fatal(out, err)
	}
	inspection, err := b.Inspect(context.Background())
	if err != nil || inspection.State != "upgrade-required" {
		t.Fatal("upgrade requirement lost from inspection", inspection, err)
	}
	still, err := memory.Open(second.Root())
	if err != nil || still.Signet.Version != 1 {
		t.Fatal("remote implicitly activated format", err)
	}
	b = upgradedFixture(t, b, memory.Authorship{DeviceID: "device-second", Actor: "Example", Harness: "test"})
	for _, s := range []*Synchronizer{b, a} {
		out, err := s.Sync(context.Background(), 30*time.Second)
		if err != nil || !out.Delivered || out.State != "synchronized" {
			t.Fatal(out, err)
		}
	}
	if gitTest(t, first.Root(), "rev-parse", "HEAD") != gitTest(t, second.Root(), "rev-parse", "HEAD") {
		t.Fatal("upgrades did not converge")
	}
	entries, err := os.ReadDir(filepath.Join(first.Root(), "provenance/upgrades"))
	if err != nil || len(entries) != 2 {
		t.Fatal("independent receipts not preserved", err)
	}
}

func TestUpgradeProofRejectsTamperingAndEvidenceEdits(t *testing.T) {
	for _, mutation := range []string{"inventory", "manifest-hash", "base", "unrelated-base", "missing-receipt", "old-evidence", "rename", "downgrade"} {
		t.Run(mutation, func(t *testing.T) {
			s, a := fixture(t)
			r := remember(t, a, "Immutable evidence")
			if _, err := s.Initialize(context.Background()); err != nil {
				t.Fatal(err)
			}
			s = upgradedFixture(t, s, r.Authorship)
			if mutation == "downgrade" {
				if _, err := s.Checkpoint(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			before := gitTest(t, a.Root(), "rev-parse", "HEAD")
			paths, err := filepath.Glob(filepath.Join(a.Root(), "provenance/upgrades/*.json"))
			if err != nil || len(paths) != 1 {
				t.Fatal(paths, err)
			}
			file := paths[0]
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var u memory.UpgradeRecord
			if err := json.Unmarshal(raw, &u); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "inventory":
				u.PortableSHA256 = strings.Repeat("a", 64)
			case "manifest-hash":
				u.OriginalManifestSHA256 = strings.Repeat("a", 64)
			case "base":
				u.BaseHead = strings.Repeat("a", 40)
			case "unrelated-base":
				u.BaseHead = gitTest(t, a.Root(), "commit-tree", u.BaseHead+"^{tree}", "-m", "Unrelated synthetic checkpoint")
			case "missing-receipt":
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
			case "old-evidence":
				file = filepath.Join(a.Root(), "memory/records", r.RecordID, r.ID+".json")
				raw, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, append(raw, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			case "rename", "downgrade":
				manifest := s.store.Signet
				if mutation == "rename" {
					manifest.Name = "Not a version-only change"
				} else {
					manifest.Version = 1
				}
				b, _ := json.Marshal(manifest)
				if err := os.WriteFile(filepath.Join(a.Root(), "signet.json"), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mutation == "inventory" || mutation == "manifest-hash" || mutation == "base" || mutation == "unrelated-base" {
				b, _ := json.Marshal(u)
				if err := os.WriteFile(file, b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := s.Checkpoint(context.Background()); err == nil || out.Checkpointed {
				t.Fatal("unsafe upgrade checkpointed", out, err)
			}
			if gitTest(t, a.Root(), "rev-parse", "HEAD") != before {
				t.Fatal("failed upgrade advanced head")
			}
		})
	}
}

func TestConcurrentVisibilityEventsSyncWithoutDisclosingContent(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	r := remember(t, first, "WITHHELD_CANARY")
	for _, s := range []*Synchronizer{a, b, a} {
		if out, err := s.Sync(context.Background(), 30*time.Second); err != nil || !out.Delivered {
			t.Fatal(out, err)
		}
	}
	a = upgradedFixture(t, a, r.Authorship)
	b = upgradedFixture(t, b, memory.Authorship{DeviceID: "device-second", Actor: "Example", Harness: "test"})
	for _, s := range []*Synchronizer{a, b, a} {
		if out, err := s.Sync(context.Background(), 30*time.Second); err != nil || !out.Delivered {
			t.Fatal(out, err)
		}
	}
	for i, s := range []*Synchronizer{a, b} {
		e := memory.VisibilityEvent{Version: 1, ID: memory.NewID("visibility"), RecordID: r.RecordID, Action: []string{"withdraw", "restore"}[i], Observed: []string{r.ID}, Parents: []string{}, Reason: "Concurrent synthetic decisions", RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: r.Authorship}
		if i == 1 {
			e.Authorship.DeviceID = "device-second"
		}
		dir := filepath.Join(s.store.Root, "memory/visibility", r.RecordID)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(e)
		if err := os.WriteFile(filepath.Join(dir, e.ID+".json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []*Synchronizer{a, b, a} {
		out, err := s.Sync(context.Background(), 30*time.Second)
		if err != nil || !out.Delivered {
			t.Fatal(out, err)
		}
		if s == b && (out.State != "conflicted" || out.SemanticConflicts != 1) {
			t.Fatal("visibility conflict omitted", out)
		}
	}
	for _, root := range []string{first.Root(), second.Root()} {
		s, err := memory.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		p, err := s.Recall(memory.Query{ExactScope: true, Scope: r.Scope, Limit: 5}, time.Now())
		if err != nil || len(p.Current) != 0 || len(p.Conflicts) != 0 {
			t.Fatal("withheld content disclosed", p, err)
		}
		history, err := s.History(r.RecordID)
		if err != nil || len(history) != 1 || history[0].Body != "WITHHELD_CANARY" {
			t.Fatal("explicit history lost", history, err)
		}
	}
}
