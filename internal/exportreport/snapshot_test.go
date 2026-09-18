package exportreport

import (
	"context"
	"github.com/acoz-labs/mandalore/internal/memory"
	"os"
	"path/filepath"
	"testing"
)

func TestReadReportSnapshotClosedDataAndDigest(t *testing.T) {
	s := snapshotService(t)
	r, err := s.Remember(memory.Write{Kind: "fact", Summary: "Selected", Body: "Synthetic value", Basis: "observation", Reason: "Observed"})
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.AppendJournal("test", "Synthetic journal")
	if err != nil {
		t.Fatal(err)
	}
	// A read must not recreate ignored operational state.
	if err := os.RemoveAll(filepath.Join(s.Root(), ".mandalore")); err != nil {
		t.Fatal(err)
	}
	a, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if a.Signet.ID != s.ID() || len(a.Revisions) != 1 || a.Revisions[0].ID != r.ID || len(a.Journal) != 1 || a.Journal[0].ID != j.ID || len(a.Digest) != 64 {
		t.Fatal("incomplete snapshot")
	}
	b, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil || a.Digest != b.Digest {
		t.Fatal("unstable snapshot", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), ".mandalore")); !os.IsNotExist(err) {
		t.Fatal("snapshot created local state")
	}
	if _, err := s.Remember(memory.Write{Kind: "fact", Summary: "Next", Body: "New data", Basis: "observation", Reason: "Observed"}); err != nil {
		t.Fatal(err)
	}
	c, err := ReadReportSnapshot(context.Background(), s.Root())
	if err != nil || c.Digest == a.Digest {
		t.Fatal("source change not captured", err)
	}
}

func TestReportSnapshotRejectsUnvalidatedBytesAndLinks(t *testing.T) {
	for _, mode := range []string{"duplicate", "path", "link"} {
		t.Run(mode, func(t *testing.T) {
			s := snapshotService(t)
			file := filepath.Join(s.Root(), "provenance/devices/device-test.json")
			switch mode {
			case "duplicate":
				if err := os.WriteFile(file, []byte(`{"schema_version":1,"id":"device-test","label":"a","label":"b"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "path":
				if err := os.Rename(file, filepath.Join(filepath.Dir(file), "device-wrong.json")); err != nil {
					t.Fatal(err)
				}
			case "link":
				if err := os.Rename(file, file+".saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(file+".saved", file); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := ReadReportSnapshot(context.Background(), s.Root()); err == nil {
				t.Fatal("unvalidated source accepted")
			}
		})
	}
}

func TestReportSnapshotReadBudgetAndCancellation(t *testing.T) {
	s := snapshotService(t)
	if _, err := readReportSnapshot(context.Background(), s.Root(), 1); err == nil {
		t.Fatal("read budget ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadReportSnapshot(ctx, s.Root()); err != context.Canceled {
		t.Fatal("cancellation lost", err)
	}
}

func snapshotService(t *testing.T) *memory.Service {
	t.Helper()
	store, err := memory.Create(filepath.Join(t.TempDir(), "bank"), "Synthetic", "device-test", "Synthetic device")
	if err != nil {
		t.Fatal(err)
	}
	service, err := memory.OpenService(store.Root, memory.Authorship{DeviceID: "device-test", Actor: "Synthetic actor", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return service
}
