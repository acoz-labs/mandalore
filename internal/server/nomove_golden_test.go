package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/memory"
)

// The no-moves goldens (#250): read outputs that 2.2.0 (main at e3c9712)
// gave on synthetic signets with no moves. They were written once, before
// any #250 code, with MANDALORE_WRITE_NOMOVE_GOLDEN=1, and are never
// regenerated; every later build must give the same bytes.
//
// The fixtures start from the 2.1.2 compat signets and add, with 2.2.0's
// code, records in every scope kind, a correction, labels, scoped and
// unscoped journal entries and, on format 2, a withdrawn record.
const (
	nomoveFixtures = "../memory/testdata/compat/v2.2.0"
	nomoveGoldens  = "testdata/nomove-golden"
)

var generatedAt = regexp.MustCompile(`"generated_at":"[^"]*"`)

type goldenCall struct {
	Name   string `json:"name"`
	Output string `json:"output"`
}

func TestNoMoveReadsMatchGoldens(t *testing.T) {
	write := os.Getenv("MANDALORE_WRITE_NOMOVE_GOLDEN") == "1"
	for _, format := range []string{"format1", "format2"} {
		t.Run(format, func(t *testing.T) {
			if write {
				writeNoMoveFixture(t, format)
			}
			root := filepath.Join(t.TempDir(), "signet")
			if err := os.CopyFS(root, os.DirFS(filepath.Join(nomoveFixtures, format))); err != nil {
				t.Fatal(err)
			}
			got := noMoveReads(t, root)
			path := filepath.Join(nomoveGoldens, format+".json")
			if write {
				b, err := json.MarshalIndent(got, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(nomoveGoldens, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(b, '\n'), 0644); err != nil {
					t.Fatal(err)
				}
				return
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var want []goldenCall
			if err := json.Unmarshal(b, &want); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(want) {
				t.Fatalf("%d calls, golden has %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("%s differs from 2.2.0:\n got %s\nwant %s", want[i].Name, got[i].Output, want[i].Output)
				}
			}
		})
	}
}

// noMoveReads runs every read whose scope selection #250 changes. Canon
// recall is memory recall on a verified snapshot (foundlings openCanon), and
// canon promotion's head lookup is an exact-scope store recall, so both are
// covered by the memory calls here on the same data.
func noMoveReads(t *testing.T, root string) []goldenCall {
	t.Helper()
	svc, err := memory.OpenReadOnlyService(root)
	if err != nil {
		t.Fatal(err)
	}
	a := api.New(svc, true)
	var out []goldenCall
	call := func(name, input string) {
		v := a.Call(context.Background(), name, []byte(input))
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, goldenCall{name + " " + input, generatedAt.ReplaceAllString(string(b), `"generated_at":"fixed"`)})
	}
	scopes := []string{`{"kind":"project","id":"copper-finch"}`, `{"kind":"task","id":"zebra-task"}`, `{"kind":"account","id":"example-account"}`}
	for _, scope := range scopes {
		call("memory_recall", `{"scope":`+scope+`,"limit":50}`)
		call("memory_recall", `{"query":"codename deploy","scope":`+scope+`,"limit":50}`)
		call("memory_prime", `{"scope":`+scope+`}`)
		call("memory_prime", `{"scope":`+scope+`,"budget_bytes":1024}`)
		call("memory_withheld", `{"scope":`+scope+`,"limit":50}`)
	}
	call("memory_recall", `{"limit":50}`)
	call("memory_scopes", `{"limit":50}`)
	call("memory_scopes", `{"query":"copper finch","limit":50}`)
	call("memory_withheld", `{"limit":50}`)
	for _, prompt := range []string{"What is the Copper Finch project codename?", "How do we deploy the synthetic service?", "Which editor does the example account prefer?"} {
		text, err := contextOf(svc, ContextInput{Event: "prompt", Prompt: prompt}, nil)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, goldenCall{"memory_context prompt " + prompt, text})
	}
	text, err := contextOf(svc, ContextInput{Event: "start"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, goldenCall{"memory_context start", text})
	store, err := memory.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []memory.Scope{{Kind: "project", ID: "copper-finch"}, {Kind: "task", ID: "zebra-task"}} {
		p, err := store.Recall(memory.Query{Scope: scope, ExactScope: true, Limit: int(^uint(0) >> 1)}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		p.GeneratedAt = "fixed"
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, goldenCall{fmt.Sprintf("store exact recall %s %s", scope.Kind, scope.ID), string(b)})
	}
	// History gains fields in #250, so only its pages of revisions are
	// pinned: the items, next_offset and truncated of each page.
	entries, err := os.ReadDir(filepath.Join(root, "memory/records"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		id := entry.Name()
		if !entry.IsDir() {
			continue
		}
		for _, limit := range []int{1, 50} {
			input := fmt.Sprintf(`{"record_id":%q,"limit":%d}`, id, limit)
			v := a.Call(context.Background(), "memory_history", []byte(input))
			b, err := json.Marshal(v.Result)
			if err != nil || !v.OK {
				t.Fatal(string(b), err)
			}
			var page map[string]json.RawMessage
			if err := json.Unmarshal(b, &page); err != nil {
				t.Fatal(err)
			}
			kept := map[string]json.RawMessage{}
			for _, k := range []string{"items", "next_offset", "truncated"} {
				if v, ok := page[k]; ok {
					kept[k] = v
				}
			}
			if b, err = json.Marshal(kept); err != nil {
				t.Fatal(err)
			}
			out = append(out, goldenCall{"memory_history page " + input, string(b)})
		}
	}
	return out
}

// writeNoMoveFixture builds the fixture with this build's code. It ran once,
// on main before #250's code; rerunning it changes every ID.
func writeNoMoveFixture(t *testing.T, format string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "signet")
	if err := os.CopyFS(root, os.DirFS(filepath.Join("../memory/testdata/compat", format))); err != nil {
		t.Fatal(err)
	}
	store, err := memory.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	devices, err := store.DeviceIDs()
	if err != nil || len(devices) == 0 {
		t.Fatal(devices, err)
	}
	svc, err := memory.OpenService(root, memory.Authorship{DeviceID: devices[0], Actor: "Synthetic actor", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	copper := memory.Scope{Kind: "project", ID: "copper-finch"}
	zebra := memory.Scope{Kind: "task", ID: "zebra-task"}
	account := memory.Scope{Kind: "account", ID: "example-account"}
	remember := func(w memory.Write) memory.Revision {
		r, err := svc.Remember(w)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	deploy := remember(memory.Write{Kind: "procedure", Scope: &copper, Summary: "Deploy the synthetic service", Body: "Run the deploy script from the example host.", Basis: "user-direction", Reason: "Synthetic fixture"})
	remember(memory.Write{Kind: "procedure", Scope: &copper, RecordID: deploy.RecordID, Supersedes: []string{deploy.ID}, Summary: "Deploy the synthetic service", Body: "Run the deploy script from the example host, then check its status page.", Basis: "user-direction", Reason: "Added the status check"})
	remember(memory.Write{Kind: "project-state", Scope: &copper, Summary: "Copper Finch: in review", Body: "Waiting for the synthetic reviewer.", Basis: "observation", Reason: "Synthetic fixture"})
	remember(memory.Write{Kind: "decision", Scope: &zebra, Summary: "Zebra task uses the deploy checklist", Body: "Follow the checklist before each deploy.", Basis: "user-direction", Reason: "Synthetic fixture"})
	remember(memory.Write{Kind: "preference", Scope: &account, Summary: "Example account editor", Body: "The example account prefers a plain text editor.", Basis: "user-direction", Reason: "Synthetic fixture"})
	withdrawn := remember(memory.Write{Kind: "fact", Scope: &account, Summary: "Example account old address", Body: "A superseded synthetic address.", Basis: "observation", Reason: "Synthetic fixture"})
	remember(memory.Write{Kind: "fact", Summary: "Synthetic signet-wide fact", Body: "Every value in this signet is synthetic.", Basis: "user-direction", Reason: "Synthetic fixture"})
	if _, err := svc.NameScope(memory.ScopeLabelWrite{Scope: copper, Name: "Copper Finch", Description: "The synthetic Copper Finch project and its deploys.", Reason: "Synthetic fixture"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.NameScope(memory.ScopeLabelWrite{Scope: zebra, Name: "Zebra task", Description: "A synthetic bounded task.", Reason: "Synthetic fixture"}); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []*memory.Scope{&copper, &zebra, nil} {
		if _, err := svc.AppendJournal("outcome", "Synthetic 2.2.0 journal entry", scope); err != nil {
			t.Fatal(err)
		}
	}
	if store.Signet.Version == 2 {
		if _, err := svc.ChangeVisibility(context.Background(), "withdraw", memory.VisibilityWrite{RecordID: withdrawn.RecordID, ContentHeads: []string{withdrawn.ID}, VisibilityHeads: []string{}, Reason: "Synthetic withdrawal"}); err != nil {
			t.Fatal(err)
		}
	}
	dest := filepath.Join(nomoveFixtures, format)
	if err := os.RemoveAll(dest); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".gitignore", "signet.json", "memory", "provenance", "foundlings"} {
		from := filepath.Join(root, name)
		info, err := os.Stat(from)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			err = os.CopyFS(filepath.Join(dest, name), os.DirFS(from))
		} else {
			var b []byte
			if b, err = os.ReadFile(from); err == nil {
				if err = os.MkdirAll(dest, 0755); err == nil {
					err = os.WriteFile(filepath.Join(dest, name), b, 0644)
				}
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}
