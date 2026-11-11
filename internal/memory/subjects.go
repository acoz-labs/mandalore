package memory

import "time"

// Two decisions (or two preferences) in one scope are on one subject when
// their summaries share at least subjectSummaryOverlap of their content terms
// and their bodies at least subjectBodyOverlap (Jaccard over content terms,
// an inflection counting as the same term). "Editor" and "Editor theme"
// share a summary word but not a subject.
const subjectSummaryOverlap, subjectBodyOverlap = 2.0 / 3, 0.3

// RankNewerFirst puts the newer of two records on one subject directly ahead
// of the older in recall. Tests turn it off to show what it changes.
var RankNewerFirst = true

// OnOneSubject reports whether two records are on one subject: same scope,
// both decisions or both preferences, near-identical summaries and
// overlapping bodies. Neither supersedes the other; both stay current.
func OnOneSubject(a, b Revision) bool { return onOneSubject(a, b, indexTerms(a), indexTerms(b)) }

func onOneSubject(a, b Revision, ia, ib termIndex) bool {
	if a.RecordID == b.RecordID || a.Scope != b.Scope || a.Kind != b.Kind || a.Kind != "decision" && a.Kind != "preference" {
		return false
	}
	return overlap(ia.summary, ib.summary) >= subjectSummaryOverlap && overlap(ia.body, ib.body) >= subjectBodyOverlap
}

// overlap is the Jaccard index of two term sets' content terms, an inflection
// of a term counting as the term.
func overlap(a, b map[string]bool) float64 {
	content := func(m map[string]bool) []string {
		out := []string{}
		for t := range m {
			if !functionWords[t] {
				out = append(out, t)
			}
		}
		return out
	}
	as, bs := content(a), content(b)
	shared := 0
	for _, x := range as {
		for _, y := range bs {
			if x == y || inflected(x, y) || inflected(y, x) {
				shared++
				break
			}
		}
	}
	union := len(as) + len(bs) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}

// newer reports whether a took effect after b: by effective time, then
// recorded time, then revision ID.
func newer(a, b Revision) bool {
	ea, _ := time.Parse(time.RFC3339Nano, a.EffectiveFrom)
	eb, _ := time.Parse(time.RFC3339Nano, b.EffectiveFrom)
	if !ea.Equal(eb) {
		return ea.After(eb)
	}
	ra, _ := time.Parse(time.RFC3339Nano, a.RecordedAt)
	rb, _ := time.Parse(time.RFC3339Nano, b.RecordedAt)
	if !ra.Equal(rb) {
		return ra.After(rb)
	}
	return a.ID > b.ID
}

// newerFirst moves the newer of each pair of ranked records on one subject
// directly ahead of the older when the older ranks above it; nothing else
// moves. It returns, by revision ID, the newer record on the same subject of
// each ranked record that has one among pool (every current record the query
// could return), matching the query or not, so a caller that keeps the older
// can bring the newer.
func (s *Store) newerFirst(ranked, pool []Revision) map[string]Revision {
	type group struct {
		scope Scope
		kind  string
	}
	guidance := map[group][]Revision{}
	for _, r := range pool {
		if r.Kind == "decision" || r.Kind == "preference" {
			g := group{r.Scope, r.Kind}
			guidance[g] = append(guidance[g], r)
		}
	}
	pairs := map[string]Revision{}
	for _, r := range ranked {
		for _, other := range guidance[group{r.Scope, r.Kind}] {
			if newer(other, r) && onOneSubject(r, other, s.reads.index(r), s.reads.index(other)) {
				if n, ok := pairs[r.ID]; !ok || newer(other, n) {
					pairs[r.ID] = other
				}
			}
		}
	}
	if !RankNewerFirst {
		return pairs
	}
	for i := 0; i < len(ranked); i++ {
		n, ok := pairs[ranked[i].ID]
		if !ok {
			continue
		}
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].ID == n.ID {
				copy(ranked[i+1:j+1], ranked[i:j])
				ranked[i] = n
				break
			}
		}
	}
	return pairs
}
