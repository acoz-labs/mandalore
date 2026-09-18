package signetsync

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// Failure preserves partial operational evidence. Adapters must check it before
// classifying its wrapped context error as a no-write cancellation.
type Failure struct {
	Cause  error
	Status Status
}

func (e *Failure) Error() string {
	return "Synchronization stopped; inspect its phase and local/remote history before retrying"
}
func (e *Failure) Unwrap() error { return e.Cause }

type Inspection struct {
	SignetID          string  `json:"signet_id"`
	State             string  `json:"state"`
	Head              string  `json:"head,omitempty"`
	Dirty             bool    `json:"dirty"`
	SemanticConflicts int     `json:"semantic_conflicts"`
	LastAttempt       *Status `json:"last_attempt,omitempty"`
	Notice            string  `json:"notice"`
}

type storedStatus struct {
	Version int    `json:"schema_version"`
	Status  Status `json:"status"`
}

var objectID = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
var fingerprint = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *Synchronizer) saveStatus(out Status) error {
	current, err := memory.Open(s.store.Root)
	if err != nil {
		return err
	}
	if current.Signet.ID != s.store.Signet.ID {
		return memory.ErrIdentityChanged
	}
	parent := filepath.Join(s.store.Root, ".mandalore")
	info, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrBoundary
	}
	path := filepath.Join(parent, "sync-status.json")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return ErrBoundary
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(parent, ".sync-status-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := json.NewEncoder(f).Encode(storedStatus{Version: 1, Status: out}); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *Synchronizer) loadStatus() (*Status, error) {
	path := filepath.Join(s.store.Root, ".mandalore", "sync-status.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 16384 {
		return nil, ErrBoundary
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil {
		return nil, err
	}
	var stored storedStatus
	if err := strictjson.Decode(data, &stored, 16384); err != nil {
		return nil, err
	}
	if stored.Version != 1 || stored.Status.SignetID != s.store.Signet.ID {
		return nil, ErrBoundary
	}
	if _, err := time.Parse(time.RFC3339Nano, stored.Status.CheckedAt); err != nil {
		return nil, ErrBoundary
	}
	switch stored.Status.State {
	case "local-only", "pending", "conflicted", "synchronized", "upgrade-required":
	default:
		return nil, ErrBoundary
	}
	r := stored.Status
	if (r.Head != "" && !objectID.MatchString(r.Head)) || (r.RemoteHead != "" && !objectID.MatchString(r.RemoteHead)) || (r.RemoteID != "" && !fingerprint.MatchString(r.RemoteID)) || r.SemanticConflicts < 0 {
		return nil, ErrBoundary
	}
	if r.Checkpointed && r.Head == "" {
		return nil, ErrBoundary
	}
	if r.Delivered && (!r.Checkpointed || r.Head == "" || r.RemoteHead != r.Head || r.RemoteID == "") {
		return nil, ErrBoundary
	}
	if r.State == "synchronized" && !r.Delivered {
		return nil, ErrBoundary
	}
	if r.State == "upgrade-required" && (r.Delivered || r.Phase != "validate" || r.RemoteHead == "") {
		return nil, ErrBoundary
	}
	switch r.Phase {
	case "checkpoint", "remote", "fetch", "validate", "integrate", "push", "complete":
	default:
		return nil, ErrBoundary
	}
	return &stored.Status, nil
}

// Inspect performs no network, enrollment, checkpoint or state-file writes.
func (s *Synchronizer) Inspect(parent context.Context) (Inspection, error) {
	out := Inspection{SignetID: s.store.Signet.ID, State: "local-only", Notice: "Local inspection only. Last-attempt delivery describes its head and time, not the current remote or already-read model context."}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := s.store.Validate(); err != nil {
		return out, err
	}
	if err := s.validateWorktree(); err != nil {
		return out, err
	}
	if _, err := os.Lstat(filepath.Join(s.store.Root, ".git")); os.IsNotExist(err) {
		return out, nil
	} else if err != nil {
		return out, err
	}
	if err := s.boundary(ctx); err != nil {
		return out, err
	}
	var err error
	out.Head, err = s.head(ctx)
	if err != nil {
		return out, err
	}
	out.LastAttempt, err = s.loadStatus()
	if err != nil {
		return out, err
	}
	out.Dirty, err = s.dirty(ctx)
	if err != nil {
		return out, err
	}
	remote, remoteErr := s.git(ctx, "remote", "get-url", "--all", "origin")
	if remoteErr == nil {
		out.State = "pending"
	} else if !isExit(remoteErr, 2) {
		return out, remoteErr
	}
	if remoteErr == nil && out.LastAttempt != nil && out.LastAttempt.RemoteID == remoteID(remote) && out.Head == out.LastAttempt.Head && !out.Dirty {
		if out.LastAttempt.State == "upgrade-required" && s.store.Signet.Version == 1 {
			out.State = "upgrade-required"
		} else if out.LastAttempt.State == "conflicted" {
			out.State = "conflicted"
		} else if out.LastAttempt.Delivered {
			out.State = "synchronized"
		}
	}
	if out.Dirty {
		out.State = "pending"
	}
	out.SemanticConflicts, err = s.semanticConflicts(ctx)
	if err != nil {
		return out, err
	}
	if out.SemanticConflicts > 0 {
		out.State = "conflicted"
	}
	return out, nil
}

func (s *Synchronizer) dirty(ctx context.Context) (bool, error) {
	status, err := s.git(ctx, "status", "--porcelain", "--untracked-files=all", "--", ".", ":!.mandalore", ":!.DS_Store")
	if err != nil {
		return false, err
	}
	if status != "" {
		return true, nil
	}
	// Valid newly written records can be ignored by custom Git rules. They are
	// still pending memory, even though ordinary git status would omit them.
	ignored, err := s.git(ctx, "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	if err != nil {
		return false, err
	}
	for _, path := range strings.Split(ignored, "\x00") {
		if allowedPath(path) {
			return true, nil
		}
	}
	return false, nil
}

func failed(out Status, err error) error {
	if err == nil {
		return nil
	}
	return &Failure{Cause: err, Status: out}
}

var _ error = (*Failure)(nil)
