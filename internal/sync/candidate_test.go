package signetsync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateRejectsUnsafeDataAndRewrittenEvidence(t *testing.T) {
	for _, kind := range []string{"symlink", "foreign-identity", "invalid-json", "rewritten-evidence", "nonempty-placeholder"} {
		t.Run(kind, func(t *testing.T) {
			s, a := fixture(t)
			r := remember(t, a, "Copper Finch")
			initial, err := s.Initialize(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(a.Root(), "memory", "records", r.RecordID, r.ID+".json")
			switch kind {
			case "symlink":
				if err := os.Symlink("/nonexistent-synthetic-target", filepath.Join(a.Root(), "README.md")); err != nil {
					t.Fatal(err)
				}
			case "foreign-identity":
				_, other := fixture(t)
				data, err := os.ReadFile(filepath.Join(other.Root(), "signet.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(a.Root(), "signet.json"), data, 0600); err != nil {
					t.Fatal(err)
				}
			case "invalid-json":
				if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "rewritten-evidence":
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(strings.Replace(string(data), "Copper Finch", "Silver Heron", 1)), 0600); err != nil {
					t.Fatal(err)
				}
			case "nonempty-placeholder":
				if err := os.WriteFile(filepath.Join(a.Root(), "memory", ".gitkeep"), []byte("Unrelated material"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			gitTest(t, a.Root(), "add", "--all")
			gitTest(t, a.Root(), "commit", "-m", "Untrusted candidate fixture")
			candidate := gitTest(t, a.Root(), "rev-parse", "HEAD")
			if err := s.validateCandidate(context.Background(), candidate, initial.Head); err == nil {
				t.Fatal("unsafe candidate accepted")
			}
		})
	}
}

func TestExplicitLocalGitAuthorshipOverridesSyntheticDefault(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "config", "--local", "user.name", "Local contributor")
	gitTest(t, a.Root(), "config", "--local", "user.email", "local@example.invalid")
	remember(t, a, "Another fact")
	if _, err := s.Checkpoint(context.Background()); err != nil {
		t.Fatal(err)
	}
	if author := gitTest(t, a.Root(), "log", "-1", "--format=%an <%ae>"); author != "Local contributor <local@example.invalid>" {
		t.Fatal(author)
	}
}
