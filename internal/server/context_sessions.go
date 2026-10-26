package server

import (
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const (
	// A session's memory of what it was given lapses after this long idle.
	contextSessionIdle = 12 * time.Hour
	// At most this many sessions are kept per signet; the least recently used
	// is dropped first, and its records may be injected again.
	contextSessionsMax = 1024
	// A session remembers at most this many revisions; past that, further
	// ones are injected without being remembered.
	contextSessionRevisions = 512
	// A longer session value is not a session ID, so it gets no de-duplication.
	contextSessionBytes = 128
)

// contextSessions remembers, per installation, client scope and session, the
// revisions memory_context has injected, so one session gets each revision
// once; a record revised since gets its new revision. It holds only revision
// IDs, in memory: nothing is persisted and a restart forgets every session.
// Without a session, every call is independent. The signet's mu guards it.
type contextSessions struct {
	now      func() time.Time
	sessions map[string]*contextSession
}

type contextSession struct {
	injected map[string]bool
	scopes   bool // listed already
	used     time.Time
}

// session returns the session's memory, or nil when the call names none.
func (cs *contextSessions) session(c Client, id string) *contextSession {
	if id == "" || len(id) > contextSessionBytes {
		return nil
	}
	now := cs.now()
	if cs.sessions == nil {
		cs.sessions = map[string]*contextSession{}
	}
	oldest := ""
	for key, s := range cs.sessions {
		if now.Sub(s.used) > contextSessionIdle {
			delete(cs.sessions, key)
		} else if oldest == "" || s.used.Before(cs.sessions[oldest].used) {
			oldest = key
		}
	}
	key := c.ID + "\x00" + c.Installation + "\x00" + string(c.Scope) + "\x00" + id
	s := cs.sessions[key]
	if s == nil {
		if len(cs.sessions) >= contextSessionsMax {
			delete(cs.sessions, oldest)
		}
		s = &contextSession{injected: map[string]bool{}}
		cs.sessions[key] = s
	}
	s.used = now
	return s
}

// fresh drops the revisions this session was already given. A nil session
// keeps them all.
func (s *contextSession) fresh(revisions []memory.Revision) []memory.Revision {
	if s == nil {
		return revisions
	}
	kept := []memory.Revision{}
	for _, r := range revisions {
		if !s.injected[r.ID] {
			kept = append(kept, r)
		}
	}
	return kept
}

// delivered records what a successful call gave the session. A nil session
// records nothing.
func (s *contextSession) delivered(r contextResult) {
	if s == nil {
		return
	}
	for _, id := range r.injected {
		if len(s.injected) < contextSessionRevisions {
			s.injected[id] = true
		}
	}
	s.scopes = s.scopes || r.listed
}
