package server

import (
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
	all := rates(r.Trials, func(evalTrial) bool { return true })
	heldOut := rates(r.Trials, func(tr evalTrial) bool { return tr.HeldOut })
	injected := 0
	for _, n := range r.NoMemory {
		injected += b2i(n.Bytes > 0)
	}
	for _, floor := range []struct {
		name      string
		got, want float64
	}{
		{"memory_context with memory_recall's first record", pct(agree, top), 90},
		{"memory_context top-1", all.Context1, 80},
		{"memory_context top-3", all.Context3, 85},
		{"held-out memory_context top-3", heldOut.Context3, 82},
		{"memory_recall top-1", all.Recall1, 65},
		{"memory_recall top-3", all.Recall3, 78},
		{"no false injection", 100 - pct(injected, len(r.NoMemory)), 85},
	} {
		if floor.got < floor.want {
			t.Errorf("%s %.1f%% is under its %.0f%% floor", floor.name, floor.got, floor.want)
		}
	}
	t.Logf("memory_context injected memory_recall's first record in %d of %d prompts (%.1f%%)", agree, top, pct(agree, top))
	if avg := average(r.Session); avg > 800 {
		t.Errorf("scripted session averages %.0f injected bytes per prompt, over the 800-byte tripwire", avg)
	}
}

// TestContextThresholdCurve logs the trade-off between the relevance
// thresholds, top-3 hits and false injections. It reruns the evaluation for
// each setting, so it runs only with MANDALORE_EVAL_CURVE=1.
func TestContextThresholdCurve(t *testing.T) {
	if os.Getenv("MANDALORE_EVAL_CURVE") == "" {
		t.Skip("set MANDALORE_EVAL_CURVE=1 to log the threshold curve")
	}
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	m.CacheReads()
	min, relative := contextMinStrength, contextRelativeStrength
	defer func() { contextMinStrength, contextRelativeStrength = min, relative }()
	t.Logf("min  rel  | top-1  top-3  held-out top-3 | false injection | session bytes/prompt")
	settings := [][2]float64{}
	for _, floor := range []float64{0, 0.1, 0.15, 0.17, 0.2, 0.25, 0.3, 0.4} {
		settings = append(settings, [2]float64{floor, relative})
	}
	for _, rel := range []float64{0, 0.5, 0.8} {
		settings = append(settings, [2]float64{min, rel})
	}
	for _, s := range settings {
		contextMinStrength, contextRelativeStrength = s[0], s[1]
		r := runEvaluation(t, m, e)
		all := rates(r.Trials, func(evalTrial) bool { return true })
		heldOut := rates(r.Trials, func(tr evalTrial) bool { return tr.HeldOut })
		injected := 0
		for _, n := range r.NoMemory {
			injected += b2i(n.Bytes > 0)
		}
		t.Logf("%.2f %.2f | %5.1f%% %5.1f%% %5.1f%%         | %5.1f%%          | %4.0f",
			s[0], s[1], all.Context1, all.Context3, heldOut.Context3, pct(injected, len(r.NoMemory)), average(r.Session))
	}
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
		text, err := memoryContext(m, in, cs.session(client, in.Session))
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
