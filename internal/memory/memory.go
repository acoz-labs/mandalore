package memory

import (
	"encoding/json"
	"errors"
	"fmt"
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
	Scope       Scope `json:"scope"`
	RecordCount int   `json:"record_count"`
}

func (s *Store) Scopes() ([]ScopeInfo, error) {
	records, err := s.revisions()
	if err != nil {
		return nil, err
	}
	if err := s.validateGraph(records); err != nil {
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
		result = append(result, ScopeInfo{Scope: scope, RecordCount: len(records)})
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
}

func (s *Store) validateGraph(records []Revision) error {
	return s.validateGraphWithSources(records, nil)
}

func (s *Store) validateGraphWithSources(records []Revision, pending map[string]Source) error {
	return validateRevisionGraph(records, s.Signet.ID, s.deviceExists, func(id string) error {
		source, exists := pending[id]
		if !exists {
			if err := readJSON(filepath.Join(s.Root, "memory/sources", id+".json"), &source); err != nil {
				return fmt.Errorf("source %s: %w", id, err)
			}
		}
		if source.ID != id {
			return errors.New("source ID mismatch")
		}
		return s.validateSource(source)
	})
}

// Resolve provenance through caller-owned lookups so a migration preflight can
// validate the same graph entirely in memory, without a filesystem fallback.
func validateRevisionGraph(records []Revision, signetID string, device, source func(string) error) error {
	schema, err := memorySchema()
	if err != nil {
		return err
	}
	byID := map[string]Revision{}
	roots := map[string]int{}
	for _, r := range records {
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

func terms(text string) []string { return words.FindAllString(strings.ToLower(text), -1) }

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

func relevance(r Revision, query string) int {
	if strings.TrimSpace(query) == "" {
		return 1
	}
	tokens := map[string]int{}
	for _, t := range terms(r.Body) {
		tokens[t]++
	}
	for _, t := range terms(r.Summary) {
		tokens[t] += 4
	}
	identifiers := map[string]int{}
	for _, t := range terms(r.ID + " " + r.RecordID + " " + r.Scope.ID) {
		identifiers[t] += 8
	}
	score := 0
	for _, t := range terms(query) {
		score += 4 * (tokens[t] + identifiers[t])
		for word, weight := range tokens {
			if word != t && (inflected(t, word) || inflected(word, t)) {
				score += weight
			}
		}
	}
	return score
}

func (s *Store) Recall(q Query, now time.Time) (Packet, error) {
	p := Packet{SignetID: s.Signet.ID, GeneratedAt: now.UTC().Format(time.RFC3339Nano), Current: []Revision{}, Conflicts: []RevisionConflict{}, Notice: "Current memory is scoped evidence. Verify volatile live state; conflicting revisions are not current guidance. An empty recall is not evidence that the fact is absent. Use memory scopes to discover stored scope IDs, then recall an explicitly selected scope; do not guess IDs or substitute cwd for a remembered entity."}
	records, err := s.revisions()
	if err != nil {
		return p, err
	}
	if err = s.validateGraph(records); err != nil {
		return p, err
	}
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
	// Scores belong to this fresh read, not a persistent index. Tokenizing the
	// same bodies inside the sort comparator makes broad recalls needlessly
	// expensive as a bank grows; compute each candidate's relevance only once.
	scores := map[string]int{}
	for recordID, revisions := range heads {
		scoped := revisions[0].Scope
		if q.ExactScope && scoped != q.Scope {
			continue
		}
		if scoped.Kind != "signet" && scoped != q.Scope {
			continue
		}
		if scoped.Kind == "signet" && scoped.ID != s.Signet.ID {
			continue
		}
		if len(revisions) > 1 {
			// All relevant scope conflicts are surfaced even if a new wording misses the query.
			sort.Slice(revisions, func(i, j int) bool { return revisions[i].ID < revisions[j].ID })
			p.Conflicts = append(p.Conflicts, RevisionConflict{RecordID: recordID, Revisions: revisions})
			continue
		}
		if score := relevance(revisions[0], q.Text); score > 0 {
			scores[revisions[0].ID] = score
			p.Current = append(p.Current, revisions[0])
		}
	}
	sort.Slice(p.Current, func(i, j int) bool {
		a, b := scores[p.Current[i].ID], scores[p.Current[j].ID]
		if a == b {
			return p.Current[i].ID < p.Current[j].ID
		}
		return a > b
	})
	sort.Slice(p.Conflicts, func(i, j int) bool { return p.Conflicts[i].RecordID < p.Conflicts[j].RecordID })
	limit := q.Limit
	if limit <= 0 {
		limit = 8
	}
	if len(p.Current) > limit {
		p.Current = p.Current[:limit]
	}
	return p, nil
}
