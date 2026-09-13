package memory

import (
	"testing"
	"time"
)

func TestRecallMatchesInflectionsWithoutChangingScopeOrHistory(t *testing.T) {
	s := fixtureStore(t)
	a := revision("revision-old-name")
	a.Summary, a.Body = "Project named Copper Finch", "Original fictional project."
	b := revision("revision-new-name", a.ID)
	b.Summary, b.Body = "Project named Silver Heron", "Renamed fictional project."
	c := revision("revision-other-name")
	c.RecordID, c.Scope.ID = "record-other", "account-other"
	c.Summary = "Project name"
	for _, r := range []Revision{a, b, c} {
		if err := s.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.Recall(Query{Text: "name", Scope: a.Scope}, time.Now())
	if err != nil || len(p.Current) != 1 || p.Current[0].ID != b.ID {
		t.Fatalf("recall: %+v %v", p, err)
	}
	p, err = s.Recall(Query{Text: "Copper", Scope: a.Scope}, time.Now())
	if err != nil || len(p.Current) != 0 {
		t.Fatalf("superseded match resurfaced: %+v %v", p, err)
	}
}

func TestLexicalMatchingKeepsExactHitsAndIdentifiersDistinct(t *testing.T) {
	for _, pair := range [][2]string{{"name", "named"}, {"name", "names"}, {"name", "naming"}, {"review", "reviewed"}, {"review", "reviews"}} {
		r := Revision{Summary: pair[1]}
		if relevance(r, pair[0]) == 0 {
			t.Errorf("missing %q -> %q", pair[0], pair[1])
		}
		if relevance(r, pair[1]) <= relevance(r, pair[0]) {
			t.Errorf("exact match not preferred: %v", pair)
		}
	}
	for _, pair := range [][2]string{{"name", "namespace"}, {"name", "filename"}, {"account-prod", "account-prods"}, {"1234", "1234s"}, {"run", "running"}} {
		if relevance(Revision{Summary: pair[1]}, pair[0]) != 0 {
			t.Errorf("overbroad match: %v", pair)
		}
	}
	if relevance(Revision{Scope: Scope{ID: "project-named"}}, "project-name") != 0 {
		t.Fatal("fuzzy identifier match")
	}
}
