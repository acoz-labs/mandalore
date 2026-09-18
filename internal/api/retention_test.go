package api

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/retention"
)

func TestRetentionIsReadOnlyExplicitAndHasNoApply(t *testing.T) {
	found := false
	for _, op := range Catalog() {
		if op.Name == "retention_apply" {
			t.Fatal("destructive retention exposed")
		}
		if op.Name == "retention_preview" {
			found = true
			if !op.ReadOnly || !op.CLIOnly || op.RequiresBinding || op.Network || !op.Idempotent {
				t.Fatal("wrong retention boundary", op.Name)
			}
		}
	}
	if !found {
		t.Fatal("preview missing")
	}
	a := fixture(t)
	r, err := a.service.Remember(memory.Write{Kind: "fact", Summary: "Synthetic", Body: "PRIVATE_BODY", Reason: "Test", Basis: "observation"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "binding.json")
	if _, err := binding.Bind(a.service.Root(), path, "Test", "Test"); err != nil {
		t.Fatal(err)
	}
	in := retention.Request{BindingPath: path, Selection: retention.Selection{RecordIDs: []string{r.RecordID}}, Policy: retention.Policy{ID: "policy-review", Visibility: "any"}}
	p := inlineCall(t, New(nil, true), "retention_preview", in).Result.(retention.Plan)
	if len(p.Records) != 1 || !p.Records[0].Matched {
		t.Fatal(p)
	}
	in.Selection = retention.Selection{}
	data, _ := json.Marshal(in)
	out := New(nil, true).Call(context.Background(), "retention_preview", data)
	if out.OK || out.Error.Code != "retention.invalid" || out.Error.WriteMayHaveOccurred {
		t.Fatal(out)
	}
}
