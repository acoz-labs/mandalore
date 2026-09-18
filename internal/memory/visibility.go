package memory

import (
	"errors"
	"sort"
)

// visibilityNode is a content-free projection of a single record's causal graph.
// Storage/schema validation belongs to the caller. References cross graph kinds:
// content observes visibility decisions; decisions observe content revisions.
type visibilityNode struct {
	ID, RecordID, Kind  string
	Parents, References []string
}

type visibilityResult struct {
	State                         string
	ContentHeads, VisibilityHeads []string
}

var errVisibilityGraph = errors.New("invalid visibility causal graph")

// evaluateVisibility validates the complete combined graph before returning any
// state. It uses no wall clock, insertion order, cached state or content text.
// Iterative traversal avoids stack growth from attacker-controlled ancestry.
func evaluateVisibility(nodes []visibilityNode) (visibilityResult, error) {
	invalid := func() (visibilityResult, error) { return visibilityResult{}, errVisibilityGraph }
	if len(nodes) == 0 {
		return invalid()
	}
	byID := make(map[string]visibilityNode, len(nodes))
	for _, n := range nodes {
		if n.ID == "" || n.RecordID == "" || n.RecordID != nodes[0].RecordID {
			return invalid()
		}
		if n.Kind != "content" && n.Kind != "withdraw" && n.Kind != "restore" {
			return invalid()
		}
		if _, exists := byID[n.ID]; exists {
			return invalid()
		}
		byID[n.ID] = n
	}
	parents := map[string]bool{}
	remaining := make(map[string]int, len(nodes))
	dependents := map[string][]string{}
	for _, n := range nodes {
		if n.Kind != "content" && len(n.References) == 0 {
			return invalid()
		}
		seen := map[string]bool{}
		for group, refs := range [][]string{n.Parents, n.References} {
			for _, id := range refs {
				other, exists := byID[id]
				if !exists || seen[id] || ((n.Kind == "content") == (other.Kind == "content")) != (group == 0) {
					return invalid()
				}
				seen[id] = true
				remaining[n.ID]++
				dependents[id] = append(dependents[id], n.ID)
				if group == 0 {
					parents[id] = true
				}
			}
		}
	}
	ready := []string{}
	for _, n := range nodes {
		if remaining[n.ID] == 0 {
			ready = append(ready, n.ID)
		}
	}
	for i := 0; i < len(ready); i++ {
		for _, id := range dependents[ready[i]] {
			remaining[id]--
			if remaining[id] == 0 {
				ready = append(ready, id)
			}
		}
	}
	if len(ready) != len(nodes) {
		return invalid()
	}
	out := visibilityResult{ContentHeads: []string{}, VisibilityHeads: []string{}}
	for _, n := range nodes {
		if parents[n.ID] {
			continue
		}
		if n.Kind == "content" {
			out.ContentHeads = append(out.ContentHeads, n.ID)
		} else {
			out.VisibilityHeads = append(out.VisibilityHeads, n.ID)
		}
	}
	sort.Strings(out.ContentHeads)
	sort.Strings(out.VisibilityHeads)
	if len(out.ContentHeads) == 0 {
		return invalid()
	}
	out.State = "visible"
	if len(out.ContentHeads) != 1 {
		out.State = "content-conflict"
	}
	if len(out.VisibilityHeads) == 0 {
		return out, nil
	}
	if len(out.VisibilityHeads) != 1 {
		out.State = "visibility-conflict"
		return out, nil
	}
	head := byID[out.VisibilityHeads[0]]
	if head.Kind == "withdraw" {
		out.State = "withdrawn"
		return out, nil
	}
	reviewed := map[string]bool{}
	queue := append([]string(nil), head.References...)
	for len(queue) > 0 {
		id := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if reviewed[id] {
			continue
		}
		reviewed[id] = true
		queue = append(queue, byID[id].Parents...)
	}
	for _, id := range out.ContentHeads {
		covered := reviewed[id]
		for _, ref := range byID[id].References {
			if ref == head.ID {
				covered = true
				break
			}
		}
		if !covered {
			out.State = "unreviewed-content"
			break
		}
	}
	return out, nil
}
