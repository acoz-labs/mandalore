package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	projectX = map[string]any{"kind": "project", "id": "example-x"}
	projectY = map[string]any{"kind": "project", "id": "example-y"}
)

func moveArgs(id, recordID string, scope map[string]any, content, moves []string) map[string]any {
	return map[string]any{"request_id": id, "move": map[string]any{"record_id": recordID, "scope": scope, "expected_content_heads": content, "expected_move_heads": moves, "reason": "Filed under the wrong project"}}
}

func inScope(id, summary string, scope map[string]any, recordID string, supersedes ...string) map[string]any {
	record := map[string]any{"kind": "fact", "summary": summary, "body": "Synthetic body for " + summary, "basis": "user-direction", "reason": "Synthetic test", "scope": scope}
	if recordID != "" {
		record["record_id"], record["supersedes"] = recordID, supersedes
	}
	return map[string]any{"request_id": id, "record": record}
}

// #250 acceptance 1, 2, 3 and 5 against a running server.
func TestMoveOverMCP(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	writer, reader := connect(t, h, "/mcp", writeToken), connect(t, h, "/mcp", readToken)

	first := receipt(t, mustCall(t, writer, "memory_remember", inScope("move-remember-00001", "Synthetic codename", projectX, "")))
	second := receipt(t, mustCall(t, writer, "memory_remember", inScope("move-remember-00002", "Synthetic codename", projectX, first.RecordID, first.ID)))

	args := moveArgs("move-request-000001", first.RecordID, projectY, []string{second.ID}, []string{})
	text := mustCall(t, writer, "memory_move", args)
	m := decoded[MoveReceipt](t, text)
	if !m.Moved || m.RecordID != first.RecordID || m.Scope.ID != "example-y" || m.ID == "" || !m.Durable {
		t.Fatalf("move receipt: %s", text)
	}
	if again := decoded[MoveReceipt](t, mustCall(t, writer, "memory_move", args)); again.ID != m.ID {
		t.Fatal("retry made a second move")
	}
	changed := moveArgs("move-request-000001", first.RecordID, projectX, []string{second.ID}, []string{m.ID})
	if e := failure(t, mustCall(t, writer, "memory_move", changed)); e.Code != "request.conflict" {
		t.Fatal(e)
	}

	for _, c := range []struct {
		scope map[string]any
		want  int
	}{{projectY, 1}, {projectX, 0}} {
		p := decoded[memory.RecallPacket](t, mustCall(t, reader, "memory_recall", map[string]any{"scope": c.scope}))
		if len(p.Current) != c.want || c.want == 1 && (p.Current[0].RecordID != first.RecordID || p.Current[0].Scope.ID != "example-y") {
			t.Fatalf("recall in %v: %+v", c.scope, p)
		}
		prime := decoded[memory.PrimePacket](t, mustCall(t, reader, "memory_prime", map[string]any{"scope": c.scope}))
		if len(prime.Current) != c.want || c.want == 1 && prime.Current[0].RecordID != first.RecordID {
			t.Fatalf("prime in %v: %+v", c.scope, prime)
		}
	}
	text = mustCall(t, reader, "memory_history", map[string]any{"record_id": first.RecordID})
	hist := decoded[memory.RecordHistory](t, text)
	if len(hist.Items) != 2 || hist.Scope == nil || hist.Scope.ID != "example-y" || len(hist.Moves) != 1 {
		t.Fatalf("history: %s", text)
	}
	if mv := hist.Moves[0]; mv.Reason != "Filed under the wrong project" || mv.Authorship.Harness == "" || mv.Authorship.DeviceID == "" || mv.From[0].ID != "example-x" {
		t.Fatalf("history move: %+v", mv)
	}

	// A correction in the new scope succeeds; one in the old scope is refused.
	if e := failure(t, mustCall(t, writer, "memory_remember", inScope("move-remember-00003", "Synthetic codename", projectX, first.RecordID, second.ID))); e.Code != "input.invalid" || !strings.Contains(e.Message, "moved to project example-y") {
		t.Fatal(e)
	}
	receipt(t, mustCall(t, writer, "memory_remember", inScope("move-remember-00004", "Synthetic codename", projectY, first.RecordID, second.ID)))

	// A stale head is refused.
	before := signetFiles(t, a.Root)
	if e := failure(t, mustCall(t, writer, "memory_move", moveArgs("move-request-stale01", first.RecordID, projectX, []string{second.ID}, []string{m.ID}))); e.Code != "memory.stale_heads" {
		t.Fatal(e)
	}
	if signetFiles(t, a.Root) != before {
		t.Fatal("refused move wrote files")
	}
	// The withdrawn-record refusal needs format 2; this server fixture is
	// format 1, so TestMoveRefusesAContentConflictAndAWithdrawnRecord in
	// internal/memory covers it.

	// A read-only client isn't served memory_move.
	if _, err := reader.CallTool(context.Background(), &sdk.CallToolParams{Name: "memory_move", Arguments: args}); err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("read-only memory_move: %v", err)
	}
}

func TestMoveToolSchemaStatesLimits(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	props := listTools(t, h, writeToken)["memory_move"]
	move, _ := props["move"].(map[string]any)
	reason, _ := move["properties"].(map[string]any)["reason"].(map[string]any)
	if reason["maxLength"] != float64(memory.MaxReasonBytes) {
		t.Fatalf("reason schema: %v", reason)
	}
	if _, ok := listTools(t, h, readToken)["memory_move"]; ok {
		t.Fatal("read-only client lists memory_move")
	}
}

// A canon source with a moved record serves it in its current scope, and
// promotion's head lookup finds it there (#250).
func TestCanonServesAMovedRecordWhereItIsNow(t *testing.T) {
	p := testfixture.NewServerSignet(t, "primary")
	c := newCanonSource(t)
	x, y := memory.Scope{Kind: "project", ID: "example-x"}, memory.Scope{Kind: "project", ID: "example-y"}
	r, err := c.memory.Remember(memory.Write{Kind: "fact", Scope: &x, Summary: "Moved canon convention", Body: "Moved: synthetic fixtures only.", Basis: "user-direction", Reason: "Synthetic canon fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.memory.Move(memory.MoveWrite{RecordID: r.RecordID, Scope: y, ContentHeads: []string{r.ID}, MoveHeads: []string{}, Reason: "Misfiled"}); err != nil {
		t.Fatal(err)
	}
	testfixture.Git(t, c.Root, "add", "-A")
	testfixture.Git(t, c.Root, "commit", "-m", "Move canon record")
	testfixture.Git(t, c.Root, "push", c.Remote, "main")
	c.refs = refsOf(t, c.RemoteDir)
	id := registerCanon(t, p, c)
	config := testConfig(t, p)
	config.Signets[0].Canon = []string{id}
	_, url := startCanonServer(t, config, time.Hour)
	reader := connect(t, url, "/mcp", readToken)
	recallIn := func(scope map[string]any) canonRecall {
		_, text := call(t, reader, "foundling_canon_recall", map[string]any{"foundling_id": id, "query": "moved canon convention", "scope": scope})
		var v canonRecall
		if err := json.Unmarshal([]byte(text), &v); err != nil {
			t.Fatal(text, err)
		}
		return v
	}
	var got canonRecall
	eventually(t, 10*time.Second, "startup refresh serves canon", func() bool { got = recallIn(projectY); return got.OK })
	if !recalls(got, "Moved: synthetic fixtures only.") || got.Result.Memory.Current[0].Scope != y {
		t.Fatalf("canon recall in the new scope: %+v", got.Result.Memory)
	}
	if old := recallIn(projectX); !old.OK || len(old.Result.Memory.Current) != 0 {
		t.Fatalf("canon recall in the old scope: %+v", old.Result.Memory)
	}
	_, text := call(t, reader, "foundling_canon_heads", map[string]any{"foundling_id": id, "record_id": r.RecordID})
	var heads struct {
		OK     bool `json:"ok"`
		Result struct {
			Heads []json.RawMessage `json:"heads"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &heads); err != nil || !heads.OK || len(heads.Result.Heads) != 1 {
		t.Fatalf("canon heads of the moved record: %s %v", text, err)
	}
}

// writeConcurrentMove writes a move as another clone would, with the given
// parents, so the record's moves conflict.
func writeConcurrentMove(t *testing.T, root string, author memory.Authorship, recordID string, to memory.Scope, observed, parents []string) {
	t.Helper()
	m := memory.RecordMove{Version: 1, ID: memory.NewID("move"), RecordID: recordID, Scope: to, Observed: observed, Parents: parents, Reason: "Moved on another clone", RecordedAt: time.Now().UTC().Format(time.RFC3339Nano), Authorship: author}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "memory/moves", recordID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, m.ID+".json"), b, 0600); err != nil {
		t.Fatal(err)
	}
}

// The context hook names a move conflict, and keeps the revision count when
// the record's revisions conflict too.
func TestMemoryContextNamesMoveConflicts(t *testing.T) {
	m, _ := contextService(t)
	x, y, z := memory.Scope{Kind: "project", ID: "example-x"}, memory.Scope{Kind: "project", ID: "example-y"}, memory.Scope{Kind: "account", ID: "example-z"}
	save := func(summary string) memory.Revision {
		r, err := m.Remember(memory.Write{Kind: "decision", Scope: &x, Summary: summary, Body: "Synthetic body for " + summary, Basis: "user-direction", Reason: "Synthetic"})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	moved := save("Heron release checklist")
	author := moved.Authorship
	if _, err := m.Move(memory.MoveWrite{RecordID: moved.RecordID, Scope: y, ContentHeads: []string{moved.ID}, MoveHeads: []string{}, Reason: "Misfiled"}); err != nil {
		t.Fatal(err)
	}
	writeConcurrentMove(t, m.Root(), author, moved.RecordID, z, []string{moved.ID}, []string{})
	text := contextFor(t, m, "prompt", "What is on the Heron release checklist?")
	if want := "- Conflict: " + moved.RecordID + " has moves to 2 different scopes; inspect memory_history"; !strings.Contains(text, want) {
		t.Fatalf("move conflict line missing:\n%s", text)
	}

	both := save("Lark rollout checklist")
	for _, body := range []string{"Silver", "Amber"} {
		if _, err := m.Remember(memory.Write{Kind: "decision", Scope: &x, RecordID: both.RecordID, Supersedes: []string{both.ID}, Summary: "Lark rollout checklist", Body: body, Basis: "user-direction", Reason: "Synthetic fork"}); err != nil {
			t.Fatal(err)
		}
	}
	h, err := m.HistoryPage(both.RecordID, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	heads := []string{h.Items[1].ID, h.Items[2].ID}
	writeConcurrentMove(t, m.Root(), author, both.RecordID, y, heads, []string{})
	writeConcurrentMove(t, m.Root(), author, both.RecordID, z, heads, []string{})
	text = contextFor(t, m, "prompt", "What is on the Lark rollout checklist?")
	if want := "- Conflict: " + both.RecordID + " has 2 competing revisions and moves to 2 different scopes;"; !strings.Contains(text, want) {
		t.Fatalf("double conflict line missing:\n%s", text)
	}
}
