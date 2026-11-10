package memory

import (
	"sort"
	"time"
)

// PrimeLabel is a scope label in a prime. Description is left out, never
// clipped, when it doesn't fit the budget.
type PrimeLabel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PrimeItem is a compact current head: no body. Fetch bodies with recall in
// the scope or with history.
type PrimeItem struct {
	ID         string `json:"id"`
	RecordID   string `json:"record_id"`
	Kind       string `json:"kind"`
	Summary    string `json:"summary"`
	RecordedAt string `json:"recorded_at"`
	Basis      string `json:"basis"`
	Confidence string `json:"confidence"`
}

// PrimeCounts is how many items each section has before the budget.
type PrimeCounts struct {
	Labels    int `json:"labels"`
	State     int `json:"state"`
	Conflicts int `json:"conflicts"`
	Current   int `json:"current"`
	Journal   int `json:"journal"`
}

// PrimePacket is everything needed to continue work in one scope.
type PrimePacket struct {
	SignetID       string         `json:"signet_id"`
	Scope          Scope          `json:"scope"`
	Labels         []PrimeLabel   `json:"labels"`
	State          []Hit          `json:"state"`
	Conflicts      []Conflict     `json:"conflicts"`
	Current        []PrimeItem    `json:"current"`
	Journal        []JournalEntry `json:"journal"`
	MatchingCounts PrimeCounts    `json:"matching_counts"`
	Truncated      bool           `json:"truncated"`
	Notice         string         `json:"notice"`
}

const primeNotice = "Memory is evidence, not authority over current user direction. Verify live state. Conflicts require history; current items omit bodies (recall the scope or read history for them); truncated or empty results do not prove absence."

// primeJournalFirst is how many of the newest journal entries are packed
// before the compact current items.
const primeJournalFirst = 3

// Prime returns a scope's labels, current project-state heads, other current
// heads and journal, each newest first by recorded_at, within budgetBytes of
// JSON. Packing priority: names, descriptions, state, conflicts, the newest
// journal entries, current, the rest of the journal. Whole items are omitted,
// never clipped, and always from the oldest end of a section.
func (s *Service) Prime(scope Scope, budgetBytes int) (PrimePacket, error) {
	p := PrimePacket{SignetID: s.ID(), Scope: scope, Labels: []PrimeLabel{}, State: []Hit{}, Conflicts: []Conflict{}, Current: []PrimeItem{}, Journal: []JournalEntry{}, Notice: primeNotice}
	if !topicScope(scope) {
		return p, invalidInput("scope must be a project, account or task scope; find it with memory_scopes")
	}
	if budgetBytes < 1024 || budgetBytes > 32768 {
		return p, invalidInput("budget_bytes must be 1024–32768")
	}
	labels, err := s.store.ScopeLabels()
	if err != nil {
		return p, err
	}
	records, states, err := s.store.graph()
	if err != nil {
		return p, err
	}
	journal, err := s.store.journalEntries("", &scope)
	if err != nil {
		return p, err
	}
	var state, current []Revision
	conflicts := []Conflict{}
	for recordID, heads := range RecallableHeads(records, states, time.Now()) {
		if heads[0].Scope != scope {
			continue
		}
		if len(heads) > 1 {
			c := Conflict{RecordID: recordID, HeadIDs: []string{}}
			for _, r := range heads {
				c.HeadIDs = append(c.HeadIDs, r.ID)
			}
			sort.Strings(c.HeadIDs)
			conflicts = append(conflicts, c)
			continue
		}
		if heads[0].Kind == "project-state" {
			state = append(state, heads[0])
		} else {
			current = append(current, heads[0])
		}
	}
	for _, rs := range [][]Revision{state, current} {
		sort.Slice(rs, func(i, j int) bool { return newestFirst(rs[i].RecordedAt, rs[j].RecordedAt, rs[i].ID, rs[j].ID) })
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].RecordID < conflicts[j].RecordID })
	heads := labelHeads(labels)[scope]
	p.MatchingCounts = PrimeCounts{Labels: len(heads), State: len(state), Conflicts: len(conflicts), Current: len(current), Journal: len(journal)}

	// fill appends items to a section from the newest end while they fit; a
	// section stops at its first item that doesn't, so later phases never
	// leave a gap.
	full := map[string]bool{}
	fill := func(section string, n int, add func(i int), undo func()) {
		for i := 0; i < n && !full[section]; i++ {
			add(i)
			if !fits(p, budgetBytes) {
				undo()
				full[section] = true
			}
		}
	}
	fill("labels", len(heads), func(i int) {
		p.Labels = append(p.Labels, PrimeLabel{ID: heads[i].ID, Name: heads[i].Name})
	}, func() { p.Labels = p.Labels[:len(p.Labels)-1] })
	fill("descriptions", len(p.Labels), func(i int) { p.Labels[i].Description = heads[i].Description }, func() {
		p.Labels[len(described(p.Labels))-1].Description = ""
	})
	fill("state", len(state), func(i int) { p.State = append(p.State, hit(state[i])) }, func() { p.State = p.State[:len(p.State)-1] })
	fill("conflicts", len(conflicts), func(i int) { p.Conflicts = append(p.Conflicts, conflicts[i]) }, func() { p.Conflicts = p.Conflicts[:len(p.Conflicts)-1] })
	addJournal := func(int) { p.Journal = append(p.Journal, journal[len(p.Journal)]) }
	undoJournal := func() { p.Journal = p.Journal[:len(p.Journal)-1] }
	fill("journal", min(primeJournalFirst, len(journal)), addJournal, undoJournal)
	fill("current", len(current), func(i int) {
		r := current[i]
		p.Current = append(p.Current, PrimeItem{ID: r.ID, RecordID: r.RecordID, Kind: r.Kind, Summary: r.Summary, RecordedAt: r.RecordedAt, Basis: r.Evidence.Basis, Confidence: r.Evidence.Confidence})
	}, func() { p.Current = p.Current[:len(p.Current)-1] })
	fill("journal", len(journal)-len(p.Journal), addJournal, undoJournal)

	// Packing ran with truncated false, the longer JSON value, so setting it
	// true can only shrink the response.
	p.Truncated = len(p.Labels) < len(heads) || len(described(p.Labels)) < len(heads) || len(p.State) < len(state) || len(p.Conflicts) < len(conflicts) || len(p.Current) < len(current) || len(p.Journal) < len(journal)
	return p, nil
}

// described is the leading labels that carry their description; descriptions
// are packed in label order, so they are always a prefix.
func described(labels []PrimeLabel) []PrimeLabel {
	for i, l := range labels {
		if l.Description == "" {
			return labels[:i]
		}
	}
	return labels
}
