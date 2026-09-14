package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/testfixture"
)

// Verification-branch benchmark: fixture setup is excluded. No CLI is built.
// Includes child process
// startup, service/recall/encoding, pipe transfer and client result validation.
// B/op describes the parent benchmark process, not the child's allocations.
func BenchmarkCompiledRetrieval(b *testing.B) {
	binary := os.Getenv("MANDALORE_EVAL_BIN")
	if !filepath.IsAbs(binary) {
		b.Fatal("explicit absolute retained candidate path required")
	}
	data, err := os.ReadFile(binary)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != "eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc" {
		b.Fatal("wrong or unavailable retained macOS ARM64 candidate")
	}
	versionContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	version := exec.CommandContext(versionContext, binary, "version")
	version.Env = []string{"PATH=/usr/bin:/bin:/usr/local/bin"}
	out, err := version.Output()
	var identity struct {
		OK     bool `json:"ok"`
		Result struct {
			Source string `json:"source_commit"`
			OS     string `json:"os"`
			Arch   string `json:"arch"`
		} `json:"result"`
	}
	if err != nil || json.Unmarshal(out, &identity) != nil || !identity.OK || identity.Result.Source != "5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f" || identity.Result.OS != "darwin" || identity.Result.Arch != "arm64" {
		b.Fatal("retained candidate version mismatch")
	}
	b.Logf("retained source=%s os=%s arch=%s binary-sha256=eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc; no candidate build", identity.Result.Source, identity.Result.OS, identity.Result.Arch)
	for _, corpus := range testfixture.ScaleCorpora[:3] {
		b.Run(corpus.Name(), func(b *testing.B) {
			f := testfixture.New(b, corpus)
			cwd := b.TempDir()
			before := retainedFixtureDigest(b, filepath.Dir(f.Binding))
			b.Logf("corpus: records=%d revisions=%d files=%d bytes=%d; setup excluded; fresh process is not cold disk", f.Records, f.Revisions, f.Files, f.Bytes)
			testfixture.Measure(b, func() ([]byte, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, binary, "memory", "recall", "--binding", f.Binding, "--query", f.Query, "--scope-kind", f.Scope.Kind, "--scope-id", f.Scope.ID, "--limit", "5", "--budget-bytes", "8192", "--read-only")
				cmd.Dir = cwd
				cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/local/bin"}
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
			if retainedFixtureDigest(b, filepath.Dir(f.Binding)) != before {
				b.Fatal("read-only benchmark changed fixture files")
			}
			entries, err := os.ReadDir(cwd)
			if err != nil || len(entries) != 0 {
				b.Fatal("benchmark changed unrelated cwd")
			}
			b.Log("read-only fixture hashes and unrelated cwd unchanged")
		})
	}
}

func retainedFixtureDigest(b *testing.B, root string) string {
	b.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%x\n", rel, sha256.Sum256(data))
		return nil
	})
	if err != nil {
		b.Fatal("cannot fingerprint synthetic fixture", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
