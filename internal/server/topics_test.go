package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var topic = map[string]any{"kind": "project", "id": "atlas-launch"}

func decoded[T any](t *testing.T, text string) T {
	t.Helper()
	var v struct {
		OK     bool `json:"ok"`
		Result T    `json:"result"`
	}
	if err := json.Unmarshal([]byte(text), &v); err != nil || !v.OK {
		t.Fatalf("not a success: %s", text)
	}
	return v.Result
}

func nameArgs(id, name string, supersedes ...string) map[string]any {
	label := map[string]any{"scope": topic, "name": name, "description": "Shipping the synthetic Atlas launch checklist.", "reason": "Named in a synthetic test"}
	if len(supersedes) > 0 {
		label["supersedes"] = supersedes
	}
	return map[string]any{"request_id": id, "label": label}
}

func scopedRemember(id, kind, summary string) map[string]any {
	return map[string]any{"request_id": id, "record": map[string]any{"kind": kind, "summary": summary, "body": "Synthetic body for " + summary, "basis": "observation", "reason": "Observed in a synthetic test", "scope": topic}}
}

// #246: a writer names a topic and works in it; a different, read-only
// client finds it by name and primes on it in one call.
func TestTopicsOverMCP(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	writer, reader := connect(t, h, "/mcp", writeToken), connect(t, h, "/mcp", readToken)

	_, text := call(t, writer, "memory_name_scope", nameArgs("name-request-atlas-01", "Atlas launch"))
	label := receipt(t, text)
	if _, again := call(t, writer, "memory_name_scope", nameArgs("name-request-atlas-01", "Atlas launch")); receipt(t, again).ID != label.ID {
		t.Fatal("retry made a second label")
	}
	if e := failure(t, mustCall(t, writer, "memory_name_scope", nameArgs("name-request-atlas-01", "Different"))); e.Code != "request.conflict" {
		t.Fatal(e)
	}
	_, text = call(t, writer, "memory_remember", scopedRemember("remember-state-0001", "project-state", "Atlas: drafting the checklist"))
	state := receipt(t, text)
	for i, summary := range []string{"Checklist lives in the synthetic tracker", "Launch needs two reviewers"} {
		_, text = call(t, writer, "memory_remember", scopedRemember("remember-fact-000"+string(rune('1'+i)), "fact", summary))
		receipt(t, text)
	}
	for i, summary := range []string{"Drafted the first half", "Drafted the second half"} {
		_, text = call(t, writer, "memory_journal_append", map[string]any{"request_id": "journal-entry-000" + string(rune('1'+i)), "entry": map[string]any{"kind": "outcome", "summary": summary, "scope": topic}})
		receipt(t, text)
	}

	_, text = call(t, reader, "memory_scopes", map[string]any{"query": "Atlas launch"})
	scopes := decoded[memory.Page[memory.ScopeInfo]](t, text)
	if len(scopes.Items) == 0 || scopes.Items[0].Scope.ID != "atlas-launch" || scopes.Items[0].Labels[0].Name != "Atlas launch" {
		t.Fatalf("find by name: %s", text)
	}
	_, text = call(t, reader, "memory_prime", map[string]any{"scope": topic})
	p := decoded[memory.PrimePacket](t, text)
	if len(p.Labels) != 1 || p.Labels[0].Description == "" || len(p.State) != 1 || p.State[0].ID != state.ID || len(p.Current) != 2 || len(p.Journal) != 2 {
		t.Fatalf("prime: %s", text)
	}
	if strings.Index(text, "Shipping the synthetic Atlas") > strings.Index(text, "Atlas: drafting") {
		t.Fatal("prime does not lead with the description")
	}

	// Correct the state; the reader's next prime shows only the new one.
	correction := scopedRemember("remember-state-0002", "project-state", "Atlas: reviewing the checklist")
	correction["record"].(map[string]any)["record_id"] = state.RecordID
	correction["record"].(map[string]any)["supersedes"] = []string{state.ID}
	_, text = call(t, writer, "memory_remember", correction)
	corrected := receipt(t, text)
	_, text = call(t, reader, "memory_prime", map[string]any{"scope": topic})
	if p = decoded[memory.PrimePacket](t, text); len(p.State) != 1 || p.State[0].ID != corrected.ID || strings.Contains(text, "drafting the checklist") {
		t.Fatalf("after correction: %s", text)
	}
	_, text = call(t, reader, "memory_journal", map[string]any{"scope": topic, "limit": 5})
	if j := decoded[memory.Page[memory.JournalEntry]](t, text); len(j.Items) != 2 || j.Items[0].Scope == nil {
		t.Fatalf("scoped journal: %s", text)
	}

	// The read-only client isn't served naming or journal writes at all.
	for name, args := range map[string]map[string]any{
		"memory_name_scope":     nameArgs("name-request-reader-1", "Reader"),
		"memory_journal_append": {"request_id": "journal-reader-00001", "entry": map[string]any{"kind": "outcome", "summary": "s", "scope": topic}},
	} {
		if _, err := reader.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args}); err == nil || !strings.Contains(err.Error(), "unknown tool") {
			t.Fatalf("read-only %s: %v", name, err)
		}
	}
	// Input errors are input.invalid with nothing written; stale heads aren't.
	before := signetFiles(t, a.Root)
	bad := nameArgs("name-request-bad-0001", strings.Repeat("n", 257), label.ID)
	if e := failure(t, mustCall(t, writer, "memory_name_scope", bad)); e.Code != "input.invalid" || e.WriteMayHaveOccurred || !strings.Contains(e.Message, "name is 257 bytes") {
		t.Fatal(e)
	}
	if signetFiles(t, a.Root) != before {
		t.Fatal("refused naming wrote files")
	}
	if e := failure(t, mustCall(t, writer, "memory_journal_append", map[string]any{"request_id": "journal-signet-00001", "entry": map[string]any{"kind": "k", "summary": "s", "scope": map[string]any{"kind": "signet", "id": a.SignetID}}})); e.Code != "input.invalid" {
		t.Fatal(e)
	}
	_, text = call(t, writer, "memory_name_scope", nameArgs("name-request-atlas-02", "Atlas rollout", label.ID))
	receipt(t, text)
}

func mustCall(t *testing.T, session *sdk.ClientSession, name string, args map[string]any) string {
	t.Helper()
	_, text := call(t, session, name, args)
	return text
}

// listTools maps each listed tool to its input schema's properties.
func listTools(t *testing.T, h *httptest.Server, token string) map[string]map[string]any {
	t.Helper()
	list, err := connect(t, h, "/mcp", token).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]any{}
	for _, tool := range list.Tools {
		b, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatal(err)
		}
		out[tool.Name] = schema.Properties
	}
	return out
}

func TestTopicToolSchemasStateLimits(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	tools := listTools(t, h, writeToken)
	label := tools["memory_name_scope"]["label"].(map[string]any)["properties"].(map[string]any)
	for field, limit := range map[string]float64{"name": 256, "description": 1024, "reason": 1024} {
		p := label[field].(map[string]any)
		if p["maxLength"] != limit || p["minLength"] != float64(1) {
			t.Fatalf("%s: %v", field, p)
		}
	}
	entry := tools["memory_journal_append"]["entry"].(map[string]any)["properties"].(map[string]any)
	if entry["scope"] == nil || tools["memory_journal"]["scope"] == nil || tools["memory_scopes"]["query"] == nil || tools["memory_prime"]["budget_bytes"] == nil {
		t.Fatal("new parameters missing from tools/list")
	}
	if _, ok := listTools(t, h, readToken)["memory_prime"]; !ok {
		t.Fatal("read clients don't get memory_prime")
	}
}

// #232 stays independent: names and scoped journal entries change nothing in
// memory_context's selection or packing.
func TestMemoryContextIgnoresTopicData(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	writer := connect(t, h, "/mcp", writeToken)
	call(t, writer, "memory_remember", scopedRemember("context-topic-0001", "fact", "Atlas checklist lives in the synthetic tracker"))
	call(t, writer, "memory_remember", rememberArgs("context-topic-0002", "The zebrafinch build cache rotates on Tuesdays."))
	events := []map[string]any{
		{"event": "start"},
		{"event": "prompt", "prompt": "Where does the Atlas launch checklist live?"},
		{"event": "prompt", "prompt": "When does the zebrafinch cache rotate?"},
	}
	snapshot := func() []string {
		out := []string{}
		for _, args := range events {
			// A fresh connection is a fresh hook session.
			_, text := call(t, connect(t, h, "/mcp", writeToken), "memory_context", args)
			out = append(out, text)
		}
		return out
	}
	before := snapshot()
	if !strings.Contains(before[0], "atlas-launch") || !strings.Contains(before[2], "zebrafinch") {
		t.Fatalf("context fixture injects nothing: %v", before)
	}
	receipt(t, mustCall(t, writer, "memory_name_scope", nameArgs("context-name-00001", "Atlas launch")))
	other := map[string]any{"request_id": "context-name-00002", "label": map[string]any{"scope": map[string]any{"kind": "task", "id": "named-only"}, "name": "Atlas side task", "description": "Atlas launch checklist side work.", "reason": "r"}}
	receipt(t, mustCall(t, writer, "memory_name_scope", other))
	receipt(t, mustCall(t, writer, "memory_journal_append", map[string]any{"request_id": "context-journal-0001", "entry": map[string]any{"kind": "outcome", "summary": "Atlas launch checklist drafted", "scope": topic}}))
	after := snapshot()
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("memory_context changed for %v:\nbefore %s\nafter  %s", events[i], before[i], after[i])
		}
	}
}
