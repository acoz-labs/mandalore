package foundlings

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func fixtureGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture Git %v: %s: %v", args, out, err)
	}
	return strings.TrimSpace(string(out))
}

func gitFixture(t *testing.T, format string) (memory.FoundlingSource, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, root, "init", "--object-format="+format)
	fixtureGit(t, root, "remote", "add", "origin", "https://example.invalid/history.git")
	writeFixture(t, root, "notes/choice.md", "Historical choice: Copper Finch.\n")
	writeFixture(t, root, "image.png", "\x00binary")
	writeFixture(t, root, "session.jsonl", "excluded transcript")
	fixtureGit(t, root, "add", ".")
	fixtureGit(t, root, "commit", "-m", "Synthetic historical source")
	writeFixture(t, root, "untracked.md", "Not part of the selected commit")
	return memory.FoundlingSource{Kind: "git", Locator: "https://example.invalid/history.git"}, root
}

func TestGitObservationVerifiesTrackedTextWithoutSourceWrites(t *testing.T) {
	for _, format := range []string{"sha1", "sha256"} {
		t.Run(format, func(t *testing.T) {
			source, root := gitFixture(t, format)
			before := fileTree(t, root)
			s, err := observe(context.Background(), source, root)
			if err != nil {
				t.Fatal(err)
			}
			if s.View.Pin.Algorithm != "git-"+format || s.View.Pin.Value != fixtureGit(t, root, "rev-parse", "HEAD") || s.View.Files != 1 || len(s.Documents) != 1 {
				t.Fatal(s.View)
			}
			after := fileTree(t, root)
			if !reflect.DeepEqual(before, after) {
				for name, content := range before {
					if next, exists := after[name]; !exists || next != content {
						t.Logf("changed or removed fixture path: %s", name)
					}
				}
				for name := range after {
					if _, exists := before[name]; !exists {
						t.Logf("added fixture path: %s", name)
					}
				}
				t.Fatal("read changed source or Git metadata")
			}
			writeFixture(t, root, "notes/choice.md", "Edited but not committed")
			if _, err := observe(context.Background(), source, root); err == nil {
				t.Fatal("dirty tracked text accepted")
			}
		})
	}
}

func TestGitObservationRefusesWrongIdentityMissingObjectsAndRedirectedFiles(t *testing.T) {
	for _, kind := range []string{"identity", "missing tree", "missing blob", "symlink", "gitfile", "multiple origins"} {
		t.Run(kind, func(t *testing.T) {
			source, root := gitFixture(t, "sha1")
			switch kind {
			case "identity":
				source.Locator = "https://example.invalid/other.git"
			case "multiple origins":
				fixtureGit(t, root, "config", "--add", "remote.origin.url", source.Locator)
			case "missing tree":
				object := fixtureGit(t, root, "rev-parse", "HEAD^{tree}")
				if err := os.Remove(filepath.Join(root, ".git", "objects", object[:2], object[2:])); err != nil {
					t.Fatal(err)
				}
			case "missing blob":
				object := fixtureGit(t, root, "rev-parse", "HEAD:notes/choice.md")
				if err := os.Remove(filepath.Join(root, ".git", "objects", object[:2], object[2:])); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(filepath.Join(root, "notes", "choice.md")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, "untracked.md"), filepath.Join(root, "notes", "choice.md")); err != nil {
					t.Fatal(err)
				}
			case "gitfile":
				if err := os.Rename(filepath.Join(root, ".git"), filepath.Join(root, "metadata")); err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, ".git", "gitdir: metadata\n")
			}
			if _, err := observe(context.Background(), source, root); err == nil {
				t.Fatal("invalid source accepted")
			}
		})
	}
}

func TestGitProcessCancellationOutputLimitsAndSanitizedErrors(t *testing.T) {
	bin := t.TempDir()
	program := filepath.Join(bin, "git")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(program, []byte("#!/bin/sh\nsleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := sourceGit(ctx, bin, "", "version"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("process group outlived cancellation bound")
	}
	for _, script := range []string{
		"#!/bin/sh\nwhile :; do printf 'PRIVATE-CANARY-OUTPUT-BOUND\\n'; done\n",
		"#!/bin/sh\nwhile :; do printf 'PRIVATE-CANARY-OUTPUT-BOUND\\n' >&2; done\n",
		"#!/bin/sh\nprintf 'PRIVATE-CANARY-FAILURE' >&2\nexit 1\n",
	} {
		if err := os.WriteFile(program, []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := sourceGit(ctx, bin, "", "version")
		cancel()
		if err == nil || out != "" || strings.Contains(err.Error(), "PRIVATE-CANARY") || errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("unbounded/reflected subprocess output", err)
		}
	}
}

func TestGitObservationDoesNotExecuteSourceConfiguration(t *testing.T) {
	source, root := gitFixture(t, "sha1")
	canary := filepath.Join(t.TempDir(), "executed")
	program := filepath.Join(root, "canary.sh")
	writeFixture(t, root, "canary.sh", "#!/bin/sh\ntouch '"+canary+"'\nexit 99\n")
	if err := os.Chmod(program, 0700); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"core.fsmonitor", "filter.canary.clean", "filter.canary.smudge", "core.sshCommand", "core.askPass"} {
		fixtureGit(t, root, "config", key, program)
	}
	fixtureGit(t, root, "config", "core.hooksPath", filepath.Dir(program))
	writeFixture(t, root, ".gitattributes", "*.md filter=canary\n")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.fsmonitor")
	t.Setenv("GIT_CONFIG_VALUE_0", program)
	t.Setenv("GIT_DIR", filepath.Join(root, "not-the-repository"))
	before := fileTree(t, root)
	if _, err := observe(context.Background(), source, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(canary); !os.IsNotExist(err) {
		t.Fatal("source program executed", err)
	}
	if !reflect.DeepEqual(before, fileTree(t, root)) {
		t.Fatal("read changed source")
	}
}
