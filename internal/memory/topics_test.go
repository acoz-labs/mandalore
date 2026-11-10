package memory

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var atlas = Scope{Kind: "project", ID: "atlas-launch"}

func nameAtlas(t *testing.T, s *Service) ScopeLabel {
	t.Helper()
	l, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Atlas launch", Description: "Shipping the synthetic Atlas launch checklist.", Reason: "Named in a synthetic test"})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestScopeLabelsAreRevisableEvidenceWithVisibleConflicts(t *testing.T) {
	s := fixture(t)
	root := nameAtlas(t, s)
	if _, err := os.Stat(filepath.Join(s.Root(), "memory/scopes/project/atlas-launch/labels", root.ID+".json")); err != nil {
		t.Fatal("label not at its canonical path:", err)
	}
	if _, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Again", Description: "d", Reason: "r"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("second root: %v", err)
	}
	// Format 1 keeps a fork visible, as records do.
	a, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Atlas launch v2", Description: "Renamed.", Reason: "r", Supersedes: []string{root.ID}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Atlas rollout", Description: "Concurrently renamed.", Reason: "r", Supersedes: []string{root.ID}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := s.ScopePage("", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	var labels []LabelHead
	for _, item := range page.Items {
		if item.Scope == atlas {
			labels = item.Labels
			if item.RecordCount != 0 {
				t.Fatal("named empty scope counts records")
			}
		}
	}
	ids := []string{}
	for _, l := range labels {
		ids = append(ids, l.ID)
	}
	slices.Sort(ids)
	want := []string{a.ID, b.ID}
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("conflicting heads %v, want %v", ids, want)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, in := range map[string]ScopeLabelWrite{
		"signet scope":    {Scope: Scope{Kind: "signet", ID: s.ID()}, Name: "n", Description: "d", Reason: "r"},
		"bad scope id":    {Scope: Scope{Kind: "project", ID: "No"}, Name: "n", Description: "d", Reason: "r"},
		"name too long":   {Scope: atlas, Name: strings.Repeat("n", 257), Description: "d", Reason: "r", Supersedes: []string{a.ID}},
		"newline in name": {Scope: atlas, Name: "two\nlines", Description: "d", Reason: "r", Supersedes: []string{a.ID}},
		"U+2028 in name":  {Scope: atlas, Name: "two\u2028lines", Description: "d", Reason: "r", Supersedes: []string{a.ID}},
		"U+2029 in name":  {Scope: atlas, Name: "two\u2029paragraphs", Description: "d", Reason: "r", Supersedes: []string{a.ID}},
		"blank desc":      {Scope: atlas, Name: "n", Description: " ", Reason: "r", Supersedes: []string{a.ID}},
		"desc too long":   {Scope: atlas, Name: "n", Description: strings.Repeat("d", 1025), Reason: "r", Supersedes: []string{a.ID}},
		"NUL reason":      {Scope: atlas, Name: "n", Description: "d", Reason: "r\x00", Supersedes: []string{a.ID}},
		"foreign label":   {Scope: Scope{Kind: "task", ID: "other-task"}, Name: "n", Description: "d", Reason: "r", Supersedes: []string{a.ID}},
		"unknown label":   {Scope: atlas, Name: "n", Description: "d", Reason: "r", Supersedes: []string{"label-missing"}},
		"repeated label":  {Scope: atlas, Name: "n", Description: "d", Reason: "r", Supersedes: []string{a.ID, a.ID}},
	} {
		if _, err := s.NameScope(in); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	at := strings.Repeat("é", 128) // 256 bytes
	if _, err := s.NameScope(ScopeLabelWrite{Scope: Scope{Kind: "account", ID: "limit-account"}, Name: at, Description: strings.Repeat("d", 1024), Reason: "r"}); err != nil {
		t.Fatal("limits refused:", err)
	}
}

func TestScopeLabelFormatTwoRefusesStaleHeads(t *testing.T) {
	s := formatTwoService(t)
	root := nameAtlas(t, s)
	next, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Atlas", Description: "d", Reason: "r", Supersedes: []string{root.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Stale", Description: "d", Reason: "r", Supersedes: []string{root.ID}}); !errors.Is(err, ErrStaleLabels) || !errors.Is(err, ErrStaleHeads) {
		t.Fatalf("stale: %v", err)
	}
	if _, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Fresh", Description: "d", Reason: "r", Supersedes: []string{next.ID}}); err != nil {
		t.Fatal(err)
	}
}

func TestScopeTreeRefusesNoncanonicalFiles(t *testing.T) {
	for name, place := range map[string]func(root string, l ScopeLabel) (string, any){
		"label under another scope": func(root string, l ScopeLabel) (string, any) {
			return filepath.Join(root, "memory/scopes/project/other-scope/labels", l.ID+".json"), l
		},
		"stray file": func(root string, l ScopeLabel) (string, any) {
			return filepath.Join(root, "memory/scopes/project/atlas-launch/notes.json"), l
		},
		"unscoped entry under scopes": func(root string, l ScopeLabel) (string, any) {
			e := JournalEntry{Version: 1, ID: "event-stray", Kind: "k", Summary: "s", RecordedAt: "2026-01-02T03:04:05Z", Authorship: l.Authorship}
			return filepath.Join(root, "memory/scopes/project/atlas-launch/events/2026/01/event-stray.json"), e
		},
		"scoped entry under events": func(root string, l ScopeLabel) (string, any) {
			e := JournalEntry{Version: 2, ID: "event-stray", Kind: "k", Summary: "s", RecordedAt: "2026-01-02T03:04:05Z", Authorship: l.Authorship, Scope: &atlas}
			return filepath.Join(root, "memory/events/2026/01/event-stray.json"), e
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := fixture(t)
			l := nameAtlas(t, s)
			path, doc := place(s.Root(), l)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(doc)
			if err := os.WriteFile(path, b, 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.Validate(); err == nil {
				t.Fatal("validated")
			}
		})
	}
}

func TestScopedJournalLeavesUnscopedEntriesUnchanged(t *testing.T) {
	s := fixture(t)
	plain, err := s.AppendJournalIdempotent("request-plain-entry-0001", "outcome", "Unscoped synthetic outcome", nil)
	if err != nil {
		t.Fatal(err)
	}
	plainPath, _ := journalPath(plain)
	before, err := os.ReadFile(filepath.Join(s.Root(), plainPath))
	if err != nil {
		t.Fatal(err)
	}
	if plain.Version != 1 || strings.Contains(string(before), `"scope"`) {
		t.Fatalf("unscoped entry changed shape: %s", before)
	}
	time.Sleep(time.Millisecond)
	scoped, err := s.AppendJournalIdempotent("request-scoped-entry-01", "outcome", "Atlas checklist drafted", &atlas)
	if err != nil {
		t.Fatal(err)
	}
	if scoped.Version != 2 || *scoped.Scope != atlas {
		t.Fatalf("scoped entry %+v", scoped)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), "memory/scopes/project/atlas-launch/events", time.Now().UTC().Format("2006/01"), scoped.ID+".json")); err != nil {
		t.Fatal(err)
	}
	all, err := s.Journal("", nil, 10)
	if err != nil || len(all) != 2 || all[0].ID != scoped.ID || all[1].ID != plain.ID {
		t.Fatalf("unfiltered %+v %v", all, err)
	}
	only, err := s.Journal("", &atlas, 10)
	if err != nil || len(only) != 1 || only[0].ID != scoped.ID {
		t.Fatalf("filtered %+v %v", only, err)
	}
	if got, err := s.Journal("unrelated", &atlas, 10); err != nil || len(got) != 0 {
		t.Fatalf("query ignored with scope: %+v %v", got, err)
	}
	if got, err := s.Journal("checklist", &atlas, 10); err != nil || len(got) != 1 {
		t.Fatalf("query with scope: %+v %v", got, err)
	}
	if _, err := s.Journal("", &Scope{Kind: "signet", ID: s.ID()}, 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("signet filter: %v", err)
	}
	if _, err := s.AppendJournal("k", "s", &Scope{Kind: "signet", ID: s.ID()}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("signet scope append: %v", err)
	}
	// A retry of the unscoped request still matches, and a scoped retry with a
	// different scope is a different request.
	if again, err := s.AppendJournalIdempotent("request-plain-entry-0001", "outcome", "Unscoped synthetic outcome", nil); err != nil || again.ID != plain.ID {
		t.Fatalf("unscoped retry %v", err)
	}
	if _, err := s.AppendJournalIdempotent("request-scoped-entry-01", "outcome", "Atlas checklist drafted", &Scope{Kind: "task", ID: "other-task"}); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("changed scope: %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(s.Root(), plainPath))
	if string(after) != string(before) {
		t.Fatal("unscoped entry bytes changed")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestUnscopedJournalIntentKeepsItsShape(t *testing.T) {
	s := fixture(t)
	if _, err := s.AppendJournalIdempotent("request-plain-entry-0001", "outcome", "Unscoped synthetic outcome", nil); err != nil {
		t.Fatal(err)
	}
	path, digest, err := s.intent("request-plain-entry-0001", "journal", []string{"outcome", "Unscoped synthetic outcome"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("unscoped intent digest or path changed:", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	var journal map[string]any
	_ = json.Unmarshal(raw["journal"], &journal)
	if len(raw) != 2 || !strings.Contains(string(raw["digest"]), digest) || journal["scope"] != nil {
		t.Fatalf("intent shape changed: %s", b)
	}
}

func TestPendingIntentsKnowNewTargets(t *testing.T) {
	s := fixture(t)
	l, err := s.NameScopeIdempotent("request-name-atlas-0001", ScopeLabelWrite{Scope: atlas, Name: "Atlas launch", Description: "d", Reason: "r"})
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.AppendJournalIdempotent("request-scoped-entry-01", "outcome", "Atlas checklist drafted", &atlas)
	if err != nil {
		t.Fatal(err)
	}
	if pending, err := s.store.PendingIntents(); err != nil || len(pending) != 0 {
		t.Fatalf("completed writes pending: %v %v", pending, err)
	}
	ePath, _ := journalPath(e)
	for _, rel := range []string{labelPath(l), ePath} {
		if err := os.Remove(filepath.Join(s.Root(), rel)); err != nil {
			t.Fatal(err)
		}
	}
	if pending, err := s.store.PendingIntents(); err != nil || len(pending) != 2 {
		t.Fatalf("interrupted writes: %v %v", pending, err)
	}
	// Retrying republishes exactly what the intents hold.
	if again, err := s.NameScopeIdempotent("request-name-atlas-0001", ScopeLabelWrite{Scope: atlas, Name: "Atlas launch", Description: "d", Reason: "r"}); err != nil || again.ID != l.ID {
		t.Fatalf("label retry %v", err)
	}
	if again, err := s.AppendJournalIdempotent("request-scoped-entry-01", "outcome", "Atlas checklist drafted", &atlas); err != nil || again.ID != e.ID {
		t.Fatalf("journal retry %v", err)
	}
	if pending, err := s.store.PendingIntents(); err != nil || len(pending) != 0 {
		t.Fatalf("after retry: %v %v", pending, err)
	}
	if _, err := s.NameScopeIdempotent("request-name-atlas-0001", ScopeLabelWrite{Scope: atlas, Name: "Other", Description: "d", Reason: "r"}); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("changed label input: %v", err)
	}
	if _, err := s.RememberIdempotent("request-name-atlas-0001", validWrite()); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("label key reused for remember: %v", err)
	}
}

func TestNameScopeRefusalWritesNothing(t *testing.T) {
	s := fixture(t)
	before := written(t, s.Root())
	if _, err := s.NameScopeIdempotent("request-name-atlas-0001", ScopeLabelWrite{Scope: atlas, Name: strings.Repeat("n", 257), Description: "d", Reason: "r"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
	if after := written(t, s.Root()); len(after) != len(before) {
		t.Fatalf("refused naming wrote %v", after)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), scopesDir)); !os.IsNotExist(err) {
		t.Fatal("refused naming created memory/scopes")
	}
}

func TestScopePageFindsTopicsByNameAndStaysBounded(t *testing.T) {
	s := fixture(t)
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Unrelated", Body: "Synthetic", Basis: "observation", Reason: "r", Scope: &Scope{Kind: "project", ID: "zeta-project"}}); err != nil {
		t.Fatal(err)
	}
	nameAtlas(t, s)
	if _, err := s.AppendJournal("outcome", "Journal only", &Scope{Kind: "task", ID: "journal-only"}); err != nil {
		t.Fatal(err)
	}
	page, err := s.ScopePage("atlas launch", 0, 50)
	if err != nil || len(page.Items) == 0 || page.Items[0].Scope != atlas || page.Items[0].Labels[0].Name != "Atlas launch" {
		t.Fatalf("query by name: %+v %v", page, err)
	}
	if page, err = s.ScopePage("checklist", 0, 50); err != nil || len(page.Items) != 1 || page.Items[0].Scope != atlas {
		t.Fatalf("query by description: %+v %v", page, err)
	}
	if page, err = s.ScopePage("zeta-project", 0, 50); err != nil || len(page.Items) != 1 || page.Items[0].Scope.ID != "zeta-project" {
		t.Fatalf("query by ID: %+v %v", page, err)
	}
	if page, err = s.ScopePage("", 0, 50); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range page.Items {
		if item.Scope.ID == "journal-only" && item.RecordCount == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("journal-only scope not listed")
	}
	// The store inventory that memory_context reads stays as it was.
	plain, err := s.Scopes()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range plain {
		if item.Labels != nil || item.Scope == atlas {
			t.Fatalf("store inventory changed: %+v", item)
		}
	}
	for i := range 40 {
		id := "bulk-" + strings.Repeat("x", 3) + string(rune('a'+i%26)) + string(rune('a'+i/26))
		if _, err := s.NameScope(ScopeLabelWrite{Scope: Scope{Kind: "project", ID: id}, Name: "Bulk topic", Description: strings.Repeat("d", 1024), Reason: "r"}); err != nil {
			t.Fatal(err)
		}
	}
	page, err = s.ScopePage("bulk", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(page)
	if len(b) > 32768 || !page.Truncated || page.NextOffset == nil {
		t.Fatalf("page %d bytes, truncated %v", len(b), page.Truncated)
	}
	if _, err := s.ScopePage(strings.Repeat("q", 2049), 0, 5); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
}

func remember(t *testing.T, s *Service, w Write) Revision {
	t.Helper()
	r, err := s.Remember(w)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPrimeOrdersSectionsNewestFirst(t *testing.T) {
	s := formatTwoService(t)
	nameAtlas(t, s)
	state := remember(t, s, Write{Kind: "project-state", Summary: "Atlas: drafting", Body: "Checklist half done.", Basis: "observation", Reason: "r", Scope: &atlas})
	var facts []Revision
	for _, summary := range []string{"First fact", "Second fact", "Third fact"} {
		time.Sleep(time.Millisecond)
		facts = append(facts, remember(t, s, Write{Kind: "fact", Summary: summary, Body: "Body of " + summary, Basis: "observation", Reason: "r", Scope: &atlas}))
	}
	remember(t, s, Write{Kind: "fact", Summary: "Elsewhere", Body: "Other scope", Basis: "observation", Reason: "r", Scope: &Scope{Kind: "project", ID: "other-project"}})
	remember(t, s, Write{Kind: "fact", Summary: "Signet-wide", Body: "Not in the topic", Basis: "observation", Reason: "r"})
	for _, summary := range []string{"Journal one", "Journal two"} {
		time.Sleep(time.Millisecond)
		if _, err := s.AppendJournal("outcome", summary, &atlas); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.AppendJournal("outcome", "Unscoped", nil); err != nil {
		t.Fatal(err)
	}
	// Correct the state, then withdraw a fact.
	corrected := remember(t, s, Write{Kind: "project-state", Summary: "Atlas: reviewing", Body: "Checklist done.", Basis: "observation", Reason: "Moved on", Scope: &atlas, RecordID: state.RecordID, Supersedes: []string{state.ID}})
	if _, err := s.ChangeVisibility(context.Background(), "withdraw", VisibilityWrite{RecordID: facts[1].RecordID, ContentHeads: []string{facts[1].ID}, VisibilityHeads: []string{}, Reason: "Wrong"}); err != nil {
		t.Fatal(err)
	}
	p, err := s.Prime(atlas, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Labels) != 1 || p.Labels[0].Description == "" || len(p.State) != 1 || p.State[0].ID != corrected.ID || p.State[0].Body != "Checklist done." {
		t.Fatalf("labels/state %+v %+v", p.Labels, p.State)
	}
	if len(p.Current) != 2 || p.Current[0].ID != facts[2].ID || p.Current[1].ID != facts[0].ID {
		t.Fatalf("current %+v", p.Current)
	}
	if len(p.Journal) != 2 || p.Journal[0].Summary != "Journal two" || p.Truncated {
		t.Fatalf("journal %+v truncated %v", p.Journal, p.Truncated)
	}
	b, _ := json.Marshal(p)
	order := []string{`"labels"`, `"state"`, `"conflicts"`, `"current"`, `"journal"`}
	for i := 1; i < len(order); i++ {
		if strings.Index(string(b), order[i-1]) > strings.Index(string(b), order[i]) {
			t.Fatalf("section order: %s", b)
		}
	}
	if strings.Contains(string(b), "Body of") {
		t.Fatal("current items carry bodies")
	}
	if _, err := s.Prime(Scope{Kind: "signet", ID: s.ID()}, 8192); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.Prime(atlas, 1023); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
}

func TestPrimeSelectsEveryHeadAndDisclosesConflicts(t *testing.T) {
	s := fixture(t)
	for i := range 12 {
		remember(t, s, Write{Kind: "fact", Summary: "Fact " + string(rune('a'+i)), Body: "b", Basis: "observation", Reason: "r", Scope: &atlas})
	}
	first := remember(t, s, Write{Kind: "decision", Summary: "Choice", Body: "One", Basis: "user-direction", Reason: "r", Scope: &atlas})
	for _, body := range []string{"Two", "Three"} {
		remember(t, s, Write{Kind: "decision", Summary: "Choice", Body: body, Basis: "user-direction", Reason: "r", Scope: &atlas, RecordID: first.RecordID, Supersedes: []string{first.ID}})
	}
	p, err := s.Prime(atlas, 32768)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) != 12 || p.MatchingCounts.Current != 12 || len(p.Conflicts) != 1 || p.Conflicts[0].RecordID != first.RecordID || len(p.Conflicts[0].HeadIDs) != 2 {
		t.Fatalf("current %d conflicts %+v", len(p.Current), p.Conflicts)
	}
	if len(p.Labels) != 0 || p.MatchingCounts.Labels != 0 {
		t.Fatal("unnamed scope has labels")
	}
}

func TestPrimeMinimumBudgetKeepsTheNameFirst(t *testing.T) {
	s := fixture(t)
	if _, err := s.NameScope(ScopeLabelWrite{Scope: atlas, Name: "Atlas launch", Description: strings.Repeat("d", 1024), Reason: "r"}); err != nil {
		t.Fatal(err)
	}
	remember(t, s, Write{Kind: "project-state", Summary: "Atlas: drafting", Body: "Short.", Basis: "observation", Reason: "r", Scope: &atlas})
	p, err := s.Prime(atlas, 1024)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	if len(b) > 1024 || len(p.Labels) != 1 || p.Labels[0].Name != "Atlas launch" || p.Labels[0].Description != "" || !p.Truncated || p.MatchingCounts.State != 1 {
		t.Fatalf("%d bytes: %s", len(b), b)
	}
	// A short description and the state both fit.
	s2 := fixture(t)
	nameAtlas(t, s2)
	remember(t, s2, Write{Kind: "project-state", Summary: "Atlas: drafting", Body: "Short.", Basis: "observation", Reason: "r", Scope: &atlas})
	if p, err = s2.Prime(atlas, 1024); err != nil || p.Labels[0].Description == "" || len(p.State) != 1 || p.Truncated {
		t.Fatalf("%+v %v", p, err)
	}
	// A name JSON escaping grows past the budget is omitted and counted.
	s3 := fixture(t)
	if _, err := s3.NameScope(ScopeLabelWrite{Scope: atlas, Name: strings.Repeat("<", 256), Description: "d", Reason: "r"}); err != nil {
		t.Fatal(err)
	}
	if p, err = s3.Prime(atlas, 1024); err != nil || len(p.Labels) != 0 || p.MatchingCounts.Labels != 1 || !p.Truncated {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestPrimeStaysUsefulOnALargeTopic(t *testing.T) {
	s := fixture(t)
	nameAtlas(t, s)
	for i := range 40 {
		remember(t, s, Write{Kind: "fact", Summary: "Large topic fact " + string(rune('a'+i%26)) + string(rune('a'+i/26)), Body: strings.Repeat("b", 4096), Basis: "observation", Reason: "r", Scope: &atlas})
	}
	var journal []string
	for i := range 5 {
		time.Sleep(time.Millisecond)
		e, err := s.AppendJournal("outcome", "Journal "+string(rune('a'+i)), &atlas)
		if err != nil {
			t.Fatal(err)
		}
		journal = append(journal, e.ID)
	}
	p, err := s.Prime(atlas, 8192)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	if len(b) > 8192 {
		t.Fatalf("%d bytes", len(b))
	}
	if len(p.Current) != 40 && !p.Truncated {
		t.Fatalf("%d of 40 summaries without disclosure", len(p.Current))
	}
	if len(p.Journal) < 3 || p.Journal[0].ID != journal[4] || p.Journal[2].ID != journal[2] {
		t.Fatalf("journal %+v", p.Journal)
	}
	for _, item := range p.Current {
		if item.Summary == "" {
			t.Fatal("current item without summary")
		}
	}
	if wide, err := s.Prime(atlas, 32768); err != nil || len(wide.Current) != 40 || len(wide.Journal) != 5 || wide.Truncated {
		t.Fatalf("largest budget: %d current, %d journal, %v", len(wide.Current), len(wide.Journal), err)
	}
	t.Logf("default budget: %d of 40 current, %d of 5 journal, %d bytes", len(p.Current), len(p.Journal), len(b))
}
