package memory

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// JournalEntry is schema 1 without a scope (under memory/events) or schema 2
// with one (under memory/scopes).
type JournalEntry struct {
	Version    int        `json:"schema_version"`
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Summary    string     `json:"summary"`
	RecordedAt string     `json:"recorded_at"`
	Authorship Authorship `json:"authorship"`
	Scope      *Scope     `json:"scope,omitempty"`
}

func validateJournalEntry(entry JournalEntry, device func(string) error) error {
	_, err := time.Parse(time.RFC3339Nano, entry.RecordedAt)
	scoped := entry.Version == 2 && entry.Scope != nil && topicScope(*entry.Scope)
	if err != nil || !(entry.Version == 1 && entry.Scope == nil || scoped) || !identifier.MatchString(entry.ID) || strings.TrimSpace(entry.Kind) == "" || strings.TrimSpace(entry.Summary) == "" {
		return errors.New("invalid journal entry")
	}
	return validateAuthorship(entry.Authorship, device)
}

func newJournalEntry(kind, summary string, scope *Scope, author Authorship) JournalEntry {
	entry := JournalEntry{Version: 1, ID: NewID("event"), Kind: kind, Summary: summary, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: author}
	if scope != nil {
		selected := *scope
		entry.Version, entry.Scope = 2, &selected
	}
	return entry
}

// publishJournal writes an entry at its canonical path. The caller holds the
// writer lock.
func (s *Store) publishJournal(entry JournalEntry) error {
	rel, err := journalPath(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(s.Root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return writeNewJSON(path, entry)
}

// RecordEvent appends a journal entry, scoped when scope is set.
func (s *Store) RecordEvent(kind, summary string, scope *Scope, author Authorship) (JournalEntry, error) {
	entry := newJournalEntry(kind, summary, scope, author)
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(summary) == "" {
		return entry, errors.New("event kind and concise summary required")
	}
	if err := validateJournalEntry(entry, s.deviceExists); err != nil {
		return entry, err
	}
	err := s.withLock(func() error { return s.publishJournal(entry) })
	return entry, err
}

// Journal returns recent matching semantic journal entries, never raw harness
// transcripts. Validation covers all entries even when the result is limited.
// A nil scope returns unscoped and scoped entries together.
func (s *Store) Journal(query string, scope *Scope, limit int) ([]JournalEntry, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("journal limit must be between 1 and 100")
	}
	entries, err := s.journalEntries(query, scope)
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, err
}

// journalEntries returns every matching entry, newest first.
func (s *Store) journalEntries(query string, scope *Scope) ([]JournalEntry, error) {
	if err := s.checkDirectories(); err != nil {
		return nil, err
	}
	tree, err := s.readScopeTree()
	if err != nil {
		return nil, err
	}
	all := []JournalEntry{}
	err = filepath.WalkDir(filepath.Join(s.Root, "memory/events"), func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in memory journal")
		}
		if info.IsDir() || info.Name() == ".gitkeep" {
			return nil
		}
		var entry JournalEntry
		if err := readJSON(path, &entry); err != nil {
			return err
		}
		if want, err := journalPath(entry); err != nil || entry.Scope != nil || path != filepath.Join(s.Root, want) {
			return errors.New("journal ID/date/path mismatch")
		}
		all = append(all, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}
	all = append(all, tree.journal...)
	entries := []JournalEntry{}
	seen := map[string]bool{}
	for _, entry := range all {
		if err := validateJournalEntry(entry, s.deviceExists); err != nil {
			return nil, err
		}
		if seen[entry.ID] {
			return nil, errors.New("duplicate journal entry")
		}
		seen[entry.ID] = true
		if scope != nil && (entry.Scope == nil || *entry.Scope != *scope) {
			continue
		}
		if strings.TrimSpace(query) == "" || relevance(Revision{Summary: entry.Summary, Body: entry.Kind}, query) > 0 {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return newestFirst(entries[i].RecordedAt, entries[j].RecordedAt, entries[i].ID, entries[j].ID)
	})
	return entries, nil
}
