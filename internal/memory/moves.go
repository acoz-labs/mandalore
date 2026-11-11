package memory

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RecordMove moves a record to another scope. It is append-only evidence in
// memory/moves/, a directory readers before 2.3.0 never open, so the record's
// revisions keep the scope their root was saved in and older readers still
// accept the graph. Every unsuperseded move is a current move head.
type RecordMove struct {
	Version    int        `json:"schema_version"`
	ID         string     `json:"id"`
	RecordID   string     `json:"record_id"`
	Scope      Scope      `json:"scope"`
	Observed   []string   `json:"observed_content_heads"`
	Parents    []string   `json:"parent_move_heads"`
	Reason     string     `json:"reason"`
	RecordedAt string     `json:"recorded_at"`
	Authorship Authorship `json:"authorship"`
}

// Placement is where a record is now. With no moves it is the scope the
// record was saved in. Move heads that agree give their scope; heads that
// disagree are a move conflict, and Scope is then zero.
type Placement struct {
	Scope     Scope
	Conflict  []Scope // the disagreeing heads' scopes, sorted
	MoveHeads []string
	Moves     int
}

// Conflicted reports a move conflict.
func (p Placement) Conflicted() bool { return len(p.Conflict) > 0 }

// In reports whether the record belongs to scope: its current scope, or any
// scope its conflicting move heads name.
func (p Placement) In(scope Scope) bool {
	if !p.Conflicted() {
		return p.Scope == scope
	}
	for _, s := range p.Conflict {
		if s == scope {
			return true
		}
	}
	return false
}

// Scopes lists every scope the record belongs to.
func (p Placement) Scopes() []Scope {
	if p.Conflicted() {
		return p.Conflict
	}
	return []Scope{p.Scope}
}

const movesDir = "memory/moves"

// ErrMoveConflict refuses a correction while a record's moves disagree.
var ErrMoveConflict = inputError("record has conflicting moves; settle them with memory_move first, naming every move head")

func movePath(m RecordMove) string {
	return filepath.Join(movesDir, m.RecordID, m.ID+".json")
}

// CanonicalMovePath is a move's slash-separated path in a signet.
func CanonicalMovePath(m RecordMove) string { return filepath.ToSlash(movePath(m)) }

func sortScopes(scopes []Scope) {
	sort.Slice(scopes, func(i, j int) bool {
		if scopes[i].Kind != scopes[j].Kind {
			return scopes[i].Kind < scopes[j].Kind
		}
		return scopes[i].ID < scopes[j].ID
	})
}

func validScope(scope Scope, signetID string) bool {
	switch scope.Kind {
	case "signet":
		return scope.ID == signetID
	case "project", "account", "task":
		return identifier.MatchString(scope.ID)
	}
	return false
}

func validateMove(m RecordMove, signetID string, device func(string) error) error {
	invalid := errors.New("invalid record move metadata")
	if m.Version != 1 || !identifier.MatchString(m.ID) || !identifier.MatchString(m.RecordID) || !validScope(m.Scope, signetID) || !textWithin(m.Reason, MaxReasonBytes) || len(m.Observed) == 0 || m.Parents == nil {
		return invalid
	}
	if _, err := time.Parse(time.RFC3339Nano, m.RecordedAt); err != nil {
		return invalid
	}
	seen := map[string]bool{}
	for _, refs := range [][]string{m.Observed, m.Parents} {
		for _, id := range refs {
			if !identifier.MatchString(id) || id == m.ID || seen[id] {
				return invalid
			}
			seen[id] = true
		}
	}
	return validateAuthorship(m.Authorship, device)
}

// ResolvePlacements validates the moves against the revisions they refer to
// and returns every record's placement. Callers validate the revisions.
func ResolvePlacements(records []Revision, moves []RecordMove, signetID string, device func(string) error) (map[string]Placement, error) {
	out := map[string]Placement{}
	revisions := map[string]string{} // revision ID to record ID
	for _, r := range records {
		revisions[r.ID] = r.RecordID
		if _, ok := out[r.RecordID]; !ok {
			out[r.RecordID] = Placement{Scope: r.Scope, MoveHeads: []string{}}
		}
	}
	byID := map[string]RecordMove{}
	for _, m := range moves {
		if err := validateMove(m, signetID, device); err != nil {
			return nil, err
		}
		if _, dup := byID[m.ID]; dup || revisions[m.ID] != "" {
			return nil, fmt.Errorf("duplicate record move %s", m.ID)
		}
		if _, ok := out[m.RecordID]; !ok {
			return nil, fmt.Errorf("record move %s names a missing record", m.ID)
		}
		for _, id := range m.Observed {
			if revisions[id] != m.RecordID {
				return nil, fmt.Errorf("record move %s observes a revision of another record", m.ID)
			}
		}
		byID[m.ID] = m
	}
	// Kahn's algorithm: every parent must exist in the same record and the
	// moves must form no cycle.
	remaining := map[string]int{}
	children := map[string][]string{}
	superseded := map[string]bool{}
	for _, m := range moves {
		for _, id := range m.Parents {
			parent, ok := byID[id]
			if !ok || parent.RecordID != m.RecordID {
				return nil, fmt.Errorf("record move %s has a missing or foreign parent", m.ID)
			}
			remaining[m.ID]++
			children[id] = append(children[id], m.ID)
			superseded[id] = true
		}
	}
	ready := []string{}
	for _, m := range moves {
		if remaining[m.ID] == 0 {
			ready = append(ready, m.ID)
		}
	}
	for i := 0; i < len(ready); i++ {
		for _, id := range children[ready[i]] {
			if remaining[id]--; remaining[id] == 0 {
				ready = append(ready, id)
			}
		}
	}
	if len(ready) != len(moves) {
		return nil, errors.New("record move cycle")
	}
	heads := map[string][]RecordMove{}
	count := map[string]int{}
	for _, m := range moves {
		count[m.RecordID]++
		if !superseded[m.ID] {
			heads[m.RecordID] = append(heads[m.RecordID], m)
		}
	}
	for recordID, hs := range heads {
		p := Placement{MoveHeads: []string{}, Moves: count[recordID]}
		distinct := map[Scope]bool{}
		for _, m := range hs {
			p.MoveHeads = append(p.MoveHeads, m.ID)
			distinct[m.Scope] = true
		}
		sort.Strings(p.MoveHeads)
		if len(distinct) == 1 {
			p.Scope = hs[0].Scope
		} else {
			for s := range distinct {
				p.Conflict = append(p.Conflict, s)
			}
			sortScopes(p.Conflict)
		}
		out[recordID] = p
	}
	return out, nil
}

// moves reads every record move. Like visibility events, there is no cache
// that could outlive a synchronization or another writer's move.
func (s *Store) moves() ([]RecordMove, error) {
	root := filepath.Join(s.Root, movesDir)
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return []RecordMove{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("invalid record move directory")
	}
	moves := []RecordMove{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in record moves")
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("nonregular record move")
		}
		var m RecordMove
		if err := readJSON(path, &m); err != nil {
			return err
		}
		if !identifier.MatchString(m.RecordID) || !identifier.MatchString(m.ID) || path != filepath.Join(s.Root, movePath(m)) {
			return errors.New("record move ID/path mismatch")
		}
		moves = append(moves, m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(moves, func(i, j int) bool { return moves[i].ID < moves[j].ID })
	return moves, nil
}

// placements validates the moves against these revisions.
func (s *Store) placements(records []Revision) (map[string]Placement, error) {
	moves, err := s.moves()
	if err != nil {
		return nil, err
	}
	return ResolvePlacements(records, moves, s.Signet.ID, memoizeDeviceValidation(s.deviceExists))
}

// placeCorrection checks a correction's scope against its record's current
// scope, under the writer lock, and stores the revision in the scope its
// record was saved in: every revision of a record keeps that scope, so
// readers before 2.3.0 still accept the graph. A record that was never moved
// is left to graph validation, as before.
func (s *Store) placeCorrection(r *Revision, records []Revision) error {
	if len(r.Supersedes) == 0 {
		return nil
	}
	placements, err := s.placements(records)
	if err != nil {
		return err
	}
	p, ok := placements[r.RecordID]
	if !ok || p.Moves == 0 {
		return nil
	}
	if p.Conflicted() {
		return ErrMoveConflict
	}
	if r.Scope != p.Scope {
		return invalidInput("record was moved to %s %s; correct it in that scope", p.Scope.Kind, p.Scope.ID)
	}
	for _, rev := range records {
		if rev.RecordID == r.RecordID {
			r.Scope = rev.Scope
			break
		}
	}
	return nil
}

// MoveWrite is a move request. The expected heads are a compare-and-swap,
// as for withdrawal: the record's current revision IDs and move IDs.
type MoveWrite struct {
	RecordID     string   `json:"record_id" jsonschema:"The record to move (record-...)."`
	Scope        Scope    `json:"scope" jsonschema:"The destination: a project, account or task scope, or signet-wide scope with this signet's ID."`
	ContentHeads []string `json:"expected_content_heads" jsonschema:"The record's current revision IDs (revision-...) from memory_history or recall; 1–32."`
	MoveHeads    []string `json:"expected_move_heads" jsonschema:"The record's current move IDs (move_heads from memory_history); an explicit [] for a record never moved; at most 32."`
	Reason       string   `json:"reason" jsonschema:"Why the record belongs in the destination."`
}

// MoveReceipt says where the record is. Moved is false when it was already
// in the destination; nothing was written then.
type MoveReceipt struct {
	SignetID string `json:"signet_id"`
	RecordID string `json:"record_id"`
	ID       string `json:"id,omitempty"`
	Scope    Scope  `json:"scope"`
	Moved    bool   `json:"moved"`
}

// moveIntent is a move's machine-local idempotency state: the move to
// publish, or none for a no-op, and the receipt to return on a retry.
type moveIntent struct {
	Move    *RecordMove `json:"document,omitempty"`
	Receipt MoveReceipt `json:"receipt"`
}

const maxMoveHeads = 32

func checkMoveHeads(field string, ids []string, least int) error {
	if ids == nil || len(ids) < least || len(ids) > maxMoveHeads {
		return invalidInput("%s requires an explicit list of %d–%d IDs", field, least, maxMoveHeads)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !identifier.MatchString(id) || seen[id] {
			return invalidInput("%s has an invalid or duplicate ID", field)
		}
		seen[id] = true
	}
	return nil
}

// prepareMove checks a move against the current graph; the caller holds the
// writer lock. A nil move with a nil error is a no-op.
func (s *Service) prepareMove(in MoveWrite) (*RecordMove, MoveReceipt, error) {
	out := MoveReceipt{SignetID: s.ID(), RecordID: in.RecordID, Scope: in.Scope}
	if !identifier.MatchString(in.RecordID) {
		return nil, out, invalidInput("record_id is invalid; copy it from recall or memory_history")
	}
	if !validScope(in.Scope, s.ID()) {
		return nil, out, invalidInput("scope must be a project, account or task scope, or signet-wide scope with this signet's ID")
	}
	if err := checkText("reason", in.Reason, MaxReasonBytes); err != nil {
		return nil, out, err
	}
	if err := checkMoveHeads("expected_content_heads", in.ContentHeads, 1); err != nil {
		return nil, out, err
	}
	if err := checkMoveHeads("expected_move_heads", in.MoveHeads, 0); err != nil {
		return nil, out, err
	}
	records, err := s.store.revisions()
	if err != nil {
		return nil, out, err
	}
	states, err := s.store.validateGraphState(records, nil)
	if err != nil {
		return nil, out, err
	}
	state, ok := states[in.RecordID]
	if !ok {
		return nil, out, invalidInput("record not found; copy record_id from recall or memory_history")
	}
	p := state.Placement
	// A record with both conflicts is settled by a move first: it must name
	// every move head and every content head (checked below).
	if len(state.ContentHeads) > 1 && !p.Conflicted() {
		return nil, out, invalidInput("record has conflicting revisions; settle the conflict with a correction that supersedes every head first")
	}
	if s.store.Signet.Version == 2 && state.Withheld() {
		return nil, out, invalidInput("record is withdrawn or its visibility is unsettled; restore it before moving it")
	}
	if !sameHeads(in.ContentHeads, state.ContentHeads) || !sameHeads(in.MoveHeads, p.MoveHeads) {
		return nil, out, ErrStaleHeads
	}
	if !p.Conflicted() && p.Scope == in.Scope {
		return nil, out, nil
	}
	m := RecordMove{Version: 1, ID: NewID("move"), RecordID: in.RecordID, Scope: in.Scope, Observed: append([]string{}, state.ContentHeads...), Parents: append([]string{}, p.MoveHeads...), Reason: in.Reason, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author}
	moves, err := s.store.moves()
	if err != nil {
		return nil, out, err
	}
	if _, err := ResolvePlacements(records, append(moves, m), s.ID(), memoizeDeviceValidation(s.store.deviceExists)); err != nil {
		return nil, out, err
	}
	out.ID, out.Moved = m.ID, true
	return &m, out, nil
}

// publishMove writes a prepared move. The caller holds the writer lock.
func (s *Store) publishMove(m RecordMove) error {
	path := filepath.Join(s.Root, movePath(m))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return writeNewJSON(path, m)
}

// Move moves a record to another scope, without idempotency.
func (s *Service) Move(in MoveWrite) (MoveReceipt, error) {
	var out MoveReceipt
	err := s.store.withLock(func() error {
		m, receipt, err := s.prepareMove(in)
		out = receipt
		if err != nil || m == nil {
			return err
		}
		return s.store.publishMove(*m)
	})
	return out, err
}

// MoveIdempotent records at most one move, or one no-op, for a
// principal-namespaced key. A retry returns the first receipt without
// rechecking the guards, so a later move doesn't make it stale.
func (s *Service) MoveIdempotent(key string, in MoveWrite) (MoveReceipt, error) {
	var out MoveReceipt
	err := s.store.withLock(func() error {
		path, digest, err := s.intent(key, "move", in)
		if err != nil {
			return err
		}
		var intent writeIntent
		err = readJSON(path, &intent)
		if err == nil {
			if intent.Digest != digest || intent.Move == nil || intent.Revision != nil || intent.Source != nil || intent.Journal != nil || intent.Label != nil {
				return ErrRequestMismatch
			}
			out = intent.Move.Receipt
			if intent.Move.Move == nil {
				return nil
			}
			return publishOnce(s.store, movePath(*intent.Move.Move), *intent.Move.Move, s.store.publishMove)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		m, receipt, err := s.prepareMove(in)
		if err != nil {
			return err
		}
		if err := writeNewJSON(path, writeIntent{Digest: digest, Move: &moveIntent{Move: m, Receipt: receipt}}); err != nil {
			return err
		}
		out = receipt
		if m == nil {
			return nil
		}
		return s.store.publishMove(*m)
	})
	return out, err
}

// recordMoves returns a record's placement and its moves, from one
// validated read of the graph.
func (s *Store) recordMoves(recordID string) (Placement, []RecordMove, error) {
	records, err := s.revisions()
	if err != nil {
		return Placement{}, nil, err
	}
	states, err := s.validateGraphState(records, nil)
	if err != nil {
		return Placement{}, nil, err
	}
	state, ok := states[recordID]
	if !ok {
		return Placement{MoveHeads: []string{}}, []RecordMove{}, nil
	}
	moves, err := s.moves()
	if err != nil {
		return Placement{}, nil, err
	}
	mine := []RecordMove{}
	for _, m := range moves {
		if m.RecordID == recordID {
			mine = append(mine, m)
		}
	}
	return state.Placement, mine, nil
}

// RecordScopes lists the scopes a record is in now: one, or each scope its
// conflicting moves name. A record without revisions has none.
func (s *Service) RecordScopes(recordID string) ([]Scope, error) {
	p, _, err := s.store.recordMoves(recordID)
	if err != nil || p.Scope == (Scope{}) && !p.Conflicted() {
		return nil, err
	}
	return p.Scopes(), nil
}

// MoveConflicts counts the records whose moves conflict among those the
// scopes count as conflicts, and how many scope listings they have: each is
// counted in every scope its moves name. That is every recallable record,
// and a visibility conflict; other withheld records are counted as
// withheld instead.
func (s *Store) MoveConflicts() (records, listings int, err error) {
	_, states, err := s.graph()
	if err != nil {
		return 0, 0, err
	}
	for _, state := range states {
		if state.Placement.Conflicted() && (!state.Withheld() || state.State == "visibility-conflict") {
			records++
			listings += len(state.Placement.Conflict)
		}
	}
	return records, listings, nil
}
