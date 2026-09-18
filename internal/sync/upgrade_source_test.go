package signetsync

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpgradeSourceRequiresExactCheckpointWithoutWrites(t *testing.T) {
	s, a := fixture(t)
	remember(t, a, "Synthetic checkpoint")
	if _, err := s.UpgradeSource(context.Background()); err == nil {
		t.Fatal("uninitialized source accepted")
	}
	checkpoint, err := s.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(a.Root(), ".git/index")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.UpgradeSource(context.Background())
	if err != nil || p.Head != checkpoint.Head || p.SignetID != a.ID() || len(p.ManifestSHA256) != 64 || len(p.PortableSHA256) != 64 || p.RootIdentity == "" {
		t.Fatal(p, err)
	}
	q, err := s.UpgradeSource(context.Background())
	if err != nil || !reflect.DeepEqual(p, q) {
		t.Fatal("preview source pins are unstable", p, q, err)
	}
	after, err := os.ReadFile(index)
	if err != nil || string(before) != string(after) {
		t.Fatal("preflight changed Git index", err)
	}
	remember(t, a, "Uncheckpointed new fact")
	if _, err := s.UpgradeSource(context.Background()); err == nil {
		t.Fatal("uncheckpointed evidence accepted")
	}
}

func TestUpgradeSourceSeesIndexHiddenChanges(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			s, a := fixture(t)
			r := remember(t, a, "Synthetic evidence")
			if _, err := s.Initialize(context.Background()); err != nil {
				t.Fatal(err)
			}
			rel := filepath.Join("memory/records", r.RecordID, r.ID+".json")
			gitTest(t, a.Root(), "update-index", flag, rel)
			file := filepath.Join(a.Root(), rel)
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, append(b, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			gitTest(t, a.Root(), "diff", "--quiet") // Establish why status alone is insufficient.
			if _, err := s.UpgradeSource(context.Background()); err == nil {
				t.Fatal("index flag concealed changed evidence")
			}
		})
	}
}

func TestUpgradeSourceRejectsStagingAndCancellation(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(a.Root(), "README.md")
	if err := os.WriteFile(file, []byte("Staged synthetic text"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, a.Root(), "add", "README.md")
	if _, err := s.UpgradeSource(context.Background()); err == nil {
		t.Fatal("staged additions accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.UpgradeSource(ctx); err != context.Canceled {
		t.Fatal("cancellation lost", err)
	}
}

func TestUpgradeInventoryIncludesPortableMetadataNotLocalState(t *testing.T) {
	s, a := fixture(t)
	if _, err := s.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	before, err := s.UpgradeSource(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.Root(), "README.md"), []byte("Synthetic portable metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Checkpoint(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, err := s.UpgradeSource(context.Background())
	if err != nil || after.PortableSHA256 == before.PortableSHA256 || after.Head == before.Head || after.ManifestSHA256 != before.ManifestSHA256 {
		t.Fatal("portable metadata not pinned separately from manifest", before, after, err)
	}
	if err := os.WriteFile(filepath.Join(a.Root(), ".mandalore/private-note"), []byte("Ignored local state"), 0600); err != nil {
		t.Fatal(err)
	}
	local, err := s.UpgradeSource(context.Background())
	if err != nil || local != after {
		t.Fatal("machine-local state entered portable digest", local, err)
	}
}
