package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixtureStore(t *testing.T) *Store {
	t.Helper()
	root := filepath.Join(t.TempDir(), "assistant")
	s, err := Create(root, "Example", "device-laptop", "Laptop")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func revision(id string, predecessors ...string) Revision {
	return Revision{Version: 1, ID: id, RecordID: "record-documents", Kind: "procedure",
		Scope: Scope{Kind: "account", ID: "account-example"}, Summary: "Keep drafts in Review", Body: "Move drafts into Review; keep other accounts separate.",
		Sensitivity: "private", Volatility: "stable", RecordedAt: "2026-09-06T12:00:00Z", EffectiveFrom: "2026-09-06T12:00:00Z",
		Authorship: Authorship{DeviceID: "device-laptop", Actor: "Example", Harness: "codex"},
		Evidence:   Evidence{Basis: "user-direction", Confidence: "high", SourceRefs: []string{}}, Supersedes: predecessors, ChangeReason: "The user changed the ongoing document policy."}
}

func TestCreateAndValidate(t *testing.T) {
	s := fixtureStore(t)
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(s.Root, "Other", "device-other", "Other"); err == nil {
		t.Fatal("overwrote existing repository")
	}
	if _, err := Open(s.Root); err != nil {
		t.Fatal(err)
	}
}

func TestRecallRankingAndTiesRemainStable(t *testing.T) {
	s := fixtureStore(t)
	for _, item := range []struct{ id, summary, body string }{
		{"revision-weak", "General note", "Project detail"},
		{"revision-tie-b", "Project", "General detail"},
		{"revision-strong", "Project project", "General detail"},
		{"revision-tie-a", "Project", "General detail"},
		{"revision-unrelated", "General note", "Unrelated detail"},
	} {
		r := revision(item.id)
		r.RecordID, r.Summary, r.Body = "record-"+item.id, item.summary, item.body
		if err := s.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"revision-strong", "revision-tie-a", "revision-tie-b", "revision-weak"}
	for _, limit := range []int{2, 10} {
		for attempt := 0; attempt < 5; attempt++ {
			p, err := s.Recall(Query{Text: "project", Scope: Scope{Kind: "account", ID: "account-example"}, Limit: limit}, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Current) != min(limit, len(want)) {
				t.Fatalf("wrong hit count: %+v", p.Current)
			}
			for n, r := range p.Current {
				if r.ID != want[n] {
					t.Fatalf("ranking at %d: got %s, want %s", n, r.ID, want[n])
				}
			}
		}
	}
}

func TestRevisionHistoryAndCrossMachineCorrection(t *testing.T) {
	s := fixtureStore(t)
	first := revision("revision-first")
	if err := s.Put(first); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDevice(Device{Version: 1, ID: "device-desktop", Label: "Desktop"}); err != nil {
		t.Fatal(err)
	}
	second := revision("revision-second", first.ID)
	second.Authorship.DeviceID = "device-desktop"
	second.Authorship.Harness = "pi"
	second.Body = "Leave drafts in the working folder until processed."
	second.Summary = "Process drafts before filing"
	if err := s.Put(second); err != nil {
		t.Fatal(err)
	}
	packet, err := s.Recall(Query{Text: "drafts", Scope: second.Scope}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Current) != 1 || packet.Current[0].ID != second.ID {
		t.Fatalf("current: %+v", packet)
	}
	history, err := s.History(first.RecordID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Authorship.DeviceID != "device-laptop" || history[1].Authorship.DeviceID != "device-desktop" {
		t.Fatalf("history: %+v", history)
	}
	if err := s.Put(second); err == nil {
		t.Fatal("duplicate revision was overwritten")
	}
}

func TestConcurrentHeadsRequireExplicitResolution(t *testing.T) {
	s := fixtureStore(t)
	a := revision("revision-root")
	b := revision("revision-branch-one", a.ID)
	c := revision("revision-branch-two", a.ID)
	for _, r := range []Revision{a, b, c} {
		if err := s.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.Recall(Query{Text: "drafts", Scope: a.Scope}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) != 0 || len(p.Conflicts) != 1 || len(p.Conflicts[0].Revisions) != 2 {
		t.Fatalf("silently chose a concurrent head: %+v", p)
	}
	d := revision("revision-resolved", b.ID, c.ID)
	if err := s.Put(d); err != nil {
		t.Fatal(err)
	}
	p, err = s.Recall(Query{Text: "drafts", Scope: a.Scope}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) != 1 || p.Current[0].ID != d.ID || len(p.Conflicts) != 0 {
		t.Fatalf("resolution: %+v", p)
	}
}

func TestRevisionGraphRejectsInvalidTransitions(t *testing.T) {
	cases := map[string]func(*Revision){
		"missing predecessor": func(r *Revision) { r.Supersedes = []string{"revision-missing"} },
		"self reference":      func(r *Revision) { r.Supersedes = []string{r.ID} },
		"scope change":        func(r *Revision) { r.Scope.ID = "account-other" },
		"kind change":         func(r *Revision) { r.Kind = "fact" },
		"unknown device":      func(r *Revision) { r.Authorship.DeviceID = "device-unknown" },
		"missing source":      func(r *Revision) { r.Evidence.SourceRefs = []string{"source-missing"} },
		"backdate":            func(r *Revision) { r.EffectiveFrom = "2026-09-05T00:00:00Z" },
		"second root":         func(r *Revision) { r.Supersedes = nil },
		"path traversal":      func(r *Revision) { r.ID = "../../outside" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := fixtureStore(t)
			a := revision("revision-root")
			if err := s.Put(a); err != nil {
				t.Fatal(err)
			}
			b := revision("revision-new", a.ID)
			change(&b)
			if err := s.Put(b); err == nil {
				t.Fatal("accepted invalid revision")
			}
			h, err := s.History(a.RecordID)
			if err != nil || len(h) != 1 {
				t.Fatalf("invalid write changed memory: %v %+v", err, h)
			}
		})
	}
}

func TestFutureRevisionAndScopeIsolation(t *testing.T) {
	s := fixtureStore(t)
	a := revision("revision-root")
	if err := s.Put(a); err != nil {
		t.Fatal(err)
	}
	b := revision("revision-future", a.ID)
	b.EffectiveFrom = "2099-01-01T00:00:00Z"
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	task := revision("revision-task")
	task.RecordID = "record-exception"
	task.Scope = Scope{Kind: "task", ID: "task-private"}
	if err := s.Put(task); err != nil {
		t.Fatal(err)
	}
	p, err := s.Recall(Query{Text: "drafts", Scope: a.Scope}, time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) != 1 || p.Current[0].ID != a.ID {
		t.Fatalf("wrong effective scope: %+v", p)
	}
	p, err = s.Recall(Query{Text: "drafts", Scope: Scope{Kind: "account", ID: "account-other"}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) != 0 {
		t.Fatalf("leaked account-scoped policy: %+v", p)
	}
}

func TestRecallDoesNotReturnSupersededKeywordHit(t *testing.T) {
	s := fixtureStore(t)
	a := revision("revision-root")
	if err := s.Put(a); err != nil {
		t.Fatal(err)
	}
	b := revision("revision-replacement", a.ID)
	b.Summary = "Revised document handling"
	b.Body = "Keep unfinished documents in the working folder."
	if err := s.Put(b); err != nil {
		t.Fatal(err)
	}
	p, err := s.Recall(Query{Text: "Review", Scope: a.Scope}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) != 0 {
		t.Fatalf("obsolete content resurfaced: %+v", p)
	}
}

func TestScopeDiscoveryDoesNotPromoteUnrelatedGuidance(t *testing.T) {
	s := fixtureStore(t)
	scopes, err := s.Scopes()
	if err != nil || scopes == nil || len(scopes) != 0 {
		t.Fatalf("empty discovery: %+v %v", scopes, err)
	}
	first := revision("revision-original")
	first.Scope = Scope{Kind: "project", ID: "project-stable-id"}
	second := first
	second.ID, second.Summary = "revision-renamed", "The project has a new display name"
	second.Supersedes = []string{first.ID}
	other := revision("revision-other")
	other.RecordID = "record-other"
	for _, r := range []Revision{first, second, other} {
		if err := s.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	scopes, err = s.Scopes()
	if err != nil || len(scopes) != 2 || scopes[0].Scope != other.Scope || scopes[1].Scope != first.Scope || scopes[1].RecordCount != 1 {
		t.Fatalf("scope discovery must be sorted and count records, not revisions: %+v %v", scopes, err)
	}
	p, err := s.Recall(Query{Scope: Scope{Kind: "project", ID: "/unrelated/cwd"}}, time.Now())
	if err != nil || len(p.Current) != 0 || !strings.Contains(p.Notice, "memory scopes") {
		t.Fatalf("empty scoped recall should offer discovery, not widen guidance: %+v %v", p, err)
	}
	p, err = s.Recall(Query{Scope: scopes[1].Scope}, time.Now())
	if err != nil || len(p.Current) != 1 || p.Current[0].ID != second.ID {
		t.Fatalf("explicit discovered scope did not return current revision: %+v %v", p, err)
	}
	// Discovery must validate the graph, just like recall, rather than trusting a stale index.
	second.Supersedes = []string{"revision-missing"}
	data, _ := json.Marshal(second)
	if err := os.WriteFile(filepath.Join(s.Root, "memory/records", second.RecordID, second.ID+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Scopes(); err == nil {
		t.Fatal("scope discovery accepted invalid memory")
	}
}

func TestStrictJSONAndFutureFormat(t *testing.T) {
	s := fixtureStore(t)
	data, err := os.ReadFile(filepath.Join(s.Root, "signet.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err = json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	v["schema_version"] = 99
	data, _ = json.Marshal(v)
	if err = os.WriteFile(filepath.Join(s.Root, "signet.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(s.Root); err == nil || !strings.Contains(err.Error(), "format") {
		t.Fatalf("newer writer format accepted: %v", err)
	}
}

func TestSymlinkRecordRefused(t *testing.T) {
	s := fixtureStore(t)
	r := revision("revision-first")
	if err := s.Put(r); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(s.Root, "memory", "records", r.RecordID, r.ID+".json")
	target := filepath.Join(t.TempDir(), "foreign.json")
	b, _ := os.ReadFile(p)
	if err := os.WriteFile(target, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err == nil {
		t.Fatal("followed symlink into external data")
	}
}
