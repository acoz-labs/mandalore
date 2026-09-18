package memory

import (
	"errors"
	"time"
)

// RecallableHeads filters before scoring or projecting content. The supplied
// states must come from validation of these exact revisions and visibility events.
// Missing or unknown states are withheld, never defaulted to visible.
func RecallableHeads(records []Revision, states map[string]VisibilityState, now time.Time) map[string][]Revision {
	heads := EffectiveHeads(records, now)
	for id := range heads {
		if visibilityWithheld(states[id].State) {
			delete(heads, id)
		}
	}
	return heads
}

// VisibilityState contains routing/history metadata, never recalled content.
type VisibilityState struct {
	State           string   `json:"state"`
	ContentHeads    []string `json:"content_heads"`
	VisibilityHeads []string `json:"visibility_heads"`
}

// Withheld separates visibility policy from ordinary content conflicts. Unknown
// or missing state is withheld rather than silently defaulting to visible.
func (s VisibilityState) Withheld() bool { return visibilityWithheld(s.State) }

// ResolveVisibility validates visibility metadata and the combined causal graph.
// Callers must also validate the closed revision/source graph and enclosing format.
// It considers all stored revisions, including future-effective revisions: a
// scheduled correction must not become an unseen restoration when time advances.
func ResolveVisibility(records []Revision, events []VisibilityEvent, device func(string) error) (map[string]VisibilityState, error) {
	groups := map[string][]visibilityNode{}
	ids := map[string]bool{}
	for _, r := range records {
		if ids[r.ID] {
			return nil, errVisibilityGraph
		}
		ids[r.ID] = true
		refs := []string{}
		switch r.Version {
		case 1:
			if r.VisibilityRefs != nil {
				return nil, errVisibilityGraph
			}
		case 2:
			if r.VisibilityRefs == nil || *r.VisibilityRefs == nil {
				return nil, errVisibilityGraph
			}
			refs = *r.VisibilityRefs
		default:
			return nil, errVisibilityGraph
		}
		groups[r.RecordID] = append(groups[r.RecordID], visibilityNode{ID: r.ID, RecordID: r.RecordID, Kind: "content", Parents: r.Supersedes, References: refs})
	}
	for _, e := range events {
		if err := validateVisibilityEvent(e, device); err != nil {
			return nil, err
		}
		if ids[e.ID] || len(groups[e.RecordID]) == 0 {
			return nil, errors.New("visibility event has duplicate identity or missing record")
		}
		ids[e.ID] = true
		groups[e.RecordID] = append(groups[e.RecordID], visibilityNode{ID: e.ID, RecordID: e.RecordID, Kind: e.Action, Parents: e.Parents, References: e.Observed})
	}
	states := map[string]VisibilityState{}
	for id, nodes := range groups {
		v, err := evaluateVisibility(nodes)
		if err != nil {
			return nil, err
		}
		states[id] = VisibilityState{State: v.State, ContentHeads: v.ContentHeads, VisibilityHeads: v.VisibilityHeads}
	}
	return states, nil
}
