package signetsync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestCallerGitEnvironmentCannotRedirectCheckpoint(t *testing.T) {
	s, a := fixture(t)
	other, otherService := fixture(t)
	initial, err := other.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(otherService.Root(), ".git"))
	t.Setenv("GIT_WORK_TREE", otherService.Root())
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "foreign-index"))
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if head := gitTest(t, otherService.Root(), "rev-parse", "HEAD"); head != initial.Head {
		t.Fatal("touched foreign repository")
	}
	if files := gitTest(t, a.Root(), "ls-files"); !strings.Contains(files, "signet.json") {
		t.Fatal("missing intended checkpoint")
	}
}

func TestSharedMemoryWriterLock(t *testing.T) {
	s, a := fixture(t)
	lock, err := os.OpenFile(filepath.Join(a.Root(), ".mandalore", "write.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if _, err := s.Initialize(context.Background()); !errors.Is(err, memory.ErrWriterBusy) {
		t.Fatal("sync bypassed memory lock", err)
	}
	if _, err := os.Lstat(filepath.Join(a.Root(), ".git")); !os.IsNotExist(err) {
		t.Fatal("busy sync initialized Git")
	}
}

func TestInProgressAndLinkedRepositoriesRefused(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(a.Root(), ".git", "MERGE_HEAD")
	if err := os.WriteFile(marker, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(context.Background()); !errors.Is(err, ErrBoundary) {
		t.Fatal(err)
	}
	if err := os.Rename(marker, filepath.Join(t.TempDir(), "merge-head")); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, a.Root(), "worktree", "add", "-b", "linked", linked)
	other, err := Open(linked, a.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Checkpoint(context.Background()); !errors.Is(err, ErrBoundary) {
		t.Fatal("linked worktree accepted", err)
	}
}

func TestGitProcessCancellationAndOutputBounds(t *testing.T) {
	s, _ := fixture(t)
	bin := t.TempDir()
	path := filepath.Join(bin, "git")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(path, []byte("#!/bin/sh\nsleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := s.git(ctx, "version"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("subprocess group outlived cancellation bound")
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nwhile :; do printf 'PRIVATE-CANARY-OUTPUT-BOUND\\n'; done\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	out, err := s.git(ctx2, "version")
	if err == nil || out != "" || strings.Contains(err.Error(), "PRIVATE-CANARY") {
		t.Fatal("unbounded/reflected output", err)
	}
	if ctx2.Err() != nil {
		t.Fatal("output bound relied on whole-operation timeout")
	}
}
