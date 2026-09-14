package distribution

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stagingFixture(t *testing.T) ([]byte, CandidateTransport, string) {
	t.Helper()
	s, docs, archive, now := transportFixture(t)
	transport, err := transportClient(t, docs).inspectCandidate(context.Background(), s, now)
	if err != nil {
		t.Fatal(err)
	}
	m, err := VerifyCandidateArchive(context.Background(), bytes.NewReader(archive), int64(len(archive)), transport)
	if err != nil {
		t.Fatal(err)
	}
	return archive, transport, m.Identity()
}

type failingStagingReader struct {
	reader io.ReaderAt
	parent string
	cancel context.CancelFunc
}

func (r failingStagingReader) ReadAt(b []byte, off int64) (int, error) {
	entries, err := os.ReadDir(r.parent)
	if err != nil {
		return 0, err
	}
	if len(entries) > 0 {
		if r.cancel != nil {
			r.cancel()
		} else {
			return 0, io.ErrUnexpectedEOF
		}
	}
	return r.reader.ReadAt(b, off)
}

func TestStageCandidateRetainsPartialEffectsWithoutVerifiedResult(t *testing.T) {
	for _, kind := range []string{"read failure", "cancellation"} {
		t.Run(kind, func(t *testing.T) {
			archive, transport, identity := stagingFixture(t)
			parent := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reader := failingStagingReader{reader: bytes.NewReader(archive), parent: parent}
			if kind == "cancellation" {
				reader.cancel = cancel
			}
			path, m, err := StageCandidateArchive(ctx, reader, int64(len(archive)), transport, identity, parent)
			if err == nil || path == "" || m.SHA256 != "" {
				t.Fatal("partial staging claimed success or lost effects", path, m, err)
			}
			st, err := os.Stat(path)
			if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
				t.Fatal("partial directory was removed or lost privacy", err)
			}
		})
	}
}

func TestStageCandidatePreservesExactPayloadWithoutExecutionOrOverwrite(t *testing.T) {
	archive, transport, identity := stagingFixture(t)
	parent := t.TempDir()
	first, m, err := StageCandidateArchive(context.Background(), bytes.NewReader(archive), int64(len(archive)), transport, identity, parent)
	if err != nil || m.Identity() != identity {
		t.Fatal(first, m, err)
	}
	verified, err := VerifyDirectory(first)
	if err != nil || verified.Identity() != identity {
		t.Fatal("staged different candidate", err)
	}
	st, err := os.Stat(first)
	if err != nil || st.Mode().Perm() != 0700 {
		t.Fatal("staging directory must be private", err)
	}
	entries, err := os.ReadDir(first)
	if err != nil || len(entries) != 8 {
		t.Fatal(err)
	}
	for _, entry := range entries {
		st, err := entry.Info()
		if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
			t.Fatal("payload must be non-executable private regular bytes", err)
		}
	}
	second, _, err := StageCandidateArchive(context.Background(), bytes.NewReader(archive), int64(len(archive)), transport, identity, parent)
	if err != nil || first == second {
		t.Fatal("staging reused an existing directory", err)
	}
	if _, err := VerifyDirectory(first); err != nil {
		t.Fatal("second staging changed the first", err)
	}
}

func TestStageCandidateRefusesUnverifiedInputsBeforeCreatingFiles(t *testing.T) {
	for _, kind := range []string{"wrong identity", "corrupt archive", "wrong size", "cancelled", "parent link", "parent file"} {
		t.Run(kind, func(t *testing.T) {
			archive, transport, identity := stagingFixture(t)
			parent := t.TempDir()
			selected := parent
			size := int64(len(archive))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "wrong identity":
				identity = "mandalore:" + strings.Repeat("f", 40) + ":sha256:" + strings.Repeat("f", 64)
			case "corrupt archive":
				archive[0] ^= 1
			case "wrong size":
				size--
			case "cancelled":
				cancel()
			case "parent link":
				selected = filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(parent, selected); err != nil {
					t.Fatal(err)
				}
			case "parent file":
				selected = filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(selected, []byte("sentinel"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			path, _, err := StageCandidateArchive(ctx, bytes.NewReader(archive), size, transport, identity, selected)
			if err == nil || path != "" {
				t.Fatal("unverified archive was staged", path, err)
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 0 {
				t.Fatal("refusal wrote files", err)
			}
		})
	}
}
