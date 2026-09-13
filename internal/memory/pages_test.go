package memory

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPagesBoundedAndContinueWithoutClipping(t *testing.T) {
	items := []string{strings.Repeat("a", 20000), strings.Repeat("b", 20000), "last"}
	p, err := page(items, 0, 50)
	if err != nil || len(p.Items) != 1 || p.Items[0] != items[0] || p.NextOffset == nil || *p.NextOffset != 1 || !p.Truncated {
		t.Fatal(p, err)
	}
	data, err := json.Marshal(p)
	if err != nil || len(data) > 32768 {
		t.Fatal("oversized page", err)
	}
	p, err = page(items, *p.NextOffset, 50)
	if err != nil || len(p.Items) != 2 || p.Truncated || p.NextOffset != nil {
		t.Fatal(p, err)
	}
	for _, args := range [][2]int{{-1, 5}, {4, 5}, {0, 0}, {0, 51}} {
		if _, err := page(items, args[0], args[1]); err == nil {
			t.Fatal("accepted invalid pagination", args)
		}
	}
	if _, err := page([]string{strings.Repeat("x", 32768)}, 0, 1); err == nil {
		t.Fatal("oversized entry was silently clipped")
	}
	empty, err := page([]string{}, 0, 5)
	if err != nil || empty.Items == nil || empty.Truncated || empty.NextOffset != nil {
		t.Fatal(empty, err)
	}
}

func TestServicePagesAndJournalTruncation(t *testing.T) {
	s := fixture(t)
	w := Write{Kind: "fact", Summary: "Example", Body: "Current evidence", Basis: "observation", Reason: "Observed"}
	r, err := s.Remember(w)
	if err != nil {
		t.Fatal(err)
	}
	w.RecordID = r.RecordID
	w.Supersedes = []string{r.ID}
	w.Body = "Updated evidence"
	if _, err := s.Remember(w); err != nil {
		t.Fatal(err)
	}
	h, err := s.HistoryPage(r.RecordID, 0, 1)
	if err != nil || len(h.Items) != 1 || !h.Truncated || h.NextOffset == nil {
		t.Fatal(h, err)
	}
	sc, err := s.ScopePage(0, 1)
	if err != nil || len(sc.Items) != 1 || sc.Truncated {
		t.Fatal(sc, err)
	}
	for n := 0; n < 3; n++ {
		if _, err := s.AppendJournal("test", "Recorded an observation"); err != nil {
			t.Fatal(err)
		}
	}
	j, err := s.JournalPage("", 2)
	if err != nil || len(j.Items) != 2 || !j.Truncated || j.NextOffset != nil {
		t.Fatal(j, err)
	}
	if _, err := s.JournalPage("", 0); err == nil {
		t.Fatal("invalid journal limit")
	}
	if _, err := s.HistoryPage("../bad", 0, 1); err == nil {
		t.Fatal("invalid history ID")
	}
	if _, err := s.Journal(strings.Repeat("x", 2049), 5); err == nil {
		t.Fatal("unbounded journal query")
	}
}
