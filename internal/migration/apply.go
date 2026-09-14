package migration

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

type ApplyInput struct {
	Plan           Plan `json:"plan"`
	WritersStopped bool `json:"writers_stopped"`
}

// The detailed receipt is stored locally, outside the converted Git-backed
// signet. Returning only paths/counts avoids flooding an agent with file hashes.
type Receipt struct {
	Version             int                 `json:"schema_version"`
	Plan                Plan                `json:"plan"`
	PreparedAt          string              `json:"prepared_at"`
	Publication         string              `json:"publication"`
	Preserved           Counts              `json:"preserved"`
	ConversionDevice    memory.Device       `json:"conversion_device"`
	ConversionEvent     memory.JournalEntry `json:"conversion_event"`
	OriginalSHA256      string              `json:"original_sha256"`
	ConvertedSHA256     string              `json:"converted_sha256"`
	OriginalFiles       map[string]string   `json:"original_files"`
	OriginalDirectories []string            `json:"original_directories"`
	ConvertedFiles      map[string]string   `json:"converted_files"`
}

type Result struct {
	SourceID  string `json:"source_id"`
	Output    string `json:"output"`
	Signet    string `json:"signet"`
	Receipt   string `json:"receipt"`
	Staging   string `json:"staging,omitempty"`
	Published bool   `json:"published"`
	Phase     string `json:"phase"`
	Notice    string `json:"notice"`
}

func Apply(ctx context.Context, in ApplyInput) (Result, error) {
	return apply(ctx, in, publishDirectory, syncDirectory)
}

func apply(ctx context.Context, in ApplyInput, publish func(string, string) error, syncDir func(string) error) (result Result, resultErr error) {
	result = Result{SourceID: in.Plan.SourceID, Output: in.Plan.Output, Phase: "preflight", Notice: "No writer activated. Inspect history, enroll a new machine binding, configure private Git sync explicitly, then review a fresh-session handoff. Retain the old source and its Git history for recovery."}
	defer func() {
		if resultErr != nil {
			result.Notice = "Migration incomplete. Inspect publication state and any retained staging/output paths before retrying. Source and published bundles are never deleted or rolled back automatically. No writer was activated."
		}
	}()
	if !in.WritersStopped {
		return result, errors.New("explicit writers_stopped acknowledgement required; stop legacy/new writers on every relevant machine")
	}
	o, err := normalize(in.Plan.Options)
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	lock, err := lockSource(o.Source)
	if err != nil {
		return result, err
	}
	defer lock.close()
	p, s, err := inspect(ctx, o, lock)
	if err != nil {
		return result, err
	}
	if !reflect.DeepEqual(p, in.Plan) {
		return result, errors.New("migration inputs or observations changed after preflight; review a new plan")
	}
	stage, err := os.MkdirTemp(filepath.Dir(o.Output), ".mandalore-migration-")
	if err != nil {
		return result, err
	}
	// Never remove a partial stage: the returned path is recovery evidence.
	result.Staging, result.Phase = stage, "staging"
	now := time.Now().UTC()
	device := memory.Device{Version: 1, ID: memory.NewID("device"), Label: o.DeviceLabel}
	event := memory.JournalEntry{Version: 1, ID: memory.NewID("event"), Kind: "migration", Summary: fmt.Sprintf("Converted My Friday memory-only format 1 to Mandalore signet format 1. Source content SHA-256 %s. Preserved %d revisions, %d sources and %d journal entries; historical authorship unchanged.", s.digest, s.counts.Revisions, s.counts.Sources, s.counts.Journal), RecordedAt: now.Format(time.RFC3339Nano), Authorship: memory.Authorship{DeviceID: device.ID, Actor: o.Actor, Harness: "cli"}}
	devicePath := "provenance/devices/" + device.ID + ".json"
	eventPath := "memory/events/" + now.Format("2006/01") + "/" + event.ID + ".json"
	if _, exists := s.converted[devicePath]; exists {
		return result, errors.New("conversion device ID collision")
	}
	if _, exists := s.converted[eventPath]; exists {
		return result, errors.New("conversion event ID collision")
	}
	s.converted[devicePath], s.converted[eventPath] = marshal(device), marshal(event)
	for _, dir := range []string{"memory/records", "memory/events", "memory/sources", "provenance/devices", "foundlings/registrations"} {
		name := dir + "/.gitkeep"
		if _, exists := s.converted[name]; !exists {
			s.converted[name] = []byte{}
		}
	}
	for _, part := range []struct {
		name  string
		files map[string][]byte
	}{{"original", s.original}, {"signet", s.converted}} {
		for _, dir := range s.directories {
			if part.name == "signet" && strings.HasPrefix(dir, "provenance/changes") {
				continue
			}
			if err := os.MkdirAll(filepath.Join(stage, part.name, filepath.FromSlash(dir)), 0700); err != nil {
				return result, err
			}
		}
		if err := writeFiles(ctx, filepath.Join(stage, part.name), part.files); err != nil {
			return result, err
		}
	}
	store, err := memory.Open(filepath.Join(stage, "signet"))
	if err != nil {
		return result, err
	}
	if err := store.Validate(); err != nil {
		return result, fmt.Errorf("staged signet failed canonical validation: %w", err)
	}
	receipt := Receipt{Version: 1, Plan: p, PreparedAt: event.RecordedAt, Publication: "prepared-and-validated; publication is established only at the plan's output path, not a retained staging path", Preserved: s.counts, ConversionDevice: device, ConversionEvent: event, OriginalSHA256: s.digest, ConvertedSHA256: filesDigest(s.converted), OriginalFiles: fileHashes(s.original), ConvertedFiles: fileHashes(s.converted), OriginalDirectories: s.directories}
	if err := writeFiles(ctx, stage, map[string][]byte{"migration.json": marshal(receipt)}); err != nil {
		return result, err
	}
	if err := syncTree(ctx, stage, syncDir); err != nil {
		return result, err
	}
	result.Phase = "validated"
	// An old lock that was absent cannot prevent a newly started writer. Detect
	// observed content/lock changes and require the explicit stop acknowledgement;
	// do not claim protection from arbitrary writers or other machines.
	again, err := readSnapshot(ctx, o.Source)
	if err != nil {
		return result, err
	}
	if again.digest != s.digest || !sameStrings(again.excluded, s.excluded) {
		return result, errors.New("source changed before publication; retain stage and preflight again")
	}
	if h, err := checkBinding(o, s.decoded); err != nil || h != p.BindingSHA256 {
		return result, errors.New("legacy binding changed before publication")
	}
	if err := lock.check(); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := publish(stage, o.Output); err != nil {
		return result, err
	}
	result.Staging, result.Phase, result.Published = "", "published", true
	result.Output, result.Signet, result.Receipt = o.Output, filepath.Join(o.Output, "signet"), filepath.Join(o.Output, "migration.json")
	if err := syncDir(filepath.Dir(o.Output)); err != nil {
		return result, err
	}
	return result, nil
}

func fileHashes(files map[string][]byte) map[string]string {
	result := map[string]string{}
	for name, b := range files {
		result[name] = hash(b)
	}
	return result
}

func writeFiles(ctx context.Context, root string, files map[string][]byte) error {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		full := filepath.Join(root, filepath.FromSlash(name))
		if !inside(root, full) || name == "." || strings.Contains(name, "\\") {
			return errors.New("invalid staged file path")
		}
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			return err
		}
		f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(files[name])
		syncErr := f.Sync()
		closeErr := f.Close()
		if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
			return err
		}
	}
	return nil
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func syncTree(ctx context.Context, root string, syncDir func(string) error) error {
	dirs := []string{}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := syncDir(dirs[i]); err != nil {
			return err
		}
	}
	return nil
}
