package server

import (
	"os"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/testfixture"
)

func TestZZDiag(t *testing.T) {
	m, e := testfixture.LoadEvaluation(t, "testdata/context-eval.json")
	prompts := append(append([]string{}, e.Collisions.Tune...), e.NoMemory.Tune...)
	if v := os.Getenv("ZZ_PROMPTS"); v != "" {
		prompts = strings.Split(v, "|")
	}
	for _, prompt := range prompts {
		text := contextFor(t, m, "prompt", prompt)
		if text == "" {
			continue
		}
		ids := contextRanks(text)
		t.Logf("INJECT %q terms=%q", prompt, significant(prompt))
		for _, q := range append([]string{prompt}, questionClauses(prompt)...) {
			p, _ := m.RecallAllNamed(significant(q), names(q), contextRecallLimit)
			for _, r := range p.Current {
				if rankOf(ids, r.RecordID) > 0 {
					t.Logf("   %-40s cov %.2f ev %.2f anc %.2f inc %v", r.RecordID[12:], p.Strength[r.ID]/contextMinStrength, p.Evidence[r.ID]/contextMinEvidence, p.Anchor[r.ID]/contextMinAnchor, p.Incidental[r.ID])
				}
			}
		}
	}
}
