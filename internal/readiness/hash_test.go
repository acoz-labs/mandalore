package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func hashFixture(t *testing.T, body []byte) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "synthetic-runtime")
	if err := os.WriteFile(path, body, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHashFileMeasuresBytesWithoutExecutionOrMutation(t *testing.T) {
	body := []byte("#!/bin/sh\nprintf executed > \"${0%/*}/execution-canary\"\nexit 97\n")
	path := hashFixture(t, body)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := hashFile(context.Background(), path, int64(len(body)), false)
	sum := sha256.Sum256(body)
	if err != nil || got.Path != path || got.SHA256 != hex.EncodeToString(sum[:]) || got.Size != int64(len(body)) || !got.Executable {
		t.Fatalf("incorrect measured identity: %+v, %v", got, err)
	}
	after, err := os.Stat(path)
	if err != nil || !sameMetadata(before, after) {
		t.Fatal("hash changed file metadata", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != string(body) {
		t.Fatal("hash changed bytes", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "execution-canary")); !os.IsNotExist(err) {
		t.Fatal("selected runtime was executed", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatal("hash created filesystem state", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	got, err = hashFile(context.Background(), path, int64(len(body)), false)
	if err != nil || got.Executable || got.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("file readability incorrectly implied executable permission", got, err)
	}
}

func TestHashFileOnlyAllowsExplicitExecutableRedirection(t *testing.T) {
	path := hashFixture(t, []byte("synthetic executable"))
	link := filepath.Join(filepath.Dir(path), "launcher")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if got, err := hashFile(context.Background(), link, 1024, false); !errors.Is(err, errMetadataUnsafe) || got != (fileDigest{}) {
		t.Fatal("managed path redirection accepted", got, err)
	}
	got, err := hashFile(context.Background(), link, 1024, true)
	if err != nil || got.Path != path || got.SHA256 == "" {
		t.Fatal("native launcher target not resolved", got, err)
	}
	// A target's digest is not the digest of this symbolic-link text.
	want, err := hashFile(context.Background(), path, 1024, false)
	if err != nil || got != want {
		t.Fatal("launcher and target identities differ", got, want, err)
	}
}

func TestHashFileRefusesSpecialPathsAndOversizeWithoutPartialIdentity(t *testing.T) {
	path := hashFixture(t, []byte("synthetic"))
	fifo := filepath.Join(filepath.Dir(path), "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Dir(path), fifo, "relative", "/synthetic/\ncontrol", "/" + strings.Repeat("a", 4096)} {
		if got, err := hashFile(context.Background(), path, 1024, true); err == nil || got != (fileDigest{}) {
			t.Fatal("unsafe path accepted", got, err)
		}
	}
	for _, limit := range []int64{-1, 0, 8, (512 << 20) + 1} {
		if got, err := hashFile(context.Background(), path, limit, false); !errors.Is(err, errMetadataLimit) || got != (fileDigest{}) {
			t.Fatal("invalid/insufficient hashing budget accepted", got, err)
		}
	}
	if got, err := hashFile(context.Background(), filepath.Join(filepath.Dir(path), "missing"), 1024, false); !os.IsNotExist(err) || got != (fileDigest{}) {
		t.Fatal("missing file incorrectly identified", got, err)
	}
}

func TestHashFileCancellationDoesNotReturnPartialDigest(t *testing.T) {
	path := hashFixture(t, make([]byte, 96<<10))
	for _, at := range []int{1, 4} {
		ctx, cancel := context.WithCancel(context.Background())
		checkpoint := &checkpointContext{Context: ctx, at: at, action: cancel}
		got, err := hashFile(checkpoint, path, 96<<10, false)
		cancel()
		if !errors.Is(err, context.Canceled) || got != (fileDigest{}) {
			t.Fatal("cancelled hash returned an identity", got, err)
		}
	}
}

func TestHashFileRejectsRetargetedLauncher(t *testing.T) {
	path := hashFixture(t, make([]byte, 96<<10))
	other := hashFixture(t, []byte("different runtime"))
	link := filepath.Join(filepath.Dir(path), "launcher")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	checkpoint := &checkpointContext{Context: context.Background(), at: 4, action: func() {
		replacement := filepath.Join(filepath.Dir(path), "new-link")
		if err := os.Symlink(other, replacement); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, link); err != nil {
			t.Fatal(err)
		}
	}}
	got, err := hashFile(checkpoint, link, 96<<10, true)
	if !errors.Is(err, errMetadataChanged) || got != (fileDigest{}) {
		t.Fatal("retargeted launcher returned a stale identity", got, err)
	}
}

func TestHashFileRejectsConcurrentGrowthReplacementAndModeChange(t *testing.T) {
	for _, change := range []string{"growth", "replacement", "mode"} {
		t.Run(change, func(t *testing.T) {
			path := hashFixture(t, make([]byte, 96<<10))
			checkpoint := &checkpointContext{Context: context.Background(), at: 4, action: func() {
				switch change {
				case "growth":
					if err := os.WriteFile(path, make([]byte, 128<<10), 0700); err != nil {
						t.Fatal(err)
					}
				case "replacement":
					replacement := filepath.Join(filepath.Dir(path), "replacement")
					if err := os.WriteFile(replacement, make([]byte, 96<<10), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(replacement, path); err != nil {
						t.Fatal(err)
					}
				case "mode":
					if err := os.Chmod(path, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}}
			got, err := hashFile(checkpoint, path, 96<<10, false)
			if got != (fileDigest{}) || (!errors.Is(err, errMetadataLimit) && !errors.Is(err, errMetadataChanged)) {
				t.Fatal("concurrent change was certified", got, err)
			}
		})
	}
}
