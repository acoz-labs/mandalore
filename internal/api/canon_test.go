package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestCanonRegistrationContractAndReadOnlyBoundary(t *testing.T) {
	local := fixture(t)
	s := local.service
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
	// Agents never refresh: the per-session refresh operation is gone, and
	// reads neither fetch nor create cache.
	if out := local.Call(context.Background(), "foundling_refresh", []byte(`{"session_id":"native-one"}`)); out.OK || out.Error.Code != "operation.unknown" {
		t.Fatal(out)
	}
	for _, input := range []string{`{"foundling_id":"` + reg.FoundlingID + `","query":""}`, `{"session_id":"ignored","foundling_id":"` + reg.FoundlingID + `","query":""}`} {
		if out := local.Call(context.Background(), "foundling_canon_recall", []byte(input)); out.OK || out.Error.Code != "foundling.unavailable" {
			t.Fatal("unrefreshed snapshot selected", out)
		}
	}
	if out := local.Call(context.Background(), "foundling_status", []byte(`{}`)); !out.OK || out.Result.(CanonStatusResult).Items[0].Reason != "not_refreshed" {
		t.Fatal(out)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), ".mandalore/canon")); !os.IsNotExist(err) {
		t.Fatal("canon read wrote cache", err)
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

func TestCanonOperationSchemasMakeSessionOptional(t *testing.T) {
	ops := map[string]Operation{}
	for _, op := range Catalog() {
		ops[op.Name] = op
	}
	for _, name := range []string{"foundling_canon_recall", "foundling_canon_scopes", "foundling_canon_heads", "foundling_status"} {
		op, ok := ops[name]
		if !ok || !op.ReadOnly || op.Network || op.CLIOnly {
			t.Fatal(name, op)
		}
		if _, ok := op.InputSchema.Properties["session_id"]; !ok || slices.Contains(op.InputSchema.Required, "session_id") {
			t.Fatal(name, op.InputSchema.Required)
		}
	}
	if op := ops["foundling_canon_promote"]; slices.Contains(op.InputSchema.Required, "session_id") {
		t.Fatal("promotion requires session_id")
	}
	if _, ok := ops["foundling_refresh"]; ok {
		t.Fatal("agent-facing refresh still in the catalog")
	}
}

func TestCanonReceiptPaginationDoesNotRequireAnotherRefresh(t *testing.T) {
	receipts := make([]foundlings.CanonReceipt, 12)
	for i := range receipts {
		receipts[i] = foundlings.CanonReceipt{FoundlingID: fmt.Sprintf("foundling-%02d", i), State: "unavailable", Description: strings.Repeat("x", 4096)}
	}
	first, err := canonReceiptPage(receipts, 0, 5)
	if err != nil || len(first.Items) != 5 || first.NextOffset == nil || *first.NextOffset != 5 || !first.Truncated {
		t.Fatal(first, err)
	}
	second, err := canonReceiptPage(receipts, *first.NextOffset, 10)
	if err != nil || len(second.Items) != 7 || second.NextOffset != nil || second.Truncated {
		t.Fatal(second, err)
	}
	if first.Items[4].FoundlingID == second.Items[0].FoundlingID {
		t.Fatal("repeated receipt")
	}
	raw, _ := json.Marshal(second)
	if len(raw) > MaxOutputBytes-1024 {
		t.Fatal("unbounded page", len(raw))
	}
	for _, selection := range [][2]int{{-1, 5}, {0, 0}, {0, 11}} {
		if _, err := canonReceiptPage(receipts, selection[0], selection[1]); err == nil {
			t.Fatal("invalid pagination accepted", selection)
		}
	}
}
