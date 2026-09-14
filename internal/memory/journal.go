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

type JournalEntry struct {
	Version    int        `json:"schema_version"`
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Summary    string     `json:"summary"`
	RecordedAt string     `json:"recorded_at"`
	Authorship Authorship `json:"authorship"`
}

func validateJournalEntry(entry JournalEntry, device func(string) error) error {
	_, err := time.Parse(time.RFC3339Nano, entry.RecordedAt)
	if err != nil || entry.Version != 1 || !identifier.MatchString(entry.ID) || strings.TrimSpace(entry.Kind) == "" || strings.TrimSpace(entry.Summary) == "" {
		return errors.New("invalid journal entry")
	}
	return validateAuthorship(entry.Authorship, device)
}

func (s *Store) RecordEvent(kind, summary string, author Authorship) (JournalEntry, error) {
	entry := JournalEntry{Version: 1, ID: NewID("event"), Kind: kind, Summary: summary, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: author}
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(summary) == "" {
		return entry, errors.New("event kind and concise summary required")
	}
	if err := s.deviceExists(author.DeviceID); err != nil {
		return entry, err
	}
	if err := s.ValidateAuthorship(author); err != nil {
		return entry, err
	}
	err := s.withLock(func() error {
		at, _ := time.Parse(time.RFC3339Nano, entry.RecordedAt)
		dir := filepath.Join(s.Root, "memory/events", at.Format("2006/01"))
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		return writeNewJSON(filepath.Join(dir, entry.ID+".json"), entry)
	})
	return entry, err
}

// Journal returns recent matching semantic journal entries, never raw harness
// transcripts. Validation covers all entries even when the result is limited.
func (s *Store) Journal(query string, limit int) ([]JournalEntry, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("journal limit must be between 1 and 100")
	}
	if err := s.checkDirectories(); err != nil {
		return nil, err
	}
	entries := []JournalEntry{}
	seen := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(s.Root, "memory/events"), func(path string, info fs.DirEntry, err error) error {
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
		if err := validateJournalEntry(entry, s.deviceExists); err != nil {
			return err
		}
		if seen[entry.ID] {
			return errors.New("duplicate journal entry")
		}
		at, _ := time.Parse(time.RFC3339Nano, entry.RecordedAt)
		if path != filepath.Join(s.Root, "memory/events", at.UTC().Format("2006/01"), entry.ID+".json") {
			return errors.New("journal ID/date/path mismatch")
		}
		seen[entry.ID] = true
		if strings.TrimSpace(query) == "" || relevance(Revision{Summary: entry.Summary, Body: entry.Kind}, query) > 0 {
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, entries[i].RecordedAt)
		b, _ := time.Parse(time.RFC3339Nano, entries[j].RecordedAt)
		if a.Equal(b) {
			return entries[i].ID < entries[j].ID
		}
		return a.After(b)
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
