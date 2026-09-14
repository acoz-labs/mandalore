package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestFileCountAndNonregularRefusals(t *testing.T) {
	t.Run("file count", func(t *testing.T) {
		o := legacyFixture(t)
		for n := 0; n < MaxFiles; n++ {
			put(t, o.Source, fmt.Sprintf("memory/sources/source-%05d.json", n), []byte(`{}`))
		}
		if _, err := Preflight(context.Background(), o); err == nil || !strings.Contains(err.Error(), "10000 files") {
			t.Fatal(err)
		}
	})
	t.Run("fifo", func(t *testing.T) {
		o := legacyFixture(t)
		if err := syscall.Mkfifo(filepath.Join(o.Source, "memory/sources/source-fifo.json"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Preflight(context.Background(), o); err == nil {
			t.Fatal("FIFO accepted")
		}
	})
}

func TestCancellationAndBusyApplyDoNotCreateOutput(t *testing.T) {
	o := legacyFixture(t)
	p, err := Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Apply(ctx, ApplyInput{Plan: p, WritersStopped: true}); err == nil {
		t.Fatal("cancelled apply accepted")
	}
	lock, err := os.OpenFile(filepath.Join(o.Source, ".my-friday/write.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	p, err = Preflight(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if _, err := Apply(context.Background(), ApplyInput{Plan: p, WritersStopped: true}); err == nil {
		t.Fatal("busy apply accepted")
	}
	if _, err := os.Lstat(o.Output); !os.IsNotExist(err) {
		t.Fatal("cancelled or busy apply wrote output")
	}
}

func TestStageSyncFailureAndCancellationRetainUnpublishedStage(t *testing.T) {
	for _, kind := range []string{"io", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			o := legacyFixture(t)
			p, err := Preflight(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, err := apply(ctx, ApplyInput{Plan: p, WritersStopped: true}, publishDirectory, func(string) error {
				if kind == "cancel" {
					cancel()
					return ctx.Err()
				}
				return fmt.Errorf("injected I/O failure")
			})
			if err == nil || r.Published || r.Staging == "" || r.Phase != "staging" {
				t.Fatal(r, err)
			}
			if _, err := os.Lstat(o.Output); !os.IsNotExist(err) {
				t.Fatal("failed stage published")
			}
		})
	}
}
