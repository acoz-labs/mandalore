package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func progressiveFixture(t *testing.T) (*API, string, string, string) {
	t.Helper()
	documents := map[string]string{}
	for i := 0; i < 12; i++ {
		documents[fmt.Sprintf("note-%02d.md", i)] = fmt.Sprintf("InventoryMarker document %02d. ", i) + strings.Repeat("Synthetic historical evidence. ", 150)
	}
	return progressiveDocuments(t, documents)
}

func progressiveDocuments(t *testing.T, documents map[string]string) (*API, string, string, string) {
	t.Helper()
	a := fixture(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range documents {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	preview := foundlingCall(t, a, "foundling_preview", FoundlingPreviewInput{Source: memory.FoundlingSource{Kind: "local", Locator: "source-progressive"}, Root: root})
	if !preview.OK {
		t.Fatal(preview)
	}
	v := preview.Result.(foundlings.Observation)
	r := foundlingCall(t, a, "foundling_register", FoundlingRegisterInput{Name: "Progressive references", Description: "Synthetic bounded retrieval", Source: v.Source, Pin: v.Pin, Root: root, Reason: "Explicit synthetic fixture"})
	if !r.OK {
		t.Fatal(r)
	}
	receipt := r.Result.(FoundlingMutationResult).Registration
	return a, root, receipt.FoundlingID, receipt.ID
}

func TestProgressiveFoundlingBudgetsAndFreshContinuation(t *testing.T) {
	documents := map[string]string{}
	for i := 0; i < 4; i++ {
		documents[fmt.Sprintf("note-%02d.md", i)] = "InventoryMarker " + strings.Repeat("\x01", 1400)
	}
	a, root, id, revision := progressiveDocuments(t, documents)
	before := inlineInventory(t, a.service.Root())
	a.ReadOnly = true
	input := map[string]any{"foundling_id": id, "query": "InventoryMarker", "budget_bytes": 2048}
	out := foundlingCall(t, a, "foundling_search", input)
	if out.OK || out.Error.Code != "foundling.budget" || out.Error.WriteMayHaveOccurred || out.Error.Retryable {
		t.Fatalf("first-item budget failure not distinguished: %+v", out.Error)
	}
	// Measure the complete one-item representation, then use that exact budget.
	// Metadata and JSON escaping are included rather than guessed from text size.
	input["budget_bytes"], input["limit"] = 32768, 1
	one := foundlingCall(t, a, "foundling_search", input)
	if !one.OK {
		t.Fatal(one.Error)
	}
	oneBytes, err := json.Marshal(one.Result)
	if err != nil || len(oneBytes) <= 2048 || len(oneBytes) > 32768 {
		t.Fatalf("fixture budget: bytes=%d err=%v", len(oneBytes), err)
	}
	input["budget_bytes"] = len(oneBytes)
	input["limit"] = 10
	out = foundlingCall(t, a, "foundling_search", input)
	if !out.OK {
		t.Fatal(out.Error)
	}
	p := out.Result.(foundlings.SearchResult)
	encoded, err := json.Marshal(p)
	if err != nil || len(encoded) != len(oneBytes) || len(p.Items) != 1 || p.NextOffset == nil || *p.NextOffset != 1 {
		t.Fatalf("JSON expansion budget: bytes=%d items=%d next=%v err=%v", len(encoded), len(p.Items), p.NextOffset, err)
	}
	input["budget_bytes"], input["registration_revision_id"], input["offset"] = 32768, revision, *p.NextOffset
	out = foundlingCall(t, a, "foundling_search", input)
	if !out.OK || len(out.Result.(foundlings.SearchResult).Items) != 3 || out.Result.(foundlings.SearchResult).NextOffset != nil {
		t.Fatal("larger-budget continuation could not retrieve remainder", out.Error)
	}
	if err := os.WriteFile(filepath.Join(root, "note-00.md"), []byte("Changed externally."), 0600); err != nil {
		t.Fatal(err)
	}
	out = foundlingCall(t, a, "foundling_search", input)
	if out.OK || out.Error.Code != "foundling.changed" || out.Error.WriteMayHaveOccurred {
		t.Fatal("stale continuation was not freshly refused", out.Error)
	}
	if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
		t.Fatal("retrieval or refusal changed signet content")
	}
}

func TestProgressiveFoundlingInvalidRangesAndEmptyPages(t *testing.T) {
	a, _, id, revision := progressiveFixture(t)
	for _, bad := range []map[string]any{
		{"budget_bytes": 0}, {"budget_bytes": 2047}, {"budget_bytes": 32769},
		{"excerpt_bytes": 0}, {"excerpt_bytes": 127}, {"excerpt_bytes": 1025},
		{"offset": -1}, {"offset": 10001}, {"offset": 1},
	} {
		bad["foundling_id"], bad["query"] = id, "InventoryMarker"
		out := foundlingCall(t, a, "foundling_search", bad)
		if out.OK || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred {
			t.Fatal("invalid explicit range accepted", bad, out.Error)
		}
	}
	for _, in := range []map[string]any{
		{"foundling_id": id, "query": "NoSuchMarker"},
		{"foundling_id": id, "query": "InventoryMarker", "offset": 12, "registration_revision_id": revision},
	} {
		out := foundlingCall(t, a, "foundling_search", in)
		if !out.OK {
			t.Fatal(out.Error)
		}
		p := out.Result.(foundlings.SearchResult)
		if len(p.Items) != 0 || p.NextOffset != nil || p.RegistrationID != revision || p.Notice == "" {
			t.Fatal("empty page lacks safe terminal metadata")
		}
	}
}

func TestProgressiveFoundlingInitialDefaults(t *testing.T) {
	a, root, id, revision := progressiveFixture(t)
	before, sourceBefore := inlineInventory(t, a.service.Root()), inlineInventory(t, root)
	a.ReadOnly = true
	search := foundlingCall(t, a, "foundling_search", FoundlingSearchInput{FoundlingID: id, Query: "InventoryMarker"})
	if !search.OK {
		t.Fatal(search)
	}
	p := search.Result.(foundlings.SearchResult)
	if len(p.Items) != 3 || p.MatchingCount != 12 || !p.Truncated {
		t.Errorf("initial page: items=%d matches=%d truncated=%v", len(p.Items), p.MatchingCount, p.Truncated)
	}
	for _, e := range p.Items {
		if len(e.Text) > 512 || !e.Unreviewed || e.Origin.RegistrationRevisionID != revision || e.Complete || e.NextOffset == nil {
			t.Errorf("invalid initial preview: content bytes=%d", len(e.Text))
		}
	}
	encoded, err := json.Marshal(p)
	if err != nil || len(encoded) > 8192 {
		t.Errorf("initial serialized result: %d bytes, %v", len(encoded), err)
	}
	read := foundlingCall(t, a, "foundling_read", FoundlingReadInput{FoundlingID: id, RegistrationID: revision, Locator: "note-00.md"})
	if !read.OK {
		t.Fatal(read)
	}
	e := read.Result.(foundlings.Excerpt)
	if len(e.Text) != 1024 || e.NextOffset == nil || *e.NextOffset != 1024 || e.Complete {
		t.Errorf("default read: content bytes=%d next=%v complete=%v", len(e.Text), e.NextOffset, e.Complete)
	}
	if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) || !reflect.DeepEqual(sourceBefore, inlineInventory(t, root)) {
		t.Fatal("read-only retrieval changed stored content")
	}
}

func TestProgressiveFoundlingPinnedPagination(t *testing.T) {
	a, _, id, revision := progressiveFixture(t)
	type page struct {
		Items          []foundlings.Excerpt `json:"items"`
		MatchingCount  int                  `json:"matching_count"`
		RegistrationID string               `json:"registration_revision_id"`
		Offset         int                  `json:"offset"`
		NextOffset     *int                 `json:"next_offset"`
		Truncated      bool                 `json:"truncated"`
	}
	names, offset := []string{}, 0
	for {
		input := map[string]any{"foundling_id": id, "query": "InventoryMarker", "registration_revision_id": revision, "offset": offset}
		out := foundlingCall(t, a, "foundling_search", input)
		if !out.OK {
			t.Fatalf("pinned page refused: %+v", out.Error)
		}
		data, err := json.Marshal(out.Result)
		if err != nil {
			t.Fatal(err)
		}
		var p page
		if err := json.Unmarshal(data, &p); err != nil {
			t.Fatal(err)
		}
		if p.RegistrationID != revision || p.Offset != offset || p.MatchingCount != 12 || !p.Truncated || len(p.Items) == 0 || len(p.Items) > 3 {
			t.Fatalf("invalid page metadata at %d", offset)
		}
		for _, e := range p.Items {
			if e.Origin.RegistrationRevisionID != revision || len(e.Origin.ContentSHA256) != 64 {
				t.Fatal("page lost verifiable provenance")
			}
			names = append(names, e.Origin.RelativeLocator)
		}
		if p.NextOffset == nil {
			break
		}
		if *p.NextOffset != offset+len(p.Items) || *p.NextOffset > 12 {
			t.Fatal("continuation skipped or repeated a rank")
		}
		offset = *p.NextOffset
	}
	if len(names) != 12 {
		t.Fatalf("only %d of twelve matches retrievable", len(names))
	}
	for i, name := range names {
		if name != fmt.Sprintf("note-%02d.md", i) {
			t.Fatal("lost deterministic ordering", names)
		}
	}
	// No stale or unpinned continuation can silently use a different registration.
	for _, input := range []map[string]any{
		{"foundling_id": id, "query": "InventoryMarker", "offset": 3},
		{"foundling_id": id, "query": "InventoryMarker", "registration_revision_id": "registration-stale", "offset": 0},
	} {
		out := foundlingCall(t, a, "foundling_search", input)
		if out.OK {
			t.Fatal("unverifiable continuation accepted")
		}
	}
}

func TestProgressiveFoundlingExplicitExpansion(t *testing.T) {
	a, root, id, revision := progressiveFixture(t)
	input := map[string]any{"foundling_id": id, "query": "InventoryMarker", "limit": 10, "excerpt_bytes": 1024, "budget_bytes": 32768}
	out := foundlingCall(t, a, "foundling_search", input)
	if !out.OK {
		t.Fatalf("explicit larger search refused: %+v", out.Error)
	}
	p := out.Result.(foundlings.SearchResult)
	if len(p.Items) != 10 || len(p.Items[0].Text) != 1024 {
		t.Fatal("explicit preview expansion lost")
	}
	want, err := os.ReadFile(filepath.Join(root, "note-00.md"))
	if err != nil {
		t.Fatal(err)
	}
	limit := 8192
	read := foundlingCall(t, a, "foundling_read", FoundlingReadInput{FoundlingID: id, RegistrationID: revision, Locator: "note-00.md", Limit: &limit})
	if !read.OK {
		t.Fatal(read)
	}
	e := read.Result.(foundlings.Excerpt)
	if e.Text != string(want) || !e.Complete || e.Truncated || e.NextOffset != nil {
		t.Fatal("deliberate full read lost source text")
	}
	if err := a.service.Validate(); err != nil {
		t.Fatal(err)
	}
	if out := a.Call(context.Background(), "memory_journal", []byte(`{}`)); !out.OK || len(out.Result.(memory.Page[memory.JournalEntry]).Items) != 0 {
		t.Fatal("retrieval caused journaling")
	}
}
