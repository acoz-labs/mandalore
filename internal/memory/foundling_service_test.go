package memory

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func foundlingWrite() FoundlingWrite {
	return FoundlingWrite{Name: "Historical notes", Description: "Reference evidence only", Source: FoundlingSource{Kind: "local", Locator: "source-old-notes"}, Pin: SourcePin{Algorithm: "sha256", Value: strings.Repeat("a", 64)}, State: "active", Reason: "User linked historical context"}
}

func TestFoundlingServiceRegistersAndListsWithoutLearningReferenceMetadata(t *testing.T) {
	s := fixture(t)
	empty, err := s.FoundlingsPage(0, 5)
	if err != nil || len(empty.Items) != 0 {
		t.Fatal(empty, err)
	}
	r, err := s.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	if r.FoundlingID == "" || r.ID == "" || r.Authorship != s.author || r.State != "active" {
		t.Fatal(r)
	}
	p, err := s.FoundlingsPage(0, 5)
	if err != nil || len(p.Items) != 1 {
		t.Fatal(p, err)
	}
	v := p.Items[0]
	if v.FoundlingID != r.FoundlingID || v.Name != r.Name || v.Description != r.Description || v.State != "active" || v.HeadCount != 1 || len(v.HeadIDs) != 1 || v.HeadIDs[0] != r.ID || v.Source == nil || *v.Source != r.Source {
		t.Fatal(v)
	}
	current, err := s.Recall("Historical notes", nil, 5, 4096)
	if err != nil || current.MatchingCount != 0 {
		t.Fatal("registration became current memory", current, err)
	}
	journal, err := s.Journal("", 10)
	if err != nil || len(journal) != 0 {
		t.Fatal("registration invented semantic journal", journal, err)
	}
}

func TestFoundlingRoutingDoesNotSelectConcurrentMetadataOrDisappearOnDisconnect(t *testing.T) {
	s := fixture(t)
	root, err := s.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	change := foundlingWrite()
	change.FoundlingID = root.FoundlingID
	change.Supersedes = []string{root.ID}
	change.Reason = "Explicit disconnect"
	change.State = "disconnected"
	off, err := s.WriteFoundling(change)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.FoundlingsPage(0, 5)
	if err != nil || len(p.Items) != 1 || p.Items[0].State != "disconnected" || p.Items[0].HeadIDs[0] != off.ID {
		t.Fatal(p, err)
	}
	change.State = "active"
	change.Name = "Competing update"
	change.Reason = "Concurrent update based on the original registration"
	on, err := s.WriteFoundling(change)
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.FoundlingsPage(0, 5)
	if err != nil || len(p.Items) != 1 {
		t.Fatal(p, err)
	}
	v := p.Items[0]
	if v.State != "conflicted" || v.Name != "" || v.Description != "" || v.Source != nil || v.Pin != nil || v.HeadCount != 2 || len(v.HeadIDs) != 2 {
		t.Fatal("picked a metadata winner", v)
	}
	history, err := s.FoundlingHistoryPage(root.FoundlingID, 0, 5)
	if err != nil || len(history.Items) != 3 {
		t.Fatal(history, err)
	}
	change.Supersedes = []string{on.ID, off.ID}
	change.Reason = "Explicitly reconcile registration heads"
	resolved, err := s.WriteFoundling(change)
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.FoundlingsPage(0, 5)
	if err != nil || p.Items[0].State != "active" || p.Items[0].HeadIDs[0] != resolved.ID {
		t.Fatal(p, err)
	}
}

func TestFoundlingServiceRejectsAmbiguousIdentityAndInvalidUpdates(t *testing.T) {
	for name, change := range map[string]func(*FoundlingWrite){
		"ID without parents": func(w *FoundlingWrite) { w.FoundlingID = "foundling-existing" },
		"parents without ID": func(w *FoundlingWrite) { w.Supersedes = []string{"registration-existing"} },
		"new disconnected":   func(w *FoundlingWrite) { w.State = "disconnected" },
		"missing name":       func(w *FoundlingWrite) { w.Name = "" },
		"absolute source":    func(w *FoundlingWrite) { w.Source.Locator = "/example/private/path" },
		"invalid pin":        func(w *FoundlingWrite) { w.Pin.Value = "main" },
	} {
		t.Run(name, func(t *testing.T) {
			s := fixture(t)
			w := foundlingWrite()
			change(&w)
			if _, err := s.WriteFoundling(w); err == nil {
				t.Fatal("invalid registration accepted")
			}
			p, err := s.FoundlingsPage(0, 5)
			if err != nil || len(p.Items) != 0 {
				t.Fatal("invalid write changed registrations", p, err)
			}
		})
	}
}

func TestFoundlingRoutingPagesAndConflictHeadsAreBounded(t *testing.T) {
	s := fixture(t)
	for n := 0; n < 10; n++ {
		w := foundlingWrite()
		w.Name = fmt.Sprintf("Reference %d", n)
		w.Description = strings.Repeat("x", 4096)
		if _, err := s.WriteFoundling(w); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.FoundlingsPage(0, 50)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(p)
	if err != nil || len(b) > 32768 || !p.Truncated || p.NextOffset == nil {
		t.Fatal("unbounded routing page", len(b), p.NextOffset, err)
	}
	next, err := s.FoundlingsPage(*p.NextOffset, 50)
	if err != nil || len(next.Items)+len(p.Items) != 10 {
		t.Fatal(next, err)
	}
	root, err := s.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 40; n++ {
		w := foundlingWrite()
		w.FoundlingID = root.FoundlingID
		w.Supersedes = []string{root.ID}
		w.Reason = fmt.Sprintf("Concurrent update %d", n)
		if _, err := s.WriteFoundling(w); err != nil {
			t.Fatal(err)
		}
	}
	seenConflict := false
	for offset := 0; ; {
		p, err := s.FoundlingsPage(offset, 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range p.Items {
			if v.FoundlingID == root.FoundlingID {
				seenConflict = true
			}
			if v.FoundlingID == root.FoundlingID && (v.HeadCount != 40 || len(v.HeadIDs) != 32 || !v.HeadsTruncated || v.State != "conflicted") {
				t.Fatal(v)
			}
		}
		if p.NextOffset == nil {
			break
		}
		offset = *p.NextOffset
	}
	if !seenConflict {
		t.Fatal("conflicting registration missing from routing")
	}
	if _, err := s.FoundlingHistoryPage("../other", 0, 5); err == nil {
		t.Fatal("invalid history selector accepted")
	}
}

func TestFoundlingPagesStayBoundAndReadFreshDataFromOtherWriter(t *testing.T) {
	s := fixture(t)
	other, err := OpenService(s.Root(), s.author)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := s.FoundlingsPage(0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal(p, err)
	}
	r, err := other.WriteFoundling(foundlingWrite())
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.FoundlingsPage(0, 5)
	if err != nil || len(p.Items) != 1 || p.Items[0].FoundlingID != r.FoundlingID {
		t.Fatal("stale routing view", p, err)
	}
	isolate := fixture(t)
	if p, err := isolate.FoundlingsPage(0, 5); err != nil || len(p.Items) != 0 {
		t.Fatal("registration crossed signet boundary", p, err)
	}
	for _, pair := range [][2]int{{-1, 5}, {2, 5}, {0, 0}, {0, 51}} {
		if _, err := s.FoundlingsPage(pair[0], pair[1]); err == nil {
			t.Fatal("invalid pagination accepted", pair)
		}
	}
}
