package signetsync

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

type Status struct {
	SignetID          string `json:"signet_id"`
	State             string `json:"state"`
	Phase             string `json:"phase"`
	Head              string `json:"head,omitempty"`
	RemoteHead        string `json:"remote_head,omitempty"`
	RemoteID          string `json:"remote_id,omitempty"`
	CheckedAt         string `json:"checked_at"`
	Checkpointed      bool   `json:"checkpointed"`
	Delivered         bool   `json:"delivered"`
	SemanticConflicts int    `json:"semantic_conflicts"`
	Notice            string `json:"notice"`
}

type Synchronizer struct{ store *memory.Store }

func Open(root, expectedID string) (*Synchronizer, error) {
	s, err := memory.Open(root)
	if err != nil {
		return nil, err
	}
	if expectedID == "" || s.Signet.ID != expectedID {
		return nil, memory.ErrIdentityChanged
	}
	return &Synchronizer{store: s}, nil
}

func (s *Synchronizer) status() Status {
	return Status{SignetID: s.store.Signet.ID, State: "local-only", Phase: "checkpoint", CheckedAt: time.Now().UTC().Format(time.RFC3339Nano), Notice: "Local history only; remote delivery has not been checked."}
}

func (s *Synchronizer) Initialize(parent context.Context) (Status, error) {
	return s.checkpointOperation(parent, true)
}
func (s *Synchronizer) Checkpoint(parent context.Context) (Status, error) {
	return s.checkpointOperation(parent, false)
}

func (s *Synchronizer) checkpointOperation(parent context.Context, initialize bool) (Status, error) {
	out := s.status()
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	err := s.store.WithExclusiveLock(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.store.Validate(); err != nil {
			return err
		}
		if initialize {
			if _, err := os.Lstat(filepath.Join(s.store.Root, ".git")); os.IsNotExist(err) {
				if _, err := s.git(ctx, "init", "--initial-branch=main", "--template="); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
		return s.checkpoint(ctx, &out)
	})
	return out, failed(out, err)
}

func (s *Synchronizer) boundary(ctx context.Context) error {
	root := filepath.Join(s.store.Root, ".git")
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrBoundary
	}
	for _, state := range []string{"commondir", "gitdir", "shallow", "MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply", "BISECT_START", "info/attributes", "info/grafts", "info/sparse-checkout", "objects/info/alternates"} {
		if _, err := os.Lstat(filepath.Join(root, state)); !os.IsNotExist(err) {
			return ErrBoundary
		}
	}
	if partial, err := filepath.Glob(filepath.Join(root, "objects", "pack", "*.promisor")); err != nil || len(partial) > 0 {
		return ErrBoundary
	}
	for _, dir := range []string{"objects", "refs"} {
		info, err := os.Lstat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrBoundary
		}
	}
	top, err := s.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	top, err = filepath.EvalSymlinks(top)
	if err != nil {
		return ErrBoundary
	}
	expectedRoot, err := filepath.EvalSymlinks(s.store.Root)
	if err != nil || top != expectedRoot {
		return ErrBoundary
	}
	branch, err := s.git(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil || branch != "main" {
		return ErrBoundary
	}
	return nil
}

func (s *Synchronizer) head(ctx context.Context) (string, error) {
	head, err := s.git(ctx, "rev-parse", "--verify", "HEAD")
	if isExit(err, 128) {
		return "", nil
	}
	return head, err
}

func allowedPath(path string) bool {
	switch path {
	case "signet.json", ".gitignore", "README.md":
		return true
	}
	for _, pattern := range portablePaths {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

var portablePaths = []*regexp.Regexp{
	regexp.MustCompile(`^(memory|provenance|foundlings)/\.gitkeep$`),
	regexp.MustCompile(`^(memory/(records|events|sources|visibility)|provenance/(devices|upgrades)|foundlings/registrations)/\.gitkeep$`),
	regexp.MustCompile(`^(memory/(records|visibility)|foundlings/registrations)/[a-z][a-z0-9-]{2,127}/([a-z][a-z0-9-]{2,127}\.json|\.gitkeep)$`),
	regexp.MustCompile(`^(memory/sources|provenance/(devices|upgrades))/[a-z][a-z0-9-]{2,127}\.json$`),
	regexp.MustCompile(`^memory/events/[0-9]{4}/[0-9]{2}/[a-z][a-z0-9-]{2,127}\.json$`),
}

// The engine validates semantic documents; this boundary also refuses material
// outside the portable layout, including ignored files that Git add -f would see.
func (s *Synchronizer) validateWorktree() error {
	return filepath.WalkDir(s.store.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(s.store.Root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".git" || rel == ".mandalore" {
			if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return ErrBoundary
			}
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if rel == ".DS_Store" {
			return nil
		}
		if !entry.Type().IsRegular() || !allowedPath(filepath.ToSlash(rel)) {
			return ErrDirty
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > 4<<20 || (entry.Name() == ".gitkeep" && info.Size() != 0) {
			return ErrDirty
		}
		return nil
	})
}

func (s *Synchronizer) appendOnly(ctx context.Context, revisions ...string) error {
	args := append([]string{"diff", "--no-ext-diff", "--no-textconv", "--name-only", "--no-renames", "--diff-filter=DMRTUXB"}, revisions...)
	args = append(args, "--", "memory", "provenance", "foundlings/registrations", "signet.json")
	changed, err := s.git(ctx, args...)
	if err != nil {
		return err
	}
	if changed != "" {
		if changed != "signet.json" {
			return ErrHistory
		}
		return s.permittedManifestChange(ctx, revisions)
	}
	return nil
}

func (s *Synchronizer) checkpoint(ctx context.Context, out *Status) error {
	if err := s.boundary(ctx); err != nil {
		return err
	}
	if err := s.store.Validate(); err != nil {
		return err
	}
	if err := s.validateWorktree(); err != nil {
		return err
	}
	head, err := s.head(ctx)
	if err != nil {
		return err
	}
	out.Head = head
	tracked, err := s.git(ctx, "ls-files", "-z")
	if err != nil {
		return err
	}
	for _, path := range strings.Split(tracked, "\x00") {
		if path != "" && !allowedPath(path) {
			return ErrDirty
		}
	}
	staged, err := s.git(ctx, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return err
	}
	for _, path := range strings.Split(staged, "\x00") {
		if path == "" {
			continue
		}
		if !allowedPath(path) {
			return ErrDirty
		}
		if _, err := s.git(ctx, "diff", "--no-ext-diff", "--no-textconv", "--quiet", "--", path); err != nil {
			return ErrDirty
		}
	}
	if head != "" {
		if err := s.appendOnly(ctx, "HEAD"); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.git(ctx, "add", "--force", "--all", "--", ".", ":!.mandalore", ":!.DS_Store"); err != nil {
		return err
	}
	_, err = s.git(ctx, "diff", "--cached", "--quiet")
	if err == nil && head != "" {
		out.Checkpointed = true
		return nil
	}
	if err != nil && !isExit(err, 1) {
		return err
	}
	tree, err := s.git(ctx, "write-tree")
	if err != nil {
		return err
	}
	if err := s.validateCandidate(ctx, tree, head); err != nil {
		return err
	}
	args := []string{"commit-tree", tree, "-m", "Checkpoint signet memory"}
	if head != "" {
		args = append(args, "-p", head)
	}
	commit, err := s.git(ctx, args...)
	if err != nil {
		return err
	}
	if err := s.boundary(ctx); err != nil {
		return err
	}
	// Compare-and-swap the exact branch; never commit a racing index snapshot
	// or overwrite a concurrently advanced ref. Staged user work is preserved.
	if _, err := s.git(ctx, "update-ref", "refs/heads/main", commit, head); err != nil {
		return err
	}
	out.Head, err = s.head(ctx)
	if err != nil {
		return err
	}
	if out.Head == "" {
		return errors.New("checkpoint commit could not be verified")
	}
	out.Checkpointed = true
	return nil
}
