package sessionsync

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

const Budget = 3 * time.Second
const StartupPairWindow = 5 * time.Second

type Boundary struct {
	Kind      string `json:"kind"`
	SessionID string `json:"session_id,omitempty"`
	EventKey  string `json:"event_key,omitempty"`
}

type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Attempt struct {
	Attempted bool               `json:"attempted"`
	Coalesced bool               `json:"coalesced"`
	Status    *signetsync.Status `json:"status,omitempty"`
	Error     *Failure           `json:"error,omitempty"`
}

func (a Attempt) Summary() string {
	if a.Error != nil {
		return "Mandalore session synchronization could not complete (" + a.Error.Code + "). Local evidence may be stale; saved content remains local. Retry at the next foreground boundary; do not repeat a save."
	}
	if a.Status != nil && a.Status.Delivered && a.Status.SemanticConflicts == 0 {
		return "Mandalore session synchronization confirmed this checkout's delivery/refresh at " + a.Status.CheckedAt + ". The remote may advance afterward; already-read context is not retroactively updated."
	}
	if a.Status != nil && a.Status.SemanticConflicts > 0 {
		return "Mandalore session synchronization found semantic conflicts. Local evidence contains unresolved heads; do not choose a winner automatically. Delivery and semantic agreement are separate."
	}
	return "Mandalore session synchronization remains local-only or pending. Local evidence may be stale; software retries at the next foreground boundary, not while all sessions are closed."
}

type receipt struct {
	Kind            string    `json:"kind"`
	Session         string    `json:"session,omitempty"`
	StartupConsumed bool      `json:"startup_consumed,omitempty"`
	Version         int       `json:"schema_version"`
	Identity        string    `json:"identity"`
	Event           string    `json:"event,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
	Attempt         Attempt   `json:"attempt"`
}

func failed(code string) Attempt {
	return Attempt{Error: &Failure{Code: code, Message: "Bounded session synchronization is unavailable; inspect the selected connection and last synchronization receipt."}}
}
func eventKey(b Boundary) string {
	if b.SessionID == "" || b.EventKey == "" {
		return ""
	}
	return hash([]byte(b.Kind + "\n" + b.SessionID + "\n" + b.EventKey))
}
func validBoundary(b Boundary) bool {
	switch b.Kind {
	case "startup", "turn", "write", "manual", "resume", "compact":
	default:
		return false
	}
	for _, v := range []string{b.SessionID, b.EventKey} {
		if len(v) > 256 || strings.IndexFunc(v, unicode.IsControl) >= 0 {
			return false
		}
	}
	return true
}

// Attempt never saves semantic content. Concurrent processes serialize per
// physical checkout; a waiter reuses completion only after checking that no
// saved content remains outside the completed head. No timer starts more work.
func (c *Coordinator) Attempt(parent context.Context, b Boundary) Attempt {
	return c.attempt(parent, b, nil)
}

type syncCall func(context.Context, time.Duration) (signetsync.Status, error)

func (c *Coordinator) attempt(parent context.Context, b Boundary, run syncCall) Attempt {
	began := time.Now().UTC()
	ctx, cancel := context.WithTimeout(parent, Budget)
	defer cancel()
	if !validBoundary(b) {
		return failed("session.boundary-invalid")
	}
	if ctx.Err() != nil {
		return failed("session.cancelled")
	}
	if err := c.Validate(); err != nil {
		return failed("session.policy-invalid")
	}
	local := filepath.Join(c.root, ".mandalore")
	if !realPath(local) {
		return failed("session.coordination-invalid")
	}
	path := filepath.Join(local, "session-sync.lock")
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return failed("session.coordination-invalid")
	}
	defer syscall.Close(fd)
	var lockStat syscall.Stat_t
	if syscall.Fstat(fd, &lockStat) != nil || lockStat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return failed("session.coordination-invalid")
	}
	for {
		err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return failed("session.coordination-invalid")
		}
		select {
		case <-ctx.Done():
			return failed("session.cancelled")
		case <-time.After(10 * time.Millisecond):
		}
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if err := c.Validate(); err != nil {
		return failed("session.policy-invalid")
	}
	g, err := signetsync.Open(c.root, c.policy.SignetID)
	if err != nil {
		return failed("session.signet-invalid")
	}
	sessionKey := ""
	if b.SessionID != "" {
		sessionKey = hash([]byte(c.selected.PolicySHA256 + "\n" + b.SessionID))
	}
	key := eventKey(b)
	if key != "" {
		key = hash([]byte(c.selected.PolicySHA256 + "\n" + key))
	}
	lastPath := filepath.Join(local, "session-sync.json")
	raw, readErr := readFile(lastPath, 32768)
	if readErr == nil {
		var last receipt
		if strictjson.Decode(raw, &last, 32768) != nil || last.Version != 1 {
			return failed("session.coordination-invalid")
		}
		startupPair := b.Kind == "turn" && sessionKey != "" && last.Session == sessionKey && last.Kind == "startup" && !last.StartupConsumed && began.Sub(last.CompletedAt) >= 0 && began.Sub(last.CompletedAt) <= StartupPairWindow && last.Attempt.Error == nil && last.Attempt.Status != nil && last.Attempt.Status.Delivered
		sameEvent := key != "" && last.Event == key && b.Kind != "write" && b.Kind != "manual" && last.Attempt.Error == nil && last.Attempt.Status != nil && last.Attempt.Status.Delivered
		if last.Identity == c.identity && (last.CompletedAt.After(began) || sameEvent || startupPair) {
			// Never claim a leader delivered a write saved after its checkpoint. Git
			// inspection includes ignored semantic files as well as ordinary dirtiness.
			state, e := g.Inspect(ctx)
			if e == nil && !state.Dirty && last.Attempt.Status != nil && state.Head == last.Attempt.Status.Head {
				if startupPair {
					last.StartupConsumed = true
					last.Event = key
					last.Kind = "turn"
					if err := writeReceipt(lastPath, last); err != nil {
						return failed("session.receipt-failed")
					}
				}
				out := last.Attempt
				out.Attempted = false
				out.Coalesced = true
				return out
			}
		}
	} else if !os.IsNotExist(readErr) {
		return failed("session.coordination-invalid")
	}
	if run == nil {
		run = g.Sync
	}
	var status signetsync.Status
	var syncErr error
	for {
		remaining := time.Until(began.Add(Budget))
		if remaining <= 0 {
			return failed("session.cancelled")
		}
		status, syncErr = run(ctx, remaining)
		if !errors.Is(syncErr, memory.ErrWriterBusy) {
			break
		}
		select {
		case <-ctx.Done():
			return failed("session.cancelled")
		case <-time.After(10 * time.Millisecond):
		}
	}
	out := Attempt{Attempted: true, Status: &status}
	if syncErr != nil {
		code := "session.sync-failed"
		if errors.Is(syncErr, context.Canceled) || errors.Is(syncErr, context.DeadlineExceeded) {
			code = "session.cancelled"
		}
		out.Error = &Failure{Code: code, Message: "The synchronization attempt stopped; preserve the saved content and inspect the partial status before retrying delivery."}
		var partial *signetsync.Failure
		if errors.As(syncErr, &partial) {
			out.Status = &partial.Status
		}
	}
	record := receipt{Kind: b.Kind, Session: sessionKey, Version: 1, Identity: c.identity, Event: key, CompletedAt: time.Now().UTC(), Attempt: out}
	if err := writeReceipt(lastPath, record); err != nil {
		out.Error = &Failure{Code: "session.receipt-failed", Message: "Synchronization may have completed, but coordination evidence could not be retained. Inspect synchronization status; do not repeat the save."}
	}
	return out
}

func writeReceipt(path string, v receipt) error {
	if st, err := os.Lstat(path); err == nil && !st.Mode().IsRegular() {
		return errors.New("receipt redirected")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".session-sync-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(raw); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
