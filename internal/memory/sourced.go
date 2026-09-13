package memory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PutSourced validates the entire proposed graph before writing either file,
// under the same lock as synchronization. A disk failure between the writes
// can leave unreferenced evidence, but never a revision with missing evidence.
func (s *Store) PutSourced(r Revision, source Source) error {
	return s.withLock(func() error {
		if err := s.validateSource(source); err != nil {
			return err
		}
		if len(r.Evidence.SourceRefs) != 1 || r.Evidence.SourceRefs[0] != source.ID || r.Authorship.DeviceID != source.DeviceID {
			return errors.New("revision and source must share evidence and authorship")
		}
		records, err := s.revisions()
		if err != nil {
			return err
		}
		if err := s.validateGraphWithSources(append(records, r), map[string]Source{source.ID: source}); err != nil {
			return err
		}
		// Size/encoding failures are invalid input, not ambiguous partial I/O.
		// Check both documents before publishing either one.
		if _, err := encodeJSON(r); err != nil {
			return err
		}
		if _, err := encodeJSON(source); err != nil {
			return err
		}
		dir := filepath.Join(s.Root, "memory/records", r.RecordID)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid record directory")
		}
		if err := writeNewJSON(filepath.Join(s.Root, "memory/sources", source.ID+".json"), source); err != nil {
			return err
		}
		return writeNewJSON(filepath.Join(dir, r.ID+".json"), r)
	})
}

func (s *Store) validateRootFiles() error {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "signet.json", ".gitignore", "README.md", ".DS_Store", ".git", ".mandalore", "memory", "provenance", "foundlings":
		default:
			return fmt.Errorf("unexpected signet entry %s; preserve it outside the bank before synchronization", entry.Name())
		}
		if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
			return errors.New("signet entries must be regular files or directories")
		}
	}
	return nil
}

func (s *Store) ValidateAuthorship(author Authorship) error {
	if strings.TrimSpace(author.Actor) == "" || strings.TrimSpace(author.Harness) == "" {
		return errors.New("authorship requires actor and harness")
	}
	return s.deviceExists(author.DeviceID)
}
