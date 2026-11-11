package server

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// assertEvaluation holds the build to #227's acceptance on the evaluation set,
// raised to the numbers #232's triage set before its work started
// (docs/retrieval.md), so a ranking or packing change that loses selection
// quality fails here.
func assertEvaluation(t *testing.T, r evalReport) {
	t.Helper()
	// Acceptance 1 and 2: when memory_recall ranks the case's record first,
	// memory_context injects it, whatever the prompt's instruction phrasing.
	// A lexical ranker misses some (docs/retrieval.md), so this is a rate.
	agree, top := 0, 0
	for _, tr := range r.Trials {
		if tr.RecallTop == tr.Expect {
			top++
			if tr.ContextHasTop {
				agree++
			} else {
				t.Logf("left out memory_recall's first record: %s %s %q", tr.Case, tr.Form, tr.Wrapper)
			}
		}
	}
	// Context and recall floors are #232's triage numbers; held-out recall
	// keeps the floor #227 set before its held-out split was first run.
	type floor struct {
		name      string
		got, want float64
	}
	floors := []floor{{"memory_context with memory_recall's first record", pct(agree, top), 90}}
	for _, split := range []struct {
		heldOut                              bool
		context1, context3, recall1, recall3 float64
		falseInjection                       int
	}{{false, 78, 86, 75, 90, 6}, {true, 75, 84, 60, 80, 2}} {
		if split.heldOut && !heldOutRequested() {
			continue
		}
		name := map[bool]string{false: "tune", true: "held-out"}[split.heldOut]
		x := rates(r.Trials, func(tr evalTrial) bool { return tr.HeldOut == split.heldOut })
		injected, n := 0, 0
		for _, nm := range r.NoMemory {
			if nm.HeldOut == split.heldOut {
				n++
				injected += b2i(nm.Bytes > 0)
			}
		}
		floors = append(floors,
			floor{name + " memory_context top-1", x.Context1, split.context1},
			floor{name + " memory_context top-3", x.Context3, split.context3},
			floor{name + " memory_recall top-1", x.Recall1, split.recall1},
			floor{name + " memory_recall top-3", x.Recall3, split.recall3})
		if injected > split.falseInjection {
			t.Errorf("%s false injection on %d of %d no-memory prompts, over %d", name, injected, n, split.falseInjection)
		}
	}
	// Acceptance 2: no instruction wrapper in the tuning split falls far behind.
	for w, x := range wrapperRates(r.Trials, false) {
		floors = append(floors, floor{"tune memory_context top-3 inside " + w, x.Context3, 75})
	}
	// #232's triage numbers for the short questions, plain and inside
	// filler, the collision and generic-word prompts, and the newer of two
	// decisions on one subject.
	for _, c := range []struct {
		name      string
		got, want int
		atMost    bool
	}{
		{"plain short questions with the answer in memory_context's top 3", shortHits(r), 35, false},
		{"filler-wrapped short questions with the answer in memory_context's top 3", top3(r.Filler), 16, false},
		{"collision prompts injected", collisionsInjected(r), 3, true},
		{"generic two-word prompts injected", genericInjected(r), 0, true},
		{"generic-word guard prompts with the answer in memory_context's top 3", top3(r.Guard), len(r.Guard), false},
		// Two of the ten lose their answer to the lone-summary-word rule
		// ("Which editor do I use when writing Rust?"), a known limit.
		{"named guard prompts with the answer in memory_context's top 3", top3(r.Named), 8, false},
		{"older-only queries with the older decision in memory_recall's top 3", top3(r.OlderOnly), len(r.OlderOnly), false},
	} {
		if c.atMost && c.got > c.want || !c.atMost && c.got < c.want {
			t.Errorf("%s: %d, want %s %d", c.name, c.got, map[bool]string{true: "at most", false: "at least"}[c.atMost], c.want)
		}
	}
	for _, n := range r.Newer {
		if n.Recall != 1 || n.Context != 1 {
			t.Errorf("newer decision for %q ranks %d in memory_recall and %d in memory_context, want first in both", n.Prompt, n.Recall, n.Context)
		}
	}
	for _, f := range floors {
		if f.got < f.want {
			t.Errorf("%s %.1f%% is under its %.0f%% floor", f.name, f.got, f.want)
		}
	}
	t.Logf("memory_context injected memory_recall's first record in %d of %d prompts (%.1f%%)", agree, top, pct(agree, top))
	if avg := average(r.Session); avg > 800 {
		t.Errorf("scripted session averages %.0f injected bytes per prompt, over the 800-byte tripwire", avg)
	}
}

// TestNewerFirstCostsNoCase is #232's item 5 no-regression check: with the
// newer-first rule on, no case's expected record ranks lower, in
// memory_recall or memory_context, than with it off.
func TestNewerFirstCostsNoCase(t *testing.T) {
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	m.CacheReads()
	on := runEvaluation(t, m, e)
	memory.RankNewerFirst = false
	defer func() { memory.RankNewerFirst = true }()
	off := runEvaluation(t, m, e)
	worse := func(a, b int) bool { return a == 0 && b > 0 || a > 0 && b > 0 && a > b }
	for i, tr := range on.Trials {
		was := off.Trials[i]
		if worse(tr.Recall, was.Recall) || worse(tr.Context, was.Context) {
			t.Errorf("%s %s %q: memory_recall rank %d (off: %d), memory_context rank %d (off: %d)", tr.Case, tr.Form, tr.Wrapper, tr.Recall, was.Recall, tr.Context, was.Context)
		}
	}
}

// TestSameSubjectPairs is a tripwire on memory.OnOneSubject: on the
// evaluation signet it pairs the newer_decisions records and nothing else,
// so a refinement such as "Editor theme" beside "Editor" stays apart.
func TestSameSubjectPairs(t *testing.T) {
	_, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	want := map[[2]string]bool{}
	for _, q := range e.NewerDecisions.Tune {
		want[[2]string{q.Older, q.Expect}] = true
	}
	var revisions []memory.Revision
	for _, r := range e.Records {
		revisions = append(revisions, memory.Revision{RecordID: r.Record, Kind: r.Kind, Scope: e.Scopes[r.Record], Summary: r.Summary, Body: r.Body})
	}
	got := map[[2]string]bool{}
	for i, a := range revisions {
		for _, b := range revisions[i+1:] {
			if memory.OnOneSubject(a, b) {
				got[[2]string{a.RecordID, b.RecordID}] = true
				if !want[[2]string{a.RecordID, b.RecordID}] {
					t.Errorf("%s and %s pair as one subject outside newer_decisions", a.RecordID, b.RecordID)
				}
			}
		}
	}
	for pair := range want {
		if !got[pair] {
			t.Errorf("%s and %s don't pair as one subject", pair[0], pair[1])
		}
	}
}

// TestContextThresholdCurve logs the trade-off between the relevance
// thresholds, hits, false injections and bytes. It reruns the evaluation for
// each setting, so it runs only with MANDALORE_EVAL_CURVE set: "tune" logs the
// tuning split alone, which is all a tuning decision may look at, and "all"
// adds the held-out split.
func TestContextThresholdCurve(t *testing.T) {
	split := os.Getenv("MANDALORE_EVAL_CURVE")
	if split != "tune" && split != "all" {
		t.Skip("set MANDALORE_EVAL_CURVE=tune (or all) to log the threshold curve")
	}
	if split == "all" && !heldOutRequested() {
		t.Skip("MANDALORE_EVAL_CURVE=all also needs MANDALORE_EVAL_HELDOUT=1")
	}
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	m.CacheReads()
	coverage, evidence, anchor, relative := contextMinStrength, contextMinEvidence, contextMinAnchor, contextRelativeStrength
	defer func() {
		contextMinStrength, contextMinEvidence, contextMinAnchor, contextRelativeStrength = coverage, evidence, anchor, relative
	}()
	const off = 1000
	settings := [][4]float64{{0.001, off, off, relative}, {coverage, off, off, relative}, {coverage, evidence, off, relative}}
	for _, c := range []float64{0.2, 0.25, 0.3} {
		for _, e := range []float64{0.6, 0.7, 0.8} {
			for _, a := range []float64{0.6, 0.7, 0.8} {
				settings = append(settings, [4]float64{c, e, a, relative})
			}
		}
	}
	for _, rel := range []float64{0, 0.5, 0.8} {
		settings = append(settings, [4]float64{coverage, evidence, anchor, rel})
	}
	t.Logf("cov  ev      anchor  rel  | split    | ctx top-1 top-3 | worst wrapper top-3 | false inj | recall top-1 top-3 | bytes | short top-3 | collisions | filler top-3 | generic")
	for _, s := range settings {
		contextMinStrength, contextMinEvidence, contextMinAnchor, contextRelativeStrength = s[0], s[1], s[2], s[3]
		r := runEvaluation(t, m, e)
		for _, heldOut := range []bool{false, true} {
			if heldOut && split != "all" {
				continue
			}
			x := rates(r.Trials, func(tr evalTrial) bool { return tr.HeldOut == heldOut })
			worst := 100.0
			for _, w := range wrapperRates(r.Trials, heldOut) {
				worst = math.Min(worst, w.Context3)
			}
			injected, n := 0, 0
			for _, nm := range r.NoMemory {
				if nm.HeldOut == heldOut {
					n++
					injected += b2i(nm.Bytes > 0)
				}
			}
			name := map[bool]string{false: "tune", true: "held-out"}[heldOut]
			t.Logf("%.2f %7.2f %7.2f %.2f | %-8s | %5.1f%% %5.1f%%   | %5.1f%%              | %2d/%2d     | %5.1f%% %5.1f%%      | %4.0f  | %2d/%2d       | %2d/%2d      | %2d/%2d        | %2d/%2d",
				s[0], s[1], s[2], s[3], name, x.Context1, x.Context3, worst, injected, n, x.Recall1, x.Recall3, average(r.Session), shortHits(r), len(r.Short), collisionsInjected(r), len(r.Collisions), top3(r.Filler), len(r.Filler), genericInjected(r), len(r.Generic))
		}
	}
}

// wrapperRates gives each wrapper's rates on the split's prompts. A held-out
// prompt is any with the variant phrasing or a held-out wrapper.
func wrapperRates(trials []evalTrial, heldOut bool) map[string]evalRates {
	out := map[string]evalRates{}
	seen := map[string]bool{}
	for _, tr := range trials {
		if tr.HeldOut == heldOut && !seen[tr.Wrapper] {
			seen[tr.Wrapper] = true
			out[tr.Wrapper] = rates(trials, func(x evalTrial) bool { return x.HeldOut == heldOut && x.Wrapper == tr.Wrapper })
		}
	}
	return out
}

// TestContextSessionBytes reports the scripted session's injected bytes per
// prompt without a session ID, as the 2.1.0 plugin calls the hook, and with
// one, as the plugin calls it since #228, which de-duplicates within the
// session.
func TestContextSessionBytes(t *testing.T) {
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	m.CacheReads()
	cs := contextSessions{now: time.Now}
	client := Client{ID: "client-evaluation", Scope: ScopeRead}
	var without, with []int
	for _, prompt := range e.Session {
		without = append(without, len(contextFor(t, m, "prompt", prompt)))
		in := ContextInput{Event: "prompt", Prompt: prompt, Session: "session-evaluation"}
		text, err := contextOf(m, in, cs.session(client, in.Session))
		if err != nil {
			t.Fatal(err)
		}
		with = append(with, len(text))
	}
	t.Logf("scripted session, injected bytes per prompt: %.0f without a session ID, %.0f with one", average(without), average(with))
	t.Logf("per turn without: %v", without)
	t.Logf("per turn with:    %v", with)
	if average(with) >= average(without) {
		t.Fatal("de-duplication saved no bytes")
	}
}
