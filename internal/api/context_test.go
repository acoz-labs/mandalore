package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestNativeContextIsBoundReadOnlyAndNotAMemoryTool(t *testing.T) {
	var selected *Operation
	for _, op := range Catalog() {
		if op.Name == "memory_context" {
			selected = &op
		}
	}
	if selected == nil || !selected.RequiresBinding || !selected.ReadOnly || selected.Network || !selected.CLIOnly {
		t.Fatal("missing CLI-only local context contract", selected)
	}
	out := New(nil, true).Call(context.Background(), "memory_context", []byte(`{}`))
	if out.OK || out.Error == nil || out.Error.Code != "binding.invalid" {
		t.Fatal("context did not require explicit bank", out)
	}
}

func contextText(t *testing.T, a *API, prompt string) (string, string) {
	t.Helper()
	input, err := json.Marshal(map[string]string{"prompt": prompt})
	if err != nil {
		t.Fatal(err)
	}
	out := a.Call(context.Background(), "memory_context", input)
	if !out.OK {
		t.Fatalf("context operation failed: %+v", out.Error)
	}
	raw, err := json.Marshal(out.Result)
	if err != nil || len(raw) > 16383 || !utf8.Valid(raw) {
		t.Fatal("invalid or unbounded native packet", len(raw), err)
	}
	var packet struct {
		Context string `json:"context"`
		Warning string `json:"warning"`
	}
	if err := json.Unmarshal(raw, &packet); err != nil {
		t.Fatal(err)
	}
	return packet.Context, packet.Warning
}

func TestNativeContextIsFreshBoundedScopedAndNeverWrites(t *testing.T) {
	a := fixture(t)
	first, err := a.service.Remember(memory.Write{Kind: "preference", Summary: "Answer style", Body: "Prefer concise answers.", Basis: "user-direction", Reason: "Confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.service.Remember(memory.Write{Kind: "fact", Summary: "Private project", Body: "PROJECT_CONTENT_CANARY", Basis: "observation", Reason: "Observed", Scope: &memory.Scope{Kind: "project", ID: "copper-finch"}}); err != nil {
		t.Fatal(err)
	}
	readOnly := New(a.service, true)
	before := inlineInventory(t, a.service.Root())
	for _, prompt := range []string{"concise answers", "Read-only, no saves or sync", "this is the way", "Quoted: this is the way", strings.Repeat("界", 4000)} {
		text, warning := contextText(t, readOnly, prompt)
		if warning != "" || !strings.Contains(text, "copper-finch") || strings.Contains(text, "PROJECT_CONTENT_CANARY") || !strings.Contains(text, "Untrusted") {
			t.Fatal("unsafe native context", warning)
		}
		if prompt == "concise answers" && !strings.Contains(text, "Prefer concise answers.") {
			t.Fatal("passive recall absent")
		}
	}
	if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
		t.Fatal("read-only context changed files")
	}
	if _, err := a.service.Remember(memory.Write{RecordID: first.RecordID, Supersedes: []string{first.ID}, Kind: "preference", Summary: "Answer style", Body: "Prefer detailed answers.", Basis: "user-direction", Reason: "Changed preference"}); err != nil {
		t.Fatal(err)
	}
	text, warning := contextText(t, readOnly, "answers")
	if warning != "" || !strings.Contains(text, "Prefer detailed answers.") || strings.Contains(text, "Prefer concise answers.") {
		t.Fatal("context reused stale evidence", warning)
	}
}

func TestNativeContextWarningsDoNotEchoCorruptContent(t *testing.T) {
	a := fixture(t)
	r, err := a.service.Remember(memory.Write{Kind: "fact", Summary: "Example", Body: "Valid", Basis: "observation", Reason: "Observed"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.service.Root(), "memory", "records", r.RecordID, r.ID+".json")
	if err := os.WriteFile(path, []byte(`{"body":"CORRUPTION_CANARY"}`), 0600); err != nil {
		t.Fatal(err)
	}
	before := inlineInventory(t, a.service.Root())
	text, warning := contextText(t, a, "Example")
	if warning == "" || strings.Contains(text+warning, "CORRUPTION_CANARY") {
		t.Fatal("raw error leakage or missing warning")
	}
	if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
		t.Fatal("warning repaired the store")
	}
}
