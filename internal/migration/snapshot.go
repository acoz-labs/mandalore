package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

var yearPattern = regexp.MustCompile(`^[0-9]{4}$`)
var monthPattern = regexp.MustCompile(`^(0[1-9]|1[0-2])$`)

func portableDirectory(name string) bool {
	switch name {
	case ".", "memory", "memory/records", "memory/sources", "memory/events", "provenance", "provenance/devices", "provenance/changes":
		return true
	}
	p := strings.Split(name, "/")
	if len(p) == 3 && p[0] == "memory" && p[1] == "records" {
		return idPattern.MatchString(p[2])
	}
	return len(p) >= 3 && len(p) <= 4 && p[0] == "memory" && p[1] == "events" && yearPattern.MatchString(p[2]) && (len(p) == 3 || monthPattern.MatchString(p[3]))
}

func readSnapshot(ctx context.Context, root string) (*snapshot, error) {
	s := &snapshot{original: map[string][]byte{}, converted: map[string][]byte{}, excluded: []string{}}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("invalid source root")
	}
	for _, marker := range []string{"agent.json", "signet.json"} {
		if _, err := os.Lstat(filepath.Join(root, marker)); !os.IsNotExist(err) {
			return nil, errors.New("assistant or mixed format is not a memory-only bank; use foundlings for historical reference material")
		}
	}
	dirs := map[string]bool{}
	entries := 0
	err = walkPortable(root, func(full string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		entries++
		if entries > MaxFiles*3 {
			return errors.New("source tree exceeds bounded entry count")
		}
		if len(rel) > 1024 || strings.IndexFunc(rel, unicode.IsControl) >= 0 {
			return errors.New("unsupported snapshot path length or control character")
		}
		if d.Type()&os.ModeSymlink != 0 || (!d.IsDir() && !d.Type().IsRegular()) {
			return fmt.Errorf("nonregular snapshot entry: %s", rel)
		}
		if rel == ".git" || rel == ".my-friday" || path.Base(rel) == ".DS_Store" {
			if len(s.excluded) >= 64 {
				return errors.New("exclusion inventory exceeds 64 entries")
			}
			s.excluded = append(s.excluded, rel)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if !portableDirectory(rel) {
				return fmt.Errorf("unsupported source directory: %s", rel)
			}
			dirs[rel] = true
			if len(dirs) > MaxFiles {
				return errors.New("too many source directories")
			}
			return nil
		}
		if len(s.original) >= MaxFiles {
			return errors.New("portable snapshot exceeds 10000 files")
		}
		b, err := readFile(full, MaxFileBytes)
		if err != nil {
			return fmt.Errorf("snapshot file %s: %w", rel, err)
		}
		s.counts.Bytes += len(b)
		if s.counts.Bytes > MaxSnapshotBytes {
			return errors.New("portable snapshot exceeds 64 MiB")
		}
		s.original[rel] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, dir := range []string{"memory/records", "memory/sources", "memory/events", "provenance/devices", "provenance/changes"} {
		if !dirs[dir] {
			return nil, fmt.Errorf("missing portable directory: %s", dir)
		}
	}
	if err := strictjson.Decode(s.original["bank.json"], &s.decoded.Signet, MaxFileBytes); err != nil || s.decoded.Signet.Version != 1 || !strings.HasPrefix(s.decoded.Signet.ID, "bank-") {
		return nil, errors.New("unsupported or invalid bank.json format; only memory-only version 1 is supported")
	}
	s.converted["signet.json"] = s.original["bank.json"]
	names := make([]string, 0, len(s.original))
	for name := range s.original {
		names = append(names, name)
	}
	sort.Strings(names)
	changes := map[string][]byte{}
	for _, name := range names {
		b := s.original[name]
		if name == "bank.json" || name == "README.md" || name == ".gitignore" {
			continue
		}
		if name != ".gitkeep" && path.Base(name) == ".gitkeep" && portableDirectory(path.Dir(name)) {
			if !strings.HasPrefix(name, "provenance/changes/") {
				s.converted[name] = b
			}
			continue
		}
		switch {
		case strings.HasPrefix(name, "memory/records/"):
			var r memory.Revision
			if err := strictjson.Decode(b, &r, MaxFileBytes); err != nil {
				return nil, fmt.Errorf("invalid revision JSON: %s", name)
			}
			if !idPattern.MatchString(r.ID) || !idPattern.MatchString(r.RecordID) || name != path.Join("memory/records", r.RecordID, r.ID+".json") {
				return nil, errors.New("revision ID/path mismatch")
			}
			if r.Scope.Kind == "assistant" {
				if r.Scope.ID != s.decoded.Signet.ID {
					return nil, errors.New("legacy bank-wide scope ID mismatch")
				}
				var raw map[string]json.RawMessage
				if err := json.Unmarshal(b, &raw); err != nil {
					return nil, err
				}
				var scope map[string]json.RawMessage
				if err := json.Unmarshal(raw["scope"], &scope); err != nil {
					return nil, err
				}
				scope["kind"] = json.RawMessage(`"signet"`)
				raw["scope"] = marshal(scope)
				b = marshal(raw)
				r.Scope.Kind = "signet"
			} else if r.Scope.Kind != "project" && r.Scope.Kind != "account" && r.Scope.Kind != "task" {
				return nil, errors.New("unsupported legacy scope kind")
			}
			s.decoded.Revisions = append(s.decoded.Revisions, r)
		case strings.HasPrefix(name, "memory/sources/"):
			var source struct {
				Version    int    `json:"schema_version"`
				ID         string `json:"id"`
				Kind       string `json:"kind"`
				Summary    string `json:"summary"`
				DeviceID   string `json:"device_id"`
				RecordedAt string `json:"recorded_at"`
			}
			if err := strictjson.Decode(b, &source, MaxFileBytes); err != nil {
				return nil, fmt.Errorf("invalid legacy source: %s", name)
			}
			if !idPattern.MatchString(source.ID) || name != "memory/sources/"+source.ID+".json" {
				return nil, errors.New("source ID/path mismatch")
			}
			s.decoded.Sources = append(s.decoded.Sources, memory.Source{Version: source.Version, ID: source.ID, Kind: source.Kind, Summary: source.Summary, DeviceID: source.DeviceID, RecordedAt: source.RecordedAt})
		case strings.HasPrefix(name, "provenance/devices/"):
			var d memory.Device
			if err := strictjson.Decode(b, &d, MaxFileBytes); err != nil {
				return nil, fmt.Errorf("invalid device JSON: %s", name)
			}
			if !idPattern.MatchString(d.ID) || name != "provenance/devices/"+d.ID+".json" {
				return nil, errors.New("device ID/path mismatch")
			}
			s.decoded.Devices = append(s.decoded.Devices, d)
		case strings.HasPrefix(name, "memory/events/"):
			var e memory.JournalEntry
			if err := strictjson.Decode(b, &e, MaxFileBytes); err != nil {
				return nil, fmt.Errorf("invalid journal JSON: %s", name)
			}
			at, err := time.Parse(time.RFC3339Nano, e.RecordedAt)
			if err != nil || !idPattern.MatchString(e.ID) || name != path.Join("memory/events", at.UTC().Format("2006/01"), e.ID+".json") {
				return nil, errors.New("journal ID/date/path mismatch")
			}
			s.decoded.Journal = append(s.decoded.Journal, e)
		case strings.HasPrefix(name, "provenance/changes/"):
			changes[name] = b
			continue
		default:
			return nil, fmt.Errorf("unsupported portable file: %s", name)
		}
		s.converted[name] = b
		if len(b) > MaxFileBytes {
			return nil, fmt.Errorf("converted file exceeds 4 MiB: %s", name)
		}
	}
	if err := memory.ValidateSnapshot(s.decoded); err != nil {
		return nil, errors.New("converted snapshot violates current memory invariants; inspect device/source identity, journal authorship and revision schema/supersession without rewriting history")
	}
	for name, b := range changes {
		if err := validateChange(name, b, s.decoded.Devices); err != nil {
			return nil, err
		}
	}
	s.counts.Files, s.counts.Devices, s.counts.Sources, s.counts.Revisions, s.counts.Journal, s.counts.SourceChanges = len(s.original), len(s.decoded.Devices), len(s.decoded.Sources), len(s.decoded.Revisions), len(s.decoded.Journal), len(changes)
	parents, heads := map[string]bool{}, map[string]int{}
	for _, r := range s.decoded.Revisions {
		for _, id := range r.Supersedes {
			parents[id] = true
		}
		heads[r.RecordID] = 0
	}
	for _, r := range s.decoded.Revisions {
		if !parents[r.ID] {
			heads[r.RecordID]++
		}
	}
	s.counts.Records = len(heads)
	for _, n := range heads {
		if n > 1 {
			s.counts.ConflictedRecords++
		}
	}
	sort.Strings(s.excluded)
	for dir := range dirs {
		s.directories = append(s.directories, dir)
	}
	sort.Strings(s.directories)
	s.counts.Directories = len(s.directories)
	s.digest = hash(marshal(struct {
		Files       string   `json:"files_sha256"`
		Directories []string `json:"directories"`
	}{filesDigest(s.original), s.directories}))
	s.converted[".gitignore"] = []byte(".mandalore/\n.DS_Store\n")
	current, err := os.Lstat(root)
	if err != nil || !os.SameFile(rootInfo, current) {
		return nil, errors.New("source root changed during snapshot")
	}
	return s, nil
}
