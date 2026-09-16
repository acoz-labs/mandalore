package signetsync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestPrivacyCorrectionAndDeletionDoNotEraseGitEvidence(t *testing.T) {
	s, a := fixture(t)
	ctx := context.Background()
	const marker = "SYNTHETIC-ORIGINAL-NOT-A-SECRET"
	first := remember(t, a, marker)
	initial, err := s.Initialize(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Remember(memory.Write{Kind: first.Kind, Summary: "Project", Body: "Replacement", Basis: "user-direction", Reason: "Synthetic correction", RecordID: first.RecordID, Supersedes: []string{first.ID}}); err != nil {
		t.Fatal(err)
	}
	current, err := s.Checkpoint(ctx)
	if err != nil || current.Head == initial.Head {
		t.Fatal("correction not checkpointed", err)
	}
	rel := "memory/records/" + first.RecordID + "/" + first.ID + ".json"
	for _, head := range []string{initial.Head, current.Head} {
		if !strings.Contains(gitTest(t, a.Root(), "show", head+":"+rel), marker) {
			t.Fatal("correction erased original Git evidence")
		}
	}
	// Only a disposable, test-created predecessor is removed.
	if err := os.Remove(filepath.Join(a.Root(), rel)); err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err == nil {
		t.Fatal("missing predecessor unexpectedly valid")
	}
	if _, err := s.Checkpoint(ctx); err == nil {
		t.Fatal("deleted predecessor checkpointed")
	}
	if gitTest(t, a.Root(), "rev-parse", "HEAD") != current.Head || !strings.Contains(gitTest(t, a.Root(), "show", initial.Head+":"+rel), marker) {
		t.Fatal("rejected deletion changed committed evidence")
	}
}

func TestPrivacyValidJournalDeletionIsRefusedByAppendOnlyGuard(t *testing.T) {
	s, a := fixture(t)
	ctx := context.Background()
	const marker = "SYNTHETIC-JOURNAL-NOT-A-SECRET"
	event, err := a.AppendJournal("fixture", marker)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := s.Initialize(ctx)
	if err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse(time.RFC3339Nano, event.RecordedAt)
	if err != nil {
		t.Fatal(err)
	}
	rel := "memory/events/" + at.UTC().Format("2006/01") + "/" + event.ID + ".json"
	if err := os.Remove(filepath.Join(a.Root(), rel)); err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err != nil {
		t.Fatal("fixture must remain structurally valid to isolate append-only guard", err)
	}
	if _, err := s.Checkpoint(ctx); !errors.Is(err, ErrHistory) {
		t.Fatal("valid evidence removal was not refused by history guard", err)
	}
	if gitTest(t, a.Root(), "rev-parse", "HEAD") != initial.Head || !strings.Contains(gitTest(t, a.Root(), "show", initial.Head+":"+rel), marker) {
		t.Fatal("rejected journal deletion changed committed history")
	}
}
