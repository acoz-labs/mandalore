package memory

import (
	"errors"
	"sort"
	"time"
)

// FoundlingWrite authors only portable registration metadata. It neither
// connects a local path nor verifies, retrieves or endorses source content.
type FoundlingWrite struct {
	FoundlingID string          `json:"foundling_id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Source      FoundlingSource `json:"source"`
	Pin         SourcePin       `json:"pin"`
	State       string          `json:"state"`
	Supersedes  []string        `json:"supersedes,omitempty"`
	Reason      string          `json:"reason"`
}

func (s *Service) WriteFoundling(input FoundlingWrite) (FoundlingRegistration, error) {
	if (input.FoundlingID == "") != (len(input.Supersedes) == 0) {
		return FoundlingRegistration{}, errors.New("registration update requires both foundling_id and supersedes; a new reference requires neither")
	}
	if input.FoundlingID == "" {
		if input.State != "active" {
			return FoundlingRegistration{}, errors.New("a new reference must be active; disconnect an existing registration explicitly")
		}
		input.FoundlingID = NewID("foundling")
	}
	if input.Supersedes == nil {
		input.Supersedes = []string{}
	}
	r := FoundlingRegistration{Version: 1, ID: NewID("registration"), FoundlingID: input.FoundlingID, Name: input.Name, Description: input.Description, Source: input.Source, Pin: input.Pin, State: input.State, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author, Supersedes: input.Supersedes, ChangeReason: input.Reason}
	if err := s.store.PutFoundlingRegistration(r); err != nil {
		return FoundlingRegistration{}, err
	}
	return r, nil
}

// FoundlingSummary is routing metadata, never an availability check or recalled
// guidance. A conflicted registration exposes no arbitrarily selected name/pin.
// History pagination retains every revision when the bounded head list truncates.
type FoundlingSummary struct {
	FoundlingID    string           `json:"foundling_id"`
	State          string           `json:"state"`
	HeadCount      int              `json:"head_count"`
	HeadIDs        []string         `json:"head_ids"`
	HeadsTruncated bool             `json:"heads_truncated"`
	Name           string           `json:"name,omitempty"`
	Description    string           `json:"description,omitempty"`
	Source         *FoundlingSource `json:"source,omitempty"`
	Pin            *SourcePin       `json:"pin,omitempty"`
}

func (s *Service) FoundlingsPage(offset, limit int) (Page[FoundlingSummary], error) {
	summaries, err := s.foundlingSummaries()
	if err != nil {
		return Page[FoundlingSummary]{}, err
	}
	return page(summaries, offset, limit)
}

// Foundling looks up fresh routing metadata, never selecting a conflicted head.
func (s *Service) Foundling(foundlingID string) (FoundlingSummary, error) {
	if !identifier.MatchString(foundlingID) {
		return FoundlingSummary{}, errors.New("invalid foundling ID")
	}
	summaries, err := s.foundlingSummaries()
	if err != nil {
		return FoundlingSummary{}, err
	}
	for _, summary := range summaries {
		if summary.FoundlingID == foundlingID {
			return summary, nil
		}
	}
	return FoundlingSummary{}, errors.New("foundling not registered in selected signet")
}

func (s *Service) foundlingSummaries() ([]FoundlingSummary, error) {
	items, err := s.store.FoundlingRegistrations()
	if err != nil {
		return nil, err
	}
	parents := map[string]bool{}
	for _, r := range items {
		for _, id := range r.Supersedes {
			parents[id] = true
		}
	}
	heads := map[string][]FoundlingRegistration{}
	for _, r := range items {
		if !parents[r.ID] {
			heads[r.FoundlingID] = append(heads[r.FoundlingID], r)
		}
	}
	summaries := make([]FoundlingSummary, 0, len(heads))
	for id, current := range heads {
		v := FoundlingSummary{FoundlingID: id, State: "conflicted", HeadCount: len(current), HeadIDs: []string{}}
		// FoundlingRegistrations already sorts by immutable revision ID.
		for _, r := range current {
			if len(v.HeadIDs) == 32 {
				v.HeadsTruncated = true
				break
			}
			v.HeadIDs = append(v.HeadIDs, r.ID)
		}
		if len(current) == 1 {
			r := current[0]
			v.State, v.Name, v.Description = r.State, r.Name, r.Description
			v.Source, v.Pin = &r.Source, &r.Pin
		}
		summaries = append(summaries, v)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].FoundlingID < summaries[j].FoundlingID })
	return summaries, nil
}

func (s *Service) FoundlingHistoryPage(foundlingID string, offset, limit int) (Page[FoundlingRegistration], error) {
	if !identifier.MatchString(foundlingID) {
		return Page[FoundlingRegistration]{}, errors.New("invalid foundling ID; discover registrations with the routing list")
	}
	items, err := s.store.FoundlingRegistrations()
	if err != nil {
		return Page[FoundlingRegistration]{}, err
	}
	matching := []FoundlingRegistration{}
	for _, r := range items {
		if r.FoundlingID == foundlingID {
			matching = append(matching, r)
		}
	}
	return page(matching, offset, limit)
}
