package signetsync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func remoteFixture(t *testing.T) (*Synchronizer, *memory.Service, *Synchronizer, *memory.Service, string) {
	t.Helper()
	a, first := fixture(t)
	if _, err := a.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, first.Root(), "init", "--bare", "--initial-branch=main", remote)
	gitTest(t, first.Root(), "remote", "add", "origin", remote)
	if out, err := a.Sync(context.Background(), 10*time.Second); err != nil || !out.Delivered {
		t.Fatal(out, err)
	}
	clone := filepath.Join(t.TempDir(), "second")
	gitTest(t, first.Root(), "clone", remote, clone)
	store, err := memory.Open(clone)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddDevice(memory.Device{Version: 1, ID: "device-second", Label: "Second"}); err != nil {
		t.Fatal(err)
	}
	second, err := memory.OpenService(clone, memory.Authorship{DeviceID: "device-second", Actor: "Example", Harness: "other-test"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(clone, first.ID())
	if err != nil {
		t.Fatal(err)
	}
	return a, first, b, second, remote
}

func TestTwoClonesReconcileAndPreserveOrigin(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	for _, service := range []*memory.Service{first, second} {
		if _, err := service.AppendJournal("session", "Synthetic device work completed"); err != nil {
			t.Fatal(err)
		}
	}
	registration := memory.FoundlingRegistration{Version: 1, ID: "registration-initial", FoundlingID: "foundling-history", Name: "Historical notes", Description: "Reference evidence", Source: memory.FoundlingSource{Kind: "git", Locator: "https://example.invalid/team/history.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("a", 40)}, State: "active", RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: memory.Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"}, Supersedes: []string{}, ChangeReason: "Explicit reference"}
	if err := a.store.PutFoundlingRegistration(registration); err != nil {
		t.Fatal(err)
	}
	r := remember(t, first, "Copper Finch")
	remember(t, second, "Other device evidence")
	for _, s := range []*Synchronizer{a, b, a} {
		out, err := s.Sync(context.Background(), 10*time.Second)
		if err != nil || out.State != "synchronized" || !out.Delivered || out.Head != out.RemoteHead {
			t.Fatal(out, err)
		}
	}
	// Both services were opened before synchronization: no reopen/cache reset.
	p, err := first.Recall("", nil, 5, 8192)
	if err != nil || len(p.Current) != 2 {
		t.Fatal(p, err)
	}
	journal, err := first.Journal("", 10)
	if err != nil || len(journal) != 2 || journal[0].Authorship.DeviceID == journal[1].Authorship.DeviceID {
		t.Fatal("journal provenance lost", journal, err)
	}
	registrations, err := b.store.FoundlingRegistrations()
	if err != nil || len(registrations) != 1 || registrations[0].ID != registration.ID || registrations[0].Authorship.DeviceID != "device-test" {
		t.Fatal("reference provenance lost", registrations, err)
	}
	revised, err := second.Remember(memory.Write{Kind: "fact", Summary: "Project", Body: "Silver Heron", Basis: "user-direction", Reason: "Renamed", RecordID: r.RecordID, Supersedes: []string{r.ID}})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Synchronizer{b, a} {
		if _, err := s.Sync(context.Background(), 10*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	history, err := first.History(r.RecordID)
	if err != nil || len(history) != 2 || history[0].Authorship.DeviceID != "device-test" || history[1].ID != revised.ID || history[1].Authorship.DeviceID != "device-second" {
		t.Fatal(history, err)
	}
}

func TestConcurrentSemanticHeadsRemainConflictedButDelivered(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	scope := memory.Scope{Kind: "project", ID: "project-example"}
	r, err := first.Remember(memory.Write{Kind: "fact", Summary: "Project", Body: "Copper Finch", Basis: "user-direction", Reason: "Confirmed", Scope: &scope})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Synchronizer{a, b} {
		if _, err := s.Sync(context.Background(), 10*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	for i, service := range []*memory.Service{first, second} {
		if _, err := service.Remember(memory.Write{Kind: "fact", Summary: "Project", Body: []string{"Silver Heron", "Amber Lark"}[i], Basis: "user-direction", Reason: "Concurrent decisions", Scope: &scope, RecordID: r.RecordID, Supersedes: []string{r.ID}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	out, err := b.Sync(context.Background(), 10*time.Second)
	if err != nil || out.State != "conflicted" || !out.Delivered || out.SemanticConflicts != 1 {
		t.Fatal(out, err)
	}
	p, err := second.Recall("", &scope, 5, 8192)
	if err != nil || len(p.Current) != 0 || p.ConflictCount != 1 {
		t.Fatal(p, err)
	}
}

func TestLocalSynchronizationTimingEvidence(t *testing.T) {
	s, a := fixture(t)
	for _, sample := range []struct {
		name string
		run  func() (Status, error)
	}{
		{"initialize", func() (Status, error) { return s.Initialize(context.Background()) }},
		{"checkpoint", func() (Status, error) { return s.Checkpoint(context.Background()) }},
		{"no-origin sync", func() (Status, error) { return s.Sync(context.Background(), 10*time.Second) }},
	} {
		start := time.Now()
		if _, err := sample.run(); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %s", sample.name, time.Since(start))
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, a.Root(), "init", "--bare", "--initial-branch=main", remote)
	gitTest(t, a.Root(), "remote", "add", "origin", remote)
	start := time.Now()
	if _, err := s.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	t.Logf("initial local-remote delivery: %s", time.Since(start))
	start = time.Now()
	if _, err := s.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	t.Logf("unchanged local-remote sync: %s", time.Since(start))
}

func TestUnavailableRemoteKeepsLocalCheckpoint(t *testing.T) {
	a, first := fixture(t)
	if _, err := a.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	gitTest(t, first.Root(), "remote", "add", "origin", filepath.Join(t.TempDir(), "absent.git"))
	r := remember(t, first, "Offline evidence")
	out, err := a.Sync(context.Background(), 10*time.Second)
	if err != nil || out.State != "pending" || !out.Checkpointed || out.Delivered || out.Head == "" {
		t.Fatal(out, err)
	}
	if got := gitTest(t, first.Root(), "status", "--porcelain"); got != "" {
		t.Fatal(got)
	}
	if _, err := first.History(r.RecordID); err != nil {
		t.Fatal(err)
	}
}

func TestUnsafeRemoteCandidateNeverChangesLocalHead(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	if _, err := b.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	before := gitTest(t, first.Root(), "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(second.Root(), "unrelated.txt"), []byte("PRIVATE-CANARY"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, second.Root(), "add", "unrelated.txt")
	gitTest(t, second.Root(), "commit", "-m", "Unsafe candidate fixture")
	gitTest(t, second.Root(), "push", "origin", "main")
	out, err := a.Sync(context.Background(), 10*time.Second)
	if err != nil || out.State != "conflicted" || out.Delivered {
		t.Fatal(out, err)
	}
	if gitTest(t, first.Root(), "rev-parse", "HEAD") != before {
		t.Fatal("integrated unsafe candidate")
	}
	if _, err := os.Stat(filepath.Join(first.Root(), "unrelated.txt")); !os.IsNotExist(err) {
		t.Fatal("checked out unsafe content")
	}
}
