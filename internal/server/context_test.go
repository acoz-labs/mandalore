package server

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

var large = testfixture.Corpus{Records: 1000, Depth: 1}

const unrelated = "What is the weather like in Lisbon this weekend, and should I pack an umbrella?"

// contextOf answers as the server does: the session records the result only
// when the call succeeds.
func contextOf(m *memory.Service, in ContextInput, session *contextSession) (string, error) {
	r, err := memoryContext(m, in, session)
	if err == nil {
		session.delivered(r)
	}
	return r.text, err
}

func contextFor(t testing.TB, m *memory.Service, event, prompt string) string {
	t.Helper()
	text, err := contextOf(m, ContextInput{Event: event, Prompt: prompt}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestMemoryContextRanksAndStaysBounded(t *testing.T) {
	f := testfixture.New(t, testfixture.Corpus{Records: 1000, Depth: 2, ConflictEvery: 17})
	// Only the prompt's first 2048 bytes count; the padding is all stopwords.
	text := contextFor(t, f.Service, "prompt", "Where is the signal-000001 checkpoint? "+strings.Repeat("please ", 400)+"unrelated")
	lines := strings.Split(text, "\n")
	if lines[0] != contextHeader || !strings.Contains(lines[1], "signal-000001") || len(text) > contextDefaultBytes {
		t.Fatalf("%d bytes:\n%s", len(text), text)
	}
	// A broad prompt matches every record; the cap still holds, and a claim
	// is whole or visibly clipped, never silently cut.
	for _, budget := range []int{contextMinBytes, contextDefaultBytes, contextMaxBytes} {
		text, err := contextOf(f.Service, ContextInput{Event: "prompt", Prompt: "fictional project checkpoint", BudgetBytes: &budget}, nil)
		if err != nil || len(text) > budget || !strings.HasPrefix(text, contextHeader) || len(contextRanks(text)) == 0 {
			t.Fatalf("budget %d: %d bytes: %v\n%s", budget, len(text), err, text)
		}
		for _, line := range strings.Split(text, "\n")[1:] {
			if strings.HasPrefix(line, "- Project") && !strings.Contains(line, "Stable evidence for a fictional project. Stable evidence for a fictional project. [") && !strings.Contains(line, contextEllipsis+" [") {
				t.Fatalf("claim cut without a mark %q", line)
			}
		}
	}
	if text := contextFor(t, f.Service, "prompt", "alternate unresolved concurrent"); !strings.Contains(text, "- Conflict: record-") {
		t.Fatalf("matching conflict not flagged:\n%s", text)
	}
	if text := contextFor(t, f.Service, "prompt", unrelated); text != "" {
		t.Fatalf("unrelated prompt returned %q", text)
	}
	if text := contextFor(t, f.Service, "prompt", "the and what"); text != "" {
		t.Fatalf("stopword prompt returned %q", text)
	}
	if text := contextFor(t, f.Service, "start", ""); !strings.HasPrefix(text, contextHeader+"\nScopes: ") || len(text) > contextDefaultBytes {
		t.Fatalf("start context %q", text)
	}
	for _, in := range []ContextInput{{Event: "stop"}, {Event: "prompt", BudgetBytes: new(int)}} {
		if _, err := contextOf(f.Service, in, nil); err == nil {
			t.Fatalf("invalid input accepted: %+v", in)
		}
	}
}

// Claude Code adds an mcp_tool hook's hookSpecificOutput.additionalContext to
// the model's context, never its plain text (#163 step 0), so the tool answers
// each event in that envelope and returns nothing when nothing is relevant.
func TestMemoryContextAnswersInHookEnvelope(t *testing.T) {
	a := testfixture.NewServerSignet(t, "alpha")
	_, h, _ := startServer(t, testConfig(t, a))
	session := connect(t, h, "/mcp", writeToken)
	call(t, session, "memory_remember", rememberArgs("context-envelope-01", "The zebrafinch build cache rotates on Tuesdays."))
	for _, tc := range []struct {
		args map[string]any
		hook string
	}{
		{map[string]any{"event": "prompt", "prompt": "When does the zebrafinch cache rotate?"}, "UserPromptSubmit"},
		{map[string]any{"event": "start"}, "SessionStart"},
	} {
		r, text := call(t, session, "memory_context", tc.args)
		var out struct {
			HookSpecificOutput struct{ HookEventName, AdditionalContext string }
		}
		if err := json.Unmarshal([]byte(text), &out); err != nil || r.IsError {
			t.Fatalf("%v: not a hook envelope: %s", tc.args, text)
		}
		got := out.HookSpecificOutput
		if got.HookEventName != tc.hook || !strings.HasPrefix(got.AdditionalContext, contextHeader+"\n") || len(got.AdditionalContext) > contextDefaultBytes {
			t.Fatalf("%v: %+v", tc.args, got)
		}
		if tc.hook == "UserPromptSubmit" && !strings.Contains(got.AdditionalContext, "rotates on Tuesdays") {
			t.Fatalf("prompt context lacks the saved fact: %s", got.AdditionalContext)
		}
	}
	if r, text := call(t, session, "memory_context", map[string]any{"event": "prompt", "prompt": unrelated}); r.IsError || text != "" {
		t.Fatalf("unrelated prompt returned %q", text)
	}
	// Acceptance 2: the policy text the hook injects stays under 200 bytes.
	if len(contextHeader) >= 200 {
		t.Fatalf("policy header is %d bytes", len(contextHeader))
	}
}

func TestMemoryContextEmptySignet(t *testing.T) {
	s := testfixture.NewServerSignet(t, "empty")
	m, err := memory.OpenReadOnlyService(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"start", "prompt"} {
		if text := contextFor(t, m, event, "anything relevant"); text != "" {
			t.Fatalf("%s on an empty signet: %q", event, text)
		}
	}
}

// Acceptance 7: on a 1,000-record signet, p95 under 300 ms, at most 4 KB,
// and empty for an unrelated prompt. Run under the race detector in bin/ci,
// so the bound has headroom on plain builds.
func TestMemoryContextLatencyOnLargeSignet(t *testing.T) {
	f := testfixture.New(t, large)
	f.Service.CacheReads() // as the server does
	max := contextMaxBytes
	samples := make([]time.Duration, 40)
	for i := range samples {
		start := time.Now()
		text, err := contextOf(f.Service, ContextInput{Event: "prompt", Prompt: fmt.Sprintf("status of checkpoint %06d in the fictional project", i*25), BudgetBytes: &max}, nil)
		samples[i] = time.Since(start)
		if err != nil || len(text) > contextMaxBytes || !strings.Contains(text, fmt.Sprintf("signal-%06d", i*25)) {
			t.Fatalf("sample %d: %d bytes: %v", i, len(text), err)
		}
	}
	slices.Sort(samples)
	p95 := samples[len(samples)*95/100-1]
	t.Logf("memory_context p95 %s over %d samples", p95, len(samples))
	if p95 >= 300*time.Millisecond {
		t.Fatalf("p95 %s exceeds 300ms", p95)
	}
	if text := contextFor(t, f.Service, "prompt", unrelated); text != "" {
		t.Fatalf("unrelated prompt returned %d bytes", len(text))
	}
}

// #227 acceptance 3: on the 250-record evaluation signet, with the 2.1 ranking,
// threshold and packing, warm p95 stays under the same 300 ms bound, far
// below the hook's 3 s timeout. bin/ci runs it on a plain build.
func TestMemoryContextLatencyOnEvaluationSignet(t *testing.T) {
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	m.CacheReads()
	contextFor(t, m, "prompt", "warm the read cache")
	var samples []time.Duration
	evalPrompts(e, func(_ int, form, text string, w evalWrapper) {
		if form == "variant" || w.heldOut {
			return // the tuning split only; see heldOutRequested
		}
		start := time.Now()
		contextFor(t, m, "prompt", strings.ReplaceAll(w.text, "{q}", text))
		samples = append(samples, time.Since(start))
	})
	slices.Sort(samples)
	p95 := samples[len(samples)*95/100-1]
	t.Logf("memory_context on %d records: p50 %s, p95 %s, max %s over %d prompts (race detector: %v)", len(e.Records), samples[len(samples)/2], p95, samples[len(samples)-1], len(samples), raceEnabled)
	if !raceEnabled && p95 >= 300*time.Millisecond {
		t.Fatalf("p95 %s exceeds 300ms", p95)
	}
}

// The read cache drops on every write, so the first call after a save reads
// and validates the whole signet (about 165 ms p95 on a development laptop).
// This is a regression tripwire, not criterion 7's warm-path bound: it fails
// only past 1 s, and only on plain builds, which bin/ci runs; the race
// detector slows this path several times over.
func TestMemoryContextAfterWriteLatency(t *testing.T) {
	f := testfixture.New(t, large)
	f.Service.CacheReads()
	samples := make([]time.Duration, 20)
	for i := range samples {
		if _, err := f.Service.Remember(memory.Write{Kind: "fact", Summary: "Fresh note", Body: fmt.Sprintf("Synthetic note %d", i), Basis: "observation", Reason: "Latency sample"}); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		text := contextFor(t, f.Service, "prompt", fmt.Sprintf("status of checkpoint %06d", i*50))
		samples[i] = time.Since(start)
		if !strings.Contains(text, fmt.Sprintf("signal-%06d", i*50)) {
			t.Fatalf("sample %d missed its record", i)
		}
	}
	slices.Sort(samples)
	p95 := samples[len(samples)*95/100-1]
	t.Logf("memory_context after a write: p95 %s over %d samples (race detector: %v)", p95, len(samples), raceEnabled)
	if !raceEnabled && p95 >= time.Second {
		t.Fatalf("p95 %s after a write exceeds the 1s tripwire", p95)
	}
}

func BenchmarkMemoryContext(b *testing.B) {
	f := testfixture.New(b, large)
	f.Service.CacheReads()
	max := contextMaxBytes
	testfixture.Measure(b, func() ([]byte, error) {
		text, err := contextOf(f.Service, ContextInput{Event: "prompt", Prompt: "status of checkpoint 000500 in the fictional project", BudgetBytes: &max}, nil)
		return []byte(text), err
	})
}

func contextService(t *testing.T, writes ...memory.Write) (*memory.Service, []string) {
	t.Helper()
	s := testfixture.NewServerSignet(t, "context")
	m, err := memory.OpenService(s.Root, memory.Authorship{DeviceID: s.DeviceID, Actor: "Synthetic server", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, w := range writes {
		w.Basis, w.Reason = "observation", "Synthetic context fixture"
		r, err := m.Remember(w)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, r.RecordID)
	}
	return m, ids
}

// #227: the packer used to skip a top hit whose whole claim didn't fit and
// inject a smaller, weaker record instead. The top hits now always appear,
// clipped at a rune boundary with a mark, and the budget still holds.
func TestMemoryContextClipsTopHitsInsteadOfDroppingThem(t *testing.T) {
	long := strings.Repeat("Beacon 日本語 é handles the front door intercom. ", 120)
	m, ids := contextService(t,
		memory.Write{Kind: "decision", Summary: "Front door assistant choice " + strings.Repeat("é", 100), Body: long},
		memory.Write{Kind: "fact", Summary: "Front door assistant fallback", Body: "Relay answers the door when Beacon is down."},
		memory.Write{Kind: "fact", Summary: "Garden hose", Body: "Coiled by the shed."},
		memory.Write{Kind: "fact", Summary: "Kitchen lights", Body: "Dimmed after nine."},
	)
	for _, budget := range []int{contextMinBytes, 512, contextDefaultBytes, contextMaxBytes} {
		text, err := contextOf(m, ContextInput{Event: "prompt", Prompt: "Who is my front door assistant? Answer from context you already have; don't call tools.", BudgetBytes: &budget}, nil)
		if err != nil || len(text) > budget || !utf8.ValidString(text) {
			t.Fatalf("budget %d: %d bytes, valid %v: %v", budget, len(text), utf8.ValidString(text), err)
		}
		ranks := contextRanks(text)
		if len(ranks) == 0 || ranks[0] != ids[0] || !strings.Contains(text, contextEllipsis+" [decision, ") {
			t.Fatalf("budget %d: top hit missing or unmarked:\n%s", budget, text)
		}
		if budget >= 512 && rankOf(ranks, ids[1]) != 2 {
			t.Fatalf("budget %d: second hit missing:\n%s", budget, text)
		}
	}
}

func TestContextEntryClipping(t *testing.T) {
	e := entryFor(memory.Revision{Kind: "fact", Scope: memory.Scope{Kind: "project", ID: "tidepool"}, RecordID: "record-a", Summary: "Résumé  of\nthe   day", Authorship: memory.Authorship{DeviceID: "device-a"}})
	if e.claim != "Résumé of the day" || e.line() != "- Résumé of the day [fact, project tidepool, record-a, device device-a]" {
		t.Fatalf("no-body entry %q", e.line())
	}
	e.claim = strings.Repeat("日本語 ", 40)
	floor := e.size() - len(e.claim) + len(contextEllipsis)
	for n := floor; n <= e.size()+1; n++ {
		c := e.clip(n)
		if c.size() > n || !utf8.ValidString(c.claim) || c.meta != e.meta || (c.size() < e.size()) != strings.HasSuffix(c.claim, contextEllipsis) {
			t.Fatalf("clip %d: %d bytes %q", n, c.size(), c.claim)
		}
	}
	// No room for one top entry's provenance: nothing is clipped into the budget.
	if got := packTop([]contextEntry{e}, e.size()-len(e.claim)); got != nil {
		t.Fatalf("packed %v", got)
	}
}

func TestMemoryContextPromptEdgeCases(t *testing.T) {
	m, _ := contextService(t,
		memory.Write{Kind: "fact", Summary: "Tide table", Body: "High water at noon."},
		memory.Write{Kind: "fact", Summary: "Tide table", Body: "High water at noon."},
		memory.Write{Kind: "fact", Summary: "Garden hose", Body: "Coiled by the shed."},
		memory.Write{Kind: "fact", Summary: "Kitchen lights", Body: "Dimmed after nine."},
	)
	for _, prompt := range []string{"", "   ", "!!! ???", "answer from context in one line, don't call tools", "ok", "thanks, go ahead", "ready", "What is the capital of Australia?"} {
		if text := contextFor(t, m, "prompt", prompt); text != "" {
			t.Fatalf("%q returned %q", prompt, text)
		}
	}
	huge := "tide table at noon " + strings.Repeat("日本語 ", 5000)
	text := contextFor(t, m, "prompt", huge)
	if len(text) > contextDefaultBytes || len(contextRanks(text)) != 2 {
		t.Fatalf("huge prompt: %d bytes:\n%s", len(text), text)
	}
	// Equal scores keep one order between calls.
	if again := contextFor(t, m, "prompt", huge); again != text {
		t.Fatalf("unstable:\n%s\n%s", text, again)
	}
}

func TestContextRelevanceThreshold(t *testing.T) {
	rev := func(id string) memory.Revision { return memory.Revision{ID: id, RecordID: "record-" + id} }
	packet := func(strengths map[string]float64) memory.Packet {
		p := memory.Packet{Strength: strengths}
		for _, id := range []string{"a", "b", "c"} {
			if _, ok := strengths[id]; ok {
				p.Current = append(p.Current, rev(id))
			}
		}
		p.Conflicts = []memory.RevisionConflict{{RecordID: "record-x", Revisions: []memory.Revision{rev("x1"), rev("x2")}}}
		return p
	}
	ids := func(records []memory.Revision, conflicts []memory.RevisionConflict) string {
		out := []string{}
		for _, r := range records {
			out = append(out, r.ID)
		}
		for _, c := range conflicts {
			out = append(out, c.RecordID)
		}
		return strings.Join(out, " ")
	}
	for _, tc := range []struct {
		strengths, evidence map[string]float64
		want                string
	}{
		// Under both floors, nothing; a conflict counts by its strongest head.
		{map[string]float64{"a": contextMinStrength - 0.01, "b": 0.1, "x1": 0.05, "x2": 0.1}, map[string]float64{"a": contextMinEvidence - 0.01}, ""},
		{map[string]float64{"a": 0.8, "b": 0.8 * contextRelativeStrength, "c": 0.8*contextRelativeStrength - 0.01, "x1": 0, "x2": 0.6}, nil, "a b record-x"},
		{map[string]float64{"a": 0.1, "x1": 0.5, "x2": 0}, nil, "record-x"},
		// Enough evidence carries a record whose coverage a long prompt diluted.
		{map[string]float64{"a": 0.05, "b": 0.05}, map[string]float64{"a": contextMinEvidence, "b": 0.1}, "a"},
		// An incidental match is never relevant, however strong.
		{map[string]float64{"a": 0.9, "b": 0.3}, nil, "b"},
	} {
		p := packet(tc.strengths)
		p.Evidence = tc.evidence
		p.Incidental = map[string]bool{"a": tc.strengths["a"] == 0.9}
		if got := ids(choose([]memory.Packet{p})); got != tc.want {
			t.Errorf("%v %v: got %q, want %q", tc.strengths, tc.evidence, got, tc.want)
		}
	}
}

func TestContextNames(t *testing.T) {
	for prompt, want := range map[string]string{
		"What is Beacon?":                               "Beacon?",
		"In one line: what is Beacon? Thanks.":          "Beacon?",
		"Beacon keeps dropping. Why does Relay answer?": "Relay",
		"Recommend a good font for a resume.":           "",
		`Is "Tidepool" on Postgres or MySQL?`:           "Tidepool\" Postgres MySQL?",
		"what is beacon":                                "",
	} {
		if got := strings.Join(names(prompt), " "); got != want {
			t.Errorf("%q: got %q, want %q", prompt, got, want)
		}
	}
}

// Round 2 of #227 review, H1: a record sharing one incidental word with a
// prompt, such as a number or a hyphenated word in its body, isn't injected;
// one whose summary names the prompt's subject is.
func TestMemoryContextIgnoresIncidentalMatches(t *testing.T) {
	writes := []memory.Write{
		{Kind: "decision", Summary: "Public API rate limit", Body: "600 requests per minute per key; two-factor sign-in for the dashboard."},
		{Kind: "entity", Summary: "Beacon is the house voice assistant", Body: "Self-hosted; runs on the hub."},
	}
	for i := range 30 { // IDF needs a signet of realistic size
		writes = append(writes, memory.Write{Kind: "project-state", Summary: fmt.Sprintf("Session notes %d", i), Body: "Checked the garden, the shed and the kitchen lights again."})
	}
	m, ids := contextService(t, writes...)
	for _, prompt := range []string{"What is 600 divided by 7?", "How do I add two-factor login to Django?", "Is self-hosted email worth it?"} {
		if text := contextFor(t, m, "prompt", prompt); text != "" {
			t.Errorf("%q injected:\n%s", prompt, text)
		}
	}
	// Round 4 of #227 review: a short question's one word isn't incidental.
	for prompt, want := range map[string]string{"What's the API rate limit?": ids[0], "Without looking anything up, what is Beacon?": ids[1], "Is it self-hosted?": ids[1]} {
		if ranks := contextRanks(contextFor(t, m, "prompt", prompt)); len(ranks) == 0 || ranks[0] != want {
			t.Errorf("%q injected %v, want %s first", prompt, ranks, want)
		}
	}
}

func TestContextQuestionClauses(t *testing.T) {
	for prompt, want := range map[string]string{
		"Before we start, remind me: Which editor do I use? One line is fine.": "Which editor do I use?",
		"Which editor do I use?":                                "",
		"Which editor do I use":                                 "",
		"Before we start: is it Apache-2.0? Thanks.":            "is it Apache-2.0?",
		"Note this. Do we meet at 10:00? Ok.":                   "Do we meet at 10:00?",
		"First: editor? Then: shell? Then: units? Then: theme?": "editor? | shell? | units?",
		"Okay? Which shell?":                                    "Which shell?",
	} {
		if got := strings.Join(questionClauses(prompt), " | "); got != want {
			t.Errorf("%q: got %q, want %q", prompt, got, want)
		}
	}
}

// #232: a clause can add or raise a record but never drop one the whole
// prompt found relevant.
func TestContextClausesOnlyAdd(t *testing.T) {
	rev := func(id string) memory.Revision { return memory.Revision{ID: id, RecordID: "record-" + id} }
	whole := memory.Packet{Current: []memory.Revision{rev("a"), rev("b")}, Strength: map[string]float64{"a": 0.5, "b": 0.1}}
	clause := memory.Packet{Current: []memory.Revision{rev("b"), rev("c")}, Strength: map[string]float64{"b": 0.6, "c": 0.4}, Incidental: map[string]bool{"a": true}}
	records, _ := choose([]memory.Packet{whole, clause})
	got := []string{}
	for _, r := range records {
		got = append(got, r.ID)
	}
	if strings.Join(got, " ") != "b a c" {
		t.Errorf("got %v, want b (raised by the clause), a (the whole prompt's), c (the clause's)", got)
	}
}

func selectionService(t *testing.T, writes ...memory.Write) (*memory.Service, []string) {
	t.Helper()
	for i := range 30 { // IDF needs a signet of realistic size
		writes = append(writes, memory.Write{Kind: "project-state", Summary: fmt.Sprintf("Session notes %d", i), Body: "Checked the garden, the shed and the kitchen lights again this year."})
	}
	return contextService(t, writes...)
}

func TestMemoryContextSelectionRules(t *testing.T) {
	m, ids := selectionService(t,
		memory.Write{Kind: "preference", Summary: "Editor", Body: "Helix, spaces not tabs."},
		memory.Write{Kind: "preference", Summary: "Editor theme", Body: "A dark theme in every terminal."},
		memory.Write{Kind: "preference", Summary: "Mail signature", Body: "A single line with no phone number."},
		memory.Write{Kind: "decision", Summary: "Tomato varieties", Body: "Sungold and San Marzano in the south bed."},
		memory.Write{Kind: "decision", Summary: "Tidepool primary database", Body: "Postgres 17; Redis is a cache only."},
		memory.Write{Kind: "entity", Summary: "Beacon is the house voice assistant", Body: "Runs on the hub."},
	)
	editor, signature, tomatoes, database, beacon := ids[0], ids[2], ids[3], ids[4], ids[5]
	for _, prompt := range []string{
		// A lone generic body word ("number", "year"), whatever the other word.
		"Is 19 a prime number?",
		"Which year is a leap?",
		// A lone summary word with another of the prompt's words in no record.
		"What theme should we pick for a wedding?",
		// The same inside a wrapper's question clause.
		"Quick one before we start: what theme should we pick for a wedding?",
		// A name no record holds, beside a match only in a record's body.
		"Which Postgres version suits the Django port?",
	} {
		if text := contextFor(t, m, "prompt", prompt); text != "" {
			t.Errorf("%q injected:\n%s", prompt, text)
		}
	}
	for prompt, want := range map[string]string{
		// A distinctive body word still counts, alone or beside a generic one.
		"Are we growing Sungold?": tomatoes,
		"What year is Postgres?":  database,
		// A one-word question on a summary word.
		"Which editor do I use?": editor,
		// A filler-wrapped short question, judged on its clause.
		"Before we start, remind me: Which editor do I use? One line is fine.": editor,
		// A name the prompt capitalizes still anchors beside an absent word.
		"Does Beacon work at a wedding?": beacon,
		// Two body words are more than a lone generic one.
		"Phone number in my signature?": signature,
		// A name no record holds, beside a match in a record's summary.
		"Which database does Tidepool use for the Django port?": database,
	} {
		if ranks := contextRanks(contextFor(t, m, "prompt", prompt)); len(ranks) == 0 || ranks[0] != want {
			t.Errorf("%q injected %v, want %s first", prompt, ranks, want)
		}
	}
}

// #232 item 5: the newer of two decisions on one subject comes first, and
// the hook never gives the older without it.
func TestMemoryContextNewerDecisionFirst(t *testing.T) {
	m, ids := selectionService(t,
		memory.Write{Kind: "decision", Summary: "Spam threshold", Body: "Spam score above 5 goes to Junk through SpamAssassin; review the Junk folder weekly; spam score is checked on every message."},
		memory.Write{Kind: "decision", Summary: "Spam threshold", Body: "Spam score above 3 goes to Junk through Rspamd; review the Junk folder weekly."},
	)
	older, newer := ids[0], ids[1]
	for prompt, want := range map[string]string{
		"What spam score sends mail to Junk?": newer + " " + older,
		// Only the older matches; the newer comes with it, ahead.
		"Do we use SpamAssassin?": newer + " " + older,
		// Only the newer matches; the older isn't added for its sake.
		"Do we use Rspamd?": newer,
	} {
		if got := strings.Join(contextRanks(contextFor(t, m, "prompt", prompt)), " "); got != want {
			t.Errorf("%q injected %q, want %q", prompt, got, want)
		}
	}
	p, err := m.Recall("What spam score sends mail to Junk?", nil, 5, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Current) < 2 || p.Current[0].RecordID != newer || p.Current[1].RecordID != older {
		t.Errorf("memory_recall ranked %+v, want %s then %s", p.Current, newer, older)
	}
}

func TestContextLead(t *testing.T) {
	for prompt, want := range map[string]string{
		"Restic or Borg?":              "Restic",
		"Restic vs Borg":               "Restic",
		"Borg or Restic?":              "Borg",
		"restic or Borg?":              "",
		"Restic?":                      "",
		"Should I use Restic or Borg?": "",
		"Format a date in JavaScript.": "",
		"What about Restic?":           "",
	} {
		if got := lead(prompt); got != want {
			t.Errorf("lead(%q) = %q, want %q", prompt, got, want)
		}
	}
}

// TestEitherOrLeads: #264's rule reaches lowercase and mixed-case bare
// either-or prompts, and no capitalised form, which keeps lead's one lead and
// a single judgement.
func TestEitherOrLeads(t *testing.T) {
	for prompt, want := range map[string][]string{
		"kopia or borg?":           {"kopia", "borg"},
		"borg vs kopia":            {"borg", "kopia"},
		"KOPIA OR borg?":           {"KOPIA", "borg"},
		"kopia or borg or restic?": {"kopia", "borg", "restic"},
		"Kopia or borg?":           {"Kopia", "borg"},
		"restic or Borg?":          {"restic"},
		// Capitalised forms: lead already gives the one lead.
		"Kopia or Borg?":             nil,
		"Kopia vs Borg":              nil,
		"Beacon or Cortana or Siri?": nil,
		// Not the bare shape.
		"Kopia vs. Borg":              nil,
		"kopia vs. borg":              nil,
		"kopia versus borg":           nil,
		"go or rust?":                 nil,
		"this or that?":               nil,
		"kopia?":                      nil,
		"In one line: kopia or borg?": nil,
		"should i use kopia or borg?": nil,
	} {
		if got := eitherOrLeads(prompt); !slices.Equal(got, want) {
			t.Errorf("eitherOrLeads(%q) = %q, want %q", prompt, got, want)
		}
	}
}

// #232's synthetic repro: on 5b87ef5 a bare "A or B?" or "A vs B", where A is
// in several records' summaries and B in none, injected nothing, while
// "B or A?" injected A's records. Each now injects all three.
func TestMemoryContextEitherOrOnASummaryName(t *testing.T) {
	m, ids := selectionService(t,
		memory.Write{Kind: "procedure", Summary: "Kopia backup schedule", Body: "Snapshots the NAS nightly to the offsite bucket."},
		memory.Write{Kind: "decision", Summary: "Kopia retention", Body: "Keep 14 daily and 8 weekly snapshots."},
		memory.Write{Kind: "fact", Summary: "Kopia repository password", Body: "Stored in the password manager under the NAS entry."},
	)
	for _, prompt := range []string{
		"Kopia or Borg?", "Borg or Kopia?", "Kopia vs Borg", "Borg vs Kopia",
		// #264: 2.5.0 injected nothing for these.
		"kopia or borg?", "borg or kopia?", "kopia vs borg", "borg vs kopia",
		"kopia or Borg?", "Kopia or borg?", "In one line: kopia or borg?",
	} {
		got := contextRanks(contextFor(t, m, "prompt", prompt))
		slices.Sort(got)
		want := slices.Sorted(slices.Values(ids[:3]))
		if !slices.Equal(got, want) {
			t.Errorf("%q injected %v, want the 3 Kopia records %v", prompt, got, want)
		}
	}
	// "vs." is outside the fix: "Borg" after it starts a sentence, so it isn't
	// a name, and these stay as in 2.5.0.
	for _, prompt := range []string{"Kopia vs. Borg", "Borg vs. Kopia"} {
		if got := contextRanks(contextFor(t, m, "prompt", prompt)); len(got) != 0 {
			t.Errorf("%q injected %v, want nothing, as in 2.5.0", prompt, got)
		}
	}
}

// #264's second judgement adds a conflict as it adds a record: a forked
// Kopia record is flagged for "kopia or borg?" as for "Kopia or Borg?".
func TestMemoryContextEitherOrAddsConflicts(t *testing.T) {
	m, ids := selectionService(t, memory.Write{Kind: "decision", Summary: "Kopia retention", Body: "Keep 14 daily snapshots."})
	h, err := m.HistoryPage(ids[0], 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"Keep 7 daily snapshots.", "Keep 30 daily snapshots."} {
		if _, err := m.Remember(memory.Write{Kind: "decision", RecordID: ids[0], Supersedes: []string{h.Items[0].ID}, Summary: "Kopia retention", Body: body, Basis: "user-direction", Reason: "Synthetic fork"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, prompt := range []string{"Kopia or Borg?", "kopia or borg?"} {
		if text := contextFor(t, m, "prompt", prompt); !strings.Contains(text, "- Conflict: "+ids[0]+" has 2 competing revisions") {
			t.Errorf("%q: conflict not flagged:\n%s", prompt, text)
		}
	}
}

// TestEitherOrProbes measures what #264's rule reaches: lowercase bare
// either-or prompts, which the evaluation set doesn't hold. For each, the rule
// may add only records the capitalised form injects, and drops none that
// 2.5.0 (the rule off) injects, so "panel or turbine?" keeps its body matches.
func TestEitherOrProbes(t *testing.T) {
	m, _ := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	var probes struct {
		Records          []memory.Write
		SummaryCollision []string `json:"summary_collision"`
		NoRecord         []string `json:"no_record"`
		BothKnown        []string `json:"both_known"`
	}
	data, err := os.ReadFile("testdata/either-or-probes.json")
	if err == nil {
		err = json.Unmarshal(data, &probes)
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(probes.SummaryCollision) != 12 || len(probes.NoRecord) != 8 || len(probes.BothKnown) != 8 {
		t.Fatalf("probe groups hold %d, %d and %d prompts, want 12, 8 and 8", len(probes.SummaryCollision), len(probes.NoRecord), len(probes.BothKnown))
	}
	for _, w := range probes.Records {
		w.Basis, w.Reason = "observation", "Synthetic either-or probe"
		if _, err := m.Remember(w); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { contextEitherOrLeads = true }()
	for _, lower := range slices.Concat(probes.SummaryCollision, probes.NoRecord, probes.BothKnown) {
		words := strings.Fields(lower)
		for i, w := range words {
			if w != "or" && w != "vs" {
				words[i] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		capital := strings.Join(words, " ")
		contextEitherOrLeads = false
		off := contextRanks(contextFor(t, m, "prompt", lower))
		contextEitherOrLeads = true
		on, c := contextRanks(contextFor(t, m, "prompt", lower)), contextRanks(contextFor(t, m, "prompt", capital))
		t.Logf("%q: off %d %v, on %d %v; %q: %d %v", lower, len(off), off, len(on), on, capital, len(c), c)
		for _, id := range on {
			if !slices.Contains(off, id) && !slices.Contains(c, id) {
				t.Errorf("%q adds %s, which %q doesn't inject", lower, id, capital)
			}
		}
		for _, id := range off {
			if !slices.Contains(on, id) {
				t.Errorf("%q drops %s, which it injects with the rule off", lower, id)
			}
		}
	}
}
