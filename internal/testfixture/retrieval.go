// Package testfixture supplies synthetic data to evaluation tests only. No
// production command imports it. Fixture publication is not a supported writer.
package testfixture

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
)

type Corpus struct {
	Records, Depth, ConflictEvery int
}

func (c Corpus) Name() string {
	return fmt.Sprintf("records-%d/depth-%d/conflict-every-%d", c.Records, c.Depth, c.ConflictEvery)
}

var ScaleCorpora = []Corpus{{Records: 100, Depth: 1}, {Records: 1000, Depth: 1}, {Records: 10000, Depth: 1}, {Records: 1000, Depth: 2}, {Records: 1000, Depth: 10}, {Records: 1000, Depth: 2, ConflictEvery: 17}}

type Fixture struct {
	Service            *memory.Service
	Author             memory.Authorship
	Binding            string
	Scope              memory.Scope
	Query, ExpectedID  string
	Records, Revisions int
	Conflicts, Files   int
	Bytes              int64
}

// New writes only a newly created test directory. Direct canonical writes avoid
// timing quadratic public-writer fixture setup; the complete bank is validated
// before returning. Each revision has its own source, like Service.Remember.
func New(t testing.TB, c Corpus) Fixture {
	t.Helper()
	if c.Records < 10 || c.Depth < 1 || c.ConflictEvery < 0 {
		t.Fatal("invalid evaluation corpus")
	}
	base := t.TempDir()
	store, err := memory.Create(filepath.Join(base, "signet"), "Synthetic retrieval evaluation", "device-evaluation", "Synthetic evaluation host")
	if err != nil {
		t.Fatal(err)
	}
	f := Fixture{Records: c.Records, Author: memory.Authorship{DeviceID: "device-evaluation", Actor: "Synthetic evaluator", Harness: "test"}, Scope: memory.Scope{Kind: "project", ID: "project-01"}, Query: "signal-000001", ExpectedID: fmt.Sprintf("revision-000001-%03d", c.Depth-1)}
	const at = "2026-01-01T00:00:00Z"
	for i := range c.Records {
		scope := memory.Scope{Kind: "project", ID: fmt.Sprintf("project-%02d", i%10)}
		if i%10 == 0 {
			scope = memory.Scope{Kind: "signet", ID: store.Signet.ID}
		}
		parent := ""
		for depth := range c.Depth {
			r := memory.Revision{Version: 1, ID: fmt.Sprintf("revision-%06d-%03d", i, depth), RecordID: fmt.Sprintf("record-%06d", i), Kind: "project-state", Scope: scope, Summary: fmt.Sprintf("Project checkpoint %06d", i), Body: fmt.Sprintf("signal-%06d verified synthetic checkpoint. %s", i, strings.Repeat("Stable evidence for a fictional project. ", 5)), Sensitivity: "private", Volatility: "stable", RecordedAt: at, EffectiveFrom: at, Authorship: f.Author, Evidence: memory.Evidence{Basis: "observation", Confidence: "high", SourceRefs: []string{}}, Supersedes: []string{}, ChangeReason: "Synthetic evaluation revision"}
			if depth > 0 {
				r.Supersedes = []string{parent}
			}
			writeRevision(t, store.Root, r)
			f.Revisions++
			parent = r.ID
			if depth == c.Depth-1 && c.ConflictEvery > 0 && i%c.ConflictEvery == 0 {
				// Concurrent siblings of a shared root; depth one gets an
				// additional pair rather than an invalid second root.
				if depth == 0 {
					r.Supersedes = []string{r.ID}
					r.ID += "-left"
					writeRevision(t, store.Root, r)
					f.Revisions++
				}
				r.ID += "-right"
				r.Body = "Alternate synthetic checkpoint; unresolved concurrent evidence."
				writeRevision(t, store.Root, r)
				f.Revisions++
				f.Conflicts++
			}
		}
	}
	if err := store.Validate(); err != nil {
		t.Fatal("invalid evaluation fixture", err)
	}
	f.Service, err = memory.OpenService(store.Root, f.Author)
	if err != nil {
		t.Fatal(err)
	}
	f.Binding = filepath.Join(base, "binding.json")
	writeJSON(t, f.Binding, binding.Binding{Version: 1, SignetID: store.Signet.ID, Root: store.Root, DeviceID: f.Author.DeviceID, Actor: f.Author.Actor})
	if err := filepath.WalkDir(store.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err == nil {
			f.Files++
			f.Bytes += info.Size()
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func writeRevision(t testing.TB, root string, r memory.Revision) {
	t.Helper()
	sourceID := "source-" + r.ID
	r.Evidence.SourceRefs = []string{sourceID}
	writeJSON(t, filepath.Join(root, "memory/sources", sourceID+".json"), memory.Source{Version: 1, ID: sourceID, Kind: "observation", Summary: "Synthetic checkpoint evidence", DeviceID: r.Authorship.DeviceID, RecordedAt: r.RecordedAt})
	writeJSON(t, filepath.Join(root, "memory/records", r.RecordID, r.ID+".json"), r)
}

func writeJSON(t testing.TB, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write(append(data, '\n'))
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
}
