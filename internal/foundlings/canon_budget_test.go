package foundlings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// bulkCanonSource builds a synthetic canon source with at least 200 eligible
// signet/provenance/memory blobs at one commit. Subtests copy it.
func bulkCanonSource(t *testing.T) string {
	t.Helper()
	_, source, _, _, root := canonFixture(t)
	for i := 0; i < 100; i++ {
		if _, e := source.Remember(memory.Write{Kind: "fact", Summary: fmt.Sprintf("Synthetic bulk fact %03d", i), Body: "Synthetic bulk body", Basis: "observation", Reason: "Budget fixture"}); e != nil {
			t.Fatal(e)
		}
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "Bulk")
	eligible := 0
	for _, name := range strings.Split(strings.TrimSpace(gitTest(t, root, "ls-tree", "-r", "--name-only", "HEAD")), "\n") {
		if name == "signet.json" || strings.HasPrefix(name, "memory/") || strings.HasPrefix(name, "provenance/") {
			eligible++
		}
	}
	if eligible < 200 {
		t.Fatalf("fixture has %d eligible blobs", eligible)
	}
	return root
}

// bulkCanonFixture registers a private copy of the bulk source in a new primary.
func bulkCanonFixture(t *testing.T, bulk string) (*Manager, memory.FoundlingRegistration) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "source")
	if e := os.CopyFS(root, os.DirFS(bulk)); e != nil {
		t.Fatal(e)
	}
	source, e := memory.OpenReadOnlyService(root)
	if e != nil {
		t.Fatal(e)
	}
	created, e := memory.Create(filepath.Join(t.TempDir(), "primary"), "primary", "device-test", "Synthetic test")
	if e != nil {
		t.Fatal(e)
	}
	primary, e := memory.OpenService(created.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if e != nil {
		t.Fatal(e)
	}
	pin := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	reg, e := primary.WriteFoundling(memory.FoundlingWrite{Name: "Bulk canon fixture", Description: "Synthetic external signet", Mode: "canon", Branch: "main", SourceSignetID: source.ID(), Source: memory.FoundlingSource{Kind: "git", Locator: "https://github.com/example/synthetic-bulk.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: pin}, State: "active", Reason: "Explicit registration"})
	if e != nil {
		t.Fatal(e)
	}
	m := New(primary)
	m.canonTransport = func(string) string { return root }
	return m, reg
}

// delayGit puts a Git wrapper first in PATH. It sleeps before the transport
// fetch or the content batch, and logs each object-read process it starts.
func delayGit(t *testing.T, fetch, batch time.Duration) string {
	t.Helper()
	realGit, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	log := filepath.Join(bin, "reads.log")
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
	script := fmt.Sprintf("#!/bin/sh\nfor arg do\ncase \"$arg\" in\nfetch) sleep %.3f; break;;\n--batch) echo batch >> %s; sleep %.3f; break;;\n--batch-check) echo check >> %s; break;;\nblob) echo blob >> %s; break;;\nesac\ndone\nexec %s \"$@\"\n",
		fetch.Seconds(), quote(log), batch.Seconds(), quote(log), quote(log), quote(realGit))
	if e = os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

type canonRefresher func(*Manager, string, memory.FoundlingRegistration) ([]CanonReceipt, error)

var canonRefreshPaths = map[string]canonRefresher{
	"native": func(m *Manager, session string, _ memory.FoundlingRegistration) ([]CanonReceipt, error) {
		return m.SessionRefresh(context.Background(), session)
	},
	"allowed": func(m *Manager, session string, reg memory.FoundlingRegistration) ([]CanonReceipt, error) {
		return m.SessionRefreshAllowed(context.Background(), session, []string{reg.FoundlingID})
	},
}

func TestCanonBulkFirstRefreshBatchesReadsWithinBudget(t *testing.T) {
	bulk := bulkCanonSource(t)
	for name, refresh := range canonRefreshPaths {
		t.Run(name, func(t *testing.T) {
			m, reg := bulkCanonFixture(t, bulk)
			log := delayGit(t, 500*time.Millisecond, 0)
			start := time.Now()
			receipts, e := refresh(m, "bulk", reg)
			elapsed := time.Since(start)
			if e != nil || len(receipts) != 1 || receipts[0].State != "available" || receipts[0].TimeoutPhase != "" {
				t.Fatal(receipts, e)
			}
			if elapsed >= 2*time.Second {
				t.Fatalf("refresh exceeded fetch budget: %s", elapsed)
			}
			reads, e := os.ReadFile(log)
			if e != nil || string(reads) != "check\nbatch\n" {
				t.Fatalf("object reads were not batched: %q %v", reads, e)
			}
			got := recallTest(t, m, reg.FoundlingID, "bulk")
			if got.Reference.State != "available" || len(got.Memory.Current) == 0 {
				t.Fatal(got)
			}
		})
	}
}

func TestCanonBulkBudgetExceededReportsPhase(t *testing.T) {
	bulk := bulkCanonSource(t)
	for _, phase := range []struct {
		name         string
		fetch, batch time.Duration
	}{{"fetch", 2500 * time.Millisecond, 0}, {"materialize", 0, 2500 * time.Millisecond}} {
		for name, refresh := range canonRefreshPaths {
			t.Run(phase.name+"/"+name+"/uncached", func(t *testing.T) {
				m, reg := bulkCanonFixture(t, bulk)
				delayGit(t, phase.fetch, phase.batch)
				start := time.Now()
				receipts, e := refresh(m, "uncached", reg)
				elapsed := time.Since(start)
				if e != nil || len(receipts) != 1 || receipts[0].State != "unavailable" || receipts[0].Reason != "timeout" || receipts[0].TimeoutPhase != phase.name {
					t.Fatal(receipts, e)
				}
				if elapsed < 1800*time.Millisecond || elapsed >= 3*time.Second {
					t.Fatalf("refresh outside shared budget: %s", elapsed)
				}
				if encoded, _ := json.Marshal(receipts); strings.Contains(string(encoded), "Synthetic bulk") {
					t.Fatal("timeout receipt exposed source content")
				}
			})
			t.Run(phase.name+"/"+name+"/stale", func(t *testing.T) {
				m, reg := bulkCanonFixture(t, bulk)
				first := refreshTest(t, m, "seed")
				delayGit(t, phase.fetch, phase.batch)
				start := time.Now()
				receipts, e := refresh(m, "stale", reg)
				elapsed := time.Since(start)
				if e != nil || len(receipts) != 1 || receipts[0].State != "stale" || receipts[0].Reason != "timeout" || receipts[0].TimeoutPhase != phase.name || receipts[0].Pin != first.Pin || receipts[0].Digest != first.Digest {
					t.Fatal(receipts, e)
				}
				if elapsed >= 3*time.Second {
					t.Fatalf("stale validation outside shared budget: %s", elapsed)
				}
				if got := recallTest(t, m, reg.FoundlingID, "stale"); got.Reference.State != "stale" || len(got.Memory.Current) == 0 {
					t.Fatal(got)
				}
			})
		}
	}
}

func TestCanonBlobsSplitsBatchesAtOutputCapAndRejectsMissingObjects(t *testing.T) {
	stage := t.TempDir()
	gitTest(t, stage, "init", "--quiet")
	files := []treeFile{}
	want := map[string][]byte{}
	for i := 0; i < 3; i++ {
		data := []byte(strings.Repeat(string(rune('a'+i)), 3<<20))
		name := fmt.Sprintf("memory/large-%d", i)
		if e := os.WriteFile(filepath.Join(stage, "blob"), data, 0600); e != nil {
			t.Fatal(e)
		}
		files = append(files, treeFile{name: name, object: strings.TrimSpace(gitTest(t, stage, "hash-object", "-w", "blob"))})
		want[name] = data
	}
	log := delayGit(t, 0, 0)
	docs, e := canonBlobs(context.Background(), stage, "git-sha1", files)
	if e != nil || len(docs) != len(want) {
		t.Fatal(len(docs), e)
	}
	for name, data := range want {
		if string(docs[name]) != string(data) {
			t.Fatalf("%s content differs", name)
		}
	}
	// 9 MiB of content exceeds one 8 MiB output cap, so exactly two batches run.
	if reads, e := os.ReadFile(log); e != nil || string(reads) != "check\nbatch\nbatch\n" {
		t.Fatalf("unexpected object reads: %q %v", reads, e)
	}
	missing := append(files[:1:1], treeFile{name: "memory/missing", object: strings.Repeat("0", 40)})
	if _, e = canonBlobs(context.Background(), stage, "git-sha1", missing); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
}
