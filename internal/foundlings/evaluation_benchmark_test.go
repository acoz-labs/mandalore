package foundlings_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

func BenchmarkFoundlingRetrieval(b *testing.B) {
	for _, files := range []int{100, 1000} {
		b.Run(fmt.Sprintf("files-%d", files), func(b *testing.B) {
			f := testfixture.New(b, testfixture.Corpus{Records: 10, Depth: 1})
			root, sourceBytes := b.TempDir(), 0
			for i := range files {
				data := []byte(fmt.Sprintf("signal-%06d. %s", i, strings.Repeat("Archived synthetic reference evidence. ", 12)))
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("note-%06d.md", i)), data, 0600); err != nil {
					b.Fatal(err)
				}
				sourceBytes += len(data)
			}
			m := foundlings.New(f.Service)
			view, err := m.Preview(context.Background(), memory.FoundlingSource{Kind: "local", Locator: "source-evaluation"}, root)
			if err != nil {
				b.Fatal(err)
			}
			r, err := f.Service.WriteFoundling(memory.FoundlingWrite{Name: "Synthetic historical notes", Description: "Evaluation only", Source: view.Source, Pin: view.Pin, State: "active", Reason: "Synthetic explicit reference"})
			if err != nil {
				b.Fatal(err)
			}
			if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
				b.Fatal(err)
			}
			b.Logf("reference: files=%d bytes=%d; signet records=10; setup excluded; source verification included", files, sourceBytes)
			for _, query := range []struct{ name, text string }{{"narrow", "signal-000001"}, {"broad", "reference"}, {"no-match", "quartzabsent"}} {
				b.Run("search/"+query.name, func(b *testing.B) {
					testfixture.Measure(b, func() ([]byte, error) {
						p, err := m.Search(context.Background(), foundlings.SearchInput{FoundlingID: r.FoundlingID, Query: query.text, Limit: 5})
						if err != nil {
							return nil, err
						}
						if query.name == "narrow" && (len(p.Items) != 1 || p.Items[0].Origin.RelativeLocator != "note-000001.md") {
							return nil, fmt.Errorf("unexpected reference evidence")
						}
						return json.Marshal(p)
					})
				})
			}
			b.Run("read", func(b *testing.B) {
				testfixture.Measure(b, func() ([]byte, error) {
					p, err := m.Read(context.Background(), foundlings.ReadInput{FoundlingID: r.FoundlingID, RegistrationID: r.ID, Locator: "note-000001.md", Limit: 4096})
					if err != nil {
						return nil, err
					}
					return json.Marshal(p)
				})
			})
		})
	}
}
