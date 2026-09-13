package signetsync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCancellationAtCheckpointIntegrationAndPush(t *testing.T) {
	for _, phase := range []string{"checkpoint", "integrate", "push"} {
		t.Run(phase, func(t *testing.T) {
			s, a, other, b, _ := remoteFixture(t)
			operation := "push"
			if phase == "checkpoint" {
				operation = "update-ref"
				remember(t, a, "Pending local evidence")
			}
			if phase == "integrate" {
				operation = "merge"
				remember(t, b, "Incoming evidence")
				if out, err := other.Sync(context.Background(), 10*time.Second); err != nil || !out.Delivered {
					t.Fatal(out, err)
				}
			}
			marker := filepath.Join(t.TempDir(), "phase-started")
			wrapGit(t, operation, ": > "+quote(marker)+"\nsleep 30\nexit 128")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := s.Sync(ctx, 10*time.Second); done <- err }()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			deadline := time.After(4 * time.Second)
			waiting := true
			for waiting {
				select {
				case <-deadline:
					t.Fatal("phase did not start")
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
				if !errors.As(err, &stopped) || !errors.Is(err, context.Canceled) || stopped.Status.Phase != phase {
					t.Fatal(phase, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("phase did not cancel")
			}
			if _, err := os.Stat(filepath.Join(a.Root(), ".git", "MERGE_HEAD")); !os.IsNotExist(err) {
				t.Fatal("cancelled phase left merge state")
			}
			if err := s.store.Validate(); err != nil {
				t.Fatal("cancel damaged memory", err)
			}
		})
	}
}
