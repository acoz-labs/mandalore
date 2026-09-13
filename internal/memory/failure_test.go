package memory

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestJSONPublicationFailurePreservesExistingData(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "record.json")
	if err := writeNewJSON(p, map[string]string{"value": "original"}); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)
	if err := writeNewJSON(p, map[string]string{"value": "replacement"}); err == nil {
		t.Fatal("overwrote immutable data")
	}
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Fatal("failure changed data or leaked temporary files")
	}
	if err := writeNewJSON(filepath.Join(dir, "missing", "record.json"), map[string]string{"value": "new"}); err == nil {
		t.Fatal("missing parent unexpectedly succeeded")
	}
	if err := writeNewJSON(filepath.Join(dir, "large.json"), strings.Repeat("x", 4<<20)); err == nil {
		t.Fatal("writer exceeded reader limit")
	}
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Fatal("failed publication left files")
	}
}

func TestJSONUnknownFieldsTrailingAndOversizeRefused(t *testing.T) {
	for _, raw := range []string{`{"schema_version":1,"id":"device-test","label":"Test","unknown":true}`, `{"schema_version":1,"id":"device-test","label":"Test"} {}`, strings.Repeat(" ", 4<<20) + "{}"} {
		t.Run("invalid", func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "record.json")
			if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			var device Device
			if err := readJSON(p, &device); err == nil {
				t.Fatal("invalid file accepted")
			}
		})
	}
}

func TestInvalidSourcedRevisionPreflightLeavesNoEvidence(t *testing.T) {
	s := fixtureStore(t)
	r := revision("revision-large")
	r.Body = strings.Repeat("x", 4<<20)
	r.Supersedes = []string{}
	source := Source{Version: 1, ID: "source-partial", Kind: "observation", Summary: "Synthetic evidence", DeviceID: r.Authorship.DeviceID, RecordedAt: r.RecordedAt}
	r.Evidence.SourceRefs = []string{source.ID}
	if err := s.PutSourced(r, source); err == nil {
		t.Fatal("oversized revision accepted")
	}
	// This is invalid-input rejection, not a genuine I/O failure. It must be
	// preflighted before publishing the source as well as before the revision.
	if _, err := os.Lstat(filepath.Join(s.Root, "memory/sources", source.ID+".json")); !os.IsNotExist(err) {
		t.Fatal("invalid revision left source evidence", err)
	}
}

func TestPartialIOKeepsEvidenceWithoutDanglingRevision(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denial fixture requires an unprivileged process")
	}
	s := fixtureStore(t)
	first := revision("revision-original")
	if err := s.Put(first); err != nil {
		t.Fatal(err)
	}
	r := revision("revision-io-failure", first.ID)
	source := Source{Version: 1, ID: "source-io-failure", Kind: "observation", Summary: "Synthetic evidence", DeviceID: r.Authorship.DeviceID, RecordedAt: r.RecordedAt}
	r.Evidence.SourceRefs = []string{source.ID}
	dir := filepath.Join(s.Root, "memory/records", r.RecordID)
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0700); err != nil {
			t.Error(err)
		}
	})
	if err := s.PutSourced(r, source); err == nil {
		t.Fatal("expected a revision publication error")
	}
	var saved Source
	if err := s.readSource(source.ID, &saved); err != nil {
		t.Fatal("published evidence disappeared", err)
	}
	h, err := s.History(r.RecordID)
	if err != nil || len(h) != 1 || h[0].ID != first.ID {
		t.Fatal("partial I/O created a dangling revision", h, err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal("orphan evidence should remain valid and inspectable", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("temporary publication artifacts leaked", err)
	}
}
