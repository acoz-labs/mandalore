package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// This is a quality characterization of the public service, not a semantic
// search score. The expected IDs and documented misses are deliberately exact.
func TestRetrievalQualityMatrix(t *testing.T) {
	store := fixtureStore(t)
	s, err := OpenService(store.Root, revision("revision-author").Authorship)
	if err != nil {
		t.Fatal(err)
	}
	first, second := Scope{"project", "copper-finch"}, Scope{"project", "green-wren"}
	add := func(id, record string, scope Scope, summary, body string, parents ...string) {
		t.Helper()
		r := revision(id, parents...)
		r.RecordID, r.Scope, r.Summary, r.Body = record, scope, summary, body
		if err := store.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	add("revision-global", "record-global", Scope{"signet", s.ID()}, "Answer preference", "Use concise answers.")
	add("revision-old-name", "record-name", first, "Fictional project name", "Copper Finch.")
	add("revision-current-name", "record-name", first, "Fictional project name", "Silver Heron, formerly Copper Finch.", "revision-old-name")
	add("revision-other-name", "record-other-name", second, "Fictional project name", "Green Wren.")
	add("revision-old-process", "record-process", first, "Routine workflow", "Require paperseal approval for every edit.")
	add("revision-current-process", "record-process", first, "Routine workflow", "Perform authorized reversible edits autonomously and verify results.", "revision-old-process")
	add("revision-project-preference", "record-project-preference", first, "Answer preference", "For this project include a detailed verification report.")
	add("revision-conflict-root", "record-conflict", first, "Deployment color", "White.")
	add("revision-conflict-blue", "record-conflict", first, "Deployment color", "Blue.", "revision-conflict-root")
	add("revision-conflict-gold", "record-conflict", first, "Deployment color", "Gold.", "revision-conflict-root")
	future := revision("revision-future-name", "revision-current-name")
	future.RecordID, future.Scope = "record-name", first
	future.Summary, future.Body = "Fictional project name", "Future Kestrel."
	future.RecordedAt, future.EffectiveFrom = "2999-01-01T00:00:00Z", "2999-01-01T00:00:00Z"
	if err := store.Put(future); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, s.Root())
	for _, tc := range []struct {
		name, query string
		scope       *Scope
		want        []string
		conflict    bool
	}{
		{"unscoped-is-only-global", "preference", nil, []string{"revision-global"}, false},
		{"scoped-does-not-inherit-global", "preference", &first, []string{"revision-project-preference"}, true},
		{"current-name", "Silver Heron", &first, []string{"revision-current-name"}, true},
		{"retained-alias-finds-current", "Copper Finch", &first, []string{"revision-current-name"}, true},
		{"terminal-period-is-known-lexical-miss", "Finch", &first, nil, true},
		{"shared-keywords-stay-in-project", "fictional project name", &second, []string{"revision-other-name"}, false},
		{"historical-only-word-not-resurrected", "paperseal", &first, nil, true},
		{"future-not-current", "Kestrel", &first, nil, true},
		{"no-match-still-signals-scope-conflict", "quartz", &first, nil, true},
		{"conflict-is-not-guidance", "deployment", &first, nil, true},
		{"prose-inflection", "edit", &first, []string{"revision-current-process"}, true},
		{"absent-synonym-is-known-lexical-miss", "brevity", nil, nil, false},
		{"empty-project-inventory", "", &second, []string{"revision-other-name"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := s.Recall(tc.query, tc.scope, 5, 4096)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, r := range p.Current {
				got = append(got, r.ID)
			}
			slices.Sort(got)
			if !slices.Equal(got, tc.want) || p.MatchingCount != len(tc.want) || p.Truncated {
				t.Fatalf("current IDs/count: got %+v, want %v", p, tc.want)
			}
			if tc.conflict {
				if p.ConflictCount != 1 || len(p.Conflicts) != 1 || p.Conflicts[0].RecordID != "record-conflict" || !slices.Equal(p.Conflicts[0].HeadIDs, []string{"revision-conflict-blue", "revision-conflict-gold"}) {
					t.Fatalf("conflict hidden or altered: %+v", p)
				}
			} else if p.ConflictCount != 0 || len(p.Conflicts) != 0 {
				t.Fatalf("conflict leaked across scopes: %+v", p)
			}
			data, err := json.Marshal(p)
			if err != nil || len(data) > 4096 || !strings.Contains(p.Notice, "do not prove absence") {
				t.Fatal("response bounds/limitations", len(data), err, p.Notice)
			}
		})
	}
	// Page by a single item: routing must retain stable pre-rename identifiers,
	// not treat names or the current working directory as stored scope IDs.
	var scopes []Scope
	for offset := 0; ; {
		p, err := s.ScopePage(offset, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range p.Items {
			scopes = append(scopes, item.Scope)
		}
		if p.NextOffset == nil {
			break
		}
		if *p.NextOffset <= offset {
			t.Fatal("inventory did not advance")
		}
		offset = *p.NextOffset
	}
	if !slices.Equal(scopes, []Scope{first, second, {"signet", s.ID()}}) {
		t.Fatal("scope discovery", scopes)
	}
	history, err := s.History("record-process")
	if err != nil || len(history) != 2 || history[0].ID != "revision-old-process" || history[1].ID != "revision-current-process" {
		t.Fatal("historical process not retained", history, err)
	}
	if !reflect.DeepEqual(before, snapshot(t, s.Root())) {
		t.Fatal("quality reads mutated the signet")
	}
}

func TestRetrievalSameServiceObservesExternalChanges(t *testing.T) {
	s := fixture(t)
	writer, err := OpenService(s.Root(), s.author)
	if err != nil {
		t.Fatal(err)
	}
	input := Write{Kind: "decision", Summary: "Project name", Body: "Copper Finch", Basis: "user-direction", Reason: "Synthetic external write"}
	first, err := writer.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	assertCurrent := func(id string, conflicts int) {
		t.Helper()
		p, err := s.Recall("project", nil, 5, 4096)
		if err != nil || p.ConflictCount != conflicts {
			t.Fatal("fresh read", p, err)
		}
		if id == "" && len(p.Current) != 0 || id != "" && (len(p.Current) != 1 || p.Current[0].ID != id) {
			t.Fatal("stale or conflicted guidance", p)
		}
	}
	assertCurrent(first.ID, 0)
	input.RecordID, input.Supersedes, input.Body = first.RecordID, []string{first.ID}, "Silver Heron"
	second, err := writer.Remember(input)
	if err != nil {
		t.Fatal(err)
	}
	assertCurrent(second.ID, 0)
	input.Body = "Amber Lark"
	if _, err := writer.Remember(input); err != nil {
		t.Fatal(err)
	}
	assertCurrent("", 1)
	// The same handle must fail, not return previously valid cached evidence.
	path := filepath.Join(s.Root(), "memory/sources", second.Evidence.SourceRefs[0]+".json")
	if err := os.WriteFile(path, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, s.Root())
	if _, err := s.Recall("", nil, 5, 4096); err == nil {
		t.Fatal("corruption hidden by prior successful read")
	}
	if !reflect.DeepEqual(before, snapshot(t, s.Root())) {
		t.Fatal("failed read repaired data")
	}
	replacement := s.store.Signet
	replacement.ID = "signet-replaced"
	data, _ := json.Marshal(replacement)
	if err := os.WriteFile(filepath.Join(s.Root(), "signet.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recall("", nil, 5, 4096); !errors.Is(err, ErrIdentityChanged) {
		t.Fatal("replacement not rejected", err)
	}
}
