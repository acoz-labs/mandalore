// Package exportreport creates reviewed derived reports, never signet backups.
package exportreport

import (
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

var fieldCategories = []string{"identity", "content", "classification", "timestamps", "authorship", "citations", "change_history", "extensions"}

type Selection struct {
	RecordIDs        []string       `json:"record_ids,omitempty"`
	Scopes           []memory.Scope `json:"scopes,omitempty"`
	JournalIDs       []string       `json:"journal_ids,omitempty"`
	IncludeHistory   bool           `json:"include_history,omitempty"`
	IncludeWithdrawn bool           `json:"include_withdrawn,omitempty" jsonschema:"Explicitly include withheld records as history; requires include_history. Not current guidance."`
	IncludeDetails   bool           `json:"include_details,omitempty"`
	OmitFields       []string       `json:"omit_fields,omitempty"`
	OmitRecordIDs    []string       `json:"omit_record_ids,omitempty"`
	OmitJournalIDs   []string       `json:"omit_journal_ids,omitempty"`
}
type Conflict struct {
	RecordID string   `json:"record_id"`
	HeadIDs  []string `json:"head_ids"`
}
type Projection struct {
	RecordIDs        []string         `json:"record_ids"`
	RevisionIDs      []string         `json:"revision_ids"`
	JournalIDs       []string         `json:"journal_ids"`
	Conflicts        []Conflict       `json:"conflicts"`
	OmittedFields    []string         `json:"omitted_fields"`
	OmittedRecords   int              `json:"omitted_records"`
	OmittedJournals  int              `json:"omitted_journals"`
	OmittedRevisions int              `json:"omitted_revisions"`
	Withheld         []WithheldRecord `json:"withheld_records,omitempty"`
}
type WithheldRecord struct {
	RecordID string `json:"record_id"`
	State    string `json:"state"`
}
type report struct {
	Kind               string           `json:"kind"`
	Version            int              `json:"version"`
	Identity           map[string]any   `json:"identity,omitempty"`
	Items              []map[string]any `json:"items"`
	OmittedFields      []string         `json:"omitted_fields"`
	OmittedRecords     int              `json:"omitted_records"`
	OmittedJournals    int              `json:"omitted_journals"`
	OmittedRevisions   int              `json:"omitted_revisions"`
	ConflictingRecords int              `json:"conflicting_records"`
	Notice             string           `json:"notice"`
}

func project(s memory.ReportSnapshot, in Selection) (Projection, []byte, error) {
	p := Projection{RecordIDs: []string{}, RevisionIDs: []string{}, JournalIDs: []string{}, Conflicts: []Conflict{}, OmittedFields: []string{}}
	bad := func() (Projection, []byte, error) {
		return p, nil, errors.New("invalid or oversized explicit report selection; inspect IDs and omission categories")
	}
	if len(in.RecordIDs)+len(in.Scopes)+len(in.JournalIDs) == 0 || len(in.RecordIDs) > 128 || len(in.Scopes) > 16 || len(in.JournalIDs) > 128 || len(in.OmitRecordIDs) > 128 || len(in.OmitJournalIDs) > 128 || len(in.OmitFields) > len(fieldCategories) {
		return bad()
	}
	if in.IncludeWithdrawn && !in.IncludeHistory {
		return bad()
	}
	if err := memory.ValidateReportSnapshot(s.Snapshot, s.Registrations); err != nil {
		return p, nil, errors.New("report snapshot is not valid")
	}
	byRecord := map[string][]memory.Revision{}
	scopes := map[memory.Scope]bool{}
	for _, r := range s.Revisions {
		byRecord[r.RecordID] = append(byRecord[r.RecordID], r)
		scopes[r.Scope] = true
	}
	selected := map[string]bool{}
	for _, id := range in.RecordIDs {
		if len(byRecord[id]) == 0 {
			return bad()
		}
		selected[id] = true
	}
	for _, scope := range in.Scopes {
		if !scopes[scope] {
			return bad()
		}
		for id, rs := range byRecord {
			if rs[0].Scope == scope {
				selected[id] = true
			}
		}
	}
	if len(selected) > 128 {
		return bad()
	}
	for id := range selected {
		p.RecordIDs = append(p.RecordIDs, id)
	}
	sort.Strings(p.RecordIDs)
	omitRecords := map[string]bool{}
	for _, id := range in.OmitRecordIDs {
		if !selected[id] {
			return bad()
		}
		omitRecords[id] = true
	}
	journals := map[string]memory.JournalEntry{}
	for _, j := range s.Journal {
		journals[j.ID] = j
	}
	selectedJournals := map[string]bool{}
	for _, id := range in.JournalIDs {
		if _, ok := journals[id]; !ok {
			return bad()
		}
		selectedJournals[id] = true
	}
	omitJournals := map[string]bool{}
	for _, id := range in.OmitJournalIDs {
		if !selectedJournals[id] {
			return bad()
		}
		omitJournals[id] = true
	}
	omit := map[string]bool{}
	for _, field := range in.OmitFields {
		if !slices.Contains(fieldCategories, field) {
			return bad()
		}
		omit[field] = true
	}
	if len(in.OmitFields) > 0 {
		omit["extensions"] = true
	}
	if omit["identity"] {
		omit["authorship"] = true
		omit["citations"] = true
		omit["change_history"] = true
	}
	if omit["content"] {
		omit["citations"] = true
		omit["change_history"] = true
	}
	// Sources and historical origins carry their own author, timestamp and
	// classification values. Omit the complete citation group rather than
	// silently retaining alternate disclosures of an omitted category.
	if omit["authorship"] || omit["timestamps"] || omit["classification"] {
		omit["citations"] = true
	}
	if !in.IncludeDetails {
		for _, k := range []string{"authorship", "citations", "change_history", "extensions"} {
			omit[k] = true
		}
	}
	if !in.IncludeHistory {
		omit["change_history"] = true
	}
	for _, k := range fieldCategories {
		if omit[k] {
			p.OmittedFields = append(p.OmittedFields, k)
		}
	}
	output := report{Kind: "mandalore-memory-report", Version: 1, Items: []map[string]any{}, OmittedFields: p.OmittedFields, Notice: "Derived selected report; not a restorable signet or safe-to-publish certification. Omitted fields and provenance are incomplete. Git, journals, original sources and other copies persist."}
	if in.IncludeWithdrawn {
		output.Notice += " Withheld records are included only as explicitly requested history, not current guidance."
	}
	if !omit["identity"] {
		output.Identity = map[string]any{"signet_id": s.Signet.ID}
	}
	devices := map[string]memory.Device{}
	for _, d := range s.Devices {
		devices[d.ID] = d
	}
	states, err := memory.ResolveVisibility(s.Revisions, s.Visibility, func(id string) error {
		if _, ok := devices[id]; !ok {
			return errors.New("missing report device")
		}
		return nil
	})
	if err != nil {
		return p, nil, errors.New("report visibility graph is not valid")
	}
	sources := map[string]memory.Source{}
	for _, source := range s.Sources {
		sources[source.ID] = source
	}
	now := time.Now().UTC()
	heads := memory.EffectiveHeads(s.Revisions, now)
	add := func(item map[string]any) {
		item["ordinal"] = len(output.Items) + 1
		output.Items = append(output.Items, item)
	}
	for _, id := range p.RecordIDs {
		rs := byRecord[id]
		withheld := states[id].Withheld()
		if withheld {
			p.Withheld = append(p.Withheld, WithheldRecord{RecordID: id, State: states[id].State})
		}
		sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
		hs := heads[id]
		headIDs := []string{}
		for _, r := range hs {
			headIDs = append(headIDs, r.ID)
		}
		if len(hs) > 1 {
			p.Conflicts = append(p.Conflicts, Conflict{RecordID: id, HeadIDs: headIDs})
		}
		if omitRecords[id] || (withheld && !in.IncludeWithdrawn) {
			p.OmittedRecords++
			p.OmittedRevisions += len(rs)
			continue
		}
		for _, r := range rs {
			current := slices.Contains(headIDs, r.ID)
			if !in.IncludeHistory && (len(hs) != 1 || !current) {
				p.OmittedRevisions++
				continue
			}
			status := "historical"
			if at, _ := time.Parse(time.RFC3339Nano, r.EffectiveFrom); at.After(now) {
				status = "future"
			}
			if current {
				status = "current"
				if len(hs) > 1 {
					status = "conflicted"
				}
			}
			if withheld {
				status = "withheld-history"
			}
			item := map[string]any{"type": "record", "status": status}
			if withheld {
				item["visibility_state"] = states[id].State
			}
			if !omit["identity"] {
				item["identity"] = map[string]any{"record_id": r.RecordID, "revision_id": r.ID, "scope": r.Scope}
			}
			if !omit["content"] {
				item["content"] = map[string]any{"summary": r.Summary, "body": r.Body}
			}
			if !omit["classification"] {
				item["classification"] = map[string]any{"kind": r.Kind, "sensitivity": r.Sensitivity, "volatility": r.Volatility, "basis": r.Evidence.Basis, "confidence": r.Evidence.Confidence}
			}
			if !omit["timestamps"] {
				item["timestamps"] = map[string]any{"recorded_at": r.RecordedAt, "effective_from": r.EffectiveFrom, "last_verified_at": r.LastVerifiedAt}
			}
			if !omit["authorship"] {
				item["authorship"] = map[string]any{"author": r.Authorship, "device": devices[r.Authorship.DeviceID]}
			}
			if !omit["citations"] {
				refs := []memory.Source{}
				for _, id := range r.Evidence.SourceRefs {
					refs = append(refs, sources[id])
				}
				item["citations"] = refs
			}
			if !omit["change_history"] {
				item["change_history"] = map[string]any{"supersedes": r.Supersedes, "reason": r.ChangeReason}
			}
			if !omit["extensions"] && len(r.Extensions) > 0 {
				item["extensions"] = r.Extensions
			}
			p.RevisionIDs = append(p.RevisionIDs, r.ID)
			add(item)
		}
	}
	ids := []string{}
	for id := range selectedJournals {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if omitJournals[id] {
			p.OmittedJournals++
			continue
		}
		j := journals[id]
		item := map[string]any{"type": "journal"}
		if !omit["identity"] {
			item["identity"] = map[string]any{"event_id": j.ID}
		}
		if !omit["content"] {
			item["content"] = map[string]any{"summary": j.Summary}
		}
		if !omit["classification"] {
			item["classification"] = map[string]any{"kind": j.Kind}
		}
		if !omit["timestamps"] {
			item["timestamps"] = map[string]any{"recorded_at": j.RecordedAt}
		}
		if !omit["authorship"] {
			item["authorship"] = map[string]any{"author": j.Authorship, "device": devices[j.Authorship.DeviceID]}
		}
		p.JournalIDs = append(p.JournalIDs, id)
		add(item)
	}
	if len(output.Items) > 128 {
		return bad()
	}
	output.OmittedRecords = p.OmittedRecords
	output.OmittedJournals = p.OmittedJournals
	output.OmittedRevisions = p.OmittedRevisions
	output.ConflictingRecords = len(p.Conflicts)
	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return p, nil, errors.New("report projection cannot be encoded")
	}
	b = append(b, '\n')
	if len(b) > 4<<20 {
		return p, nil, errors.New("report exceeds 4 MiB; narrow selection")
	}
	return p, b, nil
}
