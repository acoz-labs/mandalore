package server

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// assertEvaluation holds the build to #227's acceptance on the evaluation set,
// with floors a little under the measured rates (docs/retrieval.md), so a
// ranking or packing change that loses selection quality fails here.
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
	// Tune floors sit a little under the tuning split's rates. Held-out
	// floors were set before the held-out split was first run (round 1 of
	// #227 review) and are not moved to fit it.
	type floor struct {
		name      string
		got, want float64
	}
	floors := []floor{{"memory_context with memory_recall's first record", pct(agree, top), 90}}
	for _, split := range []struct {
		heldOut                              bool
		context1, context3, recall1, recall3 float64
		falseInjection                       float64
	}{{false, 75, 85, 72, 88, 15}, {true, 65, 80, 60, 80, 25}} {
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
			floor{name + " memory_recall top-3", x.Recall3, split.recall3},
			floor{name + " no false injection", 100 - pct(injected, n), 100 - split.falseInjection})
	}
	// Acceptance 2: no instruction wrapper in the tuning split falls far behind.
	for w, x := range wrapperRates(r.Trials, false) {
		floors = append(floors, floor{"tune memory_context top-3 inside " + w, x.Context3, 75})
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
	t.Logf("cov  ev      anchor  rel  | split    | ctx top-1 top-3 | worst wrapper top-3 | false inj | recall top-1 top-3 | bytes")
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
			t.Logf("%.2f %7.2f %7.2f %.2f | %-8s | %5.1f%% %5.1f%%   | %5.1f%%              | %2d/%2d     | %5.1f%% %5.1f%%      | %4.0f",
				s[0], s[1], s[2], s[3], name, x.Context1, x.Context3, worst, injected, n, x.Recall1, x.Recall3, average(r.Session))
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
// prompt without a session ID, as Claude Code's 2.1.0 plugin calls the hook,
// and with one, which de-duplicates within the session.
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
	if average(with) > average(without) {
		t.Fatal("de-duplication added bytes")
	}
}
