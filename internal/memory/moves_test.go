package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	scopeX = Scope{Kind: "project", ID: "example-x"}
	scopeY = Scope{Kind: "project", ID: "example-y"}
	scopeZ = Scope{Kind: "account", ID: "example-z"}
)

func bothFormats(t *testing.T, run func(t *testing.T, s *Service)) {
	t.Run("format1", func(t *testing.T) { run(t, fixture(t)) })
	t.Run("format2", func(t *testing.T) { run(t, formatTwoService(t)) })
}

// saved remembers a record in scope and returns its revision.
func saved(t *testing.T, s *Service, scope Scope, body string) Revision {
	t.Helper()
	r, err := s.Remember(Write{Kind: "fact", Scope: &scope, Summary: "Synthetic codename", Body: body, Basis: "user-direction", Reason: "Synthetic test"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func corrected(t *testing.T, s *Service, scope Scope, prev Revision, body string) Revision {
	t.Helper()
	r, err := s.Remember(Write{Kind: "fact", Scope: &scope, RecordID: prev.RecordID, Supersedes: []string{prev.ID}, Summary: "Synthetic codename", Body: body, Basis: "user-direction", Reason: "Corrected"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func moved(t *testing.T, s *Service, recordID string, to Scope, content, moves []string) MoveReceipt {
	t.Helper()
	r, err := s.Move(MoveWrite{RecordID: recordID, Scope: to, ContentHeads: content, MoveHeads: moves, Reason: "Filed under the wrong project"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// moveFiles lists the move documents and idempotency intents.
func moveFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, dir := range []string{movesDir, ".mandalore"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && (dir == movesDir || strings.HasPrefix(d.Name(), "operation-")) {
				out = append(out, path)
			}
			return err
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	return out
}

func recallIDs(t *testing.T, s *Service, scope Scope) (current, conflicts []string) {
	t.Helper()
	p, err := s.Recall("", &scope, 50, 32768)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range p.Current {
		if h.Scope != scope {
			t.Fatalf("hit in %v reports scope %v", scope, h.Scope)
		}
		current = append(current, h.RecordID)
	}
	for _, c := range p.Conflicts {
		conflicts = append(conflicts, c.RecordID)
	}
	return current, conflicts
}

func primeIDs(t *testing.T, s *Service, scope Scope) (current, conflicts []string) {
	t.Helper()
	p, err := s.Prime(scope, 32768)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range p.Current {
		current = append(current, h.RecordID)
	}
	for _, c := range p.Conflicts {
		conflicts = append(conflicts, c.RecordID)
	}
	return current, conflicts
}

func scopeInfo(t *testing.T, s *Service, scope Scope) (ScopeInfo, bool) {
	t.Helper()
	items, err := s.Scopes()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Scope == scope {
			return item, true
		}
	}
	return ScopeInfo{}, false
}

func history(t *testing.T, s *Service, recordID string, offset, limit int) RecordHistory {
	t.Helper()
	h, err := s.HistoryPage(recordID, offset, limit)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// #250 acceptance 1 and 2: the record keeps its ID and history, reads follow
// the move, and corrections take the new scope.
func TestMoveKeepsTheRecordAndReadsFollowIt(t *testing.T) {
	bothFormats(t, func(t *testing.T, s *Service) {
		first := saved(t, s, scopeX, "Copper Finch")
		second := corrected(t, s, scopeX, first, "Silver Heron")
		before := hashTree(t, s.Root())
		receipt := moved(t, s, first.RecordID, scopeY, []string{second.ID}, []string{})
		if !receipt.Moved || receipt.Scope != scopeY || receipt.RecordID != first.RecordID || !strings.HasPrefix(receipt.ID, "move-") {
			t.Fatalf("receipt %+v", receipt)
		}
		// Append-only: every earlier file keeps its bytes; only the move is new.
		after := hashTree(t, s.Root())
		for path, sum := range before {
			if after[path] != sum {
				t.Fatalf("%s changed", path)
			}
		}
		if len(after) != len(before)+1 {
			t.Fatalf("move wrote %d files", len(after)-len(before))
		}
		if _, err := os.Stat(filepath.Join(s.Root(), "memory/moves", first.RecordID, receipt.ID+".json")); err != nil {
			t.Fatal(err)
		}
		if current, _ := recallIDs(t, s, scopeY); len(current) != 1 || current[0] != first.RecordID {
			t.Fatalf("recall in the new scope: %v", current)
		}
		if current, conflicts := recallIDs(t, s, scopeX); len(current)+len(conflicts) != 0 {
			t.Fatalf("recall in the old scope: %v %v", current, conflicts)
		}
		if current, _ := primeIDs(t, s, scopeY); len(current) != 1 || current[0] != first.RecordID {
			t.Fatalf("prime in the new scope: %v", current)
		}
		if current, conflicts := primeIDs(t, s, scopeX); len(current)+len(conflicts) != 0 {
			t.Fatalf("prime in the old scope: %v %v", current, conflicts)
		}
		if info, ok := scopeInfo(t, s, scopeY); !ok || info.RecordCount != 1 || info.MoveConflicts != 0 {
			t.Fatalf("new scope counts: %+v", info)
		}
		if _, ok := scopeInfo(t, s, scopeX); ok {
			t.Fatal("the old scope is still listed")
		}
		h := history(t, s, first.RecordID, 0, 50)
		if len(h.Items) != 2 || h.Items[0].Scope != scopeX || h.Items[1].Scope != scopeX {
			t.Fatalf("history revisions: %+v", h.Items)
		}
		if h.Scope == nil || *h.Scope != scopeY || h.MoveCount != 1 || len(h.MoveHeads) != 1 || h.MoveHeads[0] != receipt.ID || len(h.Moves) != 1 {
			t.Fatalf("history placement: %+v", h)
		}
		m := h.Moves[0]
		if m.Reason != "Filed under the wrong project" || m.Authorship.Actor != "Example user" && m.Authorship.Actor != "Example" || len(m.From) != 1 || m.From[0] != scopeX || m.Scope != scopeY {
			t.Fatalf("history move: %+v", m)
		}
		// A correction after the move takes the new scope and is stored in
		// the scope the record was saved in.
		if _, err := s.Remember(Write{Kind: "fact", Scope: &scopeX, RecordID: first.RecordID, Supersedes: []string{second.ID}, Summary: "Synthetic codename", Body: "Amber Lark", Basis: "user-direction", Reason: "Corrected"}); !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "moved to project example-y") {
			t.Fatalf("old-scope correction: %v", err)
		}
		third := corrected(t, s, scopeY, second, "Amber Lark")
		var stored Revision
		if err := readJSON(filepath.Join(s.Root(), "memory/records", third.RecordID, third.ID+".json"), &stored); err != nil || stored.Scope != scopeX {
			t.Fatalf("stored correction scope %v %v", stored.Scope, err)
		}
		p, err := s.Recall("", &scopeY, 50, 32768)
		if err != nil || len(p.Current) != 1 || p.Current[0].ID != third.ID || p.Current[0].Scope != scopeY {
			t.Fatalf("recall after the correction: %+v %v", p, err)
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

// #250 requirement 4: guards, in order, and nothing written on refusal.
func TestMoveGuards(t *testing.T) {
	bothFormats(t, func(t *testing.T, s *Service) {
		r := saved(t, s, scopeX, "Copper Finch")
		ok := MoveWrite{RecordID: r.RecordID, Scope: scopeY, ContentHeads: []string{r.ID}, MoveHeads: []string{}, Reason: "Misfiled"}
		heads := func(n int) []string {
			out := []string{}
			for i := 0; i < n; i++ {
				out = append(out, NewID("revision"))
			}
			return out
		}
		for name, f := range map[string]struct {
			edit   func(*MoveWrite)
			target error
		}{
			"bad record ID":          {func(m *MoveWrite) { m.RecordID = "Bad ID" }, ErrInvalidInput},
			"missing record":         {func(m *MoveWrite) { m.RecordID = "record-missing" }, ErrInvalidInput},
			"bad scope kind":         {func(m *MoveWrite) { m.Scope = Scope{Kind: "folder", ID: "example-y"} }, ErrInvalidInput},
			"another signet":         {func(m *MoveWrite) { m.Scope = Scope{Kind: "signet", ID: "signet-other"} }, ErrInvalidInput},
			"blank reason":           {func(m *MoveWrite) { m.Reason = " " }, ErrInvalidInput},
			"reason one byte over":   {func(m *MoveWrite) { m.Reason = strings.Repeat("a", MaxReasonBytes+1) }, ErrInvalidInput},
			"no content heads":       {func(m *MoveWrite) { m.ContentHeads = []string{} }, ErrInvalidInput},
			"33 content heads":       {func(m *MoveWrite) { m.ContentHeads = heads(33) }, ErrInvalidInput},
			"omitted move heads":     {func(m *MoveWrite) { m.MoveHeads = nil }, ErrInvalidInput},
			"duplicate content head": {func(m *MoveWrite) { m.ContentHeads = []string{r.ID, r.ID} }, ErrInvalidInput},
			"stale content head":     {func(m *MoveWrite) { m.ContentHeads = []string{"revision-stale"} }, ErrStaleHeads},
			"stale move head":        {func(m *MoveWrite) { m.MoveHeads = []string{"move-stale"} }, ErrStaleHeads},
		} {
			in := ok
			f.edit(&in)
			before := moveFiles(t, s.Root())
			if _, err := s.MoveIdempotent("key-"+name, in); !errors.Is(err, f.target) {
				t.Fatalf("%s: %v", name, err)
			}
			if after := moveFiles(t, s.Root()); len(after) != len(before) {
				t.Fatalf("%s wrote %v", name, after)
			}
		}
		in := ok
		in.Reason = strings.Repeat("a", MaxReasonBytes)
		in.Scope = Scope{Kind: "signet", ID: s.ID()}
		if r, err := s.Move(in); err != nil || !r.Moved {
			t.Fatalf("reason at the limit, signet-wide: %+v %v", r, err)
		}
		// Moving to the scope it is already in is a no-op success.
		h := history(t, s, r.RecordID, 0, 50)
		before := moveFiles(t, s.Root())
		noop, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: Scope{Kind: "signet", ID: s.ID()}, ContentHeads: []string{r.ID}, MoveHeads: h.MoveHeads, Reason: "Again"})
		if err != nil || noop.Moved || noop.ID != "" {
			t.Fatalf("no-op: %+v %v", noop, err)
		}
		if after := moveFiles(t, s.Root()); len(after) != len(before) {
			t.Fatal("no-op wrote a file")
		}
	})
}

func TestMoveRefusesAContentConflictAndAWithdrawnRecord(t *testing.T) {
	s := fixture(t)
	r := saved(t, s, scopeX, "Copper Finch")
	a := corrected(t, s, scopeX, r, "Silver Heron")
	b := corrected(t, s, scopeX, r, "Amber Lark") // format 1 keeps the fork
	_, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: scopeY, ContentHeads: []string{a.ID, b.ID}, MoveHeads: []string{}, Reason: "Misfiled"})
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "settle the conflict with a correction") {
		t.Fatalf("content conflict: %v", err)
	}

	two := formatTwoService(t)
	w := saved(t, two, scopeX, "Copper Finch")
	if _, err := two.ChangeVisibility(t.Context(), "withdraw", VisibilityWrite{RecordID: w.RecordID, ContentHeads: []string{w.ID}, VisibilityHeads: []string{}, Reason: "Synthetic"}); err != nil {
		t.Fatal(err)
	}
	_, err = two.Move(MoveWrite{RecordID: w.RecordID, Scope: scopeY, ContentHeads: []string{w.ID}, MoveHeads: []string{}, Reason: "Misfiled"})
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "restore it before moving it") {
		t.Fatalf("withdrawn: %v", err)
	}
	if files := moveFiles(t, two.Root()); len(files) != 0 {
		t.Fatalf("refused moves wrote %v", files)
	}
}

// #250 acceptance 3: a repeated request_id returns the first result, for a
// move and for a no-op, even after a later move.
func TestMoveIdempotency(t *testing.T) {
	s := fixture(t)
	r := saved(t, s, scopeX, "Copper Finch")
	in := MoveWrite{RecordID: r.RecordID, Scope: scopeY, ContentHeads: []string{r.ID}, MoveHeads: []string{}, Reason: "Misfiled"}
	first, err := s.MoveIdempotent("key-move-first", in)
	if err != nil || !first.Moved {
		t.Fatal(first, err)
	}
	noopIn := MoveWrite{RecordID: r.RecordID, Scope: scopeY, ContentHeads: []string{r.ID}, MoveHeads: []string{first.ID}, Reason: "Already there"}
	noop, err := s.MoveIdempotent("key-move-noop", noopIn)
	if err != nil || noop.Moved || noop.Scope != scopeY {
		t.Fatal(noop, err)
	}
	if pending, err := s.store.PendingIntents(); err != nil || len(pending) != 0 {
		t.Fatalf("pending after a move and a no-op: %v %v", pending, err)
	}
	later := moved(t, s, r.RecordID, scopeZ, []string{r.ID}, []string{first.ID})
	if again, err := s.MoveIdempotent("key-move-first", in); err != nil || again != first {
		t.Fatalf("retried move: %+v %v", again, err)
	}
	if again, err := s.MoveIdempotent("key-move-noop", noopIn); err != nil || again != noop {
		t.Fatalf("retried no-op after a later move: %+v %v", again, err)
	}
	if h := history(t, s, r.RecordID, 0, 50); h.MoveCount != 2 || h.MoveHeads[0] != later.ID {
		t.Fatalf("retries wrote moves: %+v", h)
	}
	changed := in
	changed.Reason = "Different"
	if _, err := s.MoveIdempotent("key-move-first", changed); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("changed input: %v", err)
	}
	// Request IDs can't cross tools, either way.
	if _, err := s.RememberIdempotent("key-move-first", Write{Kind: "fact", Summary: "s", Body: "b", Basis: "observation", Reason: "r"}); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("remember reusing a move key: %v", err)
	}
	if _, err := s.AppendJournalIdempotent("key-move-first", "outcome", "s", nil); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("journal reusing a move key: %v", err)
	}
	if _, err := s.NameScopeIdempotent("key-move-noop", ScopeLabelWrite{Scope: scopeY, Name: "n", Description: "d", Reason: "r"}); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("naming reusing a move key: %v", err)
	}
	if _, err := s.RememberIdempotent("key-remember", Write{Kind: "fact", Summary: "s", Body: "b", Basis: "observation", Reason: "r"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MoveIdempotent("key-remember", in); !errors.Is(err, ErrRequestMismatch) {
		t.Fatalf("move reusing a remember key: %v", err)
	}
	// An interrupted move is pending until its retry publishes it.
	if err := os.Remove(filepath.Join(s.Root(), "memory/moves", r.RecordID, first.ID+".json")); err != nil {
		t.Fatal(err)
	}
	if pending, err := s.store.PendingIntents(); err != nil || len(pending) != 1 {
		t.Fatalf("pending after losing the move: %v %v", pending, err)
	}
	if _, err := s.MoveIdempotent("key-move-first", in); err != nil {
		t.Fatal(err)
	}
	if pending, err := s.store.PendingIntents(); err != nil || len(pending) != 0 {
		t.Fatalf("pending after the retry: %v %v", pending, err)
	}
}

// concurrentMove writes a move as another clone would: same parents as the
// current heads, without this clone's guards.
func concurrentMove(t *testing.T, s *Service, recordID string, to Scope, content, parents []string) RecordMove {
	t.Helper()
	m := RecordMove{Version: 1, ID: NewID("move"), RecordID: recordID, Scope: to, Observed: content, Parents: parents, Reason: "Moved on another clone", RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author}
	if err := s.store.publishMove(m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMoveConflictsAreListedAndSettled(t *testing.T) {
	bothFormats(t, func(t *testing.T, s *Service) {
		r := saved(t, s, scopeX, "Copper Finch")
		a := moved(t, s, r.RecordID, scopeY, []string{r.ID}, []string{})
		b := concurrentMove(t, s, r.RecordID, scopeZ, []string{r.ID}, []string{})
		for _, scope := range []Scope{scopeY, scopeZ} {
			current, conflicts := recallIDs(t, s, scope)
			if len(current) != 0 || len(conflicts) != 1 {
				t.Fatalf("recall in %v: %v %v", scope, current, conflicts)
			}
			current, conflicts = primeIDs(t, s, scope)
			if len(current) != 0 || len(conflicts) != 1 {
				t.Fatalf("prime in %v: %v %v", scope, current, conflicts)
			}
			info, ok := scopeInfo(t, s, scope)
			if !ok || info.RecordCount != 1 || info.MoveConflicts != 1 {
				t.Fatalf("counts in %v: %+v", scope, info)
			}
			if s.store.Signet.Version == 2 && (info.Visibility.Conflicted != 1 || info.Visibility.Visible != 0) {
				t.Fatalf("visibility counts in %v: %+v", scope, info.Visibility)
			}
		}
		p, err := s.Recall("", &scopeY, 50, 32768)
		if err != nil || len(p.Conflicts) != 1 || len(p.Conflicts[0].MoveScopes) != 2 {
			t.Fatalf("conflict entry: %+v %v", p.Conflicts, err)
		}
		if current, conflicts := recallIDs(t, s, scopeX); len(current)+len(conflicts) != 0 {
			t.Fatal("the original scope still lists the record")
		}
		h := history(t, s, r.RecordID, 0, 50)
		if h.Scope != nil || len(h.MoveHeads) != 2 || len(h.Moves) != 2 {
			t.Fatalf("history during the conflict: %+v", h)
		}
		if _, err := s.Remember(Write{Kind: "fact", Scope: &scopeY, RecordID: r.RecordID, Supersedes: []string{r.ID}, Summary: "s", Body: "b", Basis: "observation", Reason: "r"}); !errors.Is(err, ErrMoveConflict) {
			t.Fatalf("correction during a move conflict: %v", err)
		}
		if _, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: scopeY, ContentHeads: []string{r.ID}, MoveHeads: []string{a.ID}, Reason: "Settle"}); !errors.Is(err, ErrStaleHeads) {
			t.Fatalf("a move naming one head: %v", err)
		}
		// Settling into one of the heads' scopes still writes a move.
		settle := moved(t, s, r.RecordID, scopeY, []string{r.ID}, []string{a.ID, b.ID})
		if !settle.Moved {
			t.Fatal("settling move was a no-op")
		}
		if current, _ := recallIDs(t, s, scopeY); len(current) != 1 {
			t.Fatal("settled record not current in its scope")
		}
		if _, ok := scopeInfo(t, s, scopeZ); ok {
			t.Fatal("settled record still counted in the other scope")
		}
		corrected(t, s, scopeY, r, "Silver Heron")
	})
}

// Two clones that move a record to the same scope agree: no conflict.
func TestAgreeingMovesAreNotAConflict(t *testing.T) {
	s := fixture(t)
	r := saved(t, s, scopeX, "Copper Finch")
	a := moved(t, s, r.RecordID, scopeY, []string{r.ID}, []string{})
	b := concurrentMove(t, s, r.RecordID, scopeY, []string{r.ID}, []string{})
	if current, conflicts := recallIDs(t, s, scopeY); len(current) != 1 || len(conflicts) != 0 {
		t.Fatalf("agreeing heads: %v %v", current, conflicts)
	}
	if info, _ := scopeInfo(t, s, scopeY); info.MoveConflicts != 0 || info.RecordCount != 1 {
		t.Fatalf("counts: %+v", info)
	}
	corrected(t, s, scopeY, r, "Silver Heron")
	h := history(t, s, r.RecordID, 0, 50)
	if len(h.MoveHeads) != 2 || h.Scope == nil || *h.Scope != scopeY {
		t.Fatalf("history: %+v", h)
	}
	content := []string{h.Items[1].ID}
	if _, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: scopeZ, ContentHeads: content, MoveHeads: []string{a.ID}, Reason: "Next"}); !errors.Is(err, ErrStaleHeads) {
		t.Fatalf("one of two agreeing heads: %v", err)
	}
	moved(t, s, r.RecordID, scopeZ, content, []string{a.ID, b.ID})
}

// H1 of the plan review: a record with a content conflict and a move
// conflict is settled by a move naming every head, then a correction.
func TestBothConflictsCanBeSettled(t *testing.T) {
	s := fixture(t)
	r := saved(t, s, scopeX, "Copper Finch")
	a := corrected(t, s, scopeX, r, "Silver Heron")
	b := corrected(t, s, scopeX, r, "Amber Lark")
	ma := concurrentMove(t, s, r.RecordID, scopeY, []string{a.ID}, []string{})
	mb := concurrentMove(t, s, r.RecordID, scopeZ, []string{b.ID}, []string{})
	if _, err := s.Remember(Write{Kind: "fact", Scope: &scopeY, RecordID: r.RecordID, Supersedes: []string{a.ID, b.ID}, Summary: "s", Body: "b", Basis: "observation", Reason: "r"}); !errors.Is(err, ErrMoveConflict) {
		t.Fatalf("correction first: %v", err)
	}
	if _, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: scopeY, ContentHeads: []string{a.ID}, MoveHeads: []string{ma.ID, mb.ID}, Reason: "Settle"}); !errors.Is(err, ErrStaleHeads) {
		t.Fatalf("a settling move missing a content head: %v", err)
	}
	moved(t, s, r.RecordID, scopeY, []string{a.ID, b.ID}, []string{ma.ID, mb.ID})
	if _, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: scopeZ, ContentHeads: []string{a.ID, b.ID}, MoveHeads: history(t, s, r.RecordID, 0, 1).MoveHeads, Reason: "Again"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("a move with only the content conflict left: %v", err)
	}
	if _, err := s.Remember(Write{Kind: "fact", Scope: &scopeY, RecordID: r.RecordID, Supersedes: []string{a.ID, b.ID}, Summary: "Synthetic codename", Body: "Settled", Basis: "user-direction", Reason: "Settled"}); err != nil {
		t.Fatal(err)
	}
	if current, conflicts := recallIDs(t, s, scopeY); len(current) != 1 || len(conflicts) != 0 {
		t.Fatalf("after settling both: %v %v", current, conflicts)
	}
}

func TestMoveValidation(t *testing.T) {
	s := fixture(t)
	r := saved(t, s, scopeX, "Copper Finch")
	other := saved(t, s, scopeX, "Other")
	records, err := s.store.revisions()
	if err != nil {
		t.Fatal(err)
	}
	known := func(id string) error {
		if id != s.author.DeviceID {
			return errors.New("unknown device")
		}
		return nil
	}
	base := RecordMove{Version: 1, ID: "move-one", RecordID: r.RecordID, Scope: scopeY, Observed: []string{r.ID}, Parents: []string{}, Reason: "r", RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: s.author}
	second := base
	second.ID, second.Parents = "move-two", []string{"move-one"}
	if _, err := ResolvePlacements(records, []RecordMove{base, second}, s.ID(), known); err != nil {
		t.Fatal(err)
	}
	root2 := base
	root2.ID = "move-root"
	if p, err := ResolvePlacements(records, []RecordMove{base, root2}, s.ID(), known); err != nil || p[r.RecordID].Conflicted() {
		t.Fatalf("two agreeing roots: %+v %v", p, err)
	}
	for name, moves := range map[string][]RecordMove{
		"missing parent":     {func() RecordMove { m := base; m.Parents = []string{"move-missing"}; return m }()},
		"foreign parent":     {func() RecordMove { m := base; m.RecordID, m.Observed = other.RecordID, []string{other.ID}; return m }(), func() RecordMove { m := second; return m }()},
		"cycle":              {func() RecordMove { m := base; m.Parents = []string{"move-two"}; return m }(), second},
		"foreign observed":   {func() RecordMove { m := base; m.Observed = []string{other.ID}; return m }()},
		"missing record":     {func() RecordMove { m := base; m.RecordID = "record-missing"; return m }()},
		"unknown device":     {func() RecordMove { m := base; m.Authorship.DeviceID = "device-other"; return m }()},
		"another signet":     {func() RecordMove { m := base; m.Scope = Scope{Kind: "signet", ID: "signet-other"}; return m }()},
		"no observed heads":  {func() RecordMove { m := base; m.Observed = []string{}; return m }()},
		"missing parent nil": {func() RecordMove { m := base; m.Parents = nil; return m }()},
		"schema version":     {func() RecordMove { m := base; m.Version = 2; return m }()},
		"duplicate":          {base, base},
	} {
		if _, err := ResolvePlacements(records, moves, s.ID(), known); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	// A move at the wrong path refuses every read.
	m := moved(t, s, r.RecordID, scopeY, []string{r.ID}, []string{})
	from := filepath.Join(s.Root(), "memory/moves", r.RecordID, m.ID+".json")
	to := filepath.Join(s.Root(), "memory/moves", other.RecordID, m.ID+".json")
	if err := os.MkdirAll(filepath.Dir(to), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err == nil {
		t.Fatal("noncanonical move path validated")
	}
	if _, err := s.Recall("", &scopeY, 5, 8192); err == nil {
		t.Fatal("recall read a noncanonical move")
	}
}

// Moves sit on the first history page only, in their own budget: the
// revisions on each page are the ones 2.2.0's 32 KiB page holds.
func TestHistoryPagesAndMoves(t *testing.T) {
	s := fixture(t)
	body := strings.Repeat("Synthetic body text. ", 380) // about 8 KiB
	r := saved(t, s, scopeX, body)
	prev := r
	for i := 0; i < 5; i++ {
		prev = corrected(t, s, scopeX, prev, body+string(rune('a'+i)))
	}
	items, err := s.History(r.RecordID)
	if err != nil {
		t.Fatal(err)
	}
	want, err := page(items, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if !want.Truncated || len(want.Items) < 2 {
		t.Fatalf("fixture doesn't reach the page budget: %d items", len(want.Items))
	}
	heads := []string{}
	reason := strings.Repeat("r", MaxReasonBytes)
	for i := 0; i < 12; i++ {
		to := scopeY
		if i%2 == 1 {
			to = scopeZ
		}
		m, err := s.Move(MoveWrite{RecordID: r.RecordID, Scope: to, ContentHeads: []string{prev.ID}, MoveHeads: heads, Reason: reason})
		if err != nil {
			t.Fatal(err)
		}
		heads = []string{m.ID}
	}
	first := history(t, s, r.RecordID, 0, 50)
	if len(first.Items) != len(want.Items) || first.Items[len(first.Items)-1].ID != want.Items[len(want.Items)-1].ID || *first.NextOffset != *want.NextOffset {
		t.Fatalf("first page holds %d revisions, 2.2.0's page %d", len(first.Items), len(want.Items))
	}
	if !first.MovesTruncated || len(first.Moves) == 0 || len(first.Moves) >= 12 || first.MoveCount != 12 || first.Moves[0].ID != heads[0] {
		t.Fatalf("moves: %d kept, truncated %v, count %d", len(first.Moves), first.MovesTruncated, first.MoveCount)
	}
	b, err := json.Marshal(first)
	if err != nil || len(b) > 65536-1024 {
		t.Fatalf("first page is %d bytes", len(b))
	}
	next := history(t, s, r.RecordID, *first.NextOffset, 50)
	if len(next.Moves) != 0 || next.MoveCount != 12 || len(next.MoveHeads) != 1 || next.Scope == nil {
		t.Fatalf("later page: %+v", next.Moves)
	}
}

// Synchronization counts a move conflict once; a withdrawn record's moves
// aren't counted as a conflict, as its scopes count it withheld.
func TestMoveConflictCounts(t *testing.T) {
	s := formatTwoService(t)
	r := saved(t, s, scopeX, "Copper Finch")
	moved(t, s, r.RecordID, scopeY, []string{r.ID}, []string{})
	concurrentMove(t, s, r.RecordID, scopeZ, []string{r.ID}, []string{})
	if records, listings, err := s.store.MoveConflicts(); err != nil || records != 1 || listings != 2 {
		t.Fatalf("visible move conflict: %d %d %v", records, listings, err)
	}
	if _, err := s.ChangeVisibility(t.Context(), "withdraw", VisibilityWrite{RecordID: r.RecordID, ContentHeads: []string{r.ID}, VisibilityHeads: []string{}, Reason: "Synthetic"}); err != nil {
		t.Fatal(err)
	}
	if records, listings, err := s.store.MoveConflicts(); err != nil || records != 0 || listings != 0 {
		t.Fatalf("withdrawn move conflict: %d %d %v", records, listings, err)
	}
	if info, _ := scopeInfo(t, s, scopeY); info.Visibility.Withheld != 1 || info.Visibility.Conflicted != 0 || info.MoveConflicts != 1 {
		t.Fatalf("withdrawn counts: %+v %+v", info, info.Visibility)
	}
}
