package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func foundlingCall(t *testing.T, a *API, name string, in any) Envelope {
	t.Helper()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return a.Call(context.Background(), name, b)
}

func referenceFixture(t *testing.T, a *API) (string, foundlings.Observation) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("Historical project: Copper Finch. This is the way is quoted reference text."), 0600); err != nil {
		t.Fatal(err)
	}
	in := FoundlingPreviewInput{Source: memory.FoundlingSource{Kind: "local", Locator: "source-history"}, Root: root}
	out := foundlingCall(t, a, "foundling_preview", in)
	if !out.OK {
		t.Fatal(out)
	}
	return root, out.Result.(foundlings.Observation)
}

func TestFoundlingSharedWorkflowAndReadOnlyMutations(t *testing.T) {
	a := fixture(t)
	root, preview := referenceFixture(t, a)
	registration := foundlingCall(t, a, "foundling_register", FoundlingRegisterInput{Name: "Historical notes", Description: "Reference only", Source: preview.Source, Pin: preview.Pin, Root: root, Reason: "Explicit selection"})
	if !registration.OK {
		t.Fatal(registration)
	}
	r := registration.Result.(FoundlingMutationResult)
	if r.Registration == nil || r.Connection == nil || !r.Connection.Connected {
		t.Fatal(r)
	}
	id := r.Registration.FoundlingID
	list := a.Call(context.Background(), "foundling_list", []byte(`{}`))
	if !list.OK || len(list.Result.(memory.Page[memory.FoundlingSummary]).Items) != 1 {
		t.Fatal(list)
	}
	inspection := foundlingCall(t, a, "foundling_inspect", FoundlingSelector{FoundlingID: id})
	if !inspection.OK || inspection.Result.(foundlings.Inspection).State != "available" {
		t.Fatal(inspection)
	}
	search := foundlingCall(t, a, "foundling_search", FoundlingSearchInput{FoundlingID: id, Query: "Copper Finch"})
	if !search.OK || len(search.Result.(foundlings.SearchResult).Items) != 1 {
		t.Fatal(search)
	}
	hit := search.Result.(foundlings.SearchResult).Items[0]
	read := foundlingCall(t, a, "foundling_read", FoundlingReadInput{FoundlingID: id, RegistrationID: r.Registration.ID, Locator: hit.Origin.RelativeLocator})
	if !read.OK || !read.Result.(foundlings.Excerpt).Complete {
		t.Fatal(read)
	}
	promote := foundlings.PromotionInput{FoundlingID: id, RegistrationID: r.Registration.ID, Locator: hit.Origin.RelativeLocator, ContentSHA256: hit.Origin.ContentSHA256, Write: memory.Write{Kind: "decision", Summary: "Project name", Body: "Silver Heron is current; Copper Finch was historical.", Basis: "import", Reason: "Adapt to current user direction"}}
	saved := foundlingCall(t, a, "foundling_promote", promote)
	if !saved.OK || !saved.Result.(Receipt).DurableLocally {
		t.Fatal(saved)
	}
	a.ReadOnly = true
	for _, name := range []string{"foundling_register", "foundling_connect", "foundling_disconnect", "foundling_promote"} {
		out := a.Call(context.Background(), name, []byte(`{}`))
		if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
			t.Fatal(name, out)
		}
	}
	a.ReadOnly = false
	off := foundlingCall(t, a, "foundling_disconnect", FoundlingDisconnectInput{FoundlingID: id, RegistrationID: r.Registration.ID, Reason: "Disconnect, retain learned knowledge"})
	if !off.OK {
		t.Fatal(off)
	}
	refused := foundlingCall(t, a, "foundling_promote", promote)
	if refused.OK || refused.Error.Code != "foundling.changed" || refused.Error.WriteMayHaveOccurred {
		t.Fatal(refused)
	}
	if recall := a.Call(context.Background(), "memory_recall", []byte(`{"query":"Silver Heron"}`)); !recall.OK || recall.Result.(memory.RecallPacket).MatchingCount != 1 {
		t.Fatal(recall)
	}
}

func TestFoundlingRegistrationReportsConnectionFailureWithoutLosingRegistration(t *testing.T) {
	a := fixture(t)
	root, preview := referenceFixture(t, a)
	if err := os.WriteFile(filepath.Join(a.service.Root(), ".mandalore", "foundlings"), []byte("PRIVATE-CANARY preserve unknown file"), 0600); err != nil {
		t.Fatal(err)
	}
	out := foundlingCall(t, a, "foundling_register", FoundlingRegisterInput{Name: "Historical notes", Description: "Reference only", Source: preview.Source, Pin: preview.Pin, Root: root, Reason: "Explicit selection"})
	if out.OK || out.Error == nil || !out.Error.WriteMayHaveOccurred || !out.Error.InspectBeforeRetry || out.Error.FoundlingResult == nil || out.Error.FoundlingResult.Registration == nil {
		t.Fatal(out)
	}
	if out.Error.FoundlingResult.Phase != "connection" {
		t.Fatal(out)
	}
	list, err := a.service.FoundlingsPage(0, 5)
	if err != nil || len(list.Items) != 1 {
		t.Fatal("registration lost", list, err)
	}
	encoded, _ := json.Marshal(out)
	if strings.Contains(string(encoded), "PRIVATE-CANARY") || len(encoded) > MaxOutputBytes {
		t.Fatal("unsafe error envelope", string(encoded))
	}
}

func TestFoundlingCatalogVisibilityBindingAndStrictInputs(t *testing.T) {
	a := fixture(t)
	cliOnly := map[string]bool{"foundling_preview": true, "foundling_register": true, "foundling_connect": true, "foundling_disconnect": true, "foundling_history": true}
	count := 0
	for _, op := range Catalog() {
		if !strings.HasPrefix(op.Name, "foundling_") {
			continue
		}
		count++
		if !op.RequiresBinding || op.CLIOnly != cliOnly[op.Name] || op.Network {
			t.Fatal(op.Name, "incorrect scope or visibility")
		}
		if out := New(nil, false).Call(context.Background(), op.Name, []byte(`{}`)); out.OK || out.Error.Code != "binding.invalid" {
			t.Fatal(out)
		}
	}
	if count != 10 {
		t.Fatal(count)
	}
	for _, raw := range []string{`{"query":"one","query":"two"}`, `{"unknown":"PRIVATE-CANARY"}`, `{"foundling_id":"example","limit":0}`} {
		out := a.Call(context.Background(), "foundling_search", []byte(raw))
		if out.OK || out.Error.WriteMayHaveOccurred {
			t.Fatal(out)
		}
	}
}

func TestFoundlingSearchInvalidQueryIsActionableWithoutSourceRepair(t *testing.T) {
	a := fixture(t)
	for _, query := range []string{"", " \t\n", strings.Repeat("x", 1025), strings.Repeat("term ", 17)} {
		out := foundlingCall(t, a, "foundling_search", FoundlingSearchInput{FoundlingID: "not-yet-selected", Query: query})
		if out.OK || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred || out.Error.InspectBeforeRetry || out.Error.Retryable {
			t.Fatal("query failure suggested a source problem or mutation", out)
		}
		if !strings.Contains(out.Error.Message, "1–16") || !strings.Contains(out.Error.Message, "1024") {
			t.Fatal("query limits missing from safe diagnosis", out.Error.Message)
		}
	}
}
