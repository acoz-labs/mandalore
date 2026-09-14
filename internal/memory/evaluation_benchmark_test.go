package memory_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

func BenchmarkRetrieval(b *testing.B) {
	for _, corpus := range testfixture.ScaleCorpora {
		b.Run(corpus.Name(), func(b *testing.B) {
			f := testfixture.New(b, corpus)
			b.Logf("corpus: records=%d revisions=%d conflicts=%d files=%d bytes=%d; setup excluded", f.Records, f.Revisions, f.Conflicts, f.Files, f.Bytes)
			for _, query := range []struct{ name, text string }{{"narrow", f.Query}, {"broad", "checkpoint"}, {"empty", ""}, {"no-match", "quartzabsent"}} {
				b.Run("warm/"+query.name, func(b *testing.B) {
					testfixture.Measure(b, func() ([]byte, error) {
						p, err := f.Service.Recall(query.text, &f.Scope, 5, 8192)
						if err != nil {
							return nil, err
						}
						if query.name == "narrow" && (len(p.Current) != 1 || p.Current[0].ID != f.ExpectedID) {
							return nil, fmt.Errorf("unexpected narrow recall")
						}
						return json.Marshal(p)
					})
				})
			}
			b.Run("fresh-service/narrow", func(b *testing.B) {
				testfixture.Measure(b, func() ([]byte, error) {
					s, err := memory.OpenService(f.Service.Root(), f.Author)
					if err != nil {
						return nil, err
					}
					p, err := s.Recall(f.Query, &f.Scope, 5, 8192)
					if err != nil {
						return nil, err
					}
					return json.Marshal(p)
				})
			})
			b.Run("scopes", func(b *testing.B) {
				testfixture.Measure(b, func() ([]byte, error) {
					p, err := f.Service.ScopePage(0, 5)
					if err != nil {
						return nil, err
					}
					return json.Marshal(p)
				})
			})
		})
	}
}
