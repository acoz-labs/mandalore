package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// #243: the CLI names the field, limit and size, like the MCP tool.
func TestRememberInputErrorsOnTheCLI(t *testing.T) {
	a := fixture(t)
	in := `{"kind":"fact","summary":"` + strings.Repeat("s", 257) + `","body":"Copper Finch","basis":"user-direction","reason":"Confirmed"}`
	out := a.Call(context.Background(), "memory_remember", []byte(in))
	if out.OK || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred || ExitCode(out) != 2 || !strings.Contains(out.Error.Message, "summary is 257 bytes; the limit is 256 UTF-8 bytes") {
		t.Fatal(out)
	}
}

func TestRememberStaleHeadsOnTheCLI(t *testing.T) {
	a, r := upgradedVisibilityAPI(t)
	correct := func(body string) Envelope {
		raw, err := json.Marshal(memory.Write{Kind: r.Kind, Scope: &r.Scope, RecordID: r.RecordID, Supersedes: []string{r.ID}, Summary: r.Summary, Body: body, Basis: "observation", Reason: "Correction"})
		if err != nil {
			t.Fatal(err)
		}
		return a.Call(context.Background(), "memory_remember", raw)
	}
	if out := correct("First correction"); !out.OK {
		t.Fatal(out.Error)
	}
	if out := correct("Stale correction"); out.OK || out.Error.Code != "memory.stale_heads" || out.Error.WriteMayHaveOccurred {
		t.Fatal(out)
	}
}
