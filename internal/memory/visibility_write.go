package memory

import (
	"errors"
	"sort"
)

var ErrStaleHeads = errors.New("memory heads changed; inspect current content and visibility history before retrying")

func sameHeads(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	a, b = append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] || (i > 0 && a[i] == a[i-1]) {
			return false
		}
	}
	return true
}

// Called only under the shared writer lock, after validation of the current
// graph. Remote/imported revisions are validated without this local-write step:
// their original causal context must never be rewritten as a fresh observation.
func prepareContentWrite(r *Revision, format int, states map[string]VisibilityState) error {
	if r.Version != 1 && r.Version != 2 {
		return errors.New("unsupported revision version")
	}
	if r.Version == 1 && r.VisibilityRefs != nil {
		return errors.New("legacy revision cannot carry visibility context")
	}
	if format == 1 {
		if r.Version != 1 {
			return errors.New("revision requires upgraded signet format")
		}
		return nil
	}
	if format != 2 {
		return errors.New("unsupported signet format")
	}
	state := states[r.RecordID]
	if !sameHeads(r.Supersedes, state.ContentHeads) {
		return ErrStaleHeads
	}
	if r.Version == 2 && (r.VisibilityRefs == nil || *r.VisibilityRefs == nil || !sameHeads(*r.VisibilityRefs, state.VisibilityHeads)) {
		return ErrStaleHeads
	}
	refs := append([]string{}, state.VisibilityHeads...)
	r.Version, r.VisibilityRefs = 2, &refs
	return nil
}
