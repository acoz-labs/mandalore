package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// Opt-in benchmark: build and fixture setup are excluded. Includes child process
// startup, service/recall/encoding, pipe transfer and client result validation.
// B/op describes the parent benchmark process, not the child's allocations.
func BenchmarkCompiledRetrieval(b *testing.B) {
	binary := filepath.Join(b.TempDir(), "mandalore")
	buildContext, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(buildContext, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		b.Fatal("compile evaluation CLI", string(out), err)
	}
	for _, corpus := range testfixture.ScaleCorpora[:3] {
		b.Run(corpus.Name(), func(b *testing.B) {
			f := testfixture.New(b, corpus)
			cwd := b.TempDir()
			b.Logf("corpus: records=%d revisions=%d files=%d bytes=%d; setup/build excluded; fresh process is not cold disk", f.Records, f.Revisions, f.Files, f.Bytes)
			testfixture.Measure(b, func() ([]byte, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, binary, "memory", "recall", "--binding", f.Binding, "--query", f.Query, "--scope-kind", f.Scope.Kind, "--scope-id", f.Scope.ID, "--limit", "5", "--budget-bytes", "8192", "--read-only")
				cmd.Dir = cwd
				data, err := cmd.Output()
				if err != nil {
					return nil, err
				}
				var result struct {
					OK     bool                `json:"ok"`
					Result memory.RecallPacket `json:"result"`
				}
				if err := json.Unmarshal(data, &result); err != nil {
					return nil, err
				}
				if !result.OK || len(result.Result.Current) != 1 || result.Result.Current[0].ID != f.ExpectedID {
					return nil, fmt.Errorf("compiled recall returned unexpected evidence")
				}
				return data, nil
			})
		})
	}
}
