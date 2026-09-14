package codex_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/codex"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

func BenchmarkPromptHook(b *testing.B) {
	for _, corpus := range testfixture.ScaleCorpora[:3] {
		b.Run(corpus.Name(), func(b *testing.B) {
			f := testfixture.New(b, corpus)
			b.Logf("corpus: records=%d revisions=%d files=%d bytes=%d; setup excluded", f.Records, f.Revisions, f.Files, f.Bytes)
			testfixture.Measure(b, func() ([]byte, error) {
				var out bytes.Buffer
				err := codex.Run(f.Binding, strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"Recall the fictional project checkpoint, read-only."}`), &out)
				if err == nil && (out.Len() > 16384 || !bytes.Contains(out.Bytes(), []byte("scope inventory"))) {
					err = fmt.Errorf("unexpected prompt-hook response")
				}
				return out.Bytes(), err
			})
		})
	}
}
