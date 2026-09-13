package signetsync

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func quote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func wrapGit(t *testing.T, operation, body string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nfor arg in \"$@\"; do\nif [ \"$arg\" = " + quote(operation) + " ]; then\n" + body + "\nfi\ndone\nexec " + quote(real) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestControlledAuthenticationFailureIsPendingWithoutEcho(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "remote", "add", "origin", "https://example.invalid/signet.git")
	wrapGit(t, "ls-remote", "printf 'PRIVATE-CANARY-AUTH\\n' >&2\nexit 128")
	out, err := s.Sync(context.Background(), 10*time.Second)
	if err != nil || out.State != "pending" || !out.Checkpointed || out.Delivered || strings.Contains(out.Notice, "PRIVATE-CANARY") {
		t.Fatal(out, err)
	}
}

func TestCancellationAfterCheckpointRetainsPhaseAndReceipt(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "remote", "add", "origin", "https://example.invalid/signet.git")
	marker := filepath.Join(t.TempDir(), "fetch-started")
	wrapGit(t, "ls-remote", ": > "+quote(marker)+"\nsleep 30\nexit 128")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := s.Sync(ctx, 10*time.Second); done <- err }()
	deadline := time.After(4 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	waiting := true
	for waiting {
		select {
		case <-deadline:
			t.Fatal("fetch fixture did not start")
		case <-ticker.C:
			if _, err := os.Stat(marker); err == nil {
				waiting = false
			}
		}
	}
	cancel()
	select {
	case err := <-done:
		var stopped *Failure
		if !errors.As(err, &stopped) || !errors.Is(err, context.Canceled) || !stopped.Status.Checkpointed || stopped.Status.Phase != "fetch" {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled network subprocess retained")
	}
	status, err := s.Inspect(context.Background())
	if err != nil || status.LastAttempt == nil || status.LastAttempt.State != "pending" || status.LastAttempt.Phase != "fetch" {
		t.Fatal(status, err)
	}
}

func TestFileConflictLeavesNoMergeStateOrLostLocalWork(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	if _, err := b.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	for i, root := range []string{first.Root(), second.Root()} {
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte([]string{"First decision\n", "Other decision\n"}[i]), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	out, err := b.Sync(context.Background(), 10*time.Second)
	if err != nil || out.State != "conflicted" || out.Delivered {
		t.Fatal(out, err)
	}
	data, err := os.ReadFile(filepath.Join(second.Root(), "README.md"))
	if err != nil || string(data) != "Other decision\n" {
		t.Fatal(string(data), err)
	}
	if _, err := os.Stat(filepath.Join(second.Root(), ".git", "MERGE_HEAD")); !os.IsNotExist(err) {
		t.Fatal("left Git in merge state")
	}
	inspection, err := b.Inspect(context.Background())
	if err != nil || inspection.State != "conflicted" {
		t.Fatal("forgot unresolved file conflict", inspection, err)
	}
}

func TestRacingPushRemainsPendingWithoutOverwritingRemote(t *testing.T) {
	a, first, b, second, remote := remoteFixture(t)
	remember(t, second, "Rival pending evidence")
	rival, err := b.Checkpoint(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	remember(t, first, "Local pending evidence")
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	body := quote(real) + " -C " + quote(second.Root()) + " push origin main >&2"
	wrapGit(t, "push", body)
	out, err := a.Sync(context.Background(), 10*time.Second)
	if err != nil || out.State != "pending" || out.Delivered || !out.Checkpointed {
		t.Fatal(out, err)
	}
	if got := gitTest(t, remote, "rev-parse", "main"); got != rival.Head {
		t.Fatal("remote overwritten", got)
	}
	if got := gitTest(t, first.Root(), "rev-parse", "main"); got != out.Head {
		t.Fatal("local checkpoint lost", got)
	}
}

func TestRemoteChangeInvalidatesLastDeliveredState(t *testing.T) {
	a, first, _, _, _ := remoteFixture(t)
	gitTest(t, first.Root(), "remote", "set-url", "origin", filepath.Join(t.TempDir(), "different.git"))
	out, err := a.Inspect(context.Background())
	if err != nil || out.State != "pending" || out.LastAttempt == nil || !out.LastAttempt.Delivered {
		t.Fatal(out, err)
	}
}

func TestRemoteTransportValidation(t *testing.T) {
	for _, remote := range []string{"https://example.invalid/bank.git", "git@example.invalid:bank.git", "ssh://git@example.invalid/bank.git", "file:///tmp/synthetic.git", "/tmp/synthetic.git"} {
		if !validRemote(remote) {
			t.Fatal("valid transport refused", remote)
		}
	}
	for _, remote := range []string{"ext::sh payload", "https://token@example.invalid/bank.git", "https://example.invalid/bank.git?token=private", "-host:path", "ssh://-host/bank.git", "ssh://git:password@example.invalid/bank.git", "git://example.invalid/bank.git", "/tmp/one\n/tmp/two"} {
		if validRemote(remote) {
			t.Fatal("unsafe transport accepted")
		}
	}
}
