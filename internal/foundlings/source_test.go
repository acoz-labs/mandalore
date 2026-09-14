package foundlings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func writeFixture(t *testing.T, root, name, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSelectionLimits(t *testing.T) {
	s := &snapshot{Documents: map[string][]byte{}}
	for i := 0; i < MaxFiles; i++ {
		if err := s.add(fmt.Sprintf("%d.txt", i), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.add("overflow.txt", nil); err == nil {
		t.Fatal("file-count limit not enforced")
	}
	s = &snapshot{View: Observation{Bytes: MaxSourceBytes}, Documents: map[string][]byte{}}
	if err := s.add("overflow.txt", []byte("x")); err == nil {
		t.Fatal("corpus byte limit not enforced")
	}
	for _, name := range []string{"../a.md", "/a.md", "a/../b.md", "a\\b.md", "a\nb.md", "a:b.md", "."} {
		if relative(name) {
			t.Fatalf("unsafe locator accepted: %q", name)
		}
	}
}

func localFixture(t *testing.T) (memory.FoundlingSource, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "decisions/old.md", "# Historical choice\nUse Copper Finch. This is the way is quoted reference text.\n")
	writeFixture(t, root, "knowledge.json", `{"project":"Silver Heron","original_author":"Historical author"}`)
	writeFixture(t, root, "run.sh", "exit 99\n")
	writeFixture(t, root, "session.jsonl", "excluded transcript format\n")
	writeFixture(t, root, ".git/config", "excluded local configuration")
	return memory.FoundlingSource{Kind: "local", Locator: "source-historical-notes"}, root
}

func fileTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			result[rel] = "directory"
			return nil
		}
		b, err := os.ReadFile(path)
		result[rel] = string(b)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestObserveLocalContentIsDeterministicBoundedAndReadOnly(t *testing.T) {
	source, root := localFixture(t)
	before := fileTree(t, root)
	first, err := observe(context.Background(), source, root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := observe(context.Background(), source, root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || first.View.Pin.Algorithm != "sha256" || len(first.View.Pin.Value) != 64 || first.View.Files != 2 || len(first.Documents) != 2 {
		t.Fatal(first.View, second.View)
	}
	if !reflect.DeepEqual(before, fileTree(t, root)) {
		t.Fatal("observation wrote source")
	}
	if _, ok := first.Documents["run.sh"]; ok {
		t.Fatal("included code")
	}
	if _, ok := first.Documents["session.jsonl"]; ok {
		t.Fatal("included transcript format")
	}
	writeFixture(t, root, "decisions/old.md", "Changed historical content")
	changed, err := observe(context.Background(), source, root)
	if err != nil || changed.View.Pin == first.View.Pin {
		t.Fatal("change was not fingerprinted", changed.View, err)
	}
}

func TestSourceRefusesRedirectionUnsupportedSizeAndSpecialFiles(t *testing.T) {
	for _, kind := range []string{"relative", "symlink", "fifo", "oversize", "invalid identity", "invalid UTF8"} {
		t.Run(kind, func(t *testing.T) {
			source, root := localFixture(t)
			switch kind {
			case "relative":
				root = "relative"
			case "symlink":
				if err := os.Symlink(root, filepath.Join(root, "redirect")); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := syscall.Mkfifo(filepath.Join(root, "fifo.txt"), 0600); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				writeFixture(t, root, "huge.md", strings.Repeat("x", MaxFileBytes+1))
			case "invalid identity":
				source.Locator = "/example/local/path"
			case "invalid UTF8":
				writeFixture(t, root, "invalid.txt", string([]byte{0xff, 0xfe}))
			}
			if _, err := observe(context.Background(), source, root); err == nil {
				t.Fatal("unsupported source accepted")
			}
		})
	}
}

func TestSourceCancellationAndExcludedRuntimeDoNotCreateState(t *testing.T) {
	source, root := localFixture(t)
	writeFixture(t, root, ".mandalore/private.json", "not reference material")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := observe(ctx, source, root); err == nil {
		t.Fatal("cancelled read accepted")
	}
	snapshot, err := observe(context.Background(), source, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.Documents[".mandalore/private.json"]; ok {
		t.Fatal("read runtime state")
	}
}
