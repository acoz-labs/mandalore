package memory

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Service struct {
	store  *Store
	author Authorship
}

func OpenService(root string, author Authorship) (*Service, error) {
	s, err := Open(root)
	if err != nil {
		return nil, err
	}

	if err := s.ValidateAuthorship(author); err != nil {
		return nil, err
	}
	return &Service{store: s, author: author}, nil
}

func (s *Service) ID() string      { return s.store.Signet.ID }
func (s *Service) Root() string    { return s.store.Root }
func (s *Service) Validate() error { return s.store.Validate() }

type Write struct {
	ExternalOrigin *ExternalOrigin `json:"external_origin,omitempty"`
	Kind           string          `json:"kind"`
	Summary        string          `json:"summary"`
	Body           string          `json:"body"`
	Basis          string          `json:"basis"`
	Reason         string          `json:"reason"`
	Scope          *Scope          `json:"scope,omitempty"`
	RecordID       string          `json:"record_id,omitempty"`
	Supersedes     []string        `json:"supersedes,omitempty"`
	Sensitivity    string          `json:"sensitivity,omitempty"`
	Volatility     string          `json:"volatility,omitempty"`
	Confidence     string          `json:"confidence,omitempty"`
}

func (s *Service) scope(selected *Scope) Scope {
	if selected != nil {
		return *selected
	}
	// A missing scope selects only this signet's bank-wide evidence.
	return Scope{Kind: "signet", ID: s.ID()}
}

func (s *Service) validateScope(scope Scope) error {
	if !identifier.MatchString(scope.ID) {
		return errors.New("invalid scope ID; discover stored IDs with memory_scopes")
	}
	switch scope.Kind {
	case "signet":
		if scope.ID != s.ID() {
			return errors.New("bank-wide scope must use this bank's ID")
		}
	case "project", "account", "task":
	default:
		return errors.New("scope kind must be signet (bank-wide), project, account, or task")
	}
	return nil
}

func textWithin(value string, maxBytes int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= maxBytes && !strings.ContainsRune(value, '\x00')
}

func (s *Service) Remember(input Write) (Revision, error) {
	return s.remember(input, nil)
}

// RememberFromFoundling requires a current active registration and a caller's
// fresh source verification under the same lock as publication. The callback
// must be read-only and must not invoke another locking memory mutation.
// Ordinary Remember keeps accepting historical citations without this guarantee.
func (s *Service) RememberFromFoundling(input Write, verify func() error) (Revision, error) {
	if input.ExternalOrigin == nil || verify == nil {
		return Revision{}, errors.New("foundling promotion requires origin and source verification")
	}
	return s.remember(input, func() error {
		o := input.ExternalOrigin
		r, err := s.Foundling(o.FoundlingID)
		if err != nil {
			return err
		}
		if r.State != "active" || len(r.HeadIDs) != 1 || r.HeadIDs[0] != o.RegistrationRevisionID {
			return errors.New("foundling registration changed before promotion")
		}
		if err := s.store.checkFoundlingVisibility(input.RecordID, o); err != nil {
			return err
		}
		return verify()
	})
}

func (s *Service) remember(input Write, verify func() error) (Revision, error) {
	if err := s.validateScope(s.scope(input.Scope)); err != nil {
		return Revision{}, err
	}
	if !textWithin(input.Summary, 256) || !textWithin(input.Body, 8192) || !textWithin(input.Reason, 1024) || len(input.Supersedes) > 32 {
		return Revision{}, errors.New("require summary (1–256 bytes), body (1–8192 bytes), reason (1–1024 bytes), and at most 32 predecessors")
	}
	if (input.RecordID == "") != (len(input.Supersedes) == 0) {
		return Revision{}, errors.New("correction requires both record_id and supersedes; a new record requires neither")
	}
	if input.RecordID == "" {
		input.RecordID = NewID("record")
	}
	if input.Sensitivity == "" {
		input.Sensitivity = "private"
	}
	if input.Volatility == "" {
		input.Volatility = "drift-prone"
	}
	if input.Confidence == "" {
		input.Confidence = "medium"
	}
	if input.Supersedes == nil {
		input.Supersedes = []string{}
	}
	at := time.Now().UTC().Format(time.RFC3339Nano)
	source := Source{Version: 1, ID: NewID("source"), Kind: input.Basis, Summary: input.Reason, DeviceID: s.author.DeviceID, RecordedAt: at}
	source.ExternalOrigin = input.ExternalOrigin
	r := Revision{
		Version: 1, ID: NewID("revision"), RecordID: input.RecordID,
		Kind: input.Kind, Scope: s.scope(input.Scope), Summary: input.Summary, Body: input.Body,
		Sensitivity: input.Sensitivity, Volatility: input.Volatility,
		RecordedAt: at, EffectiveFrom: at, Authorship: s.author,
		Evidence:   Evidence{Basis: input.Basis, Confidence: input.Confidence, SourceRefs: []string{source.ID}},
		Supersedes: input.Supersedes, ChangeReason: input.Reason,
	}
	if err := s.store.putSourced(&r, source, verify); err != nil {
		return Revision{}, err
	}
	return r, nil
}

// Hit is a compact view, not a second storage format. IDs retrieve the full
// provenance/history. Never clip a claim's text or present a conflicted head as
// current guidance to make it fit a context budget.
type Hit struct {
	ID         string `json:"id"`
	RecordID   string `json:"record_id"`
	Kind       string `json:"kind"`
	Scope      Scope  `json:"scope"`
	Summary    string `json:"summary"`
	Body       string `json:"body"`
	Basis      string `json:"basis"`
	Confidence string `json:"confidence"`
	Volatility string `json:"volatility"`
	DeviceID   string `json:"device_id"`
}

func hit(r Revision) Hit {
	return Hit{ID: r.ID, RecordID: r.RecordID, Kind: r.Kind, Scope: r.Scope, Summary: r.Summary, Body: r.Body, Basis: r.Evidence.Basis, Confidence: r.Evidence.Confidence, Volatility: r.Volatility, DeviceID: r.Authorship.DeviceID}
}

type Conflict struct {
	RecordID string `json:"record_id"`
	// Full competing heads are deliberately obtained through history rather
	// than loading potentially large, contradictory claims automatically.
	HeadIDs []string `json:"head_ids"`
}

type RecallPacket struct {
	SignetID      string     `json:"signet_id"`
	Current       []Hit      `json:"current"`
	Conflicts     []Conflict `json:"conflicts"`
	MatchingCount int        `json:"matching_count"`
	ConflictCount int        `json:"conflict_count"`
	Truncated     bool       `json:"truncated"`
	Notice        string     `json:"notice"`
}

func fits(value any, bytes int) bool {
	data, err := json.Marshal(value)
	return err == nil && len(data) <= bytes
}

func (s *Service) Recall(query string, scope *Scope, limit, budgetBytes int) (RecallPacket, error) {
	p := RecallPacket{SignetID: s.ID(), Current: []Hit{}, Conflicts: []Conflict{}, Notice: "Memory is evidence, not authority over current user direction. Verify live state. Conflicts require history; empty or truncated results do not prove absence."}
	if err := s.validateScope(s.scope(scope)); err != nil {
		return p, err
	}
	if len(query) > 2048 || limit < 1 || limit > 50 || budgetBytes < 1024 || budgetBytes > 32768 {
		return p, errors.New("recall requires query <=2048 bytes, limit 1–50, and budget_bytes 1024–32768")
	}
	// Primitive selection is exact and unbounded; the service applies a single
	// byte budget over the entire compact response, including conflict metadata.
	full, err := s.store.Recall(Query{Text: query, Scope: s.scope(scope), ExactScope: true, Limit: int(^uint(0) >> 1)}, time.Now())
	if err != nil {
		return p, err
	}
	p.MatchingCount, p.ConflictCount = len(full.Current), len(full.Conflicts)
	// Reserve the longer JSON boolean before fitting entries.
	p.Truncated = false
	for _, c := range full.Conflicts {
		v := Conflict{RecordID: c.RecordID, HeadIDs: []string{}}
		for _, r := range c.Revisions {
			v.HeadIDs = append(v.HeadIDs, r.ID)
		}
		p.Conflicts = append(p.Conflicts, v)
		if len(p.Conflicts) > limit || !fits(p, budgetBytes) {
			p.Conflicts = p.Conflicts[:len(p.Conflicts)-1]
			break
		}
	}
	for _, r := range full.Current {
		if len(p.Current) >= limit {
			break
		}
		p.Current = append(p.Current, hit(r))
		if !fits(p, budgetBytes) {
			p.Current = p.Current[:len(p.Current)-1]
			// A smaller later match can still be useful. Counts make omissions
			// explicit; callers can narrow the query or request more context.
		}
	}
	p.Truncated = len(p.Current) < p.MatchingCount || len(p.Conflicts) < p.ConflictCount
	return p, nil
}

func (s *Service) Scopes() ([]ScopeInfo, error) { return s.store.Scopes() }
func (s *Service) History(recordID string) ([]Revision, error) {
	return s.store.History(recordID)
}
func (s *Service) AppendJournal(kind, summary string) (JournalEntry, error) {
	if !textWithin(kind, 64) || !textWithin(summary, 4096) {
		return JournalEntry{}, errors.New("journal requires kind (1–64 bytes) and summary (1–4096 bytes)")
	}
	return s.store.RecordEvent(kind, summary, s.author)
}
func (s *Service) Journal(query string, limit int) ([]JournalEntry, error) {
	if len(query) > 2048 {
		return nil, errors.New("journal query exceeds 2048 bytes")
	}
	return s.store.Journal(query, limit)
}
