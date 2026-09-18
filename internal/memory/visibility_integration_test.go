package memory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Synthetic format fixture only, not the production upgrade transaction. Real
// preview/apply must prove Git pins, preservation and crash recovery separately.
func visibilityStore(t *testing.T, old *Store) *Store {
	t.Helper()
	manifest, err := os.ReadFile(filepath.Join(old.Root, "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(manifest)
	u := UpgradeRecord{Version: 1, ID: "upgrade-test", SignetID: old.Signet.ID, From: 1, To: 2, OriginalManifestSHA256: hex.EncodeToString(h[:]), PortableSHA256: strings.Repeat("b", 64), BaseHead: strings.Repeat("c", 40), RecordedAt: "2026-09-18T12:00:00Z", Authorship: revision("revision-template").Authorship}
	dir := filepath.Join(old.Root, "provenance/upgrades")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeNewJSON(filepath.Join(dir, u.ID+".json"), u); err != nil {
		t.Fatal(err)
	}
	manifest2 := old.Signet
	manifest2.Version = 2
	b, err := encodeJSON(manifest2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old.Root, "signet.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(old.Root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func storeVisibilityFixture(t *testing.T, s *Store, id, action string, parents, observed []string) {
	t.Helper()
	e := VisibilityEvent{Version: 1, ID: id, RecordID: "record-documents", Action: action, Parents: parents, Observed: observed, Reason: "Explicit synthetic decision", RecordedAt: "2026-09-18T12:00:00Z", Authorship: revision("revision-template").Authorship}
	dir := filepath.Join(s.Root, "memory/visibility", e.RecordID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeNewJSON(filepath.Join(dir, id+".json"), e); err != nil {
		t.Fatal(err)
	}
}

func TestVisibilityStorageRecallAndLegacyPreservation(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-first")
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(old.Root, "memory/records", r.RecordID, r.ID+".json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := visibilityStore(t, old)
	if _, err := old.Recall(Query{Scope: r.Scope}, time.Now()); err == nil {
		t.Fatal("stale connection ignored format change")
	}
	storeVisibilityFixture(t, s, "visibility-withdraw", "withdraw", []string{}, []string{r.ID})
	p, err := s.Recall(Query{Scope: r.Scope}, time.Now())
	if err != nil || len(p.Current) != 0 || len(p.Conflicts) != 0 {
		t.Fatal("withheld content escaped", p, err)
	}
	scopes, err := s.Scopes()
	if err != nil || len(scopes) != 1 || scopes[0].Visibility == nil || scopes[0].Visibility.Withheld != 1 {
		t.Fatal(scopes, err)
	}
	next := revision("revision-next", r.ID)
	if err := s.Put(next); err != nil {
		t.Fatal(err)
	}
	p, err = s.Recall(Query{Scope: r.Scope}, time.Now())
	if err != nil || len(p.Current) != 0 {
		t.Fatal("correction restored memory", p, err)
	}
	history, err := s.History(r.RecordID)
	if err != nil || len(history) != 2 || history[1].Version != 2 || history[1].VisibilityRefs == nil {
		t.Fatal(history, err)
	}
	storeVisibilityFixture(t, s, "visibility-restore", "restore", []string{"visibility-withdraw"}, []string{next.ID})
	p, err = s.Recall(Query{Scope: r.Scope}, time.Now())
	if err != nil || len(p.Current) != 1 || p.Current[0].ID != next.ID {
		t.Fatal("explicit restore failed", p, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("legacy evidence changed", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestVisibilityServiceReceiptAndIndependentJournal(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-first")
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	s := visibilityStore(t, old)
	storeVisibilityFixture(t, s, "visibility-withdraw", "withdraw", []string{}, []string{r.ID})
	service, err := OpenService(s.Root, r.Authorship)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := service.Remember(Write{Kind: r.Kind, Scope: &r.Scope, RecordID: r.RecordID, Supersedes: []string{r.ID}, Summary: "Corrected synthetic policy", Body: "Keep the corrected history, still withdrawn.", Basis: "user-direction", Reason: "Synthetic correction"})
	if err != nil || saved.Version != 2 || saved.VisibilityRefs == nil || len(*saved.VisibilityRefs) != 1 || (*saved.VisibilityRefs)[0] != "visibility-withdraw" {
		t.Fatal("receipt omitted actual causal context", saved, err)
	}
	if _, err := service.AppendJournal("test", "Independent synthetic journal evidence"); err != nil {
		t.Fatal(err)
	}
	entries, err := service.Journal("Independent", 10)
	if err != nil || len(entries) != 1 {
		t.Fatal("record withdrawal suppressed independent journal", entries, err)
	}
	p, err := service.Recall("", &r.Scope, 5, 8192)
	if err != nil || len(p.Current) != 0 || len(p.Conflicts) != 0 {
		t.Fatal("service disclosed withdrawn correction", p, err)
	}
}

func TestVisibilityConcurrentFutureCorrectionNeverAppears(t *testing.T) {
	old := fixtureStore(t)
	r := revision("revision-first")
	if err := old.Put(r); err != nil {
		t.Fatal(err)
	}
	s := visibilityStore(t, old)
	storeVisibilityFixture(t, s, "visibility-withdraw", "withdraw", []string{}, []string{r.ID})
	future := revision("revision-future", r.ID)
	now := time.Now().UTC()
	future.EffectiveFrom = now.Add(time.Hour).Format(time.RFC3339Nano)
	if err := s.Put(future); err != nil {
		t.Fatal(err)
	}
	// Simulates a valid concurrent restore delivered from a clone that did not
	// observe the scheduled correction; local expected-head checks would reject it.
	storeVisibilityFixture(t, s, "visibility-restore", "restore", []string{"visibility-withdraw"}, []string{r.ID})
	for _, at := range []time.Time{now, now.Add(2 * time.Hour)} {
		p, err := s.Recall(Query{Scope: r.Scope}, at)
		if err != nil || len(p.Current) != 0 || len(p.Conflicts) != 0 {
			t.Fatal("clock activated unseen correction", p, err)
		}
	}
}
