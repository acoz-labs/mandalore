package signetsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

// UpgradeSource pins an existing format1 checkpoint. It neither activates a
// format nor authorizes a binding; the caller must additionally pin its binding.
// PortableSHA256 frames every checkpointed portable file's path and raw bytes in
// lexical path order, including README, .gitignore and empty placeholders.
type UpgradeSource struct {
	SignetID       string `json:"signet_id"`
	Head           string `json:"head"`
	RootIdentity   string `json:"root_identity"`
	ManifestSHA256 string `json:"manifest_sha256"`
	PortableSHA256 string `json:"portable_sha256"`
}

var ErrUpgradeSource = errors.New("upgrade requires a valid format1 signet with an exact clean checkpoint; inspect source and Git state before previewing again")

// UpgradeSource is read-only: no lockfile, index refresh, checkpoint, fetch or
// temporary candidate is created. Actual file bytes are compared with Git blobs;
// assume-unchanged, skip-worktree and filemode configuration cannot hide changes.
func (s *Synchronizer) UpgradeSource(ctx context.Context) (out UpgradeSource, err error) {
	defer func() {
		if err != nil {
			out = UpgradeSource{}
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = ErrUpgradeSource
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return
	}
	if s.store.Signet.Version != 1 {
		return out, ErrUpgradeSource
	}
	if err = s.boundary(ctx); err != nil {
		return
	}
	if err = s.store.Validate(); err != nil {
		return
	}
	if err = s.validateWorktree(); err != nil {
		return
	}
	head, err := s.head(ctx)
	if err != nil || head == "" {
		return out, ErrUpgradeSource
	}
	if _, err = s.git(ctx, "diff", "--cached", "--no-ext-diff", "--no-textconv", "--quiet", head, "--"); err != nil {
		return
	}
	tree, err := s.git(ctx, "ls-tree", "-r", "-z", "--full-tree", head)
	if err != nil {
		return
	}
	expected := map[string]string{}
	for _, entry := range strings.Split(tree, "\x00") {
		if entry == "" {
			continue
		}
		meta, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[0] != "100644" || fields[1] != "blob" || !allowedPath(path) || expected[path] != "" {
			return out, ErrUpgradeSource
		}
		expected[path] = fields[2]
	}
	root, err := os.OpenRoot(s.store.Root)
	if err != nil {
		return
	}
	defer root.Close()
	identity, err := root.Stat(".")
	if err != nil {
		return
	}
	stat, ok := identity.Sys().(*syscall.Stat_t)
	if !ok {
		return out, ErrUpgradeSource
	}
	inventory := sha256.New()
	seen := map[string]bool{}
	var total int64
	manifestHash := ""
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == ".git" || path == ".mandalore" {
			if !d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return ErrUpgradeSource
			}
			return fs.SkipDir
		}
		if path == ".DS_Store" {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return ErrUpgradeSource
		}
		if d.IsDir() {
			return nil
		}
		oid, exists := expected[path]
		if !exists || !d.Type().IsRegular() {
			return ErrUpgradeSource
		}
		before, err := root.Lstat(path)
		if err != nil {
			return err
		}
		if before.Size() > 4<<20 || before.Mode().Perm()&0111 != 0 {
			return ErrUpgradeSource
		}
		f, err := root.Open(path)
		if err != nil {
			return err
		}
		opened, err := f.Stat()
		if err != nil || !os.SameFile(before, opened) {
			f.Close()
			return ErrUpgradeSource
		}
		data, readErr := io.ReadAll(io.LimitReader(f, (4<<20)+1))
		closeErr := f.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		total += int64(len(data))
		if len(data) > 4<<20 || total > 128<<20 || blobID(data, len(oid)) != oid {
			return ErrUpgradeSource
		}
		after, err := root.Lstat(path)
		if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
			return ErrUpgradeSource
		}
		fmt.Fprintf(inventory, "%d:%s:%d:", len(path), path, len(data))
		_, _ = inventory.Write(data)
		if path == "signet.json" {
			h := sha256.Sum256(data)
			manifestHash = hex.EncodeToString(h[:])
		}
		seen[path] = true
		return nil
	})
	if err != nil {
		return
	}
	if len(seen) != len(expected) || manifestHash == "" {
		return out, ErrUpgradeSource
	}
	current, err := os.Lstat(s.store.Root)
	if err != nil || !current.IsDir() || !os.SameFile(identity, current) {
		return out, ErrUpgradeSource
	}
	if err = s.boundary(ctx); err != nil {
		return
	}
	again, err := s.head(ctx)
	if err != nil || again != head {
		return out, ErrUpgradeSource
	}
	if _, err = s.git(ctx, "diff", "--cached", "--no-ext-diff", "--no-textconv", "--quiet", head, "--"); err != nil {
		return
	}
	return UpgradeSource{SignetID: s.store.Signet.ID, Head: head, RootIdentity: fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), ManifestSHA256: manifestHash, PortableSHA256: hex.EncodeToString(inventory.Sum(nil))}, nil
}
