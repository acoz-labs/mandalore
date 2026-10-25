package signetsync

import (
	"context"
	"errors"
	"strings"
)

// ErrRemoteAhead means the backup remote names commits this checkout does not
// contain. A single writer must not push over them; an operator reconciles.
var ErrRemoteAhead = errors.New("remote ahead: run mandalore server reconcile")

var ErrDiverged = errors.New("diverged: local and remote both have commits; manual recovery needed")

func (s *Synchronizer) remoteHead(ctx context.Context, remote string) (string, error) {
	if !validRemote(remote) {
		return "", errors.New("unsupported or ambiguous remote")
	}
	observed, err := s.git(ctx, "ls-remote", "--exit-code", "--heads", remote, "refs/heads/main")
	if isExit(err, 2) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	head, _, _ := strings.Cut(observed, "\t")
	return head, nil
}

// Remote states relative to local main.
const (
	RemoteCurrent  = "current"  // remote main is empty or in local history
	RemoteAheadOf  = "ahead"    // remote has commits; local main can fast-forward
	RemoteDiverged = "diverged" // both sides have commits the other lacks
)

// RemoteState compares remote main with local main without fetching. When
// the remote commit is not local, pushed (the last head this checkout pushed,
// if known) tells ahead from diverged: local commits after it are unpushed.
func (s *Synchronizer) RemoteState(ctx context.Context, remote, pushed string) (string, error) {
	observed, err := s.remoteHead(ctx, remote)
	if err != nil {
		return "", err
	}
	if observed == "" {
		return RemoteCurrent, nil
	}
	local, err := s.head(ctx)
	if err != nil {
		return "", err
	}
	if local == "" {
		return RemoteAheadOf, nil
	}
	if _, err := s.git(ctx, "cat-file", "-e", observed+"^{commit}"); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if pushed != "" && pushed != local {
			return RemoteDiverged, nil
		}
		return RemoteAheadOf, nil
	}
	for _, check := range [][2]string{{observed, local}, {local, observed}} {
		_, err = s.git(ctx, "merge-base", "--is-ancestor", check[0], check[1])
		if err == nil {
			if check[0] == observed {
				return RemoteCurrent, nil
			}
			return RemoteAheadOf, nil
		}
		if !isExit(err, 1) {
			return "", err
		}
	}
	return RemoteDiverged, nil
}

// RemoteAhead reports whether remote main has a commit outside local history.
// It reads only the remote ref and local objects: it never fetches.
func (s *Synchronizer) RemoteAhead(ctx context.Context, remote string) (bool, error) {
	state, err := s.RemoteState(ctx, remote, "")
	return state != RemoteCurrent, err
}

// Push delivers committed local main to remote main without fetching or
// forcing. It takes no writer lock: it reads only committed objects, so memory
// writes continue during the network call. It returns the pushed head.
func (s *Synchronizer) Push(ctx context.Context, remote string) (string, error) {
	if err := s.boundary(ctx); err != nil {
		return "", err
	}
	head, err := s.head(ctx)
	if err != nil || head == "" {
		return "", err
	}
	ahead, err := s.RemoteAhead(ctx, remote)
	if err != nil {
		return "", err
	}
	if ahead {
		return "", ErrRemoteAhead
	}
	if _, err := s.git(ctx, "push", remote, head+":refs/heads/main"); err != nil {
		return "", err
	}
	return head, nil
}

// FastForward fetches remote main, validates it as a signet with append-only
// history from local main, and fast-forwards. Uncommitted or divergent local
// work is refused, never merged or discarded.
func (s *Synchronizer) FastForward(ctx context.Context, remote string) (Status, error) {
	out := s.status()
	out.RemoteID = remoteID(remote)
	err := s.store.WithExclusiveLock(func() error {
		out.Phase = "fetch"
		observed, err := s.remoteHead(ctx, remote)
		if err != nil {
			return err
		}
		local, err := s.head(ctx)
		if err != nil {
			return err
		}
		out.Head = local
		if err := s.cleanAt(ctx, local); err != nil {
			return err
		}
		if observed == "" || observed == local {
			out.State, out.Phase, out.RemoteHead = "synchronized", "complete", observed
			out.Notice = "Already current with the remote; nothing to fast-forward."
			return nil
		}
		if _, err := s.git(ctx, "fetch", "--no-tags", "--no-recurse-submodules", remote, "refs/heads/main:refs/remotes/origin/main"); err != nil {
			return err
		}
		out.RemoteHead, err = s.git(ctx, "rev-parse", "--verify", "refs/remotes/origin/main^{commit}")
		if err != nil {
			return err
		}
		if local != "" {
			if _, err := s.git(ctx, "merge-base", "--is-ancestor", local, out.RemoteHead); err != nil {
				if isExit(err, 1) {
					return ErrDiverged
				}
				return err
			}
		}
		out.Phase = "validate"
		if err := s.validateCandidate(ctx, out.RemoteHead, local); err != nil {
			return err
		}
		out.Phase = "integrate"
		if _, err := s.git(ctx, "merge", "--ff-only", "--no-edit", out.RemoteHead); err != nil {
			return err
		}
		if err := s.cleanAt(ctx, out.RemoteHead); err != nil {
			return err
		}
		out.Head, out.State, out.Phase = out.RemoteHead, "synchronized", "complete"
		out.Notice = "Fast-forwarded to the validated remote head."
		return nil
	})
	return out, err
}

// Head returns committed local main, or empty before the first commit.
func (s *Synchronizer) Head(ctx context.Context) (string, error) { return s.head(ctx) }
