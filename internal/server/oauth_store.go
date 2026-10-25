package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	accessTTL      = time.Hour
	refreshTTL     = 30 * 24 * time.Hour
	pendingTTL     = 24 * time.Hour
	maxPending     = 20
	enrollmentTTL  = 10 * time.Minute
	tokenAccess    = "access"
	tokenRefresh   = "refresh"
	oauthStateFile = "oauth.json"
)

// oauthState is everything OAuth keeps across restarts. Tokens and the owner
// enrollment link are stored only as SHA-256 hashes.
type oauthState struct {
	Clients map[string]*oauthClient `json:"clients"`
	Tokens  map[string]*oauthToken  `json:"tokens"`
	Owner   ownerState              `json:"owner"`
}

// oauthClient is one identity: a DCR registration or a client ID metadata
// document URL. It is pending until the owner grants it a signet.
type oauthClient struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Kind         string           `json:"kind"`
	RedirectURIs []string         `json:"redirect_uris"`
	Document     bool             `json:"metadata_document,omitempty"`
	Registered   time.Time        `json:"registered"`
	Grants       map[string]Scope `json:"grants,omitempty"` // signet slug → scope
}

type oauthToken struct {
	Kind    string    `json:"kind"`
	Client  string    `json:"client"`
	Signet  string    `json:"signet"`
	Family  string    `json:"family"`
	Scope   Scope     `json:"scope"`
	Expires time.Time `json:"expires"`
	// Used marks a rotated refresh token, kept until it expires so a replay
	// can be recognised and its family revoked.
	Used bool `json:"used,omitempty"`
}

type ownerState struct {
	UserID            []byte                `json:"user_id,omitempty"`
	Credentials       []webauthn.Credential `json:"credentials,omitempty"`
	Enrollment        string                `json:"enrollment,omitempty"`
	EnrollmentExpires time.Time             `json:"enrollment_expires,omitzero"`
	// EnrollmentReplaces makes the enrolled passkey the only one.
	EnrollmentReplaces bool `json:"enrollment_replaces,omitempty"`
}

// oauthStore is the state file in state_dir. The server and the admin CLI
// share it: writers hold a file lock, and readers reload whenever the file
// changes, so a revoke from the CLI takes effect on the next request.
type oauthStore struct {
	dir string
	now func() time.Time

	mu    sync.Mutex
	state oauthState
	seen  os.FileInfo
}

func openOAuthStore(stateDir string, now func() time.Time) (*oauthStore, error) {
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, errors.New("cannot create state_dir")
	}
	s := &oauthStore{dir: stateDir, now: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reload(); err != nil {
		return nil, errors.New("OAuth state unreadable")
	}
	return s, nil
}

func (s *oauthStore) path() string { return filepath.Join(s.dir, oauthStateFile) }

// reload reads the file if it changed since the last read. The caller holds mu.
func (s *oauthStore) reload() error {
	info, err := os.Stat(s.path())
	if errors.Is(err, os.ErrNotExist) {
		s.state, s.seen = oauthState{}, nil
		s.state.init()
		return nil
	}
	if err != nil {
		return err
	}
	if s.seen != nil && os.SameFile(info, s.seen) && info.ModTime().Equal(s.seen.ModTime()) && info.Size() == s.seen.Size() {
		return nil
	}
	var next oauthState
	if err := readState(s.path(), &next); err != nil {
		return err
	}
	next.init()
	s.state, s.seen = next, info
	return nil
}

func (st *oauthState) init() {
	if st.Clients == nil {
		st.Clients = map[string]*oauthClient{}
	}
	if st.Tokens == nil {
		st.Tokens = map[string]*oauthToken{}
	}
}

// view runs fn on the current state. fn must not modify it.
func (s *oauthStore) view(fn func(*oauthState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reload(); err != nil {
		return errors.New("OAuth state unreadable")
	}
	return fn(&s.state)
}

// update applies fn to the latest state under the file lock and saves it if
// fn succeeds. Expired tokens, pending clients and enrollments are pruned.
func (s *oauthStore) update(fn func(*oauthState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fd, err := syscall.Open(filepath.Join(s.dir, "oauth.lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return errors.New("OAuth state unavailable")
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return errors.New("OAuth state unavailable")
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if err := s.reload(); err != nil {
		return errors.New("OAuth state unreadable")
	}
	next := s.state.clone()
	if err := fn(&next); err != nil {
		return err
	}
	next.prune(s.now())
	if err := writeState(s.path(), next); err != nil {
		return errors.New("cannot save OAuth state")
	}
	s.seen = nil
	return s.reload()
}

// clone copies the state deeply enough that a failed update leaves the
// cached state untouched.
func (st oauthState) clone() oauthState {
	out := oauthState{Clients: map[string]*oauthClient{}, Tokens: map[string]*oauthToken{}, Owner: st.Owner}
	out.Owner.Credentials = append([]webauthn.Credential(nil), st.Owner.Credentials...)
	for id, c := range st.Clients {
		cp := *c
		cp.RedirectURIs = append([]string(nil), c.RedirectURIs...)
		cp.Grants = map[string]Scope{}
		for k, v := range c.Grants {
			cp.Grants[k] = v
		}
		out.Clients[id] = &cp
	}
	for h, t := range st.Tokens {
		cp := *t
		out.Tokens[h] = &cp
	}
	return out
}

func (st *oauthState) prune(now time.Time) {
	for h, t := range st.Tokens {
		if !now.Before(t.Expires) || st.Clients[t.Client] == nil {
			delete(st.Tokens, h)
		}
	}
	for id, c := range st.Clients {
		if len(c.Grants) == 0 && !now.Before(c.Registered.Add(pendingTTL)) {
			delete(st.Clients, id)
		}
	}
	if !now.Before(st.Owner.EnrollmentExpires) {
		st.Owner.Enrollment, st.Owner.EnrollmentExpires, st.Owner.EnrollmentReplaces = "", time.Time{}, false
	}
}

func (st *oauthState) pending(now time.Time) int {
	n := 0
	for _, c := range st.Clients {
		if len(c.Grants) == 0 && now.Before(c.Registered.Add(pendingTTL)) {
			n++
		}
	}
	return n
}

// revokeFamily deletes every token issued from one authorization.
func (st *oauthState) revokeFamily(family string) {
	for h, t := range st.Tokens {
		if t.Family == family {
			delete(st.Tokens, h)
		}
	}
}

func (st *oauthState) revokeClient(id string) {
	for h, t := range st.Tokens {
		if t.Client == id {
			delete(st.Tokens, h)
		}
	}
}

// secret returns a random URL-safe token of 32 bytes.
func secret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashSecret(v string) string {
	h := sha256.Sum256([]byte(v))
	return hex.EncodeToString(h[:])
}
