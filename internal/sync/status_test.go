package signetsync

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func digest(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[path] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestStatusNeverWritesAndInvalidatesOldDeliveryOnNewMemory(t *testing.T) {
	s, a := fixture(t)
	before := digest(t, a.Root())
	out, err := s.Inspect(context.Background())
	if err != nil || out.State != "local-only" || out.Head != "" {
		t.Fatal(out, err)
	}
	if !reflect.DeepEqual(before, digest(t, a.Root())) {
		t.Fatal("uninitialized inspection wrote files")
	}
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	gitTest(t, a.Root(), "init", "--bare", "--initial-branch=main", remote)
	gitTest(t, a.Root(), "remote", "add", "origin", remote)
	if _, err := s.Sync(context.Background(), 10*time.Second); err != nil {
		t.Fatal(err)
	}
	before = digest(t, a.Root())
	out, err = s.Inspect(context.Background())
	if err != nil || out.State != "synchronized" || out.LastAttempt == nil || !out.LastAttempt.Delivered {
		t.Fatal(out, err)
	}
	if !reflect.DeepEqual(before, digest(t, a.Root())) {
		t.Fatal("inspection changed files/index/receipt")
	}
	remember(t, a, "New local memory")
	out, err = s.Inspect(context.Background())
	if err != nil || out.State != "pending" || !out.Dirty || out.LastAttempt == nil {
		t.Fatal("old receipt implied current delivery", out, err)
	}
}

func TestMalformedStatusIsNotSilentlyRepaired(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.Root(), ".mandalore", "sync-status.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	before := digest(t, a.Root())
	if _, err := s.Inspect(context.Background()); err == nil {
		t.Fatal("accepted invalid receipt")
	}
	if !reflect.DeepEqual(before, digest(t, a.Root())) {
		t.Fatal("repaired receipt during inspection")
	}
}

// largeCanonCache fills the ignored .mandalore/canon with enough files that
// listing every ignored path would exceed the Git output limit.
func largeCanonCache(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".mandalore", "canon", "source-"+strings.Repeat("a", 200))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for size, i := 0, 0; size <= commandOutputLimit; i++ {
		name := fmt.Sprintf("%06d-%s.md", i, strings.Repeat("b", 200))
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
		size += len(".mandalore/canon//") + len(filepath.Base(dir)) + len(name) + 1
	}
}

func TestLargeIgnoredCanonCacheKeepsInspectionWithinTheOutputLimit(t *testing.T) {
	s, a := fixture(t)
	ctx := context.Background()
	if _, err := s.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	largeCanonCache(t, a.Root())
	var limit *OutputLimitError
	if _, err := s.gitOutput(ctx, commandOutputLimit, "ls-files", "--others", "--ignored", "--exclude-standard", "-z"); !errors.As(err, &limit) || limit.Command != "ls-files" {
		t.Fatal("fixture does not exceed the old listing's limit", err)
	}
	if dirty, err := s.dirty(ctx); err != nil || dirty {
		t.Fatal("dirty on a clean clone with a large canon cache", dirty, err)
	}
	state, err := s.InspectClone(ctx)
	if err != nil || len(state.Uncommitted) != 0 || state.Remote != "none" {
		t.Fatal("inspect on a clean clone with a large canon cache", state, err)
	}

	// Ignored memory outside .mandalore is still found.
	if err := os.Mkdir(filepath.Join(a.Root(), ".git", "info"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.Root(), ".git", "info", "exclude"), []byte("memory/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := remember(t, a, "Ignored record")
	if dirty, err := s.dirty(ctx); err != nil || !dirty {
		t.Fatal("dirty missed ignored memory", dirty, err)
	}
	state, err = s.InspectClone(ctx)
	if err != nil || !strings.Contains(strings.Join(state.Uncommitted, "\n"), r.ID+".json") {
		t.Fatal("inspect missed ignored memory", state, err)
	}
}
