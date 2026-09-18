package memory

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

// WithheldRecord is explicit historical routing metadata, never current guidance.
// Contents, summaries, reasons and authorship require intentional history reads.
type WithheldRecord struct {
	RecordID string `json:"record_id"`
	Scope    Scope  `json:"scope"`
	Kind     string `json:"kind"`
	State    string `json:"visibility_state"`
}

func (s *Service) WithheldRecords(query string, selected *Scope, offset, limit int) (Page[WithheldRecord], error) {
	scope := s.scope(selected)
	if err := s.validateScope(scope); err != nil {
		return Page[WithheldRecord]{}, err
	}
	if len(query) > 2048 || !utf8.ValidString(query) || strings.ContainsRune(query, '\x00') || offset < 0 || limit < 1 || limit > 50 {
		return Page[WithheldRecord]{}, errors.New("withheld discovery requires query up to 2048 bytes, nonnegative offset and limit 1–50")
	}
	records, err := s.store.revisions()
	if err != nil {
		return Page[WithheldRecord]{}, err
	}
	states, err := s.store.validateGraphState(records, nil)
	if err != nil {
		return Page[WithheldRecord]{}, err
	}
	heads := map[string]bool{}
	for _, state := range states {
		for _, id := range state.ContentHeads {
			heads[id] = true
		}
	}
	terms := strings.Fields(strings.ToLower(query))
	matched := map[string]WithheldRecord{}
	for _, r := range records {
		state := states[r.RecordID]
		if r.Scope != scope || !state.Withheld() || !heads[r.ID] {
			continue
		}
		// Only current structural-head summaries and the stable ID are searchable;
		// never recover superseded names or match a hidden body through this route.
		text := strings.ToLower(r.RecordID + " " + r.Summary)
		matches := true
		for _, term := range terms {
			if !strings.Contains(text, term) {
				matches = false
				break
			}
		}
		if matches {
			matched[r.RecordID] = WithheldRecord{RecordID: r.RecordID, Scope: r.Scope, Kind: r.Kind, State: state.State}
		}
	}
	items := make([]WithheldRecord, 0, len(matched))
	for _, item := range matched {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RecordID < items[j].RecordID })
	return page(items, offset, limit)
}
