package memory

import (
	"errors"
	"sort"
	"strings"
)

type Page[T any] struct {
	Items      []T  `json:"items"`
	NextOffset *int `json:"next_offset,omitempty"`
	Truncated  bool `json:"truncated"`
}

func page[T any](items []T, offset, limit int) (Page[T], error) {
	p := Page[T]{Items: []T{}}
	if offset < 0 || offset > len(items) || limit < 1 || limit > 50 {
		return p, errors.New("page requires an existing offset and limit 1–50")
	}
	for n := offset; n < len(items) && len(p.Items) < limit; n++ {
		next := n + 1
		p.NextOffset = &next
		p.Items = append(p.Items, items[n])
		if !fits(p, 32768) {
			p.Items = p.Items[:len(p.Items)-1]
			if len(p.Items) == 0 {
				return p, errors.New("entry exceeds the 32 KiB page budget; inspect it through local files")
			}
			break
		}
	}
	next := offset + len(p.Items)
	p.NextOffset = nil
	if next < len(items) {
		p.Truncated, p.NextOffset = true, &next
	}
	return p, nil
}

func (s *Service) HistoryPage(recordID string, offset, limit int) (Page[Revision], error) {
	items, err := s.History(recordID)
	if err != nil {
		return Page[Revision]{}, err
	}
	return page(items, offset, limit)
}

// ScopePage lists scopes with their current labels. Scopes with only a name
// or scoped journal entries are listed with record_count 0. A query keeps the
// scopes whose ID, name or description match, best match first.
func (s *Service) ScopePage(query string, offset, limit int) (Page[ScopeInfo], error) {
	if len(query) > 2048 {
		return Page[ScopeInfo]{}, invalidInput("scopes query is %d bytes; the limit is 2048", len(query))
	}
	items, err := s.Scopes()
	if err != nil {
		return Page[ScopeInfo]{}, err
	}
	labels, err := s.store.ScopeLabels()
	if err != nil {
		return Page[ScopeInfo]{}, err
	}
	journal, err := s.store.journalEntries("", nil)
	if err != nil {
		return Page[ScopeInfo]{}, err
	}
	listed := map[Scope]bool{}
	for _, item := range items {
		listed[item.Scope] = true
	}
	extra := []Scope{}
	for _, l := range labels {
		extra = append(extra, l.Scope)
	}
	for _, e := range journal {
		if e.Scope != nil {
			extra = append(extra, *e.Scope)
		}
	}
	for _, scope := range extra {
		if !listed[scope] {
			listed[scope] = true
			items = append(items, ScopeInfo{Scope: scope})
		}
	}
	heads := labelHeads(labels)
	scores := map[Scope]int{}
	matching := []ScopeInfo{}
	for _, item := range items {
		if h := heads[item.Scope]; len(h) > 0 {
			item.Labels = headsOf(h)
		}
		if strings.TrimSpace(query) != "" {
			text := Revision{Scope: item.Scope}
			for _, l := range item.Labels {
				text.Summary += l.Name + "\n"
				text.Body += l.Description + "\n"
			}
			if scores[item.Scope] = relevance(text, query); scores[item.Scope] == 0 {
				continue
			}
		}
		matching = append(matching, item)
	}
	sort.SliceStable(matching, func(i, j int) bool {
		a, b := matching[i], matching[j]
		if scores[a.Scope] != scores[b.Scope] {
			return scores[a.Scope] > scores[b.Scope]
		}
		if a.Scope.Kind != b.Scope.Kind {
			return a.Scope.Kind < b.Scope.Kind
		}
		return a.Scope.ID < b.Scope.ID
	})
	return page(matching, offset, limit)
}

func (s *Service) JournalPage(query string, scope *Scope, limit int) (Page[JournalEntry], error) {
	// Journal is newest-first lexical search, not a stable transcript cursor.
	// An extra result makes truncation explicit without claiming completeness.
	if limit < 1 || limit > 50 {
		return Page[JournalEntry]{}, errors.New("journal limit must be 1–50")
	}
	items, err := s.Journal(query, scope, limit+1)
	if err != nil {
		return Page[JournalEntry]{}, err
	}
	p, err := page(items, 0, limit)
	p.NextOffset = nil // Refine the query for older journals; no unstable cursor.
	return p, err
}
