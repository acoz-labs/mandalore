package memory

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func rememberAll(t *testing.T, s *Service, writes ...Write) []string {
	t.Helper()
	ids := []string{}
	for _, w := range writes {
		if w.Basis == "" {
			w.Basis, w.Reason = "observation", "Synthetic ranking fixture"
		}
		r, err := s.Remember(w)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, r.RecordID)
	}
	return ids
}

func recalled(t *testing.T, s *Service, query string, scope *Scope) []string {
	t.Helper()
	p, err := s.Recall(query, scope, 50, 32768)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, h := range p.Current {
		ids = append(ids, h.RecordID)
	}
	return ids
}

// #227: a long record full of the question's common words used to outrank
// the short record that answers it.
func TestRecallRanksSpecificAnswerAboveLongCommonWordRecords(t *testing.T) {
	s := fixture(t)
	noise := strings.Repeat("The front door panel and the assistant on the home network were checked again in this session. ", 12)
	ids := rememberAll(t, s,
		Write{Kind: "project-state", Summary: "Homelab status after the rebuild", Body: noise},
		Write{Kind: "research-claim", Summary: "Assistant comparison notes", Body: noise},
		Write{Kind: "decision", Summary: "Front door assistant choice", Body: "The hallway panel at door.memory.example.com runs Beacon."},
	)
	for _, query := range []string{"Who is my front door assistant on door.memory.example.com?", "front door assistant"} {
		if got := recalled(t, s, query, nil); len(got) != 3 || got[0] != ids[2] {
			t.Fatalf("%q ranked %v, want %s first", query, got, ids[2])
		}
	}
}

// Ranking reorders matches; it never adds or drops one. Equal scores order
// by revision ID, so repeated reads agree.
func TestRecallRankingKeepsMatchesAndBreaksTiesByID(t *testing.T) {
	s := fixture(t)
	rememberAll(t, s,
		Write{Kind: "fact", Summary: "Tide table", Body: "High water at noon."},
		Write{Kind: "fact", Summary: "Tide table", Body: "High water at noon."},
		Write{Kind: "fact", Summary: "Tide table", Body: "High water at noon."},
		Write{Kind: "fact", Summary: "Harbour notes", Body: "The water taxi stops at noon and the tide turns later."},
		Write{Kind: "fact", Summary: "Unrelated", Body: "Something else entirely."},
	)
	records, states, err := s.store.graph()
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"tide", "water noon", "the", "tides turning", "quartz", "", "!!!"} {
		want := []string{}
		for _, heads := range RecallableHeads(records, states, time.Now()) {
			if relevance(heads[0], query) > 0 {
				want = append(want, heads[0].RecordID)
			}
		}
		got := recalled(t, s, query, nil)
		if !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))) {
			t.Fatalf("%q matched %v, want %v", query, got, want)
		}
		if again := recalled(t, s, query, nil); !slices.Equal(got, again) {
			t.Fatalf("%q unstable: %v then %v", query, got, again)
		}
	}
	p, err := s.Recall("tide table", nil, 50, 32768)
	if err != nil {
		t.Fatal(err)
	}
	ties := []string{p.Current[0].ID, p.Current[1].ID, p.Current[2].ID}
	if !slices.IsSorted(ties) {
		t.Fatalf("tied revisions out of ID order: %v", ties)
	}
}

// Term statistics come only from the records the query can return, so
// records in another scope can't change a scoped recall's order.
func TestScopedRankingIgnoresOtherScopes(t *testing.T) {
	s := fixture(t)
	a, b := Scope{"project", "tidepool"}, Scope{"project", "lanternfish"}
	rememberAll(t, s,
		Write{Kind: "decision", Scope: &a, Summary: "Primary database", Body: "Postgres 17 for the service."},
		Write{Kind: "project-state", Scope: &a, Summary: "Service status", Body: "The database and the service cache were reviewed."},
		Write{Kind: "fact", Scope: &a, Summary: "Cache", Body: "Redis is a cache for the service and may be flushed."},
	)
	before := recalled(t, s, "service database cache", &a)
	for range 6 {
		rememberAll(t, s, Write{Kind: "project-state", Scope: &b, Summary: "Database notes", Body: "Database database cache service postgres."})
	}
	if after := recalled(t, s, "service database cache", &a); !slices.Equal(before, after) || len(after) != 3 {
		t.Fatalf("other scope changed the order: %v then %v", before, after)
	}
}
