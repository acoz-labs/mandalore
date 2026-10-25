package foundlings

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestConfiguredCanonDoesNotFetchOrExposeOtherReferences(t *testing.T) {
	m, _, allowed, _, sourceRoot := canonFixture(t)
	denied, err := m.memory.WriteFoundling(memory.FoundlingWrite{Name: "Unconfigured reference", Description: "Registered but not served", Mode: "canon", Branch: "main", SourceSignetID: allowed.SourceSignetID, Source: memory.FoundlingSource{Kind: "git", Locator: "https://github.com/example/unconfigured.git"}, Pin: allowed.Pin, State: "active", Reason: "Synthetic authorization fixture"})
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	transport := func(locator string) string { fetched = append(fetched, locator); return sourceRoot }
	served := NewConfigured(m.memory, Config{Canon: []string{allowed.FoundlingID, "foundling-unknown"}, Transport: transport})
	receipts, err := served.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 2 || receipts[0].FoundlingID != allowed.FoundlingID || receipts[0].State != "available" || receipts[1].FoundlingID != "foundling-unknown" || receipts[1].State != "unavailable" || receipts[1].Reason != "not_registered" {
		t.Fatalf("unexpected configured receipts: %+v", receipts)
	}
	if len(fetched) != 1 || fetched[0] != allowed.Source.Locator {
		t.Fatalf("unexpected source fetches: %v", fetched)
	}
	root, err := m.canonRoot(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{referenceFile(denied.FoundlingID), latestFile(denied.FoundlingID)} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("unconfigured snapshot exists: %s %v", relative, err)
		}
	}
	if _, err := served.CanonRecall(context.Background(), CanonRecallInput{FoundlingID: denied.FoundlingID, Limit: 10, BudgetBytes: 8192}); err == nil {
		t.Fatal("unconfigured source readable")
	}
	status, err := served.CanonStatus(context.Background())
	if err != nil || len(status) != 2 || status[0].State != "available" || status[1].Reason != "not_registered" {
		t.Fatal(status, err)
	}
	for _, selection := range [][]string{nil, {}} {
		fetched = nil
		receipts, err = NewConfigured(m.memory, Config{Canon: selection, Transport: transport}).Refresh(context.Background())
		if err != nil || len(receipts) != 0 || len(fetched) != 0 {
			t.Fatalf("empty configuration fetched/exposed references: %v %v %v", receipts, fetched, err)
		}
	}
}
