package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Scope struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// ScopeInfo is a routing inventory, not recalled guidance. Counts include all
// stored records (including future-effective records), not revision counts.
type ScopeInfo struct {
	Scope       Scope            `json:"scope"`
	RecordCount int              `json:"record_count"`
	Visibility  *ScopeVisibility `json:"visibility_counts,omitempty"`
}

type ScopeVisibility struct {
	Visible    int `json:"visible"`
	Withheld   int `json:"withheld"`
	Conflicted int `json:"conflicted"`
}

func (s *Store) Scopes() ([]ScopeInfo, error) {
	records, states, err := s.graph()
	if err != nil {
		return nil, err
	}
	byScope := map[Scope]map[string]bool{}
	for _, r := range records {
		if byScope[r.Scope] == nil {
			byScope[r.Scope] = map[string]bool{}
		}
		byScope[r.Scope][r.RecordID] = true
	}
	result := []ScopeInfo{}
	for scope, records := range byScope {
		info := ScopeInfo{Scope: scope, RecordCount: len(records)}
		if s.Signet.Version == 2 {
			counts := &ScopeVisibility{}
			for id := range records {
				switch states[id].State {
				case "visible":
					counts.Visible++
				case "content-conflict", "visibility-conflict":
					counts.Conflicted++
				default:
					counts.Withheld++
				}
			}
			info.Visibility = counts
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Scope.Kind == result[j].Scope.Kind {
			return result[i].Scope.ID < result[j].Scope.ID
		}
		return result[i].Scope.Kind < result[j].Scope.Kind
	})
	return result, nil
}

type Authorship struct {
	DeviceID  string  `json:"device_id"`
	Actor     string  `json:"actor"`
	Harness   string  `json:"harness"`
	Model     *string `json:"model"`
	SessionID *string `json:"session_id"`
}
type Evidence struct {
	Basis      string   `json:"basis"`
	Confidence string   `json:"confidence"`
	SourceRefs []string `json:"source_refs"`
}
type Revision struct {
	VisibilityRefs *[]string      `json:"visibility_refs,omitempty"`
	Version        int            `json:"schema_version"`
	ID             string         `json:"id"`
	RecordID       string         `json:"record_id"`
	Kind           string         `json:"kind"`
	Scope          Scope          `json:"scope"`
	Summary        string         `json:"summary"`
	Body           string         `json:"body"`
	Sensitivity    string         `json:"sensitivity"`
	Volatility     string         `json:"volatility"`
	LastVerifiedAt *string        `json:"last_verified_at,omitempty"`
	RecordedAt     string         `json:"recorded_at"`
	EffectiveFrom  string         `json:"effective_from"`
	Authorship     Authorship     `json:"authorship"`
	Evidence       Evidence       `json:"evidence"`
	Supersedes     []string       `json:"supersedes"`
	ChangeReason   string         `json:"change_reason"`
	Extensions     map[string]any `json:"extensions,omitempty"`
}
type Query struct {
	Text  string
	Scope Scope
	Limit int
	// ExactScope omits bank-wide records when a narrower scope is selected.
	// False includes signet-wide evidence alongside the selected scope.
	ExactScope bool
	// AllScopes ranks every stored scope in one pass and surfaces only the
	// conflicts whose heads match the query. It overrides Scope and ExactScope.
	AllScopes bool
	// Names, when not nil, are the query's words written as names; only they
	// can anchor a record (Packet.Anchor). Ranking ignores them.
	Names []string
}
type RevisionConflict struct {
	RecordID  string     `json:"record_id"`
	Revisions []Revision `json:"revisions"`
}
type Packet struct {
	SignetID    string             `json:"signet_id"`
	GeneratedAt string             `json:"generated_at"`
	Current     []Revision         `json:"current"`
	Conflicts   []RevisionConflict `json:"conflicts"`
	Notice      string             `json:"notice"`
	// Strength is each current or competing head's match as a share of the
	// most a revision could score on this query, by revision ID: from 0 to 1,
	// times the standing-guidance boost for a decision or preference.
	Strength map[string]float64 `json:"-"`
	// Evidence is each current or competing head's score in units of one
	// fully matched term that no other revision contains, by revision ID.
	// Unlike Strength it doesn't fall as the query grows.
	Evidence map[string]float64 `json:"-"`
	// Anchor is, for each current or competing head, the rarest query term it
	// contains exactly in its summary or identifiers, as that term's IDF over
	// the most a term can have, by revision ID; 0 when there is none. With
	// Query.Names, only those terms anchor.
	Anchor map[string]float64 `json:"-"`
	// Incidental marks the current or competing heads that match just one of
	// the query's content terms, and only outside their summary and
	// identifiers, by revision ID.
	Incidental map[string]bool `json:"-"`
}

func (s *Store) validateGraph(records []Revision) error {
	return s.validateGraphWithSources(records, nil)
}

func memoizeDeviceValidation(check func(string) error) func(string) error {
	validated := map[string]bool{}
	return func(id string) error {
		if validated[id] {
			return nil
		}
		if err := check(id); err != nil {
			return err
		}
		validated[id] = true
		return nil
	}
}

func (s *Store) validateGraphWithSources(records []Revision, pending map[string]Source) error {
	_, err := s.validateGraphState(records, pending)
	return err
}

func (s *Store) validateGraphState(records []Revision, pending map[string]Source) (map[string]VisibilityState, error) {
	if err := s.validateUpgradeState(); err != nil {
		return nil, err
	}
	// One graph validation shares successful device checks between revisions
	// and their sources. Do not retain this memo on Store/Service: every later
	// read or write must check current provenance again (a sole writer's
	// CacheReads reuses a validated graph only until its next write). Multi-file reads do
	// not claim atomic snapshots against uncoordinated external filesystem edits.
	device := memoizeDeviceValidation(s.deviceExists)
	err := validateRevisionGraph(records, s.Signet.ID, s.Signet.Version, device, func(id string) error {
		source, exists := pending[id]
		if !exists {
			if err := readJSON(filepath.Join(s.Root, "memory/sources", id+".json"), &source); err != nil {
				return fmt.Errorf("source %s: %w", id, err)
			}
		}
		if source.ID != id {
			return errors.New("source ID mismatch")
		}
		return s.validateSourceWithDevice(source, device)
	})
	if err != nil {
		return nil, err
	}
	return s.visibilityStates(records)
}

// Resolve provenance through caller-owned lookups so a migration preflight can
// validate the same graph entirely in memory, without a filesystem fallback.
func validateRevisionGraph(records []Revision, signetID string, format int, device, source func(string) error) error {
	if format != 1 && format != 2 {
		return errors.New("unsupported revision graph format")
	}
	schema, err := memorySchema()
	if err != nil {
		return err
	}
	byID := map[string]Revision{}
	roots := map[string]int{}
	for _, r := range records {
		if format == 1 && r.Version != 1 {
			return errors.New("revision requires upgraded signet format")
		}
		b, err := json.Marshal(r)
		if err != nil {
			return err
		}
		var value any
		if err = json.Unmarshal(b, &value); err != nil {
			return err
		}
		if err = schema.Validate(value); err != nil {
			return fmt.Errorf("revision %s: %w", r.ID, err)
		}
		if _, exists := byID[r.ID]; exists {
			return fmt.Errorf("duplicate revision %s", r.ID)
		}
		byID[r.ID] = r
		if err = device(r.Authorship.DeviceID); err != nil {
			return err
		}
		if r.Scope.Kind == "signet" && r.Scope.ID != signetID {
			return errors.New("signet-wide scope must match the selected signet")
		}
		for _, id := range r.Evidence.SourceRefs {
			if err := source(id); err != nil {
				return err
			}
		}
		if len(r.Supersedes) == 0 {
			roots[r.RecordID]++
		}
	}
	for _, r := range records {
		if roots[r.RecordID] != 1 {
			return fmt.Errorf("record %s must have exactly one root", r.RecordID)
		}
		effective, _ := time.Parse(time.RFC3339Nano, r.EffectiveFrom)
		for _, id := range r.Supersedes {
			parent, exists := byID[id]
			if !exists {
				return fmt.Errorf("missing predecessor %s", id)
			}
			if parent.RecordID != r.RecordID || parent.Kind != r.Kind || parent.Scope != r.Scope {
				return fmt.Errorf("successor %s changes record identity, kind, or scope", r.ID)
			}
			previous, _ := time.Parse(time.RFC3339Nano, parent.EffectiveFrom)
			if effective.Before(previous) {
				return errors.New("successor cannot take effect before its predecessor")
			}
		}
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return errors.New("supersession cycle")
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, parent := range byID[id].Supersedes {
			if err := visit(parent); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range byID {
		if err = visit(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Put(r Revision) error {
	if r.Supersedes == nil {
		r.Supersedes = []string{}
	}
	if r.Evidence.SourceRefs == nil {
		r.Evidence.SourceRefs = []string{}
	}
	return s.withLock(func() error {
		records, err := s.revisions()
		if err != nil {
			return err
		}
		var states map[string]VisibilityState
		if s.Signet.Version == 2 {
			states, err = s.validateGraphState(records, nil)
			if err != nil {
				return err
			}
		}
		if err := prepareContentWrite(&r, s.Signet.Version, states); err != nil {
			return err
		}
		if err = s.validateGraph(append(records, r)); err != nil {
			return err
		}
		dir := filepath.Join(s.Root, "memory/records", r.RecordID)
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid record directory")
		}
		return writeNewJSON(filepath.Join(dir, r.ID+".json"), r)
	})
}

func (s *Store) History(recordID string) ([]Revision, error) {
	if !identifier.MatchString(recordID) {
		return nil, errors.New("invalid record ID")
	}
	records, err := s.revisions()
	if err != nil {
		return nil, err
	}
	if err = s.validateGraph(records); err != nil {
		return nil, err
	}
	history := []Revision{}
	for _, r := range records {
		if r.RecordID == recordID {
			history = append(history, r)
		}
	}
	// Stable topological ordering also preserves concurrent branches with equal times.
	result := []Revision{}
	seen := map[string]bool{}
	for len(result) < len(history) {
		sort.Slice(history, func(i, j int) bool {
			if history[i].RecordedAt == history[j].RecordedAt {
				return history[i].ID < history[j].ID
			}
			return history[i].RecordedAt < history[j].RecordedAt
		})
		progress := false
		for _, r := range history {
			if seen[r.ID] {
				continue
			}
			ready := true
			for _, id := range r.Supersedes {
				if !seen[id] {
					ready = false
				}
			}
			if ready {
				result = append(result, r)
				seen[r.ID] = true
				progress = true
			}
		}
		if !progress {
			return nil, errors.New("invalid history graph")
		}
	}
	return result, nil
}

var words = regexp.MustCompile(`[\pL\pN][\pL\pN._-]*`)

func terms(text string) []string {
	tokens := words.FindAllString(strings.ToLower(text), -1)
	for i := range tokens {
		// Keep internal dots/hyphens intact (domains, versions, identifiers),
		// but do not let a sentence-ending period hide an otherwise exact hit.
		tokens[i] = strings.TrimRight(tokens[i], ".")
	}
	return tokens
}

// A small English inflection fallback for prose, not identifiers or synonyms.
// Exact hits receive four times the weight. Do not expand arbitrary substrings.
func inflected(base, word string) bool {
	if len(base) < 4 {
		return false
	}
	for _, r := range base + word {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	for _, suffix := range []string{"s", "es", "ed", "ing"} {
		if word == base+suffix {
			return true
		}
	}
	return strings.HasSuffix(base, "e") && (word == base+"d" || word == strings.TrimSuffix(base, "e")+"ing")
}

func (s *Store) anyRelevant(revisions []Revision, query []string) bool {
	for _, r := range revisions {
		if s.reads.index(r).score(query) > 0 {
			return true
		}
	}
	return false
}

// termIndex is a revision's weighted terms: summary terms outweigh body terms
// and identifiers outweigh both. summary holds the summary's own terms, and
// length counts its summary and body terms.
type termIndex struct {
	tokens, identifiers map[string]int
	summary             map[string]bool
	length              int
}

func indexTerms(r Revision) termIndex {
	ix := termIndex{tokens: map[string]int{}, identifiers: map[string]int{}, summary: map[string]bool{}}
	for _, t := range terms(r.Body) {
		ix.tokens[t]++
		ix.length++
	}
	for _, t := range terms(r.Summary) {
		ix.tokens[t] += 4
		ix.summary[t] = true
		ix.length++
	}
	for _, t := range terms(r.ID + " " + r.RecordID + " " + r.Scope.ID) {
		ix.identifiers[t] += 8
	}
	return ix
}

// queryTerms is nil for a blank query, which matches every revision; a query
// with no word characters matches none.
func queryTerms(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return append([]string{}, terms(text)...)
}

func (ix termIndex) score(query []string) int {
	if query == nil {
		return 1
	}
	score := 0
	for _, t := range query {
		score += ix.weight(t)
	}
	return score
}

// weight is the term's matches in the revision: four per exact body hit, 16
// per summary hit, 32 per identifier hit, and the plain weight of an inflection.
func (ix termIndex) weight(t string) int {
	w := 4 * (ix.tokens[t] + ix.identifiers[t])
	for word, weight := range ix.tokens {
		if word != t && (inflected(t, word) || inflected(word, t)) {
			w += weight
		}
	}
	return w
}

// BM25 parameters: term-frequency saturation and length normalization.
const bm25K1, bm25B = 1.2, 0.75

// Decisions and preferences are standing guidance, so on similar matches they
// rank above status notes and research that mention the same words.
const standingGuidanceBoost = 1.2

// functionWords are English words that carry no topic. They still make a
// record match, but rank only between records that tie on the other terms.
var functionWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields("a about after again all also am an and any are as at be because been before being between both but by can could did do does doing done each else even ever every for from had has have having he her here hers him his how i if in into is it its just like made make many may me more most much must my need not now of off on once only or other our ours out over own please same she should so some such than that the their them then there these they this those through to too under until up upon us use used very want was way we were what when where which while who whom why will with would yes yet you your yours") {
		functionWords[w] = true
	}
}

// IsFunctionWord reports whether the term is a function word, which ranks
// no record above another that matches the query's other terms.
func IsFunctionWord(term string) bool { return functionWords[term] }

// rank scores candidates by BM25 over the query's content terms; a query of
// function words alone ranks by those. Function words score separately, as a
// tie-break, so a record matching only them still matches but ranks last.
// A nil query matches every candidate equally. strengths gives each content
// score as a share of the most a record could score, which is IDF-normalized:
// matching the query's rare terms counts for more than matching common ones.
func rank(query []string, names map[string]bool, candidates []termIndex) (content, function, strengths, evidence, anchors []float64, incidental []bool) {
	if query == nil {
		content = make([]float64, len(candidates))
		for i := range content {
			content[i] = 1
		}
		return content, make([]float64, len(candidates)), content, content, content, make([]bool, len(candidates))
	}
	seen := map[string]bool{}
	var topical, common []string
	for _, t := range query {
		if !seen[t] {
			seen[t] = true
			if functionWords[t] {
				common = append(common, t)
			} else {
				topical = append(topical, t)
			}
		}
	}
	if len(topical) == 0 {
		topical, common = common, nil
	}
	content, strengths, evidence, anchors, incidental = bm25(topical, names, candidates)
	function, _, _, _, _ = bm25(common, names, candidates)
	return content, function, strengths, evidence, anchors, incidental
}

// bm25 scores candidates on distinct terms, with document frequencies and
// average length taken from the candidates themselves, so a term common
// across them counts for little and a long record gains little from repeating
// it. A candidate that matches none of the terms scores 0.
func bm25(terms []string, names map[string]bool, candidates []termIndex) (scores, strengths, evidence, anchors []float64, incidental []bool) {
	scores = make([]float64, len(candidates))
	strengths = make([]float64, len(candidates))
	evidence = make([]float64, len(candidates))
	anchors = make([]float64, len(candidates))
	incidental = make([]bool, len(candidates))
	matched, inTitle := make([]int, len(candidates)), make([]bool, len(candidates))
	weights := make([][]int, len(candidates))
	df := make([]int, len(terms))
	total := 0
	for i, ix := range candidates {
		total += ix.length
		weights[i] = make([]int, len(terms))
		for j, t := range terms {
			if w := ix.weight(t); w > 0 {
				weights[i][j] = w
				df[j]++
			}
		}
	}
	n := float64(len(candidates))
	average := math.Max(1, float64(total)/math.Max(1, n))
	idf := make([]float64, len(terms))
	most := 0.0
	for j := range terms {
		idf[j] = math.Log(1 + (n-float64(df[j])+0.5)/(float64(df[j])+0.5))
		most += idf[j] * (bm25K1 + 1)
	}
	rarest := math.Log(1 + (n+0.5)/0.5)
	for i, ix := range candidates {
		norm := bm25K1 * (1 - bm25B + bm25B*float64(ix.length)/average)
		for j, w := range weights[i] {
			if w == 0 {
				continue
			}
			f := float64(w) / 4
			scores[i] += idf[j] * f * (bm25K1 + 1) / (f + norm)
			titled := ix.summary[terms[j]] || ix.identifiers[terms[j]] > 0
			matched[i]++
			if titled {
				inTitle[i] = true
				if names == nil || names[terms[j]] {
					anchors[i] = max(anchors[i], idf[j]/rarest)
				}
			}
		}
		if most > 0 {
			strengths[i] = scores[i] / most
		}
		evidence[i] = scores[i] / (rarest * (bm25K1 + 1))
		incidental[i] = matched[i] == 1 && !inTitle[i]
	}
	return scores, strengths, evidence, anchors, incidental
}

func relevance(r Revision, query string) int {
	return indexTerms(r).score(queryTerms(query))
}

func (s *Store) Recall(q Query, now time.Time) (Packet, error) {
	p := Packet{SignetID: s.Signet.ID, GeneratedAt: now.UTC().Format(time.RFC3339Nano), Current: []Revision{}, Conflicts: []RevisionConflict{}, Notice: "Current memory is scoped evidence. Verify volatile live state; conflicting revisions are not current guidance. An empty recall is not evidence that the fact is absent. Use memory scopes to discover stored scope IDs, then recall an explicitly selected scope; do not guess IDs or substitute cwd for a remembered entity."}
	records, states, err := s.graph()
	if err != nil {
		return p, err
	}
	heads := RecallableHeads(records, states, now)
	return s.recallHeads(q, p, heads), nil
}

// EffectiveHeads resolves validated revisions at a point in time without
// selecting among concurrent branches. Callers must validate the closed graph.
func EffectiveHeads(records []Revision, now time.Time) map[string][]Revision {
	applicable := map[string]Revision{}
	superseded := map[string]bool{}
	for _, r := range records {
		at, _ := time.Parse(time.RFC3339Nano, r.EffectiveFrom)
		if at.After(now) {
			continue
		}
		applicable[r.ID] = r
		for _, id := range r.Supersedes {
			superseded[id] = true
		}
	}
	heads := map[string][]Revision{}
	for id, r := range applicable {
		if !superseded[id] {
			heads[r.RecordID] = append(heads[r.RecordID], r)
		}
	}
	for id := range heads {
		sort.Slice(heads[id], func(i, j int) bool { return heads[id][i].ID < heads[id][j].ID })
	}
	return heads
}

func (s *Store) recallHeads(q Query, p Packet, heads map[string][]Revision) Packet {
	// Scores belong to this fresh read, not a persistent index. Tokenizing the
	// same bodies inside the sort comparator makes broad recalls needlessly
	// expensive as a bank grows; compute each candidate's relevance only once.
	query := queryTerms(q.Text)
	var candidates []termIndex
	for recordID, revisions := range heads {
		scoped := revisions[0].Scope
		if !q.AllScopes && q.ExactScope && scoped != q.Scope {
			continue
		}
		if !q.AllScopes && scoped.Kind != "signet" && scoped != q.Scope {
			continue
		}
		if scoped.Kind == "signet" && scoped.ID != s.Signet.ID {
			continue
		}
		if len(revisions) > 1 && q.AllScopes && !s.anyRelevant(revisions, query) {
			continue
		}
		if len(revisions) > 1 {
			// All relevant scope conflicts are surfaced even if a new wording misses the query.
			sort.Slice(revisions, func(i, j int) bool { return revisions[i].ID < revisions[j].ID })
			p.Conflicts = append(p.Conflicts, RevisionConflict{RecordID: recordID, Revisions: revisions})
			continue
		}
		p.Current = append(p.Current, revisions[0])
		candidates = append(candidates, s.reads.index(revisions[0]))
	}
	// Competing heads count toward term statistics and get strengths too, so
	// a caller can weigh a matching conflict like a current record.
	var competing []Revision
	for _, c := range p.Conflicts {
		for _, r := range c.Revisions {
			competing = append(competing, r)
			candidates = append(candidates, s.reads.index(r))
		}
	}
	var names map[string]bool
	if q.Names != nil {
		names = map[string]bool{}
		for _, t := range terms(strings.Join(q.Names, " ")) {
			names[t] = true
		}
	}
	content, function, strengths, evidence, anchors, incidental := rank(query, names, candidates)
	p.Strength, p.Evidence, p.Anchor, p.Incidental = map[string]float64{}, map[string]float64{}, map[string]float64{}, map[string]bool{}
	for i, r := range competing {
		j := len(p.Current) + i
		p.Strength[r.ID], p.Evidence[r.ID], p.Anchor[r.ID], p.Incidental[r.ID] = strengths[j], evidence[j], anchors[j], incidental[j]
	}
	scores := map[string][2]float64{}
	matching := p.Current[:0]
	for i, r := range p.Current {
		if content[i] > 0 || function[i] > 0 {
			if query != nil && (r.Kind == "decision" || r.Kind == "preference") {
				content[i] *= standingGuidanceBoost
				strengths[i] *= standingGuidanceBoost
				evidence[i] *= standingGuidanceBoost
			}
			scores[r.ID] = [2]float64{content[i], function[i]}
			p.Strength[r.ID], p.Evidence[r.ID], p.Anchor[r.ID], p.Incidental[r.ID] = strengths[i], evidence[i], anchors[i], incidental[i]
			matching = append(matching, r)
		}
	}
	p.Current = matching
	sort.Slice(p.Current, func(i, j int) bool {
		a, b := scores[p.Current[i].ID], scores[p.Current[j].ID]
		if a != b {
			return a[0] > b[0] || a[0] == b[0] && a[1] > b[1]
		}
		return p.Current[i].ID < p.Current[j].ID
	})
	sort.Slice(p.Conflicts, func(i, j int) bool { return p.Conflicts[i].RecordID < p.Conflicts[j].RecordID })
	limit := q.Limit
	if limit <= 0 {
		limit = 8
	}
	if len(p.Current) > limit {
		p.Current = p.Current[:limit]
	}
	return p
}
