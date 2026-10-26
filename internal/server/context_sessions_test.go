package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

// #227: within one session a record is injected once, until it is revised;
// other sessions, installations and client scopes are unaffected.
func TestMemoryContextInjectsEachRevisionOncePerSession(t *testing.T) {
	m, ids := contextService(t,
		memory.Write{Kind: "decision", Summary: "Tidepool primary database", Body: "Postgres 17."},
		memory.Write{Kind: "fact", Summary: "Tidepool cache", Body: "Redis, flushable."},
	)
	clock := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	cs := contextSessions{now: func() time.Time { return clock }}
	laptop := Client{ID: "https://client.memory.example.com/metadata.json", Installation: "inst-0000000000000001", Scope: ScopeReadWrite}
	ask := func(c Client, session, prompt string) []string {
		t.Helper()
		text, err := contextOf(m, ContextInput{Event: "prompt", Prompt: prompt, Session: session}, cs.session(c, session))
		if err != nil {
			t.Fatal(err)
		}
		return contextRanks(text)
	}
	const prompt = "Which database and cache does tidepool use?"
	if got := ask(laptop, "session-a", prompt); len(got) != 2 {
		t.Fatalf("first prompt injected %v", got)
	}
	if got := ask(laptop, "session-a", prompt); len(got) != 0 {
		t.Fatalf("repeat in the session injected %v", got)
	}
	desktop := laptop
	desktop.Installation = "inst-0000000000000002"
	readOnly := laptop
	readOnly.Scope = ScopeRead
	for _, c := range []struct {
		client  Client
		session string
	}{{laptop, "session-b"}, {desktop, "session-a"}, {readOnly, "session-a"}, {laptop, ""}, {laptop, strings.Repeat("s", contextSessionBytes+1)}} {
		if got := ask(c.client, c.session, prompt); len(got) != 2 {
			t.Fatalf("%+v session %.20q injected %v", c.client, c.session, got)
		}
	}
	// A revision since the injection is new evidence for the session.
	revised, err := m.Remember(memory.Write{Kind: "decision", Summary: "Tidepool primary database", Body: "Postgres 18 since the upgrade.", RecordID: ids[0], Supersedes: []string{headOf(t, m, ids[0])}, Basis: "observation", Reason: "Synthetic revision"})
	if err != nil {
		t.Fatal(err)
	}
	if got := ask(laptop, "session-a", prompt); len(got) != 1 || got[0] != revised.RecordID {
		t.Fatalf("after a revision injected %v", got)
	}
	// An idle session is forgotten.
	clock = clock.Add(contextSessionIdle + time.Minute)
	if got := ask(laptop, "session-a", prompt); len(got) != 2 {
		t.Fatalf("expired session injected %v", got)
	}
}

func headOf(t *testing.T, m *memory.Service, recordID string) string {
	t.Helper()
	history, err := m.History(recordID)
	if err != nil {
		t.Fatal(err)
	}
	return history[len(history)-1].ID
}

func TestContextSessionsStayBounded(t *testing.T) {
	clock := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	cs := contextSessions{now: func() time.Time { return clock }}
	c := Client{ID: "client-a", Scope: ScopeRead}
	first := cs.session(c, "session-0000")
	first.delivered(contextResult{injected: []string{"revision-a"}})
	for i := 1; i <= contextSessionsMax; i++ {
		clock = clock.Add(time.Second)
		cs.session(c, fmt.Sprintf("session-%04d", i))
	}
	if len(cs.sessions) != contextSessionsMax {
		t.Fatalf("%d sessions kept", len(cs.sessions))
	}
	if again := cs.session(c, "session-0000"); again.injected["revision-a"] {
		t.Fatal("the least recently used session was not dropped")
	}
	s := cs.session(c, "session-full")
	for i := range contextSessionRevisions + 10 {
		s.delivered(contextResult{injected: []string{fmt.Sprintf("revision-%d", i)}})
	}
	if len(s.injected) != contextSessionRevisions {
		t.Fatalf("%d revisions kept", len(s.injected))
	}
}

// Round 1 of #227 review, L3: a call records nothing in the session until
// it has succeeded, so a result that never reaches the hook doesn't
// suppress its records for the rest of the session.
func TestMemoryContextSessionRecordsOnlyDeliveredResults(t *testing.T) {
	m, _ := contextService(t, memory.Write{Kind: "decision", Summary: "Tidepool primary database", Body: "Postgres 17."})
	cs := contextSessions{now: time.Now}
	session := cs.session(Client{ID: "client-a", Scope: ScopeRead}, "session-a")
	in := ContextInput{Event: "prompt", Prompt: "Which database does tidepool use?", Session: "session-a"}
	first, err := memoryContext(m, in, session)
	if err != nil || len(contextRanks(first.text)) != 1 || !first.listed {
		t.Fatalf("first call: %+v %v", first, err)
	}
	// Not delivered: the next call injects the record and lists scopes again.
	again, err := memoryContext(m, in, session)
	if err != nil || again.text != first.text {
		t.Fatalf("undelivered result suppressed the next one: %q", again.text)
	}
	session.delivered(again)
	if after, err := memoryContext(m, in, session); err != nil || after.text != "" {
		t.Fatalf("delivered result injected again: %q %v", after.text, err)
	}
}
