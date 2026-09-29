package foundlings

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestSessionRefreshAllowedDoesNotFetchOrExposeForbiddenReferences(t *testing.T) {
	m, _, allowed, _, sourceRoot := canonFixture(t)
	denied, err := m.memory.WriteFoundling(memory.FoundlingWrite{Name: "Forbidden reference", Description: "Must remain outside the remote session", Mode: "canon", Branch: "main", SourceSignetID: allowed.SourceSignetID, Source: memory.FoundlingSource{Kind: "git", Locator: "https://github.com/example/forbidden.git"}, Pin: allowed.Pin, State: "active", Reason: "Synthetic authorization fixture"})
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	m.canonTransport = func(locator string) string { fetched = append(fetched, locator); return sourceRoot }
	receipts, err := m.SessionRefreshAllowed(context.Background(), "restricted-session", []string{allowed.FoundlingID})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 1 || receipts[0].FoundlingID != allowed.FoundlingID || receipts[0].State != "available" {
		t.Fatalf("unexpected authorized receipts: %+v", receipts)
	}
	if len(fetched) != 1 || fetched[0] != allowed.Source.Locator {
		t.Fatalf("unexpected source fetches: %v", fetched)
	}
	root, err := m.canonRoot(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{sessionFile("restricted-session", denied.FoundlingID), latestFile(denied.FoundlingID)} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("forbidden snapshot exists: %s %v", relative, err)
		}
	}
	if _, err := m.CanonRecall(context.Background(), CanonRecallInput{SessionID: "restricted-session", FoundlingID: denied.FoundlingID, Limit: 10, BudgetBytes: 8192}); err == nil {
		t.Fatal("forbidden source unexpectedly readable in fresh restricted session")
	}
	for _, selection := range [][]string{nil, {}, {"foundling-unknown"}} {
		fetched = nil
		receipts, err = m.SessionRefreshAllowed(context.Background(), "empty-session", selection)
		if err != nil || len(receipts) != 0 || len(fetched) != 0 {
			t.Fatalf("empty or unmatched allowlist fetched/exposed references: %v %v %v", receipts, fetched, err)
		}
	}
}

func TestSessionRefreshAllowedPreservesSessionPins(t *testing.T) {
	m, source, reg, revision, root := canonFixture(t)
	first, err := m.SessionRefreshAllowed(context.Background(), "first", []string{reg.FoundlingID})
	if err != nil || len(first) != 1 || first[0].State != "available" {
		t.Fatalf("first refresh: %v %v", first, err)
	}
	newer, err := source.Remember(memory.Write{Kind: "fact", Summary: "Current color", Body: "Synthetic color green", Basis: "observation", Reason: "Changed source", RecordID: revision.RecordID, Supersedes: []string{revision.ID}})
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Advance source")
	second, err := m.SessionRefreshAllowed(context.Background(), "second", []string{reg.FoundlingID})
	if err != nil || len(second) != 1 || second[0].State != "available" {
		t.Fatalf("second refresh: %v %v", second, err)
	}
	if first[0].Pin == second[0].Pin {
		t.Fatal("new session did not fetch new source revision")
	}
	for session, want := range map[string]string{"first": revision.ID, "second": newer.ID} {
		got := recallTest(t, m, reg.FoundlingID, session)
		if len(got.Memory.Current) != 1 || got.Memory.Current[0].ID != want {
			t.Fatalf("session %s lost its pin: %+v", session, got)
		}
	}
	// Explicitly advancing one authorized session must not alter another.
	advanced, err := m.SessionRefreshAllowed(context.Background(), "first", []string{reg.FoundlingID})
	if err != nil || len(advanced) != 1 || advanced[0].Pin != second[0].Pin {
		t.Fatalf("explicit refresh did not advance first session: %v %v", advanced, err)
	}
	if got := recallTest(t, m, reg.FoundlingID, "first"); len(got.Memory.Current) != 1 || got.Memory.Current[0].ID != newer.ID {
		t.Fatal("explicit session refresh not reflected", got)
	}
}
