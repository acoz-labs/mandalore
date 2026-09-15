package api

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func TestRealSynchronizationOperations(t *testing.T) {
	a := fixture(t)
	ctx := context.Background()
	for _, name := range []string{"memory_git_init", "memory_checkpoint", "memory_sync"} {
		out := a.Call(ctx, name, []byte(`{}`))
		if !out.OK {
			t.Fatal(name, out.Error)
		}
		receipt := out.Result.(signetsync.Status)
		if receipt.State != "local-only" || !receipt.Checkpointed || receipt.Delivered {
			t.Fatal(receipt)
		}
	}
	out := a.Call(ctx, "memory_sync_status", []byte(`{}`))
	if !out.OK {
		t.Fatal(out.Error)
	}
	a.ReadOnly = true
	for _, name := range []string{"memory_git_init", "memory_checkpoint", "memory_sync"} {
		out := a.Call(ctx, name, []byte(`{}`))
		if out.OK || out.Error.Code != "operation.read_only" {
			t.Fatal(out)
		}
	}
	if out := a.Call(ctx, "memory_sync_status", []byte(`{}`)); !out.OK {
		t.Fatal(out.Error)
	}
}

func TestSyncTimeoutIsStrictInput(t *testing.T) {
	a := fixture(t)
	for _, raw := range []string{`{"timeout_seconds":0}`, `{"timeout_seconds":31}`, `{"timeout_seconds":"2"}`} {
		out := a.Call(context.Background(), "memory_sync", []byte(raw))
		if out.OK || out.Error.Code != "input.invalid" {
			t.Fatal(out)
		}
	}
}

func TestSyncCancellationExposesPartialWriteEvidence(t *testing.T) {
	for _, startup := range []string{"normal", "slow"} {
		t.Run(startup, func(t *testing.T) {
			a, _, fetch := cancellationFixture(t, startup)
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan Envelope, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				result <- a.Call(ctx, "memory_sync", []byte(`{"timeout_seconds":30}`))
			}()
			t.Cleanup(func() {
				cancel()
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					t.Error("cancelled sync worker did not finish")
				}
			})
			deadline := time.NewTimer(10 * time.Second)
			defer deadline.Stop()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
		waitForFetch:
			for {
				select {
				case out := <-result:
					encoded, _ := json.Marshal(out)
					t.Fatalf("sync returned before observed fetch: %s", encoded)
				case <-deadline.C:
					t.Fatal("fetch did not start before readiness deadline")
				case <-ticker.C:
					if data, err := os.ReadFile(fetch); err == nil && len(strings.TrimSpace(string(data))) > 0 {
						break waitForFetch
					} else if err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
				}
			}
			cancel()
			select {
			case out := <-result:
				assertSyncCancellation(t, out, "fetch", true)
			case <-time.After(5 * time.Second):
				t.Fatal("observed fetch did not cancel")
			}
			assertStoppedGit(t, fetch)
			if err := a.service.Validate(); err != nil {
				t.Fatal("cancel damaged memory:", err)
			}
		})
	}
}

func TestSyncDeadlineBeforeCheckpointRetainsEvidence(t *testing.T) {
	a, first, fetch := cancellationFixture(t, "blocked")
	// This parent only bounds a broken implementation. It must not be what ends
	// the call: the configured one-second operation deadline is under test.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out := a.Call(ctx, "memory_sync", []byte(`{"timeout_seconds":1}`))
	if ctx.Err() != nil {
		t.Fatal("sync did not honor its operation deadline before the parent deadline")
	}
	assertSyncCancellation(t, out, "checkpoint", false)
	assertStoppedGit(t, first)
	if _, err := os.Stat(fetch); !os.IsNotExist(err) {
		t.Fatalf("deadline fixture unexpectedly reached fetch: %v", err)
	}
	if err := a.service.Validate(); err != nil {
		t.Fatal("deadline damaged memory:", err)
	}
}

func cancellationFixture(t *testing.T, startup string) (*API, string, string) {
	t.Helper()
	a := fixture(t)
	if out := a.Call(context.Background(), "memory_git_init", []byte(`{}`)); !out.OK {
		t.Fatal(out.Error)
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(real, "-C", a.service.Root(), "remote", "add", "origin", "https://example.invalid/signet.git").CombinedOutput(); err != nil {
		t.Fatal(string(output), err)
	}
	bin := t.TempDir()
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	first, fetch := filepath.Join(bin, "first-git"), filepath.Join(bin, "fetch-git")
	delay := ""
	fetchDelay := ""
	switch startup {
	case "normal":
	case "slow-marker":
		// Creating a redirection target is observable before printf writes its
		// PID. Widen that real window to verify readers await complete evidence.
		fetchDelay = ": > " + quote(fetch) + "\nsleep 0.2\n"
	case "slow":
		delay = "sleep 2\n" // Deliberately exceeds the old whole-operation budget.
	case "blocked":
		delay = "exec sleep 30\n"
	default:
		t.Fatal("unknown cancellation fixture startup")
	}
	script := "#!/bin/sh\nset -eu\nif [ ! -e " + quote(first) + " ]; then\nprintf '%s\\n' \"$$\" > " + quote(first) + "\n" + delay + "fi\n" +
		"for arg in \"$@\"; do\nif [ \"$arg\" = ls-remote ]; then\n" + fetchDelay + "printf '%s\\n' \"$$\" > " + quote(fetch) + "\nexec sleep 30\nfi\ndone\nexec " + quote(real) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return a, first, fetch
}

func assertSyncCancellation(t *testing.T, out Envelope, phase string, checkpointed bool) {
	t.Helper()
	if out.OK || out.Error == nil || out.Error.Code != "operation.cancelled" || out.Error.Retryable || !out.Error.WriteMayHaveOccurred || !out.Error.InspectBeforeRetry || out.Error.SyncStatus == nil || out.Error.SyncStatus.Checkpointed != checkpointed || out.Error.SyncStatus.Phase != phase || out.Error.SyncStatus.Delivered {
		encoded, _ := json.Marshal(out)
		t.Fatalf("expected cancelled %s (checkpointed=%t): %s", phase, checkpointed, encoded)
	}
}

func assertStoppedGit(t *testing.T, marker string) {
	t.Helper()
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal("Git entry was not observed:", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid Git process marker: %q", data)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("Git process still exists after cancellation and wait: %v", err)
	}
}
