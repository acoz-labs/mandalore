package server

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// evalTrial is one prompt's outcome: the expected record's 1-based rank in
// memory_recall (scoped, tool defaults) and in memory_context's injected
// records, or 0 when it is absent.
type evalTrial struct {
	Case, Expect    string
	Form, Wrapper   string
	HeldOut         bool
	Recall, Context int
	// RecallTop is memory_recall's first record for the phrasing and
	// ContextHasTop whether memory_context injected it (acceptance 1).
	RecallTop     string
	ContextHasTop bool
	// Injected counts the records memory_context injected for the prompt.
	Injected int
}

// evalReport is everything one build measured on the evaluation set.
type evalReport struct {
	Trials []evalTrial
	// NoMemory holds the injected bytes for each prompt no record answers.
	NoMemory []struct {
		Prompt  string
		HeldOut bool
		Bytes   int
	}
	// Session holds the injected bytes for each turn of the scripted session.
	Session []int
}

var contextRecordID = regexp.MustCompile(`, (record-[a-z0-9-]+), device `)

// contextRanks lists the record IDs memory_context injected, in order.
func contextRanks(text string) []string {
	ids := []string{}
	for _, line := range strings.Split(text, "\n") {
		if m := contextRecordID.FindStringSubmatch(line); strings.HasPrefix(line, "- ") && m != nil {
			ids = append(ids, m[1])
		}
	}
	return ids
}

func rankOf(ids []string, want string) int {
	for i, id := range ids {
		if id == want {
			return i + 1
		}
	}
	return 0
}

type evalWrapper struct {
	text    string
	heldOut bool
}

func evalWrappers(e testfixture.Evaluation) []evalWrapper {
	wrappers := []evalWrapper{{"{q}", false}}
	for _, w := range e.Wrappers.Tune {
		wrappers = append(wrappers, evalWrapper{w, false})
	}
	for _, w := range e.Wrappers.HeldOut {
		wrappers = append(wrappers, evalWrapper{w, true})
	}
	return wrappers
}

// evalPrompts calls each case's phrasings with every wrapper. A prompt is held
// out when it uses the variant phrasing or a held-out wrapper, which no
// tuning decision looked at.
func evalPrompts(e testfixture.Evaluation, each func(c int, form string, text string, w evalWrapper)) {
	for i, c := range e.Cases {
		for _, form := range []struct{ name, text string }{{"question", c.Question}, {"paraphrase", c.Paraphrase}, {"variant", c.Variant}} {
			for _, w := range evalWrappers(e) {
				each(i, form.name, form.text, w)
			}
		}
	}
}

// heldOutRequested reports whether a run may compute the held-out split. It is
// computed once, after the code is frozen (docs/retrieval.md#evaluation-set),
// so every other run, bin/ci's included, scores the tuning split alone.
func heldOutRequested() bool { return os.Getenv("MANDALORE_EVAL_HELDOUT") == "1" }

// runEvaluation scores testdata/context-eval.json: memory_recall on each
// case's phrasings in the record's scope, memory_context on each
// prompt, on the prompts no record answers and on the scripted session.
func runEvaluation(t *testing.T, m *memory.Service, e testfixture.Evaluation) evalReport {
	t.Helper()
	var report evalReport
	recalled := map[string][]string{}
	evalPrompts(e, func(i int, form, text string, w evalWrapper) {
		c := e.Cases[i]
		if (form == "variant" || w.heldOut) && !heldOutRequested() {
			return
		}
		if _, ok := recalled[text]; !ok {
			scope := e.Scopes[c.Expect]
			p, err := m.Recall(text, &scope, 5, 8192)
			if err != nil {
				t.Fatal(err)
			}
			for _, h := range p.Current {
				recalled[text] = append(recalled[text], h.RecordID)
			}
		}
		top := ""
		if r := recalled[text]; len(r) > 0 {
			top = r[0]
		}
		injected := contextRanks(contextFor(t, m, "prompt", strings.ReplaceAll(w.text, "{q}", text)))
		report.Trials = append(report.Trials, evalTrial{Case: c.Name, Expect: c.Expect, Form: form, Wrapper: w.text, HeldOut: form == "variant" || w.heldOut,
			Recall: rankOf(recalled[text], c.Expect), Context: rankOf(injected, c.Expect), RecallTop: top, ContextHasTop: top != "" && rankOf(injected, top) > 0, Injected: len(injected)})
	})
	for _, set := range []struct {
		prompts []string
		heldOut bool
	}{{e.NoMemory.Tune, false}, {e.NoMemory.HeldOut, true}} {
		if set.heldOut && !heldOutRequested() {
			continue
		}
		for _, prompt := range set.prompts {
			report.NoMemory = append(report.NoMemory, struct {
				Prompt  string
				HeldOut bool
				Bytes   int
			}{prompt, set.heldOut, len(contextFor(t, m, "prompt", prompt))})
		}
	}
	for _, prompt := range e.Session {
		report.Session = append(report.Session, len(contextFor(t, m, "prompt", prompt)))
	}
	return report
}

// evalRates is one split's hit rates (percent) and prompt count.
type evalRates struct {
	Context1, Context3, Recall1, Recall3 float64
	Prompts, Queries                     int
}

func rates(trials []evalTrial, keep func(evalTrial) bool) evalRates {
	var r evalRates
	var c1, c3, r1, r3 int
	for _, tr := range trials {
		if !keep(tr) {
			continue
		}
		r.Prompts++
		c1 += b2i(tr.Context == 1)
		c3 += b2i(tr.Context >= 1 && tr.Context <= 3)
		// memory_recall is scored once per phrasing: on the bare prompt.
		if tr.Wrapper == "{q}" {
			r.Queries++
			r1 += b2i(tr.Recall == 1)
			r3 += b2i(tr.Recall >= 1 && tr.Recall <= 3)
		}
	}
	r.Context1, r.Context3 = pct(c1, r.Prompts), pct(c3, r.Prompts)
	r.Recall1, r.Recall3 = pct(r1, r.Queries), pct(r3, r.Queries)
	return r
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func pct(k, n int) float64 { return 100 * float64(k) / max(float64(n), 1) }

func average(bytes []int) float64 {
	total := 0
	for _, b := range bytes {
		total += b
	}
	return float64(total) / max(float64(len(bytes)), 1)
}

func logReport(t *testing.T, r evalReport) {
	t.Helper()
	for _, split := range []struct {
		name string
		keep func(evalTrial) bool
	}{{"tune", func(tr evalTrial) bool { return !tr.HeldOut }}, {"held-out", func(tr evalTrial) bool { return tr.HeldOut }}, {"all", func(evalTrial) bool { return true }}} {
		if split.name != "tune" && !heldOutRequested() {
			continue
		}
		x := rates(r.Trials, split.keep)
		t.Logf("%-8s memory_context top-1 %5.1f%% top-3 %5.1f%% (%d prompts); memory_recall top-1 %5.1f%% top-3 %5.1f%% (%d queries)",
			split.name, x.Context1, x.Context3, x.Prompts, x.Recall1, x.Recall3, x.Queries)
	}
	injected, heldOut, heldOutInjected := 0, 0, 0
	for _, n := range r.NoMemory {
		injected += b2i(n.Bytes > 0)
		if n.HeldOut {
			heldOut++
			heldOutInjected += b2i(n.Bytes > 0)
		}
	}
	t.Logf("false injection %d of %d no-memory prompts (%.1f%%; held out %d of %d)", injected, len(r.NoMemory), pct(injected, len(r.NoMemory)), heldOutInjected, heldOut)
	maxBytes := 0
	for _, b := range r.Session {
		maxBytes = max(maxBytes, b)
	}
	t.Logf("scripted session: %.0f injected bytes per prompt on average over %d turns, at most %d", average(r.Session), len(r.Session), maxBytes)
}

// TestContextSelectionEvaluation is #227's checked-in evaluation (acceptance
// 1, 2 and 4). Set MANDALORE_EVAL_REPORT to a file to write the whole report
// as JSON for a per-case comparison between two builds.
func TestContextSelectionEvaluation(t *testing.T) {
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	m.CacheReads() // as the server does
	r := runEvaluation(t, m, e)
	if path := os.Getenv("MANDALORE_EVAL_REPORT"); path != "" {
		data, _ := json.MarshalIndent(r, "", " ")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	logReport(t, r)
	assertEvaluation(t, r)
}
