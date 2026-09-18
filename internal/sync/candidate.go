package signetsync

import (
	"archive/tar"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const candidateArchiveLimit = 64 << 20

// Validate Git data without checking it out or executing attributes/hooks.
// The tree inventory is checked first, then every archived blob is hash-verified.
// The disposable directory belongs only to this operation, not to a user clone.
func (s *Synchronizer) validateCandidate(ctx context.Context, candidate string, bases ...string) error {
	for _, base := range bases {
		if base != "" {
			if err := s.appendOnly(ctx, base, candidate); err != nil {
				return err
			}
		}
	}
	files, err := s.candidateFiles(ctx, candidate)
	if err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "mandalore-candidate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	for name, data := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			return err
		}
	}
	other, err := memory.Open(root)
	if err != nil {
		return err
	}
	if other.Signet.ID != s.store.Signet.ID {
		return memory.ErrIdentityChanged
	}
	if err := (&Synchronizer{store: other}).validateWorktree(); err != nil {
		return err
	}
	if err := other.Validate(); err != nil {
		return err
	}
	anchors := append([]string{}, bases...)
	kind, err := s.git(ctx, "cat-file", "-t", candidate)
	if err != nil {
		return err
	}
	if kind == "commit" {
		anchors = []string{candidate}
	} else if kind == "tree" {
		head, err := s.head(ctx)
		if err != nil {
			return err
		}
		anchors = append(anchors, head)
	} else {
		return ErrHistory
	}
	return s.validateUpgradeProofs(ctx, files, anchors)
}

// candidateFiles verifies the complete bounded archive against its immutable
// tree before returning bytes. It does not trust archive attributes or paths.
func (s *Synchronizer) candidateFiles(ctx context.Context, candidate string) (map[string][]byte, error) {
	tree, err := s.git(ctx, "ls-tree", "-r", "-z", "--full-tree", candidate)
	if err != nil {
		return nil, err
	}
	expected := map[string]string{}
	for _, entry := range strings.Split(tree, "\x00") {
		if entry == "" {
			continue
		}
		meta, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[0] != "100644" || fields[1] != "blob" || !allowedPath(path) {
			return nil, ErrDirty
		}
		if _, exists := expected[path]; exists {
			return nil, ErrDirty
		}
		expected[path] = fields[2]
	}
	archive, err := s.gitOutput(ctx, candidateArchiveLimit, "archive", "--format=tar", candidate)
	if err != nil {
		return nil, err
	}
	reader := tar.NewReader(strings.NewReader(archive))
	seen := map[string][]byte{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		h, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeXGlobalHeader || h.Typeflag == tar.TypeDir {
			continue
		}
		oid, exists := expected[h.Name]
		if _, duplicate := seen[h.Name]; !exists || duplicate || h.Typeflag != tar.TypeReg || h.Size > 4<<20 || h.Size < 0 {
			return nil, ErrDirty
		}
		data, err := io.ReadAll(io.LimitReader(reader, (4<<20)+1))
		if err != nil {
			return nil, err
		}
		if int64(len(data)) != h.Size || blobID(data, len(oid)) != oid {
			return nil, ErrDirty
		}
		seen[h.Name] = data
	}
	if len(seen) != len(expected) {
		return nil, ErrDirty
	}
	return seen, nil
}

func blobID(data []byte, size int) string {
	encoded := append([]byte(fmt.Sprintf("blob %d\x00", len(data))), data...)
	if size == 40 {
		hash := sha1.Sum(encoded)
		return hex.EncodeToString(hash[:])
	}
	if size == 64 {
		hash := sha256.Sum256(encoded)
		return hex.EncodeToString(hash[:])
	}
	return ""
}
