package memory

import (
	"testing"
	"time"
)

func TestOnOneSubject(t *testing.T) {
	scope := Scope{Kind: "project", ID: "project-example"}
	rec := func(id, kind, summary, body string) Revision {
		return Revision{ID: "revision-" + id, RecordID: "record-" + id, Kind: kind, Scope: scope, Summary: summary, Body: body}
	}
	queue := rec("queue", "decision", "Tidepool task queue", "Background tasks run on the Postgres-backed queue with five workers.")
	queue2 := rec("queue-2", "decision", "Tidepool task queue", "Background tasks move to the NATS queue with five workers.")
	other := queue2
	other.Scope = Scope{Kind: "project", ID: "project-other"}
	for _, tc := range []struct {
		name string
		a, b Revision
		want bool
	}{
		{"same subject", queue, queue2, true},
		{"one summary word differs of four, inflections alike", queue, rec("q3", "decision", "Tidepool tasks queues backlog", "Background tasks move to the NATS queue with five workers."), true},
		{"one summary word differs of two", rec("e1", "decision", "Tidepool cache", queue.Body), rec("e2", "decision", "Tidepool database", queue2.Body), false},
		{"a refinement", rec("editor", "preference", "Editor", "Helix, spaces not tabs, indent width 4."), rec("theme", "preference", "Editor theme", "A dark theme in the editor, spaces not tabs, indent width 4."), false},
		{"different scopes", queue, other, false},
		{"different kinds", queue, rec("pref", "preference", "Tidepool task queue", "Background tasks move to the NATS queue with five workers."), false},
		{"neither decision nor preference", rec("f1", "fact", "Tidepool task queue", queue.Body), rec("f2", "fact", "Tidepool task queue", queue2.Body), false},
		{"bodies apart", queue, rec("q4", "decision", "Tidepool task queue", "Nightly exports go to cold storage."), false},
	} {
		if got := OnOneSubject(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestNewerOrdersByEffectiveThenRecordedThenID(t *testing.T) {
	at := func(id, effective, recorded string) Revision {
		return Revision{ID: id, EffectiveFrom: effective, RecordedAt: recorded}
	}
	for _, tc := range []struct {
		a, b Revision
		want bool
	}{
		{at("revision-a", "2026-10-02T09:00:00Z", "2026-01-01T00:00:00Z"), at("revision-b", "2026-08-10T09:00:00Z", "2026-12-01T00:00:00Z"), true},
		{at("revision-a", "2026-10-02T09:00:00Z", "2026-10-03T00:00:00Z"), at("revision-b", "2026-10-02T09:00:00Z", "2026-10-02T09:00:00Z"), true},
		{at("revision-a", "2026-10-02T09:00:00Z", "2026-10-02T09:00:00Z"), at("revision-b", "2026-10-02T09:00:00Z", "2026-10-02T09:00:00Z"), false},
	} {
		if got := newer(tc.a, tc.b); got != tc.want {
			t.Errorf("newer(%+v, %+v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestRecallRanksNewerOnOneSubjectFirst(t *testing.T) {
	s := fixtureStore(t)
	for i, body := range []string{
		"Background tasks run on the Postgres-backed queue with five workers; the queue retries tasks.",
		"Background tasks move to the NATS queue with five workers.",
	} {
		r := revision([]string{"revision-old", "revision-new"}[i])
		r.RecordID, r.Kind, r.Summary, r.Body = []string{"record-old", "record-new"}[i], "decision", "Tidepool task queue", body
		r.EffectiveFrom = []string{"2026-08-10T09:00:00Z", "2026-10-02T09:00:00Z"}[i]
		if err := s.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	recall := func(text string) []string {
		p, err := s.Recall(Query{Text: text, Scope: Scope{Kind: "account", ID: "account-example"}, Limit: 5}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		ids := []string{}
		for _, r := range p.Current {
			ids = append(ids, r.ID)
		}
		return ids
	}
	if got := recall("queue tasks"); len(got) != 2 || got[0] != "revision-new" {
		t.Errorf("got %v, want revision-new first", got)
	}
	RankNewerFirst = false
	got := recall("queue tasks")
	RankNewerFirst = true
	if len(got) != 2 || got[0] != "revision-old" {
		t.Errorf("without the rule got %v, want BM25's revision-old first", got)
	}
	if got := recall("Postgres-backed"); len(got) != 1 || got[0] != "revision-old" {
		t.Errorf("an older-only query got %v, want revision-old", got)
	}
}
