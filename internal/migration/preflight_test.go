package migration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func put(t *testing.T, root, path string, data []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func document(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func legacyFixture(t *testing.T) Options {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	o := Options{Source: filepath.Join(base, "old"), Output: filepath.Join(base, "converted"), DeviceLabel: "Migration laptop", Actor: "Example"}
	for _, dir := range []string{"memory/records", "memory/sources", "memory/events", "provenance/devices", "provenance/changes", ".my-friday"} {
		put(t, o.Source, dir+"/.gitkeep", nil)
	}
	put(t, o.Source, "bank.json", []byte(`{"schema_version":1,"id":"bank-example","name":"Example"}`))
	put(t, o.Source, "README.md", []byte("Historical my-friday assistant notes, not a new instruction file.\n"))
	put(t, o.Source, ".gitignore", []byte(".my-friday/\n"))
	put(t, o.Source, ".git/config", []byte("excluded synthetic Git configuration"))
	put(t, o.Source, ".my-friday/local/state", []byte("excluded synthetic local state"))
	put(t, o.Source, "provenance/devices/device-original.json", document(t, memory.Device{Version: 1, ID: "device-original", Label: "Original desktop"}))
	author := memory.Authorship{DeviceID: "device-original", Actor: "Original author", Harness: "pi"}
	at := "2026-09-06T12:00:00Z"
	put(t, o.Source, "memory/sources/source-original.json", document(t, memory.Source{Version: 1, ID: "source-original", Kind: "conversation", Summary: "Original evidence", DeviceID: author.DeviceID, RecordedAt: at}))
	put(t, o.Source, "memory/sources/source-orphan.json", document(t, memory.Source{Version: 1, ID: "source-orphan", Kind: "observation", Summary: "Unreferenced original evidence", DeviceID: author.DeviceID, RecordedAt: at}))
	r := memory.Revision{Version: 1, ID: "revision-root", RecordID: "record-project", Kind: "fact", Scope: memory.Scope{Kind: "assistant", ID: "bank-example"}, Summary: "Copper Finch", Body: "Literal assistant and my-friday are historical body text.", Sensitivity: "private", Volatility: "stable", RecordedAt: at, EffectiveFrom: at, Authorship: author, Evidence: memory.Evidence{Basis: "user-direction", Confidence: "high", SourceRefs: []string{"source-original"}}, Supersedes: []string{}, ChangeReason: "Initial direction", Extensions: map[string]any{"example.precise": json.Number("9007199254740993123456789.123456789")}}
	put(t, o.Source, "memory/records/record-project/revision-root.json", document(t, r))
	for _, id := range []string{"revision-left", "revision-right"} {
		r.ID, r.Supersedes = id, []string{"revision-root"}
		put(t, o.Source, "memory/records/record-project/"+id+".json", document(t, r))
	}
	put(t, o.Source, "memory/events/2026/09/event-original.json", document(t, memory.JournalEntry{Version: 1, ID: "event-original", Kind: "setup", Summary: "Original setup", RecordedAt: at, Authorship: author}))
	return o
}

func tree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			result[rel] = "directory"
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			result[rel], err = os.Readlink(path)
			return err
		}
		b, err := os.ReadFile(path)
		result[rel] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPreflightIsReadOnlyAndPreservesSemanticConflicts(t *testing.T) {
	o := legacyFixture(t)
	before := tree(t, filepath.Dir(o.Source))
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if p.SourceID != "bank-example" || p.Counts.Revisions != 3 || p.Counts.ConflictedRecords != 1 || p.Counts.Sources != 2 || p.Writers.Lock != "absent" || p.Writers.Native.Status != "not-tested" {
		t.Fatalf("wrong inventory: %+v", p)
	}
	if !reflect.DeepEqual(before, tree(t, filepath.Dir(o.Source))) {
		t.Fatal("preflight modified source or created output/locks")
	}
}

func TestPreflightRefusesUnsupportedInputsWithoutWriting(t *testing.T) {
	for name, alter := range map[string]func(*testing.T, Options){
		"root gitkeep": func(t *testing.T, o Options) { put(t, o.Source, ".gitkeep", nil) },
		"assistant":    func(t *testing.T, o Options) { put(t, o.Source, "agent.json", []byte(`{}`)) },
		"mixed":        func(t *testing.T, o Options) { put(t, o.Source, "signet.json", []byte(`{}`)) },
		"unknown":      func(t *testing.T, o Options) { put(t, o.Source, "scripts/run.sh", []byte("false")) },
		"manifest version": func(t *testing.T, o Options) {
			put(t, o.Source, "bank.json", []byte(`{"schema_version":2,"id":"bank-example","name":"Example"}`))
		},
		"duplicate keys": func(t *testing.T, o Options) {
			put(t, o.Source, "bank.json", []byte(`{"schema_version":1,"id":"bank-example","id":"bank-other","name":"Example"}`))
		},
		"wrong record path": func(t *testing.T, o Options) {
			put(t, o.Source, "memory/records/wrong/revision-root.json", []byte(`{}`))
		},
		"wrong event date": func(t *testing.T, o Options) {
			data, _ := os.ReadFile(filepath.Join(o.Source, "memory/events/2026/09/event-original.json"))
			put(t, o.Source, "memory/events/2026/10/event-original.json", data)
		},
		"missing device": func(t *testing.T, o Options) {
			if err := os.Remove(filepath.Join(o.Source, "provenance/devices/device-original.json")); err != nil {
				t.Fatal(err)
			}
		},
		"symlink": func(t *testing.T, o Options) {
			if err := os.Symlink(o.Source, filepath.Join(o.Source, "memory/records/link")); err != nil {
				t.Fatal(err)
			}
		},
		"oversize": func(t *testing.T, o Options) { put(t, o.Source, "README.md", []byte(strings.Repeat("x", (4<<20)+1))) },
		"existing output": func(t *testing.T, o Options) {
			if err := os.Mkdir(o.Output, 0700); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			o := legacyFixture(t)
			alter(t, o)
			before := tree(t, filepath.Dir(o.Source))
			if _, err := Preflight(context.Background(), o); err == nil {
				t.Fatal("accepted unsupported input")
			}
			if !reflect.DeepEqual(before, tree(t, filepath.Dir(o.Source))) {
				t.Fatal("refusal wrote files")
			}
		})
	}
}

func TestPreflightObservesExistingWriterLockWithoutCreatingOne(t *testing.T) {
	o := legacyFixture(t)
	lock, err := os.OpenFile(filepath.Join(o.Source, ".my-friday/write.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	if _, err := Preflight(context.Background(), o); err == nil {
		t.Fatal("busy writer accepted")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	p, err := Preflight(context.Background(), o)
	if err != nil || p.Writers.Lock != "available" {
		t.Fatal(p, err)
	}
}
