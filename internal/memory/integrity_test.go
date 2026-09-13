package memory

import (
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func snapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReadWithoutLocalStateAndWithoutMutation(t *testing.T) {
	s := fixture(t)
	r, err := s.Remember(Write{Kind: "fact", Summary: "Example", Body: "Remembered evidence", Basis: "observation", Reason: "Observed"})
	if err != nil {
		t.Fatal(err)
	}
	// Simulates the absent ignored directory in a fresh clone; only fixture state.
	if err := os.RemoveAll(filepath.Join(s.Root(), ".mandalore")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, s.Root())
	if _, err := s.Recall("", nil, 5, 4096); err != nil {
		t.Fatal(err)
	}
	if _, err := s.History(r.RecordID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Journal("", 5); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Scopes(); err != nil {
		t.Fatal(err)
	}
	if err := s.store.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, snapshot(t, s.Root())) {
		t.Fatal("read wrote state")
	}
	if _, err := s.AppendJournal("test", "First write after clone"); err != nil {
		t.Fatal(err)
	}
}

func TestCorruptSourceRefusedOnReadAndWrite(t *testing.T) {
	s := fixture(t)
	r, err := s.Remember(Write{Kind: "fact", Summary: "Example", Body: "Evidence", Basis: "observation", Reason: "Observed"})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(s.Root(), "memory/sources", r.Evidence.SourceRefs[0]+".json")
	var source Source
	if err := readJSON(p, &source); err != nil {
		t.Fatal(err)
	}
	source.Version = 99
	b, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, s.Root())
	if _, err := s.Recall("", nil, 5, 4096); err == nil {
		t.Fatal("recalled invalid source")
	}
	if _, err := s.Remember(Write{Kind: "fact", Summary: "Other", Body: "Evidence", Basis: "observation", Reason: "Observed"}); err == nil {
		t.Fatal("wrote over corrupt evidence")
	}
	if !reflect.DeepEqual(before, snapshot(t, s.Root())) {
		t.Fatal("rejected write mutated files")
	}
}

func TestNestedJournalSymlinkDoesNotWriteOutside(t *testing.T) {
	s := fixture(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(s.Root(), "memory/events", time.Now().UTC().Format("2006"))); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendJournal("test", "Do not follow the symlink"); err == nil {
		t.Fatal("accepted a symlink journal tree")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal("outside modified", err)
	}
}

func TestForeignSignetScopeRejected(t *testing.T) {
	s := fixtureStore(t)
	r := revision("revision-foreign")
	r.Scope = Scope{Kind: "signet", ID: "signet-other"}
	if err := s.Put(r); err == nil {
		t.Fatal("accepted foreign signet scope")
	}
}

func TestStoreHandleDetectsReplacement(t *testing.T) {
	s := fixtureStore(t)
	replacement := s.Signet
	replacement.ID = "signet-replacement"
	b, err := json.Marshal(replacement)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "signet.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Scopes(); err == nil {
		t.Fatal("stale store handle accepted")
	}
	if err := s.Put(revision("revision-replaced")); err == nil {
		t.Fatal("stale store handle wrote")
	}
}
