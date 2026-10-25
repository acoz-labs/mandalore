package server

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

var large = testfixture.Corpus{Records: 1000, Depth: 1}

const unrelated = "What is the weather like in Lisbon this weekend, and should I pack an umbrella?"

func contextFor(t testing.TB, m *memory.Service, event, prompt string) string {
	t.Helper()
	text, err := memoryContext(m, ContextInput{Event: event, Prompt: prompt})
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestMemoryContextRanksAndStaysBounded(t *testing.T) {
	f := testfixture.New(t, testfixture.Corpus{Records: 1000, Depth: 2, ConflictEvery: 17})
	text := contextFor(t, f.Service, "prompt", "Where did we leave signal-000001? "+strings.Repeat("padding ", 400))
	lines := strings.Split(text, "\n")
	if lines[0] != contextHeader || !strings.Contains(lines[1], "signal-000001") || len(text) > contextDefaultBytes {
		t.Fatalf("%d bytes:\n%s", len(text), text)
	}
	// A broad prompt matches every record; the cap still holds, claims are whole.
	for _, budget := range []int{contextMinBytes, contextDefaultBytes, contextMaxBytes} {
		text, err := memoryContext(f.Service, ContextInput{Event: "prompt", Prompt: "fictional project checkpoint", BudgetBytes: &budget})
		if err != nil || len(text) > budget || !strings.HasPrefix(text, contextHeader) {
			t.Fatalf("budget %d: %d bytes: %v", budget, len(text), err)
		}
		for _, line := range strings.Split(text, "\n")[1:] {
			if strings.HasPrefix(line, "- Project") && !strings.Contains(line, "Stable evidence for a fictional project. Stable") {
				t.Fatalf("clipped claim %q", line)
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
		if _, err := memoryContext(f.Service, in); err == nil {
			t.Fatalf("invalid input accepted: %+v", in)
		}
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
		text, err := memoryContext(f.Service, ContextInput{Event: "prompt", Prompt: fmt.Sprintf("status of signal-%06d in the fictional project", i*25), BudgetBytes: &max})
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

func BenchmarkMemoryContext(b *testing.B) {
	f := testfixture.New(b, large)
	f.Service.CacheReads()
	max := contextMaxBytes
	testfixture.Measure(b, func() ([]byte, error) {
		text, err := memoryContext(f.Service, ContextInput{Event: "prompt", Prompt: "status of signal-000500 in the fictional project", BudgetBytes: &max})
		return []byte(text), err
	})
}
