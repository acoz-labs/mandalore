package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

// Idempotency intents are machine-local durable state. The signet volume,
// including .mandalore, must survive restarts. Callers namespace keys by the
// authenticated principal; request digests additionally bind local authorship.
type writeIntent struct {
	Digest   string        `json:"digest"`
	Revision *Revision     `json:"revision,omitempty"`
	Source   *Source       `json:"source,omitempty"`
	Journal  *JournalEntry `json:"journal,omitempty"`
}

func (s *Service) intent(key, operation string, payload any) (string, string, error) {
	if key == "" || len(key) > 1024 {
		return "", "", errors.New("idempotency key requires 1–1024 bytes")
	}
	b, err := json.Marshal([]any{operation, s.ID(), s.author, payload})
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(b)
	name := sha256.Sum256([]byte(key))
	// The existing store lock validates .mandalore before this is called.
	return filepath.Join(s.Root(), ".mandalore", "operation-"+hex.EncodeToString(name[:])+".json"), hex.EncodeToString(digest[:]), nil
}

// RememberIdempotent publishes at most one revision for a principal-namespaced
// key. An intent precedes publication, closing the publication/receipt crash gap.
func (s *Service) RememberIdempotent(key string, input Write) (Revision, error) {
	var result Revision
	err := s.store.withLock(func() error {
		path, digest, err := s.intent(key, "remember", input)
		if err != nil {
			return err
		}
		var intent writeIntent
		err = readJSON(path, &intent)
		if err == nil {
			if intent.Digest != digest || intent.Revision == nil || intent.Source == nil || intent.Journal != nil {
				return errors.New("idempotency key payload mismatch")
			}
			result = *intent.Revision
			var existing Revision
			p := filepath.Join(s.Root(), "memory/records", result.RecordID, result.ID+".json")
			err = readJSON(p, &existing)
			if err == nil {
				if !reflect.DeepEqual(existing, result) {
					return errors.New("idempotency revision mismatch")
				}
				var source Source
				if err := readJSON(filepath.Join(s.Root(), "memory/sources", intent.Source.ID+".json"), &source); err != nil {
					return err
				}
				if !reflect.DeepEqual(source, *intent.Source) {
					return errors.New("idempotency source mismatch")
				}
				return nil
			}
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return s.store.putSourcedLocked(&result, *intent.Source, nil, true)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		var source Source
		result, source, err = s.prepareRemember(input)
		if err != nil {
			return err
		}
		// putSourced prepares visibility metadata and validates the graph before
		// persisting the intent, but invokes this callback before publishing content.
		err = s.store.putSourcedLocked(&result, source, func() error {
			return writeNewJSON(path, writeIntent{Digest: digest, Revision: &result, Source: &source})
		}, false)
		return err
	})
	return result, err
}

func (s *Service) AppendJournalIdempotent(key, kind, summary string) (JournalEntry, error) {
	var entry JournalEntry
	if !textWithin(kind, 64) || !textWithin(summary, 4096) {
		return entry, errors.New("journal requires kind (1–64 bytes) and summary (1–4096 bytes)")
	}
	err := s.store.withLock(func() error {
		path, digest, err := s.intent(key, "journal", []string{kind, summary})
		if err != nil {
			return err
		}
		var intent writeIntent
		err = readJSON(path, &intent)
		if err == nil {
			if intent.Digest != digest || intent.Journal == nil || intent.Revision != nil || intent.Source != nil {
				return errors.New("idempotency key payload mismatch")
			}
			entry = *intent.Journal
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			entry = JournalEntry{Version: 1, ID: NewID("event"), Kind: kind, Summary: summary, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author}
			if err = validateJournalEntry(entry, s.store.deviceExists); err != nil {
				return err
			}
			if err = writeNewJSON(path, writeIntent{Digest: digest, Journal: &entry}); err != nil {
				return err
			}
		}
		at, err := time.Parse(time.RFC3339Nano, entry.RecordedAt)
		if err != nil {
			return err
		}
		dir := filepath.Join(s.Root(), "memory/events", at.UTC().Format("2006/01"))
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		dest := filepath.Join(dir, entry.ID+".json")
		var existing JournalEntry
		err = readJSON(dest, &existing)
		if err == nil {
			if !reflect.DeepEqual(existing, entry) {
				return errors.New("idempotency journal mismatch")
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return writeNewJSON(dest, entry)
	})
	return entry, err
}
