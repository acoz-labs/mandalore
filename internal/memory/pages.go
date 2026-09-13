package memory

import (
	"errors"
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

func (s *Service) ScopePage(offset, limit int) (Page[ScopeInfo], error) {
	items, err := s.Scopes()
	if err != nil {
		return Page[ScopeInfo]{}, err
	}
	return page(items, offset, limit)
}

func (s *Service) JournalPage(query string, limit int) (Page[JournalEntry], error) {
	// Journal is newest-first lexical search, not a stable transcript cursor.
	// An extra result makes truncation explicit without claiming completeness.
	if limit < 1 || limit > 50 {
		return Page[JournalEntry]{}, errors.New("journal limit must be 1–50")
	}
	items, err := s.Journal(query, limit+1)
	if err != nil {
		return Page[JournalEntry]{}, err
	}
	p, err := page(items, 0, limit)
	p.NextOffset = nil // Refine the query for older journals; no unstable cursor.
	return p, err
}
