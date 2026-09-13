package signetsync

import (
	"context"
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
