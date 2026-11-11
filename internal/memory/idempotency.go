package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// Idempotency intents are machine-local durable state. The signet volume,
// including .mandalore, must survive restarts. Callers namespace keys by the
// authenticated principal; request digests additionally bind local authorship.
type writeIntent struct {
	Digest   string        `json:"digest"`
	Revision *Revision     `json:"revision,omitempty"`
	Source   *Source       `json:"source,omitempty"`
	Journal  *JournalEntry `json:"journal,omitempty"`
	Label    *ScopeLabel   `json:"label,omitempty"`
	Move     *moveIntent   `json:"move,omitempty"`
}

// ErrRequestMismatch means a key was reused with different input or author.
var ErrRequestMismatch = errors.New("idempotency key payload mismatch")

func (s *Service) intent(key, operation string, payload any) (string, string, error) {
	if key == "" || len(key) > 1024 {
		return "", "", errors.New("idempotency key requires 1–1024 bytes")
	}
	// The transport session labels the connection, not the author: a retry
	// from a new connection is the same request. Leaving it out also keeps
	// every digest from before 2.4.0, which had none.
	author := s.author
	author.SessionID = nil
	b, err := json.Marshal([]any{operation, s.ID(), author, payload})
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
			if intent.Digest != digest || intent.Revision == nil || intent.Source == nil || intent.Journal != nil || intent.Label != nil || intent.Move != nil {
				return ErrRequestMismatch
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
			// The intent is already written, so a resumed save never reports
			// invalid input or stale heads; break the error chain.
			if err := s.store.putSourcedLocked(&result, *intent.Source, nil, true); err != nil {
				return fmt.Errorf("resume save: %v", err)
			}
			return nil
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

func (s *Service) AppendJournalIdempotent(key, kind, summary string, scope *Scope) (JournalEntry, error) {
	var entry JournalEntry
	if err := checkJournal(kind, summary, scope); err != nil {
		return entry, err
	}
	// An unscoped entry keeps the 2.1 digest input, so its retries still match.
	var payload any = []string{kind, summary}
	if scope != nil {
		payload = []any{kind, summary, *scope}
	}
	err := s.store.withLock(func() error {
		path, digest, err := s.intent(key, "journal", payload)
		if err != nil {
			return err
		}
		var intent writeIntent
		err = readJSON(path, &intent)
		if err == nil {
			if intent.Digest != digest || intent.Journal == nil || intent.Revision != nil || intent.Source != nil || intent.Label != nil || intent.Move != nil {
				return ErrRequestMismatch
			}
			entry = *intent.Journal
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			entry = newJournalEntry(kind, summary, scope, s.author)
			if err = validateJournalEntry(entry, s.store.deviceExists); err != nil {
				return err
			}
			if err = writeNewJSON(path, writeIntent{Digest: digest, Journal: &entry}); err != nil {
				return err
			}
		}
		rel, err := journalPath(entry)
		if err != nil {
			return err
		}
		return publishOnce(s.store, rel, entry, s.store.publishJournal)
	})
	return entry, err
}

// NameScopeIdempotent publishes at most one label for a principal-namespaced
// key, like RememberIdempotent.
func (s *Service) NameScopeIdempotent(key string, in ScopeLabelWrite) (ScopeLabel, error) {
	var label ScopeLabel
	err := s.store.withLock(func() error {
		path, digest, err := s.intent(key, "scope-label", in)
		if err != nil {
			return err
		}
		var intent writeIntent
		err = readJSON(path, &intent)
		if err == nil {
			if intent.Digest != digest || intent.Label == nil || intent.Revision != nil || intent.Source != nil || intent.Journal != nil || intent.Move != nil {
				return ErrRequestMismatch
			}
			label = *intent.Label
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			existing, err := s.store.ScopeLabels()
			if err != nil {
				return err
			}
			if label, err = s.prepareLabel(in, existing); err != nil {
				return err
			}
			if err = writeNewJSON(path, writeIntent{Digest: digest, Label: &label}); err != nil {
				return err
			}
		}
		return publishOnce(s.store, labelPath(label), label, s.store.publishLabel)
	})
	return label, err
}

// publishOnce publishes an intent's document unless an identical one is
// already at its relative path. The caller holds the writer lock.
func publishOnce[T any](s *Store, rel string, doc T, publish func(T) error) error {
	var existing T
	err := readJSON(filepath.Join(s.Root, rel), &existing)
	if err == nil {
		if !reflect.DeepEqual(existing, doc) {
			return errors.New("idempotency document mismatch")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return publish(doc)
}

// PendingIntents names the machine-local write intents whose publication has
// not completed, and a prepared but inactive format upgrade. It only reads.
func (s *Store) PendingIntents() ([]string, error) {
	pending := []string{}
	if err := s.validateUpgradeState(); errors.Is(err, ErrUpgradePending) {
		pending = append(pending, "format-upgrade")
	} else if err != nil {
		return nil, err
	}
	names, err := filepath.Glob(filepath.Join(s.Root, ".mandalore", "operation-*.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range names {
		var intent writeIntent
		if err := readJSON(path, &intent); err != nil {
			return nil, err
		}
		targets := []string{}
		if intent.Revision != nil {
			targets = append(targets, filepath.Join("memory/records", intent.Revision.RecordID, intent.Revision.ID+".json"))
		}
		if intent.Source != nil {
			targets = append(targets, filepath.Join("memory/sources", intent.Source.ID+".json"))
		}
		if intent.Journal != nil {
			target, err := journalPath(*intent.Journal)
			if err != nil {
				return nil, err
			}
			targets = append(targets, target)
		}
		if intent.Label != nil {
			targets = append(targets, labelPath(*intent.Label))
		}
		if intent.Move != nil {
			// A no-op move publishes nothing, so it is never pending.
			if intent.Move.Move == nil {
				continue
			}
			targets = append(targets, movePath(*intent.Move.Move))
		}
		if len(targets) == 0 {
			pending = append(pending, filepath.Base(path))
		}
		for _, target := range targets {
			if _, err := os.Lstat(filepath.Join(s.Root, target)); errors.Is(err, os.ErrNotExist) {
				pending = append(pending, filepath.Base(path))
				break
			} else if err != nil {
				return nil, err
			}
		}
	}
	return pending, nil
}
