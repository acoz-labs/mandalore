package signetsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

var errUpgradeRequired = errors.New("explicit local format upgrade required")

var endpointName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+-]*$`)

func validRemote(remote string) bool {
	if remote == "" || strings.ContainsAny(remote, "\x00\r\n\t") {
		return false
	}
	if filepath.IsAbs(remote) {
		return true
	}
	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return false
		}
		switch u.Scheme {
		case "file":
			return u.User == nil && (u.Host == "" || u.Host == "localhost") && filepath.IsAbs(u.Path)
		case "https":
			return u.Host != "" && u.User == nil && u.Path != ""
		case "ssh":
			if u.Path == "" || u.Path == "/" || (!endpointName.MatchString(u.Hostname()) && net.ParseIP(u.Hostname()) == nil) {
				return false
			}
			if u.User != nil {
				if _, password := u.User.Password(); password || !endpointName.MatchString(u.User.Username()) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
	endpoint, path, ok := strings.Cut(remote, ":")
	if !ok || path == "" || strings.HasPrefix(path, ":") || strings.HasPrefix(path, "-") {
		return false
	}
	if user, host, ok := strings.Cut(endpoint, "@"); ok {
		return endpointName.MatchString(user) && endpointName.MatchString(host)
	}
	return endpointName.MatchString(endpoint)
}

func (s *Synchronizer) Sync(parent context.Context, timeout time.Duration) (Status, error) {
	out := s.status()
	if timeout <= 0 || timeout > 30*time.Second {
		return out, errors.New("sync timeout must be greater than zero and at most 30 seconds")
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	err := s.store.WithExclusiveLock(func() (operationErr error) {
		defer func() {
			if operationErr != nil {
				out.State = "pending"
				out.Notice = "Synchronization stopped; a write or delivery may have occurred. Inspect this phase and history before retrying."
			}
			if out.Checkpointed {
				if err := s.saveStatus(out); operationErr == nil {
					operationErr = err
				}
			}
		}()
		if err := s.checkpoint(ctx, &out); err != nil {
			return err
		}
		out.Phase = "remote"
		remote, err := s.git(ctx, "remote", "get-url", "--all", "origin")
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if isExit(err, 2) {
				return nil
			}
			return err
		}
		out.State = "pending"
		if !validRemote(remote) {
			out.Notice = "Origin transport is unsupported or ambiguous; local checkpoint preserved."
			return nil
		}
		out.RemoteID = remoteID(remote)
		out.Phase = "fetch"
		observed, err := s.git(ctx, "ls-remote", "--exit-code", "--heads", remote, "refs/heads/main")
		if err != nil && !isExit(err, 2) {
			return s.pending(ctx, &out, "Remote unavailable; local changes remain committed.")
		}
		if observed != "" {
			if _, err := s.git(ctx, "fetch", "--no-tags", "--no-recurse-submodules", remote, "refs/heads/main:refs/remotes/origin/main"); err != nil {
				return s.pending(ctx, &out, "Fetch not confirmed; local history preserved.")
			}
			out.RemoteHead, err = s.git(ctx, "rev-parse", "--verify", "refs/remotes/origin/main^{commit}")
			if err != nil {
				return err
			}
			out.Phase = "validate"
			if err := s.reconcile(ctx, &out); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if errors.Is(err, errUpgradeRequired) {
					out.State = "upgrade-required"
					out.Notice = "Remote uses format2. Local format1 was not upgraded or replaced. Explicitly preview and apply a local format upgrade, then request synchronization again."
					return nil
				}
				out.State = "conflicted"
				out.Notice = "Remote candidate could not be safely reconciled; inspect both histories. Local work was not discarded."
				return nil
			}
		}
		out.Phase = "push"
		if err := s.cleanAt(ctx, out.Head); err != nil {
			return err
		}
		if _, err := s.git(ctx, "push", remote, out.Head+":refs/heads/main"); err != nil {
			return s.pending(ctx, &out, "Push not confirmed; inspect the remote before retrying. Local history is preserved.")
		}
		out.RemoteHead, out.Delivered = out.Head, true
		out.CheckedAt = time.Now().UTC().Format(time.RFC3339Nano)
		out.SemanticConflicts, err = s.semanticConflicts(ctx)
		if err != nil {
			return err
		}
		out.State, out.Phase = "synchronized", "complete"
		out.Notice = "This head was delivered at checked_at; the remote may advance afterward. Already-read model context is not refreshed."
		if out.SemanticConflicts > 0 {
			out.State = "conflicted"
			out.Notice = "Git delivery succeeded, but competing memory revisions still require an explicit superseding decision."
		}
		return nil
	})
	return out, failed(out, err)
}

func remoteID(remote string) string {
	sum := sha256.Sum256([]byte(remote))
	return hex.EncodeToString(sum[:])
}

func (s *Synchronizer) pending(ctx context.Context, out *Status, notice string) error {
	out.State, out.Notice = "pending", notice
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (s *Synchronizer) cleanAt(ctx context.Context, head string) error {
	if err := s.boundary(ctx); err != nil {
		return err
	}
	current, err := s.head(ctx)
	if err != nil {
		return err
	}
	if current != head {
		return ErrDirty
	}
	if err := s.validateWorktree(); err != nil {
		return err
	}
	status, err := s.git(ctx, "status", "--porcelain", "--untracked-files=all", "--", ".", ":!.mandalore", ":!.DS_Store")
	if err != nil {
		return err
	}
	if status != "" {
		return ErrDirty
	}
	return nil
}

func (s *Synchronizer) reconcile(ctx context.Context, out *Status) error {
	if _, err := s.git(ctx, "merge-base", out.Head, out.RemoteHead); err != nil {
		return err
	}
	if err := s.validateCandidate(ctx, out.RemoteHead); err != nil {
		return err
	}
	manifest, err := s.gitOutput(ctx, 4<<20, "show", out.RemoteHead+":signet.json")
	if err != nil {
		return err
	}
	var remote memory.Signet
	if err := strictjson.Decode([]byte(manifest), &remote, 4<<20); err != nil {
		return err
	}
	if s.store.Signet.Version == 1 && remote.Version == 2 {
		return errUpgradeRequired
	}
	if _, err := s.git(ctx, "merge-base", "--is-ancestor", out.RemoteHead, out.Head); err == nil {
		return nil
	} else if !isExit(err, 1) {
		return err
	}
	candidate := out.RemoteHead
	if _, err := s.git(ctx, "merge-base", "--is-ancestor", out.Head, out.RemoteHead); err != nil {
		if !isExit(err, 1) {
			return err
		}
		tree, err := s.git(ctx, "merge-tree", "--write-tree", "--no-messages", out.Head, out.RemoteHead)
		if err != nil {
			return err
		}
		candidate, err = s.git(ctx, "commit-tree", tree, "-p", out.Head, "-p", out.RemoteHead, "-m", "Reconcile signet memory")
		if err != nil {
			return err
		}
	}
	if err := s.validateCandidate(ctx, candidate, out.Head, out.RemoteHead); err != nil {
		return err
	}
	if err := s.cleanAt(ctx, out.Head); err != nil {
		return err
	}
	out.Phase = "integrate"
	if _, err := s.git(ctx, "merge", "--ff-only", "--no-edit", candidate); err != nil {
		return err
	}
	out.Head = candidate
	return s.cleanAt(ctx, candidate)
}

func (s *Synchronizer) semanticConflicts(ctx context.Context) (int, error) {
	scopes, err := s.store.Scopes()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, scope := range scopes {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if scope.Visibility != nil {
			count += scope.Visibility.Conflicted
			continue
		}
		packet, err := s.store.Recall(memory.Query{Scope: scope.Scope, ExactScope: true, Limit: 1}, time.Now())
		if err != nil {
			return 0, err
		}
		count += len(packet.Conflicts)
	}
	return count, nil
}
