package memory

import (
	"fmt"
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

// Round 1 of #227 review, L2: an empty query lists current memory in
// revision-ID order whatever the kinds, as before.
func TestEmptyRecallListsInRevisionOrder(t *testing.T) {
	s := fixture(t)
	rememberAll(t, s,
		Write{Kind: "fact", Summary: "Tide table", Body: "High water at noon."},
		Write{Kind: "decision", Summary: "Harbour", Body: "Moor at the east pontoon."},
		Write{Kind: "project-state", Summary: "Boat", Body: "Antifouled in spring."},
		Write{Kind: "preference", Summary: "Sailing", Body: "Prefer mornings."},
	)
	p, err := s.Recall("", nil, 50, 32768)
	if err != nil || len(p.Current) != 4 {
		t.Fatal(p, err)
	}
	ids := []string{}
	for _, h := range p.Current {
		ids = append(ids, h.ID)
	}
	if !slices.IsSorted(ids) {
		t.Fatalf("empty recall out of revision order: %v", ids)
	}
}

// Round 2 of #227 review: a record matching one query term only in its body
// can be incidental, and with Names only those words anchor.
func TestRecallMarksIncidentalMatchesAndNamedAnchors(t *testing.T) {
	s := fixture(t)
	ids := rememberAll(t, s,
		Write{Kind: "decision", Summary: "Public API rate limit", Body: "600 requests per minute per key."},
		Write{Kind: "entity", Summary: "Beacon is the house voice assistant", Body: "Runs on the hub."},
	)
	for i := range 20 {
		rememberAll(t, s, Write{Kind: "project-state", Summary: fmt.Sprintf("Notes %d", i), Body: "Garden and shed."})
	}
	head := func(p Packet, record string) string {
		for _, r := range p.Current {
			if r.RecordID == record {
				return r.ID
			}
		}
		t.Fatalf("%s not recalled", record)
		return ""
	}
	// Round 4: it is incidental only while two of the query's other terms go
	// unmatched, or one does and the match is a number.
	for query, want := range map[string]bool{"600 divided by seven": true, "600 percent": true, "requests elsewhere today": true,
		"requests": false, "requests allowed": false} {
		p, err := s.store.Recall(Query{Text: query, AllScopes: true}, time.Now())
		if err != nil || p.Incidental[head(p, ids[0])] != want {
			t.Errorf("%q: incidental %v, want %v (%v)", query, p.Incidental[head(p, ids[0])], want, err)
		}
	}
	p, _ := s.store.Recall(Query{Text: "api rate limit", AllScopes: true}, time.Now())
	if p.Incidental[head(p, ids[0])] {
		t.Fatal("summary match marked incidental")
	}
	named, _ := s.store.Recall(Query{Text: "what is beacon", AllScopes: true, Names: []string{"Beacon?"}}, time.Now())
	unnamed, _ := s.store.Recall(Query{Text: "what is beacon", AllScopes: true, Names: []string{}}, time.Now())
	if named.Anchor[head(named, ids[1])] == 0 || unnamed.Anchor[head(unnamed, ids[1])] != 0 {
		t.Fatalf("names: anchored %v, unnamed %v", named.Anchor, unnamed.Anchor)
	}
}
