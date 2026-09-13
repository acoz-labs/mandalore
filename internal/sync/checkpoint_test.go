package signetsync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func fixture(t *testing.T) (*Synchronizer, *memory.Service) {
	t.Helper()
	s, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Example", "device-test", "Test")
	if err != nil {
		t.Fatal(err)
	}
	a, err := memory.OpenService(s.Root, memory.Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	sy, err := Open(s.Root, s.Signet.ID)
	if err != nil {
		t.Fatal(err)
	}
	return sy, a
}

func gitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Env = append(cleanEnvironment(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git fixture %s: %v: %s", args[0], err, output)
	}
	return strings.TrimSpace(string(output))
}

func remember(t *testing.T, a *memory.Service, body string) memory.Revision {
	t.Helper()
	r, err := a.Remember(memory.Write{Kind: "fact", Summary: "Project", Body: body, Basis: "user-direction", Reason: "Confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestInitializeAndCheckpointWithoutRemote(t *testing.T) {
	s, a := fixture(t)
	ctx := context.Background()
	if _, err := s.Checkpoint(ctx); err == nil {
		t.Fatal("implicit Git initialization")
	}
	initial, err := s.Initialize(ctx)
	if err != nil || initial.State != "local-only" || initial.Head == "" || !initial.Checkpointed {
		t.Fatal(initial, err)
	}
	if got := gitTest(t, a.Root(), "status", "--porcelain"); got != "" {
		t.Fatal(got)
	}
	if tracked := gitTest(t, a.Root(), "ls-files"); strings.Contains(tracked, ".mandalore/") {
		t.Fatal("local state tracked", tracked)
	}
	repeated, err := s.Checkpoint(ctx)
	if err != nil || repeated.Head != initial.Head {
		t.Fatal("empty checkpoint commit", repeated, err)
	}
	remember(t, a, "Copper Finch")
	changed, err := s.Checkpoint(ctx)
	if err != nil || changed.Head == initial.Head {
		t.Fatal(changed, err)
	}
	if got := gitTest(t, a.Root(), "status", "--porcelain"); got != "" {
		t.Fatal(got)
	}
}

func TestCheckpointPreservesUnrelatedAndPartialStaging(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(a.Root(), "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("PRIVATE-CANARY"), 0600); err != nil {
		t.Fatal(err)
	}
	before := gitTest(t, a.Root(), "status", "--porcelain")
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("unrelated file accepted")
	}
	if after := gitTest(t, a.Root(), "status", "--porcelain"); after != before {
		t.Fatal("changed unrelated work")
	}
	if err := os.Rename(unrelated, filepath.Join(t.TempDir(), "unrelated.txt")); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(a.Root(), "README.md")
	if err := os.WriteFile(readme, []byte("Staged text"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "add", "README.md")
	if err := os.WriteFile(readme, []byte("Unstaged text"), 0600); err != nil {
		t.Fatal(err)
	}
	before = gitTest(t, a.Root(), "diff", "--cached")
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("partial staging accepted")
	}
	if after := gitTest(t, a.Root(), "diff", "--cached"); after != before {
		t.Fatal("overwrote staged content")
	}
}

func TestCheckpointRefusesRewrittenEvidenceAndWrongBranch(t *testing.T) {
	s, a := fixture(t)
	r := remember(t, a, "Copper Finch")
	initial, err := s.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.Root(), "memory", "records", r.RecordID, r.ID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), "Copper Finch", "Silver Heron", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("rewritten evidence committed")
	}
	if gitTest(t, a.Root(), "rev-parse", "HEAD") != initial.Head {
		t.Fatal("HEAD moved on rejection")
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "switch", "-c", "other")
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("wrong branch accepted")
	}
}

func TestCancelledCheckpointDoesNotInitializeGit(t *testing.T) {
	s, a := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Initialize(ctx); err == nil {
		t.Fatal("cancelled init succeeded")
	}
	if _, err := os.Lstat(filepath.Join(a.Root(), ".git")); !os.IsNotExist(err) {
		t.Fatal("cancelled init touched Git", err)
	}
}

func TestCheckpointRefusesNestedUnrelatedFiles(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.Root(), "memory", "notes.txt"), []byte("PRIVATE-CANARY"), 0600); err != nil {
		t.Fatal(err)
	}
	before := gitTest(t, a.Root(), "status", "--porcelain")
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("nested unrelated material committed")
	}
	if after := gitTest(t, a.Root(), "status", "--porcelain"); after != before {
		t.Fatal("nested unrelated work changed")
	}
}

func TestIgnoreRulesCannotHideValidMemoryFromCheckpoint(t *testing.T) {
	s, a := fixture(t)
	if err := os.WriteFile(filepath.Join(a.Root(), ".gitignore"), []byte(".mandalore/\nmemory/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := remember(t, a, "Copper Finch")
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gitTest(t, a.Root(), "ls-files"), r.ID+".json") {
		t.Fatal("checkpoint omitted ignored valid memory")
	}
}
