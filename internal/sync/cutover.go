package signetsync

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// CloneState compares a 1.x clone with its origin for the 2.0 cutover. It is
// read-only: it lists the remote ref but never fetches.
type CloneState struct {
	Head       string `json:"head"`
	RemoteHead string `json:"remote_head,omitempty"`
	// Remote is current, ahead (the clone is behind), diverged,
	// ahead-or-diverged (the remote commit is not local), unreachable or none.
	Remote      string   `json:"remote"`
	Uncommitted []string `json:"uncommitted"`
	Unpushed    int      `json:"unpushed_commits"`
}

func (s *Synchronizer) InspectClone(ctx context.Context) (CloneState, error) {
	out := CloneState{Uncommitted: []string{}}
	if err := s.boundary(ctx); err != nil {
		return out, err
	}
	var err error
	if out.Head, err = s.head(ctx); err != nil {
		return out, err
	}
	status, err := s.gitOutput(ctx, commandOutputLimit, "status", "--porcelain", "-z", "--untracked-files=all", "--", ".", ":!.mandalore", ":!.DS_Store")
	if err != nil {
		return out, err
	}
	for _, entry := range strings.Split(status, "\x00") {
		if len(entry) > 3 {
			out.Uncommitted = append(out.Uncommitted, entry[3:])
		}
	}
	ignored, err := s.git(ctx, "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	if err != nil {
		return out, err
	}
	for _, path := range strings.Split(ignored, "\x00") {
		if allowedPath(path) {
			out.Uncommitted = append(out.Uncommitted, path)
		}
	}
	remote, err := s.git(ctx, "remote", "get-url", "--all", "origin")
	if isExit(err, 2) {
		out.Remote = "none"
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if out.RemoteHead, err = s.remoteHead(ctx, remote); err != nil {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		out.Remote = "unreachable"
		return out, nil
	}
	if out.Remote, err = s.RemoteState(ctx, remote, "", false); err != nil {
		return out, err
	}
	if out.Head != "" && (out.Remote == RemoteCurrent || out.Remote == RemoteDiverged) {
		args := []string{"rev-list", "--count", out.Head}
		if out.RemoteHead != "" {
			args = append(args, "^"+out.RemoteHead)
		}
		count, err := s.git(ctx, args...)
		if err != nil {
			return out, err
		}
		if out.Unpushed, err = strconv.Atoi(count); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Contains reports whether commit is in the history of local main.
func (s *Synchronizer) Contains(ctx context.Context, commit string) (bool, error) {
	if !objectID.MatchString(commit) {
		return false, nil
	}
	if _, err := s.git(ctx, "cat-file", "-e", commit+"^{commit}"); err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, nil
	}
	_, err := s.git(ctx, "merge-base", "--is-ancestor", commit, "HEAD")
	if isExit(err, 1) {
		return false, nil
	}
	return err == nil, err
}

// RemoteHead returns remote main without fetching, or empty when it has none.
func (s *Synchronizer) RemoteHead(ctx context.Context, remote string) (string, error) {
	return s.remoteHead(ctx, remote)
}

// Clone makes a fresh checkout of remote main at root, which must not exist.
// It uses the same hardened Git environment as synchronization.
func Clone(parent context.Context, remote, root string) error {
	if !validRemote(remote) {
		return errors.New("unsupported or ambiguous remote")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "core.askPass=", "-c", "credential.interactive=false", "-c", "protocol.ext.allow=never",
		"clone", "--quiet", "--no-tags", "--single-branch", "--branch", "main", "--template=", "--", remote, root)
	cmd.Env = cleanEnvironment()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return &commandError{operation: "clone", exitCode: -1, cause: err}
	}
	return nil
}
