package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

// Backup is a signet's Git delivery state, persisted for `server status`.
type Backup struct {
	State     string `json:"state"`  // pushed, pending, failed or unknown before the first push
	Remote    string `json:"remote"` // current, ahead, unreachable or unknown
	Pending   bool   `json:"pending_push"`
	Head      string `json:"pushed_head,omitempty"`
	LastPush  string `json:"last_push,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

// signet owns one checkout. Its mutex serializes memory work and checkpoints;
// pushes run outside it so a slow remote never blocks reads or saves.
type signet struct {
	config SignetConfig
	owner  string
	state  string
	auth   Authenticator
	// protected serves MCP behind OAuth bearer tokens; nil for local auth.
	protected http.Handler
	log       *log.Logger
	timing    timing
	memory    *memory.Service
	git       *signetsync.Synchronizer
	blocked   bool
	// foundlings serves the configured canon and legacy references; canon
	// refreshes them through its own read-only view, outside mu.
	foundlings foundlings.Config
	canon      *foundlings.Manager

	mu       sync.Mutex
	devices  map[string]string // Client.device() → enrolled device ID
	handlers map[string]http.Handler

	backupMu sync.Mutex
	backup   Backup
	writes   uint64
	kick     chan struct{}
}

type timing struct {
	debounce, retry, push, canon time.Duration
}

// canonTransport rewrites canon source locators to local test remotes. Only
// tests set it; production fetches the registered locator.
var canonTransport func(string) string

func openSignet(c Config, sc SignetConfig, logger *log.Logger, t timing) (*signet, error) {
	m, err := memory.OpenService(sc.Root, memory.Authorship{DeviceID: sc.DeviceID, Actor: c.Owner.Name, Harness: "mandalore-server"})
	if err != nil {
		return nil, fmt.Errorf("signet %s: cannot open root as a valid signet with the server device", sc.Slug)
	}
	if m.ID() != sc.SignetID {
		return nil, fmt.Errorf("signet %s: identity does not match signet_id", sc.Slug)
	}
	// The server is the signet's only writer, so reads may reuse the
	// validated graph between its own writes.
	m.CacheReads()
	g, err := signetsync.Open(sc.Root, sc.SignetID)
	if err != nil {
		return nil, fmt.Errorf("signet %s: cannot open checkout", sc.Slug)
	}
	s := &signet{config: sc, owner: c.Owner.Name, state: filepath.Join(c.StateDir, "signets", sc.Slug), log: logger, timing: t, memory: m, git: g,
		devices: map[string]string{}, handlers: map[string]http.Handler{}, backup: Backup{State: "unknown", Remote: "unknown"}, kick: make(chan struct{}, 1)}
	switch c.auth() {
	case AuthLoopbackTrust:
		s.auth = loopbackTrust{scope: c.LocalScope}
	case AuthInsecureTest:
		s.auth = newTestAuthenticator(c.TestClients, sc.Slug)
	}
	if err := os.MkdirAll(s.state, 0700); err != nil {
		return nil, fmt.Errorf("signet %s: cannot create its state directory", sc.Slug)
	}
	if err := readState(filepath.Join(s.state, "devices.json"), &s.devices); err != nil {
		return nil, fmt.Errorf("signet %s: client device registry unreadable", sc.Slug)
	}
	if err := readState(filepath.Join(s.state, "backup.json"), &s.backup); err != nil {
		return nil, fmt.Errorf("signet %s: backup state unreadable", sc.Slug)
	}
	s.foundlings = foundlingConfig(sc)
	if len(sc.Canon) > 0 {
		view, err := memory.OpenReadOnlyService(sc.Root)
		if err != nil {
			return nil, fmt.Errorf("signet %s: cannot open a canon refresh view", sc.Slug)
		}
		s.canon = foundlings.NewConfigured(view, s.foundlings)
	}
	for id, root := range sc.Legacy {
		if _, err := os.ReadDir(root); err != nil {
			logger.Printf("signet %s: legacy %s root unreadable; its reads report unavailable", sc.Slug, id)
		}
	}
	return s, nil
}

func foundlingConfig(sc SignetConfig) foundlings.Config {
	return foundlings.Config{Canon: sc.Canon, Legacy: sc.Legacy, Transport: canonTransport}
}

// refreshCanon refreshes the configured canon references now and then on
// every interval. A failed refresh keeps the last verified snapshot, which
// reads report stale; it never blocks serving.
func (s *signet) refreshCanon(ctx context.Context) {
	if s.canon == nil {
		return
	}
	tick := time.NewTicker(s.timing.canon)
	defer tick.Stop()
	for {
		receipts, err := s.canon.Refresh(ctx)
		if err != nil && ctx.Err() == nil {
			s.log.Printf("signet %s: canon refresh failed", s.config.Slug)
		}
		for _, r := range receipts {
			if r.State != "available" {
				s.log.Printf("signet %s: canon %s %s (%s)", s.config.Slug, r.FoundlingID, r.State, r.Reason)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

func readState(path string, v any) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// writeState replaces a server state file atomically. Its temp file has a
// unique name, so one an interrupted write left behind cannot fail it. State
// never lives in the signet repository.
func writeState(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(append(raw, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// author resolves the client's device, enrolling one on its first write. The
// caller holds s.mu.
func (s *signet) author(c Client) (memory.Authorship, error) {
	device := s.devices[c.device()]
	if device == "" {
		var err error
		name := c.Name
		if name == "" {
			name = c.ID
		}
		if c.Installation != "" {
			name += " installation " + c.Installation
		}
		device, err = s.memory.EnrollClientDevice(fmt.Sprintf("%s (%s), approved %s", name, c.Kind, time.Now().UTC().Format("2006-01-02")))
		if err != nil {
			return memory.Authorship{}, err
		}
		s.devices[c.device()] = device
		if err := writeState(filepath.Join(s.state, "devices.json"), s.devices); err != nil {
			return memory.Authorship{}, err
		}
	}
	return memory.Authorship{DeviceID: device, Actor: s.owner, Harness: c.Kind}, nil
}

// enroll records an approved client's device at approval, so provenance
// names it before its first write. While writes are paused it waits for the
// first write instead.
func (s *signet) enroll(c Client) error {
	if s.blocked || s.writesPaused() {
		return nil
	}
	s.mu.Lock()
	_, known := s.devices[c.device()]
	_, err := s.author(c)
	s.mu.Unlock()
	if err == nil && !known {
		s.wrote()
	}
	return err
}

// startup refuses to serve a signet whose remote has commits the checkout
// lacks: a surprise remote commit means another writer is still alive.
func (s *signet) startup(ctx context.Context) {
	s.backupMu.Lock()
	pushed := s.backup.Head
	s.backupMu.Unlock()
	check, cancel := context.WithTimeout(ctx, s.timing.push)
	defer cancel()
	state, err := s.git.RemoteState(check, s.config.Remote, pushed, true)
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	// A previous run's failure is rechecked here; the first push settles it.
	s.backup.State = "unknown"
	if s.backup.Pending {
		s.backup.State = "pending"
	}
	s.backup.Remote = "" // recheck, so a blocked signet logs why on every start
	switch {
	case err != nil:
		s.backup.Remote = "unreachable"
		s.log.Printf("signet %s: remote unreachable at startup; serving and retrying backup", s.config.Slug)
	case state == signetsync.RemoteCurrent:
		s.backup.Remote = state
	default:
		s.blocked = true
		s.remoteMoved(state)
	}
	s.saveBackup()
}

// remoteMoved records that the remote has commits this checkout lacks and
// pauses writes, so a fast-forward reconcile stays possible. It logs once per
// change. The caller holds backupMu.
func (s *signet) remoteMoved(state string) {
	if s.backup.Remote != state {
		switch state {
		case signetsync.RemoteDiverged:
			s.log.Printf("signet %s: diverged: local and remote both have commits; manual recovery needed (see docs/server.md)", s.config.Slug)
		case signetsync.RemoteMoved:
			s.log.Printf("signet %s: remote ahead or diverged (no pushed head recorded): run mandalore server reconcile, which refuses diverged history (see docs/server.md)", s.config.Slug)
		default:
			s.log.Printf("signet %s: remote ahead: run mandalore server reconcile", s.config.Slug)
		}
	}
	s.backup.State, s.backup.Remote = "failed", state
}

// writesPaused reports whether the remote is ahead or diverged, or may be. Reads continue;
// writes would only widen what reconcile has to repair.
func (s *signet) writesPaused() bool {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	switch s.backup.Remote {
	case signetsync.RemoteAheadOf, signetsync.RemoteDiverged, signetsync.RemoteMoved:
		return true
	}
	return false
}

// wrote records a published write and schedules a debounced push. It returns
// the backup state for the write's receipt.
func (s *signet) wrote() string {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	s.writes++
	s.backup.Pending = true
	if s.backup.State != "failed" {
		s.backup.State = "pending"
	}
	s.saveBackup()
	select {
	case s.kick <- struct{}{}:
	default:
	}
	return s.backup.State
}

// saveBackup persists the state; the caller holds backupMu. Losing it only
// makes `server status` stale, so a failure is logged, not fatal.
func (s *signet) saveBackup() {
	if err := writeState(filepath.Join(s.state, "backup.json"), s.backup); err != nil {
		s.log.Printf("signet %s: cannot record backup state", s.config.Slug)
	}
}

// deliver pushes after writes settle for the debounce interval, and retries
// on a timer while a push is pending or the remote is not known current. A
// failed push is never fatal: the memory is already durable in the checkout.
func (s *signet) deliver(ctx context.Context) {
	retry := time.NewTicker(s.timing.retry)
	defer retry.Stop()
	s.push(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.kick:
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.timing.debounce):
			}
		case <-retry.C:
			// Retry pending pushes, and keep checking a remote not known to be
			// current, so one that returns ahead pauses writes before the next.
			s.backupMu.Lock()
			idle := !s.backup.Pending && s.backup.Remote == signetsync.RemoteCurrent
			s.backupMu.Unlock()
			if idle {
				continue
			}
		}
		s.push(ctx)
	}
}

func (s *signet) push(ctx context.Context) {
	s.backupMu.Lock()
	seen := s.writes
	s.backupMu.Unlock()
	s.mu.Lock()
	_, err := s.git.Checkpoint(ctx)
	s.mu.Unlock()
	var head, state string
	if err == nil {
		s.backupMu.Lock()
		pushed := s.backup.Head
		s.backupMu.Unlock()
		pushCtx, cancel := context.WithTimeout(ctx, s.timing.push)
		head, err = s.git.Push(pushCtx, s.config.Remote)
		if errors.Is(err, signetsync.ErrRemoteAhead) {
			if state, err = s.git.RemoteState(pushCtx, s.config.Remote, pushed, true); err == nil {
				err = signetsync.ErrRemoteAhead
			}
		}
		cancel()
	}
	if ctx.Err() != nil {
		return
	}
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	defer s.saveBackup()
	switch {
	case errors.Is(err, signetsync.ErrRemoteAhead) && state != signetsync.RemoteCurrent:
		s.remoteMoved(state)
		s.backup.LastError = state
	case err != nil:
		s.backup.Remote, s.backup.LastError = "unreachable", "push not confirmed; retrying"
	default:
		s.backup.Remote, s.backup.LastError = signetsync.RemoteCurrent, ""
		if head != "" {
			s.backup.Head, s.backup.LastPush = head, time.Now().UTC().Format(time.RFC3339)
		}
		if s.writes == seen {
			s.backup.State, s.backup.Pending = "pushed", false
		}
	}
	if s.backup.Pending && s.backup.State != "failed" {
		s.backup.State = "pending"
	}
}
