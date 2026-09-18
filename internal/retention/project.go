// Package retention provides metadata-only review, never expiry or deletion.
package retention

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

type Selection struct {
	RecordIDs  []string       `json:"record_ids,omitempty"`
	Scopes     []memory.Scope `json:"scopes,omitempty"`
	JournalIDs []string       `json:"journal_ids,omitempty"`
}
type AgePolicy struct {
	Timestamp string `json:"timestamp"`
	Before    string `json:"before"`
}
type Policy struct {
	ID         string     `json:"id"`
	Visibility string     `json:"visibility"`
	Age        *AgePolicy `json:"age,omitempty"`
}
type Record struct {
	RecordID        string                 `json:"record_id"`
	Scope           memory.Scope           `json:"scope"`
	Visibility      memory.VisibilityState `json:"visibility"`
	Matched         bool                   `json:"matched"`
	Reason          string                 `json:"reason"`
	RevisionIDs     []string               `json:"revision_ids"`
	SourceIDs       []string               `json:"source_ids"`
	NewestTimestamp *string                `json:"newest_selected_timestamp,omitempty"`
}
type Journal struct {
	ID         string `json:"id"`
	Matched    bool   `json:"matched"`
	Reason     string `json:"reason"`
	RecordedAt string `json:"recorded_at"`
}
type SourceRelationship struct {
	SourceID       string   `json:"source_id"`
	RecordIDs      []string `json:"referenced_by_record_ids"`
	FoundlingID    string   `json:"foundling_id,omitempty"`
	RegistrationID string   `json:"registration_id,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{2,127}$`)
var errSelection = errors.New("invalid or oversized explicit retention selection or policy")

func validatePolicy(p Policy) error {
	if !identifier.MatchString(p.ID) || (p.Visibility != "any" && p.Visibility != "withheld") {
		return errSelection
	}
	if p.Age != nil {
		if p.Age.Timestamp != "recorded_at" && p.Age.Timestamp != "effective_from" && p.Age.Timestamp != "last_verified_at" {
			return errSelection
		}
		if _, err := time.Parse(time.RFC3339Nano, p.Age.Before); err != nil {
			return errSelection
		}
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Age compares every structural current head, including future-effective heads.
// This is a review predicate, not a clock-based resolution of memory conflicts.
func matchAge(rs []memory.Revision, heads []string, age *AgePolicy) (bool, string, *string) {
	if age == nil {
		return true, "selected", nil
	}
	wanted := map[string]bool{}
	for _, id := range heads {
		wanted[id] = true
	}
	var latest time.Time
	found := false
	for _, r := range rs {
		if !wanted[r.ID] {
			continue
		}
		var value string
		switch age.Timestamp {
		case "recorded_at":
			value = r.RecordedAt
		case "effective_from":
			value = r.EffectiveFrom
		case "last_verified_at":
			if r.LastVerifiedAt == nil {
				return false, "timestamp-missing", nil
			}
			value = *r.LastVerifiedAt
		}
		at, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return false, "timestamp-missing", nil
		}
		if !found || at.After(latest) {
			latest = at
		}
		found = true
	}
	if !found {
		return false, "timestamp-missing", nil
	}
	value := latest.UTC().Format(time.RFC3339Nano)
	before, _ := time.Parse(time.RFC3339Nano, age.Before)
	if !latest.Before(before) {
		return false, "not-before-cutoff", &value
	}
	return true, "before-cutoff", &value
}

func project(ctx context.Context, s memory.ReportSnapshot, in Selection, policy Policy) ([]Record, []Journal, []SourceRelationship, error) {
	bad := func() ([]Record, []Journal, []SourceRelationship, error) { return nil, nil, nil, errSelection }
	if err := validatePolicy(policy); err != nil {
		return bad()
	}
	if len(in.RecordIDs)+len(in.Scopes)+len(in.JournalIDs) == 0 || len(in.RecordIDs) > 128 || len(in.Scopes) > 16 || len(in.JournalIDs) > 128 {
		return bad()
	}
	if err := memory.ValidateReportSnapshot(s.Snapshot, s.Registrations); err != nil {
		return bad()
	}
	byRecord := map[string][]memory.Revision{}
	scopes := map[memory.Scope]bool{}
	sourceRecords := map[string]map[string]bool{}
	for _, r := range s.Revisions {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		byRecord[r.RecordID] = append(byRecord[r.RecordID], r)
		scopes[r.Scope] = true
		for _, id := range r.Evidence.SourceRefs {
			if sourceRecords[id] == nil {
				sourceRecords[id] = map[string]bool{}
			}
			sourceRecords[id][r.RecordID] = true
		}
	}
	selected := map[string]bool{}
	for _, id := range in.RecordIDs {
		if len(byRecord[id]) == 0 || selected[id] {
			return bad()
		}
		selected[id] = true
	}
	selectedScopes := map[memory.Scope]bool{}
	for _, scope := range in.Scopes {
		if !scopes[scope] || selectedScopes[scope] {
			return bad()
		}
		selectedScopes[scope] = true
		for id, rs := range byRecord {
			if rs[0].Scope == scope {
				selected[id] = true
			}
		}
	}
	if len(selected) > 128 {
		return bad()
	}
	devices := map[string]bool{}
	for _, d := range s.Devices {
		devices[d.ID] = true
	}
	states, err := memory.ResolveVisibility(s.Revisions, s.Visibility, func(id string) error {
		if !devices[id] {
			return errSelection
		}
		return nil
	})
	if err != nil {
		return bad()
	}
	records := []Record{}
	selectedSources := map[string]bool{}
	for _, id := range sortedKeys(selected) {
		rs := byRecord[id]
		state := states[id]
		r := Record{RecordID: id, Scope: rs[0].Scope, Visibility: state, RevisionIDs: []string{}, SourceIDs: []string{}}
		refs := map[string]bool{}
		for _, revision := range rs {
			r.RevisionIDs = append(r.RevisionIDs, revision.ID)
			for _, source := range revision.Evidence.SourceRefs {
				refs[source] = true
				selectedSources[source] = true
			}
		}
		sort.Strings(r.RevisionIDs)
		r.SourceIDs = sortedKeys(refs)
		r.Matched, r.Reason, r.NewestTimestamp = matchAge(rs, state.ContentHeads, policy.Age)
		if policy.Visibility == "withheld" && !state.Withheld() {
			r.Matched = false
			r.Reason = "visibility-not-matched"
		}
		records = append(records, r)
	}
	byJournal := map[string]memory.JournalEntry{}
	for _, j := range s.Journal {
		byJournal[j.ID] = j
	}
	wantedJournals := map[string]bool{}
	for _, id := range in.JournalIDs {
		if _, ok := byJournal[id]; !ok || wantedJournals[id] {
			return bad()
		}
		wantedJournals[id] = true
	}
	journals := []Journal{}
	for _, id := range sortedKeys(wantedJournals) {
		j := byJournal[id]
		item := Journal{ID: id, Matched: true, Reason: "explicitly-selected-journal", RecordedAt: j.RecordedAt}
		if policy.Age != nil {
			if policy.Age.Timestamp != "recorded_at" {
				item.Matched = false
				item.Reason = "timestamp-not-applicable"
			} else {
				at, _ := time.Parse(time.RFC3339Nano, j.RecordedAt)
				before, _ := time.Parse(time.RFC3339Nano, policy.Age.Before)
				item.Matched = at.Before(before)
				item.Reason = "before-cutoff"
				if !item.Matched {
					item.Reason = "not-before-cutoff"
				}
			}
		}
		journals = append(journals, item)
	}
	sources := map[string]memory.Source{}
	for _, s := range s.Sources {
		sources[s.ID] = s
	}
	relationships := []SourceRelationship{}
	for _, id := range sortedKeys(selectedSources) {
		source := sources[id]
		item := SourceRelationship{SourceID: id, RecordIDs: sortedKeys(sourceRecords[id])}
		if source.ExternalOrigin != nil {
			item.FoundlingID = source.ExternalOrigin.FoundlingID
			item.RegistrationID = source.ExternalOrigin.RegistrationRevisionID
		}
		relationships = append(relationships, item)
	}
	return records, journals, relationships, ctx.Err()
}
