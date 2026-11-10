package memory

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
)

// ScopeLabel names a project, account or task scope: the topic a user calls a
// bounty. Labels are append-only evidence. Every unsuperseded label is a
// current head, so two clones that name the same scope before syncing, or
// rename it concurrently, leave several heads visible until a revision
// supersedes them all.
type ScopeLabel struct {
	Version      int        `json:"schema_version"`
	ID           string     `json:"id"`
	Scope        Scope      `json:"scope"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	RecordedAt   string     `json:"recorded_at"`
	Authorship   Authorship `json:"authorship"`
	Supersedes   []string   `json:"supersedes"`
	ChangeReason string     `json:"change_reason"`
}

// LabelHead is a scope's current name and description.
type LabelHead struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const (
	MaxScopeNameBytes        = 256
	MaxScopeDescriptionBytes = 1024
	maxLabelSupersedes       = 32
)

// ErrStaleLabels means a revision's supersedes no longer matches the scope's
// current label heads (format 2), checked before any write.
var ErrStaleLabels = fmt.Errorf("scope label heads changed; reread them with memory_scopes before retrying: %w", ErrStaleHeads)

// scopesDir holds the files added in 2.2.0: labels and scoped journal entries.
// Readers before 2.2.0 never open it.
const scopesDir = "memory/scopes"

func topicScope(scope Scope) bool {
	switch scope.Kind {
	case "project", "account", "task":
		return identifier.MatchString(scope.ID)
	}
	return false
}

func scopePath(scope Scope) string {
	return filepath.Join(scopesDir, scope.Kind, scope.ID)
}

func labelPath(l ScopeLabel) string {
	return filepath.Join(scopePath(l.Scope), "labels", l.ID+".json")
}

// journalPath is an entry's canonical path: unscoped entries stay under
// memory/events, scoped ones live under their scope.
func journalPath(e JournalEntry) (string, error) {
	at, err := time.Parse(time.RFC3339Nano, e.RecordedAt)
	if err != nil {
		return "", err
	}
	month := at.UTC().Format("2006/01")
	if e.Scope == nil {
		return filepath.Join("memory/events", month, e.ID+".json"), nil
	}
	return filepath.Join(scopePath(*e.Scope), "events", month, e.ID+".json"), nil
}

// oneLine refuses control characters and the Unicode line and paragraph
// separators, which clients may render as line breaks.
func oneLine(value string) bool {
	return !strings.ContainsFunc(value, func(r rune) bool { return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' })
}

func validateLabelGraph(items []ScopeLabel, device func(string) error) error {
	byID := map[string]ScopeLabel{}
	for _, l := range items {
		if l.Version != 1 || !identifier.MatchString(l.ID) || !topicScope(l.Scope) || !textWithin(l.Name, MaxScopeNameBytes) || !oneLine(l.Name) || !textWithin(l.Description, MaxScopeDescriptionBytes) || !textWithin(l.ChangeReason, MaxReasonBytes) || l.Supersedes == nil || len(l.Supersedes) > maxLabelSupersedes {
			return errors.New("invalid scope label")
		}
		if _, err := time.Parse(time.RFC3339Nano, l.RecordedAt); err != nil {
			return errors.New("invalid scope label time")
		}
		if err := validateAuthorship(l.Authorship, device); err != nil {
			return err
		}
		if _, exists := byID[l.ID]; exists {
			return errors.New("duplicate scope label")
		}
		byID[l.ID] = l
	}
	for _, l := range items {
		seen := map[string]bool{}
		for _, id := range l.Supersedes {
			parent, exists := byID[id]
			if !exists || seen[id] || parent.Scope != l.Scope {
				return errors.New("invalid scope label predecessor")
			}
			seen[id] = true
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return errors.New("scope label supersession cycle")
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
		visiting[id], visited[id] = false, true
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

// newestFirst orders by recorded time, newest first, then by ID.
func newestFirst(atI, atJ, idI, idJ string) bool {
	a, _ := time.Parse(time.RFC3339Nano, atI)
	b, _ := time.Parse(time.RFC3339Nano, atJ)
	if a.Equal(b) {
		return idI < idJ
	}
	return a.After(b)
}

// labelHeads returns each named scope's unsuperseded labels, newest first.
func labelHeads(items []ScopeLabel) map[Scope][]ScopeLabel {
	superseded := map[string]bool{}
	for _, l := range items {
		for _, id := range l.Supersedes {
			superseded[id] = true
		}
	}
	heads := map[Scope][]ScopeLabel{}
	for _, l := range items {
		if !superseded[l.ID] {
			heads[l.Scope] = append(heads[l.Scope], l)
		}
	}
	for _, h := range heads {
		sort.Slice(h, func(i, j int) bool { return newestFirst(h[i].RecordedAt, h[j].RecordedAt, h[i].ID, h[j].ID) })
	}
	return heads
}

func headsOf(labels []ScopeLabel) []LabelHead {
	out := []LabelHead{}
	for _, l := range labels {
		out = append(out, LabelHead{ID: l.ID, Name: l.Name, Description: l.Description})
	}
	return out
}

// scopeTree is everything under memory/scopes, strictly read.
type scopeTree struct {
	labels  []ScopeLabel
	journal []JournalEntry
}

// readScopeTree reads memory/scopes. Every file must be a label or a scoped
// journal entry at its canonical path; anything else refuses.
func (s *Store) readScopeTree() (scopeTree, error) {
	tree := scopeTree{labels: []ScopeLabel{}, journal: []JournalEntry{}}
	root := filepath.Join(s.Root, scopesDir)
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return tree, nil
	}
	if err != nil {
		return tree, err
	}
	if !info.IsDir() {
		return tree, errors.New("invalid scopes directory")
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in scope evidence")
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("nonregular scope evidence")
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		switch {
		case len(parts) == 6 && parts[4] == "labels":
			var l ScopeLabel
			if err := readJSON(path, &l); err != nil {
				return err
			}
			if !topicScope(l.Scope) || !identifier.MatchString(l.ID) || rel != labelPath(l) {
				return errors.New("scope label ID/path mismatch")
			}
			tree.labels = append(tree.labels, l)
		case len(parts) == 8 && parts[4] == "events":
			var e JournalEntry
			if err := readJSON(path, &e); err != nil {
				return err
			}
			if e.Scope == nil || !topicScope(*e.Scope) || !identifier.MatchString(e.ID) {
				return errors.New("scoped journal ID/path mismatch")
			}
			if want, err := journalPath(e); err != nil || rel != want {
				return errors.New("scoped journal ID/path mismatch")
			}
			tree.journal = append(tree.journal, e)
		default:
			return errors.New("unexpected file under memory/scopes")
		}
		return nil
	})
	return tree, err
}

// ScopeLabels returns every validated label revision, sorted by ID.
func (s *Store) ScopeLabels() ([]ScopeLabel, error) {
	tree, err := s.topicTree()
	return tree.labels, err
}

// topicTree reads memory/scopes and validates its labels and scoped journal
// entries, without walking the unscoped journal.
func (s *Store) topicTree() (scopeTree, error) {
	if err := s.checkDirectories(); err != nil {
		return scopeTree{}, err
	}
	tree, err := s.readScopeTree()
	if err != nil {
		return scopeTree{}, err
	}
	if err := validateLabelGraph(tree.labels, s.deviceExists); err != nil {
		return scopeTree{}, err
	}
	for _, e := range tree.journal {
		if err := validateJournalEntry(e, s.deviceExists); err != nil {
			return scopeTree{}, err
		}
	}
	sort.Slice(tree.labels, func(i, j int) bool { return tree.labels[i].ID < tree.labels[j].ID })
	return tree, nil
}

// ScopeLabelWrite names a scope or revises its name and description.
type ScopeLabelWrite struct {
	Scope       Scope    `json:"scope" jsonschema:"The project, account or task scope to name. Signet-wide scope can't be named."`
	Name        string   `json:"name" jsonschema:"The topic's display name, one line."`
	Description string   `json:"description" jsonschema:"What the topic is about and what belongs in it."`
	Reason      string   `json:"reason" jsonschema:"Why it is named or renamed."`
	Supersedes  []string `json:"supersedes,omitempty" jsonschema:"To revise a name, the scope's current label IDs (label-...) from memory_scopes; omit only for a scope that has no name yet."`
}

// prepareLabel checks caller input against the scope's labels and builds the
// new revision. The caller holds the writer lock.
func (s *Service) prepareLabel(in ScopeLabelWrite, existing []ScopeLabel) (ScopeLabel, error) {
	var l ScopeLabel
	if !topicScope(in.Scope) {
		return l, invalidInput("scope must be a project, account or task scope with an ID matching ^[a-z][a-z0-9-]{2,127}$")
	}
	if err := checkText("name", in.Name, MaxScopeNameBytes); err != nil {
		return l, err
	}
	if !oneLine(in.Name) {
		return l, invalidInput("name must be one line, without control characters")
	}
	if err := checkText("description", in.Description, MaxScopeDescriptionBytes); err != nil {
		return l, err
	}
	if err := checkText("reason", in.Reason, MaxReasonBytes); err != nil {
		return l, err
	}
	if len(in.Supersedes) > maxLabelSupersedes {
		return l, invalidInput("supersedes has %d IDs; the limit is %d", len(in.Supersedes), maxLabelSupersedes)
	}
	all := map[string]bool{}
	for _, e := range existing {
		if e.Scope == in.Scope {
			all[e.ID] = true
		}
	}
	if len(in.Supersedes) == 0 {
		if len(all) > 0 {
			return l, invalidInput("this scope already has a name; to revise it, set supersedes to its current label IDs from memory_scopes")
		}
	} else if s.store.Signet.Version == 2 {
		heads := []string{}
		for _, h := range labelHeads(existing)[in.Scope] {
			heads = append(heads, h.ID)
		}
		given := slices.Clone(in.Supersedes)
		slices.Sort(given)
		slices.Sort(heads)
		if !slices.Equal(given, heads) {
			return l, ErrStaleLabels
		}
	} else {
		seen := map[string]bool{}
		for _, id := range in.Supersedes {
			if !all[id] || seen[id] {
				return l, invalidInput("supersedes must name distinct label IDs of this scope; read them with memory_scopes")
			}
			seen[id] = true
		}
	}
	supersedes := in.Supersedes
	if supersedes == nil {
		supersedes = []string{}
	}
	l = ScopeLabel{Version: 1, ID: NewID("label"), Scope: in.Scope, Name: in.Name, Description: in.Description, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author, Supersedes: supersedes, ChangeReason: in.Reason}
	if err := validateLabelGraph(append(slices.Clone(existing), l), s.store.deviceExists); err != nil {
		return l, err
	}
	return l, nil
}

// publishLabel writes a prepared label. The caller holds the writer lock.
func (s *Store) publishLabel(l ScopeLabel) error {
	path := filepath.Join(s.Root, labelPath(l))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return writeNewJSON(path, l)
}

// NameScope names a scope or revises its name, without idempotency.
func (s *Service) NameScope(in ScopeLabelWrite) (ScopeLabel, error) {
	var l ScopeLabel
	err := s.store.withLock(func() error {
		existing, err := s.store.ScopeLabels()
		if err != nil {
			return err
		}
		if l, err = s.prepareLabel(in, existing); err != nil {
			return err
		}
		return s.store.publishLabel(l)
	})
	return l, err
}

// CanonicalLabelPath is a label's slash-separated path in a signet.
func CanonicalLabelPath(l ScopeLabel) string { return filepath.ToSlash(labelPath(l)) }

// CanonicalJournalPath is a journal entry's slash-separated path in a signet.
func CanonicalJournalPath(e JournalEntry) (string, error) {
	p, err := journalPath(e)
	return filepath.ToSlash(p), err
}
