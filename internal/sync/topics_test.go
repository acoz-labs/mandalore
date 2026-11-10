package signetsync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

var topic = memory.Scope{Kind: "project", ID: "atlas-launch"}

func nameTopic(t *testing.T, s *memory.Service) memory.ScopeLabel {
	t.Helper()
	l, err := s.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "Atlas launch", Description: "Synthetic launch topic.", Reason: "Named in a synthetic test"})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestTopicFilesSyncBetweenClones(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	l := nameTopic(t, first)
	e, err := first.AppendJournal("outcome", "Atlas checklist drafted", &topic)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := a.Sync(context.Background(), 30*time.Second); err != nil || !out.Delivered {
		t.Fatal(out, err)
	}
	if out, err := b.Sync(context.Background(), 30*time.Second); err != nil || out.State != "synchronized" {
		t.Fatal(out, err)
	}
	labels, err := second.ScopePage("atlas", 0, 5)
	if err != nil || len(labels.Items) != 1 || labels.Items[0].Labels[0].ID != l.ID {
		t.Fatalf("label not synchronized: %+v %v", labels, err)
	}
	journal, err := second.Journal("", &topic, 5)
	if err != nil || len(journal) != 1 || journal[0].ID != e.ID {
		t.Fatalf("scoped journal not synchronized: %+v %v", journal, err)
	}
}

func TestCheckpointStillRefusesStrayFilesUnderScopes(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	nameTopic(t, a)
	stray := filepath.Join(a.Root(), "memory/scopes/project/atlas-launch/notes.json")
	if err := os.WriteFile(stray, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("stray file under memory/scopes committed")
	}
	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	if out, err := s.Checkpoint(context.Background()); err != nil || !out.Checkpointed {
		t.Fatal(out, err)
	}
	if files := gitTest(t, a.Root(), "ls-files"); !strings.Contains(files, "memory/scopes/project/atlas-launch/labels/") {
		t.Fatal("label not checkpointed")
	}
}

func TestFormatOneSignetWithTopicsUpgradesAndSyncs(t *testing.T) {
	s, a := fixture(t)
	r := remember(t, a, "Preserved evidence")
	nameTopic(t, a)
	if _, err := a.AppendJournal("outcome", "Atlas checklist drafted", &topic); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	s = upgradedFixture(t, s, r.Authorship)
	out, err := s.Checkpoint(context.Background())
	if err != nil || !out.Checkpointed {
		t.Fatal(out, err)
	}
	if err := s.validateCandidate(context.Background(), out.Head); err != nil {
		t.Fatal(err)
	}
	// The upgrade proof protects the topic files as append-only evidence.
	label := gitTest(t, a.Root(), "ls-files", "memory/scopes/project/atlas-launch/labels")
	if label == "" {
		t.Fatal("label missing after upgrade")
	}
	if err := os.Remove(filepath.Join(a.Root(), strings.TrimSpace(label))); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "add", "-A")
	gitTest(t, a.Root(), "commit", "-q", "-m", "drop label")
	if err := s.validateCandidate(context.Background(), gitTest(t, a.Root(), "rev-parse", "HEAD")); err == nil {
		t.Fatal("deleting a label passed as an append-only history")
	}
}

// Two clones that name the same scope before syncing both keep syncing; the
// two first names show as conflicting labels until a rename supersedes both.
func TestConcurrentFirstNamesSyncAsAConflict(t *testing.T) {
	a, first, b, second, _ := remoteFixture(t)
	mine := nameTopic(t, first)
	theirs, err := second.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "Atlas rollout", Description: "Named on the other clone.", Reason: "Named in a synthetic test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Synchronizer{a, b, a} {
		if out, err := s.Sync(context.Background(), 30*time.Second); err != nil || out.State != "synchronized" {
			t.Fatal(out, err)
		}
	}
	heads := func(s *memory.Service) []string {
		t.Helper()
		page, err := s.ScopePage("atlas", 0, 5)
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("%+v %v", page, err)
		}
		ids := []string{}
		for _, l := range page.Items[0].Labels {
			ids = append(ids, l.ID)
		}
		return ids
	}
	for _, s := range []*memory.Service{first, second} {
		if got := heads(s); len(got) != 2 || !slices.Contains(got, mine.ID) || !slices.Contains(got, theirs.ID) {
			t.Fatalf("conflict not visible: %v", got)
		}
		if p, err := s.Prime(topic, 8192); err != nil || len(p.Labels) != 2 {
			t.Fatalf("prime: %+v %v", p.Labels, err)
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	// Naming again without supersedes is still refused on one writer.
	if _, err := second.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "Third", Description: "d", Reason: "r"}); !errors.Is(err, memory.ErrInvalidInput) {
		t.Fatal(err)
	}
	settled, err := second.NameScope(memory.ScopeLabelWrite{Scope: topic, Name: "Atlas launch", Description: "Settled with the user.", Reason: "Resolved the naming conflict", Supersedes: []string{mine.ID, theirs.ID}})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Synchronizer{b, a} {
		if out, err := s.Sync(context.Background(), 30*time.Second); err != nil || out.State != "synchronized" {
			t.Fatal(out, err)
		}
	}
	for _, s := range []*memory.Service{first, second} {
		if got := heads(s); len(got) != 1 || got[0] != settled.ID {
			t.Fatalf("not settled: %v", got)
		}
	}
}
