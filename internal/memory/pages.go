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

// RecordHistory is a page of a record's stored revisions, with where the
// record is now and, on the first page, its moves (2.3.0). Each revision's
// scope is the scope the record was saved in; Scope is where it is now.
type RecordHistory struct {
	Page[Revision]
	Scope          *Scope      `json:"scope,omitempty"`
	MoveHeads      []string    `json:"move_heads"`
	MoveCount      int         `json:"move_count"`
	Moves          []MoveEntry `json:"moves"`
	MovesTruncated bool        `json:"moves_truncated,omitempty"`
}

// MoveEntry is a stored move with the scope it moved the record from: the
// root's scope, or its parent moves' scopes.
type MoveEntry struct {
	RecordMove
	From []Scope `json:"from"`
}

// movesBudget bounds a history page's moves. The revisions keep 2.2.0's
// 32 KiB page budget on their own, so a page holds the same revisions it
// did before moves existed; the whole response stays under the 64 KiB
// output limit.
const movesBudget = 16384

func (s *Service) HistoryPage(recordID string, offset, limit int) (RecordHistory, error) {
	items, err := s.History(recordID)
	if err != nil {
		return RecordHistory{}, err
	}
	p, err := page(items, offset, limit)
	if err != nil {
		return RecordHistory{}, err
	}
	out := RecordHistory{Page: p, Moves: []MoveEntry{}}
	placement, moves, err := s.store.recordMoves(recordID)
	if err != nil {
		return RecordHistory{}, err
	}
	if !placement.Conflicted() && placement.Scope != (Scope{}) {
		out.Scope = &placement.Scope
	}
	out.MoveHeads, out.MoveCount = placement.MoveHeads, placement.Moves
	if offset != 0 {
		return out, nil
	}
	byID := map[string]RecordMove{}
	for _, m := range moves {
		byID[m.ID] = m
	}
	sort.Slice(moves, func(i, j int) bool {
		return newestFirst(moves[i].RecordedAt, moves[j].RecordedAt, moves[i].ID, moves[j].ID)
	})
	for _, m := range moves {
		entry := MoveEntry{RecordMove: m, From: []Scope{}}
		if len(m.Parents) == 0 && len(items) > 0 {
			entry.From = append(entry.From, items[0].Scope)
		}
		seen := map[Scope]bool{}
		for _, id := range m.Parents {
			if scope := byID[id].Scope; !seen[scope] {
				seen[scope] = true
				entry.From = append(entry.From, scope)
			}
		}
		sortScopes(entry.From)
		out.Moves = append(out.Moves, entry)
		if !fits(out.Moves, movesBudget) {
			out.Moves = out.Moves[:len(out.Moves)-1]
			out.MovesTruncated = true
			break
		}
	}
	return out, nil
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
	tree, err := s.store.topicTree()
	if err != nil {
		return Page[ScopeInfo]{}, err
	}
	labels := tree.labels
	listed := map[Scope]bool{}
	for _, item := range items {
		listed[item.Scope] = true
	}
	extra := []Scope{}
	for _, l := range labels {
		extra = append(extra, l.Scope)
	}
	for _, e := range tree.journal {
		extra = append(extra, *e.Scope)
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
