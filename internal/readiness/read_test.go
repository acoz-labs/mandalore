package readiness

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Invoke a fixture action at a deterministic cancellation checkpoint. This
// uses real files and a real cancellable context, not timing-dependent races.
type checkpointContext struct {
	context.Context
	checks int
	at     int
	action func()
}

func (c *checkpointContext) Err() error {
	c.checks++
	if c.checks == c.at {
		c.action()
	}
	return c.Context.Err()
}

func TestReadMetadataRegularAndBounded(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "metadata.json")
	const body = `{"schema_version":1}`
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readMetadata(context.Background(), file, int64(len(body)))
	if err != nil || string(got) != body {
		t.Fatalf("exact-bound read: got %q, error %v", got, err)
	}
	if got, err := readMetadata(context.Background(), file, int64(len(body)-1)); !errors.Is(err, errMetadataLimit) || got != nil {
		t.Fatalf("oversized read must return no partial data: %q, %v", got, err)
	}
	after, err := os.Stat(file)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("read changed metadata identity/mode/modification time", err)
	}
	actual, err := os.ReadFile(file)
	if err != nil || string(actual) != body {
		t.Fatal("read changed contents", err)
	}
}

func TestReadMetadataRefusesSpecialFilesAndRedirection(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "source")
	if err := os.WriteFile(file, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "redirected")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, link, fifo, "relative.json"} {
		got, err := readMetadata(context.Background(), path, 1024)
		if !errors.Is(err, errMetadataUnsafe) || got != nil {
			t.Fatalf("unsafe type/path was not refused: %q, %v", got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if got, err := readMetadata(context.Background(), filepath.Join(dir, "missing"), 1024); !os.IsNotExist(err) || got != nil {
		t.Fatalf("missing metadata was not distinguished: %q, %v", got, err)
	}
}

func TestReadMetadataCancellationAndInvalidBudgetReturnNoData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := readMetadata(ctx, "/absent/synthetic/metadata", 1024); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("cancelled read must not begin inspection: %q, %v", got, err)
	}
	for _, limit := range []int64{-1, 0, (4 << 20) + 1} {
		if got, err := readMetadata(context.Background(), "/absent/synthetic/metadata", limit); !errors.Is(err, errMetadataLimit) || got != nil {
			t.Fatalf("invalid read budget: %q, %v", got, err)
		}
	}
}

func TestReadMetadataCancellationBetweenChunksDiscardsPartialData(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "large-metadata")
	if err := os.WriteFile(file, make([]byte, 96<<10), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checkpoint := &checkpointContext{Context: ctx, at: 4, action: cancel}
	got, err := readMetadata(checkpoint, file, 96<<10)
	if !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("mid-read cancellation leaked partial data: %d bytes, %v", len(got), err)
	}
}

func TestReadMetadataGrowthAfterOpenRemainsBounded(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "growing-metadata")
	if err := os.WriteFile(file, []byte("small"), 0600); err != nil {
		t.Fatal(err)
	}
	checkpoint := &checkpointContext{Context: context.Background(), at: 3, action: func() {
		if err := os.WriteFile(file, make([]byte, 96<<10), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	got, err := readMetadata(checkpoint, file, 16)
	if !errors.Is(err, errMetadataLimit) || got != nil {
		t.Fatalf("growth after open escaped bound: %d bytes, %v", len(got), err)
	}
}
