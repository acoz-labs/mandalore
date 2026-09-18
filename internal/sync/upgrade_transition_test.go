package signetsync

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestUpgradeTransitionInspectionOnlyPermitsExactPreparedChanges(t *testing.T) {
	s, a := fixture(t)
	r := remember(t, a, "Retain synthetic evidence")
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := s.UpgradeSource(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(filepath.Join(a.Root(), "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := s.store.Signet
	manifest.Version = 2
	next, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	next = append(next, '\n')
	u := memory.UpgradeRecord{Version: 1, ID: "upgrade-test", SignetID: a.ID(), From: 1, To: 2, OriginalManifestSHA256: p.ManifestSHA256, PortableSHA256: p.PortableSHA256, BaseHead: p.Head, RecordedAt: r.RecordedAt, Authorship: r.Authorship}
	transition := UpgradeTransition{Source: p, OriginalManifest: old, Manifest: next, Record: u}
	state, err := s.InspectUpgradeTransition(context.Background(), transition)
	if err != nil || state.Activated || state.Published {
		t.Fatal(state, err)
	}
	dir := filepath.Join(a.Root(), "provenance/upgrades")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	receipt, _ := json.MarshalIndent(u, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, u.ID+".json"), append(receipt, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	state, err = s.InspectUpgradeTransition(context.Background(), transition)
	if err != nil || state.Activated || !state.Published {
		t.Fatal(state, err)
	}
	if err := os.WriteFile(filepath.Join(a.Root(), "signet.json"), next, 0600); err != nil {
		t.Fatal(err)
	}
	sy, err := Open(a.Root(), a.ID())
	if err != nil {
		t.Fatal(err)
	}
	state, err = sy.InspectUpgradeTransition(context.Background(), transition)
	if err != nil || !state.Activated || !state.Published {
		t.Fatal(state, err)
	}
	file := filepath.Join(a.Root(), "memory/records", r.RecordID, r.ID+".json")
	b, _ := os.ReadFile(file)
	if err := os.WriteFile(file, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := sy.InspectUpgradeTransition(context.Background(), transition); err == nil {
		t.Fatal("transition exception admitted an old-evidence edit")
	}
}
