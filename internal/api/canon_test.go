package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestCanonRegistrationContractAndReadOnlyBoundary(t *testing.T) {
	a, s, _ := sessionFixture(t)
	local := New(s, false)
	in := FoundlingRegisterInput{Mode: "canon", Branch: "main", SourceSignetID: "signet-source", Name: "Work", Description: "Synthetic work reference", Source: memory.FoundlingSource{Kind: "git", Locator: "https://example.invalid/work.git"}, Pin: memory.SourcePin{Algorithm: "git-sha1", Value: strings.Repeat("a", 40)}, Reason: "Explicit directional registration"}
	raw, _ := json.Marshal(in)
	out := local.Call(context.Background(), "foundling_register", raw)
	if !out.OK {
		t.Fatal(out.Error)
	}
	reg := out.Result.(FoundlingMutationResult).Registration
	view, err := s.Foundling(reg.FoundlingID)
	if err != nil || view.Mode != "canon" {
		t.Fatal(view, err)
	}
	// Local-only context and refresh denial cannot upgrade transport or create cache.
	if out := local.Call(context.Background(), "memory_context", []byte(`{}`)); !out.OK {
		t.Fatal(out.Error)
	}
	if out := local.Call(context.Background(), "foundling_refresh", []byte(`{"session_id":"native-one"}`)); out.OK {
		t.Fatal("local-only refresh enabled")
	}
	if _, err := os.Stat(filepath.Join(s.Root(), ".mandalore/canon")); !os.IsNotExist(err) {
		t.Fatal("local-only path wrote cache", err)
	}
	ro := New(s, true)
	if out := ro.Call(context.Background(), "foundling_refresh", []byte(`{"session_id":"native-one"}`)); out.OK || out.Error.Code != "operation.read_only" {
		t.Fatal(out)
	}
	if out := local.Call(context.Background(), "foundling_canon_recall", []byte(`{"session_id":"native-one","foundling_id":"`+reg.FoundlingID+`","query":""}`)); out.OK {
		t.Fatal("unknown snapshot selected")
	}
	// Missing native identity is refused before any remote/cache attempt.
	out = a.Call(context.Background(), "foundling_refresh", []byte(`{"session_id":""}`))
	if out.OK {
		t.Fatal("missing ID accepted")
	}
	if _, err := os.Stat(filepath.Join(s.Root(), ".mandalore/canon")); !os.IsNotExist(err) {
		t.Fatal("invalid session wrote cache", err)
	}
	raw, _ = json.Marshal(FoundlingDisconnectInput{FoundlingID: reg.FoundlingID, RegistrationID: reg.ID, Reason: "Explicit disconnect"})
	if out := local.Call(context.Background(), "foundling_disconnect", raw); !out.OK {
		t.Fatal(out.Error)
	}
	view, err = s.Foundling(reg.FoundlingID)
	if err != nil || view.Mode != "canon" || view.Branch != "main" || view.SourceSignetID != "signet-source" || view.State != "disconnected" {
		t.Fatal(view, err)
	}
}

func TestCanonOperationSchemasExposeSessionAndEffects(t *testing.T) {
	ops := map[string]Operation{}
	for _, op := range Catalog() {
		ops[op.Name] = op
	}
	for _, name := range []string{"foundling_canon_recall", "foundling_canon_scopes", "foundling_canon_heads", "foundling_status"} {
		op, ok := ops[name]
		if !ok || !op.ReadOnly || op.Network || op.CLIOnly {
			t.Fatal(name, op)
		}
		raw, err := json.Marshal(op.InputSchema)
		if err != nil || !strings.Contains(string(raw), "session_id") {
			t.Fatal(name, string(raw), err)
		}
	}
	if op := ops["foundling_refresh"]; op.ReadOnly || op.Network {
		t.Fatal("local catalog permits network", op)
	}
	a, _, _ := sessionFixture(t)
	for _, op := range a.Catalog() {
		if op.Name == "foundling_refresh" && (!op.Network || op.ReadOnly) {
			t.Fatal(op)
		}
	}
}
