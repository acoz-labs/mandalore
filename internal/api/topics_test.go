package api

import (
	"context"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestTopicOperations(t *testing.T) {
	a := fixture(t)
	ctx := context.Background()
	topic := `{"kind":"project","id":"copper-finch"}`
	if out := a.Call(ctx, "memory_name_scope", []byte(`{"scope":`+topic+`,"name":"Copper Finch","description":"Fictional project.","reason":"Named"}`)); !out.OK {
		t.Fatal(out.Error)
	}
	if out := a.Call(ctx, "memory_journal_append", []byte(`{"kind":"outcome","summary":"Drafted","scope":`+topic+`}`)); !out.OK {
		t.Fatal(out.Error)
	}
	out := a.Call(ctx, "memory_scopes", []byte(`{"query":"Copper Finch"}`))
	if page := out.Result.(memory.Page[memory.ScopeInfo]); !out.OK || len(page.Items) != 1 || page.Items[0].Labels[0].Name != "Copper Finch" {
		t.Fatal(out)
	}
	out = a.Call(ctx, "memory_prime", []byte(`{"scope":`+topic+`,"budget_bytes":1024}`))
	if p := out.Result.(memory.PrimePacket); !out.OK || len(p.Labels) != 1 || len(p.Journal) != 1 {
		t.Fatal(out)
	}
	out = a.Call(ctx, "memory_journal", []byte(`{"scope":`+topic+`}`))
	if page := out.Result.(memory.Page[memory.JournalEntry]); !out.OK || len(page.Items) != 1 {
		t.Fatal(out)
	}
	if out = a.Call(ctx, "memory_prime", []byte(`{"scope":{"kind":"signet","id":"signet-x"}}`)); out.OK || out.Error.Code != "input.invalid" {
		t.Fatal(out)
	}
	// A second first name is the caller's error.
	if out = a.Call(ctx, "memory_name_scope", []byte(`{"scope":`+topic+`,"name":"Again","description":"d","reason":"r"}`)); out.OK || out.Error.Code != "input.invalid" || !strings.Contains(out.Error.Message, "already has a name") {
		t.Fatal(out)
	}
	readOnly := New(a.service, true)
	for _, raw := range []string{`{"scope":` + topic + `,"name":"n","description":"d","reason":"r"}`} {
		if out = readOnly.Call(ctx, "memory_name_scope", []byte(raw)); out.OK || out.Error.Code != "operation.read_only" {
			t.Fatal(out)
		}
	}
	if out = readOnly.Call(ctx, "memory_journal_append", []byte(`{"kind":"k","summary":"s","scope":`+topic+`}`)); out.OK || out.Error.Code != "operation.read_only" {
		t.Fatal(out)
	}
}
