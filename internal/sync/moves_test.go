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

var (
	homeX = memory.Scope{Kind: "project", ID: "example-x"}
	homeY = memory.Scope{Kind: "project", ID: "example-y"}
	homeZ = memory.Scope{Kind: "account", ID: "example-z"}
)

func syncAll(t *testing.T, clones ...*Synchronizer) {
	t.Helper()
	syncAs(t, "synchronized", 0, clones...)
}

// syncAs syncs each clone in turn and checks the last result.
func syncAs(t *testing.T, state string, conflicts int, clones ...*Synchronizer) {
	t.Helper()
	for i, s := range clones {
		out, err := s.Sync(context.Background(), 30*time.Second)
		if err != nil || !out.Delivered && out.State != "synchronized" {
			t.Fatal(out, err)
		}
		if i == len(clones)-1 && (out.State != state || out.SemanticConflicts != conflicts) {
			t.Fatalf("sync state %s with %d conflicts, want %s with %d", out.State, out.SemanticConflicts, state, conflicts)
		}
	}
}

// sharedRecord saves a record in homeX on the first clone and syncs it to both.
func sharedRecord(t *testing.T) (*Synchronizer, *memory.Service, *Synchronizer, *memory.Service, memory.Revision) {
	t.Helper()
	a, first, b, second, _ := remoteFixture(t)
	r, err := first.Remember(memory.Write{Kind: "fact", Scope: &homeX, Summary: "Synthetic codename", Body: "Copper Finch", Basis: "user-direction", Reason: "Synthetic test"})
	if err != nil {
		t.Fatal(err)
	}
	syncAll(t, a, b)
	return a, first, b, second, r
}

func move(t *testing.T, s *memory.Service, recordID string, to memory.Scope, content, moves []string) memory.MoveReceipt {
	t.Helper()
	m, err := s.Move(memory.MoveWrite{RecordID: recordID, Scope: to, ContentHeads: content, MoveHeads: moves, Reason: "Misfiled"})
	if err != nil || !m.Moved {
		t.Fatal(m, err)
	}
	return m
}

func correct(t *testing.T, s *memory.Service, scope memory.Scope, recordID string, supersedes []string, body string) memory.Revision {
	t.Helper()
	r, err := s.Remember(memory.Write{Kind: "fact", Scope: &scope, RecordID: recordID, Supersedes: supersedes, Summary: "Synthetic codename", Body: body, Basis: "user-direction", Reason: "Corrected"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func conflictsIn(t *testing.T, s *memory.Service, scope memory.Scope) (current, conflicts int) {
	t.Helper()
	p, err := s.Recall("", &scope, 50, 32768)
	if err != nil {
		t.Fatal(err)
	}
	return len(p.Current), len(p.Conflicts)
}

// #250 acceptance 4: two clones move the same record to different scopes
// and sync. The conflict is listed and neither move is lost.
func TestConcurrentMovesSyncAsAConflict(t *testing.T) {
	a, first, b, second, r := sharedRecord(t)
	mine := move(t, first, r.RecordID, homeY, []string{r.ID}, []string{})
	theirs := move(t, second, r.RecordID, homeZ, []string{r.ID}, []string{})
	// The sync result lists the conflict, counted once.
	syncAs(t, "conflicted", 1, a, b, a)
	for _, s := range []*memory.Service{first, second} {
		h, err := s.HistoryPage(r.RecordID, 0, 50)
		if err != nil || len(h.MoveHeads) != 2 || h.Scope != nil || len(h.Moves) != 2 {
			t.Fatalf("history after sync: %+v %v", h, err)
		}
		for _, scope := range []memory.Scope{homeY, homeZ} {
			if current, conflicts := conflictsIn(t, s, scope); current != 0 || conflicts != 1 {
				t.Fatalf("recall in %v: %d current, %d conflicts", scope, current, conflicts)
			}
		}
	}
	settle := move(t, first, r.RecordID, homeY, []string{r.ID}, []string{mine.ID, theirs.ID})
	syncAll(t, a, b)
	h, err := second.HistoryPage(r.RecordID, 0, 50)
	if err != nil || h.Scope == nil || *h.Scope != homeY || len(h.MoveHeads) != 1 || h.MoveHeads[0] != settle.ID || h.MoveCount != 3 {
		t.Fatalf("settled history on the other clone: %+v %v", h, err)
	}
	if current, conflicts := conflictsIn(t, second, homeY); current != 1 || conflicts != 0 {
		t.Fatalf("settled recall: %d %d", current, conflicts)
	}
}

func TestAgreeingConcurrentMovesSyncWithoutAConflict(t *testing.T) {
	a, first, b, second, r := sharedRecord(t)
	move(t, first, r.RecordID, homeY, []string{r.ID}, []string{})
	move(t, second, r.RecordID, homeY, []string{r.ID}, []string{})
	syncAll(t, a, b, a)
	if current, conflicts := conflictsIn(t, second, homeY); current != 1 || conflicts != 0 {
		t.Fatalf("agreeing moves: %d current, %d conflicts", current, conflicts)
	}
	correct(t, second, homeY, r.RecordID, []string{r.ID}, "Silver Heron")
	h, err := second.HistoryPage(r.RecordID, 0, 50)
	if err != nil || len(h.MoveHeads) != 2 || h.MoveCount != 2 {
		t.Fatalf("both moves kept: %+v %v", h, err)
	}
}

// A move on one clone and a correction on the other keep both: the
// corrected content, in the new scope.
func TestConcurrentMoveAndCorrectionKeepBoth(t *testing.T) {
	a, first, b, second, r := sharedRecord(t)
	move(t, first, r.RecordID, homeY, []string{r.ID}, []string{})
	c := correct(t, second, homeX, r.RecordID, []string{r.ID}, "Silver Heron")
	syncAll(t, a, b, a)
	for _, s := range []*memory.Service{first, second} {
		p, err := s.Recall("", &homeY, 50, 32768)
		if err != nil || len(p.Current) != 1 || p.Current[0].ID != c.ID {
			t.Fatalf("recall: %+v %v", p, err)
		}
	}
}

// Each clone corrects and moves the same record to a different scope: both
// conflicts are settled, the move first and then a correction.
func TestBothConflictsFromTwoClonesAreSettled(t *testing.T) {
	a, first, b, second, r := sharedRecord(t)
	ca := correct(t, first, homeX, r.RecordID, []string{r.ID}, "Silver Heron")
	ma := move(t, first, r.RecordID, homeY, []string{ca.ID}, []string{})
	cb := correct(t, second, homeX, r.RecordID, []string{r.ID}, "Amber Lark")
	mb := move(t, second, r.RecordID, homeZ, []string{cb.ID}, []string{})
	syncAs(t, "conflicted", 1, a, b, a)
	both := []string{ca.ID, cb.ID}
	if _, err := first.Remember(memory.Write{Kind: "fact", Scope: &homeY, RecordID: r.RecordID, Supersedes: both, Summary: "s", Body: "b", Basis: "observation", Reason: "r"}); err == nil {
		t.Fatal("correction accepted during a move conflict")
	}
	if _, err := first.Move(memory.MoveWrite{RecordID: r.RecordID, Scope: homeY, ContentHeads: []string{ca.ID}, MoveHeads: []string{ma.ID, mb.ID}, Reason: "Settle"}); err == nil {
		t.Fatal("settling move missing a content head accepted")
	}
	move(t, first, r.RecordID, homeY, both, []string{ma.ID, mb.ID})
	settled := correct(t, first, homeY, r.RecordID, both, "Settled")
	syncAll(t, a, b)
	for _, s := range []*memory.Service{first, second} {
		p, err := s.Recall("", &homeY, 50, 32768)
		if err != nil || len(p.Current) != 1 || p.Current[0].ID != settled.ID || len(p.Conflicts) != 0 {
			t.Fatalf("after settling: %+v %v", p, err)
		}
		h, err := s.HistoryPage(r.RecordID, 0, 50)
		if err != nil || h.Scope == nil || *h.Scope != homeY || len(h.MoveHeads) != 1 {
			t.Fatalf("history after settling: %+v %v", h, err)
		}
	}
}

func TestCheckpointRefusesStrayFilesUnderMoves(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	r, err := a.Remember(memory.Write{Kind: "fact", Scope: &homeX, Summary: "s", Body: "b", Basis: "observation", Reason: "r"})
	if err != nil {
		t.Fatal(err)
	}
	move(t, a, r.RecordID, homeY, []string{r.ID}, []string{})
	stray := filepath.Join(a.Root(), "memory/moves", r.RecordID, "notes.txt")
	if err := os.WriteFile(stray, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(context.Background()); err == nil {
		t.Fatal("stray file under memory/moves committed")
	}
	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	if out, err := s.Checkpoint(context.Background()); err != nil || !out.Checkpointed {
		t.Fatal(out, err)
	}
	if files := gitTest(t, a.Root(), "ls-files"); !strings.Contains(files, "memory/moves/"+r.RecordID+"/move-") {
		t.Fatal("move not checkpointed")
	}
}

// A format 1 signet with a move upgrades, and the upgrade proof protects the
// move as append-only evidence.
func TestFormatOneSignetWithAMoveUpgrades(t *testing.T) {
	s, a := fixture(t)
	r, err := a.Remember(memory.Write{Kind: "fact", Scope: &homeX, Summary: "s", Body: "b", Basis: "observation", Reason: "r"})
	if err != nil {
		t.Fatal(err)
	}
	move(t, a, r.RecordID, homeY, []string{r.ID}, []string{})
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
	upgraded, err := memory.OpenReadOnlyService(a.Root())
	if err != nil {
		t.Fatal(err)
	}
	if p, err := upgraded.Recall("", &homeY, 5, 8192); err != nil || len(p.Current) != 1 {
		t.Fatalf("moved record after upgrade: %+v %v", p, err)
	}
	path := strings.TrimSpace(gitTest(t, a.Root(), "ls-files", "memory/moves"))
	if path == "" {
		t.Fatal("move missing after upgrade")
	}
	if err := os.Remove(filepath.Join(a.Root(), path)); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "add", "-A")
	gitTest(t, a.Root(), "commit", "-q", "-m", "drop move")
	if err := s.validateCandidate(context.Background(), gitTest(t, a.Root(), "rev-parse", "HEAD")); err == nil {
		t.Fatal("deleting a move passed as an append-only history")
	}
}
