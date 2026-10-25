package server

import (
	"cmp"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	// stateVersion 2 keeps grants per installation. Version 1 (2.0.0 and
	// 2.0.1, which write no version) kept them per client.
	stateVersion = 2
)

// oauthState is everything OAuth keeps across restarts. Tokens and the owner
// enrollment link are stored only as SHA-256 hashes.
type oauthState struct {
	Version       int                      `json:"version,omitempty"`
	Clients       map[string]*oauthClient  `json:"clients"`
	Installations map[string]*installation `json:"installations"`
	Tokens        map[string]*oauthToken   `json:"tokens"`
	Owner         ownerState               `json:"owner"`
}

// oauthClient is one identity: a DCR registration or a client ID metadata
// document URL. It is pending until the owner approves an installation of it.
type oauthClient struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	RedirectURIs []string  `json:"redirect_uris"`
	Document     bool      `json:"metadata_document,omitempty"`
	Registered   time.Time `json:"registered"`
	// Legacy is the per-client grant that version 1 reads: signet slug →
	// the widest scope of the client's installations. It is rewritten on
	// every save, so rolling back to 2.0.1 keeps the grants, and read only
	// to migrate a version 1 state.
	Legacy map[string]Scope `json:"grants,omitempty"`
}

// installation is one approved copy of a client, holding the owner's grants.
// Every installation of a metadata-document client shares its client_id, so
// each approval creates a new installation, and its token family is the
// installation's. A DCR registration belongs to one installation, so it has
// one, which later authorizations reuse.
type installation struct {
	ID        string           `json:"id"`
	Client    string           `json:"client"`
	Grants    map[string]Scope `json:"grants"` // signet slug → scope
	FirstSeen time.Time        `json:"first_seen"`
	// LastUsed is the last token issue or, to within lastUsedEvery, the
	// last authenticated MCP request.
	LastUsed time.Time `json:"last_used"`
	// Label is the owner's name for it, set with `server clients label`.
	Label string `json:"label,omitempty"`
	// Hint is the truncated User-Agent of the browser that approved it.
	Hint string `json:"hint,omitempty"`
}

type oauthToken struct {
	Kind         string    `json:"kind"`
	Client       string    `json:"client"`
	Installation string    `json:"installation,omitempty"`
	Signet       string    `json:"signet"`
	Family       string    `json:"family"`
	Scope        Scope     `json:"scope"`
	Expires      time.Time `json:"expires"`
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

// errStore matches every failure to lock, read or save the OAuth state, so
// callers can tell the server's failure from an error fn returns.
var errStore = errors.New("OAuth state store failure")

// storeError is a store failure with the message the client sees.
type storeError string

func (e storeError) Error() string        { return string(e) }
func (e storeError) Is(target error) bool { return target == errStore }

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
	if st.Installations == nil {
		st.Installations = map[string]*installation{}
	}
	if st.Tokens == nil {
		st.Tokens = map[string]*oauthToken{}
	}
	if st.Version < stateVersion {
		st.migrate()
	}
}

// migrate turns version 1's per-client grants into installations, so every
// live token keeps working with no new consent. A DCR client becomes one
// installation holding all its tokens. A metadata-document client becomes one
// installation per live token family, each with the client's grant, since
// version 1 can't tell which installation was approved with which scope. A
// document client with no live tokens has no installation left to keep.
// Installation IDs derive from the client and family, so the server and the
// admin CLI agree on them before the migrated state is first saved.
func (st *oauthState) migrate() {
	for _, c := range st.Clients {
		if len(c.Legacy) == 0 {
			continue
		}
		install := func(key string) *installation {
			id := "inst-" + hashSecret("migrated\x00" + key)[:16]
			if st.Installations[id] == nil {
				st.Installations[id] = &installation{ID: id, Client: c.ID, Grants: maps.Clone(c.Legacy), FirstSeen: c.Registered, LastUsed: c.Registered}
			}
			return st.Installations[id]
		}
		if !c.Document {
			install(c.ID)
		}
		for _, t := range st.Tokens {
			if t.Client != c.ID {
				continue
			}
			key := c.ID
			if c.Document {
				key += "\x00" + t.Family
			}
			in := install(key)
			if issued := t.issued(); issued.After(in.LastUsed) {
				in.LastUsed = issued
			}
			t.Installation = in.ID
		}
	}
	st.Version = stateVersion
}

// issued is when a token was issued, from its fixed lifetime.
func (t *oauthToken) issued() time.Time {
	if t.Kind == tokenRefresh {
		return t.Expires.Add(-refreshTTL)
	}
	return t.Expires.Add(-accessTTL)
}

// installationsOf returns a client's installations, oldest first.
func (st *oauthState) installationsOf(client string) []*installation {
	var out []*installation
	for _, in := range st.Installations {
		if in.Client == client {
			out = append(out, in)
		}
	}
	slices.SortFunc(out, func(x, y *installation) int {
		return cmp.Or(x.FirstSeen.Compare(y.FirstSeen), strings.Compare(x.ID, y.ID))
	})
	return out
}

// newInstallation records an installation of a client with no grants yet.
func (st *oauthState) newInstallation(client string, now time.Time) *installation {
	in := &installation{ID: "inst-" + hashSecret(secret())[:16], Client: client, Grants: map[string]Scope{}, FirstSeen: now, LastUsed: now}
	st.Installations[in.ID] = in
	return in
}

// liveTokens counts each installation's tokens that can still be used.
func (st *oauthState) liveTokens(now time.Time) map[string]int {
	live := map[string]int{}
	for _, t := range st.Tokens {
		if !t.Used && now.Before(t.Expires) {
			live[t.Installation]++
		}
	}
	return live
}

// expired reports whether an installation can never be used again: one of a
// document client with no live token, past the time its approval's code
// could still be redeemed. Its next authorization is a new installation. A
// DCR installation never expires, since its client authorizes again with
// its grant.
func (st *oauthState) expired(in *installation, live map[string]int, now time.Time) bool {
	c := st.Clients[in.Client]
	return c != nil && c.Document && live[in.ID] == 0 && !now.Before(in.LastUsed.Add(codeTTL))
}

// grantFor returns a token's current grant on its signet, or "" when the
// client, the installation or the grant is gone.
func (st *oauthState) grantFor(t *oauthToken) Scope {
	in := st.Installations[t.Installation]
	if in == nil || in.Client != t.Client || st.Clients[t.Client] == nil {
		return ""
	}
	return in.Grants[t.Signet]
}

// writeLegacy rewrites each client's version 1 grant from its installations.
func (st *oauthState) writeLegacy() {
	for _, c := range st.Clients {
		c.Legacy = nil
		for _, in := range st.installationsOf(c.ID) {
			for slug, scope := range in.Grants {
				if c.Legacy == nil {
					c.Legacy = map[string]Scope{}
				}
				if c.Legacy[slug] != ScopeReadWrite {
					c.Legacy[slug] = scope
				}
			}
		}
	}
}

// view runs fn on the current state. fn must not modify it.
func (s *oauthStore) view(fn func(*oauthState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reload(); err != nil {
		return storeError("OAuth state unreadable")
	}
	return fn(&s.state)
}

// update applies fn to the latest state under the file lock and saves it if
// fn succeeds. Expired tokens, pending clients and enrollments are pruned.
// Once the save succeeds the update has happened, so it reports no error even
// if rereading the saved file fails.
func (s *oauthStore) update(fn func(*oauthState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fd, err := syscall.Open(filepath.Join(s.dir, "oauth.lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return storeError("OAuth state unavailable")
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return storeError("OAuth state unavailable")
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if err := s.reload(); err != nil {
		return storeError("OAuth state unreadable")
	}
	next := s.state.clone()
	if err := fn(&next); err != nil {
		return err
	}
	next.prune(s.now())
	next.writeLegacy()
	if err := writeState(s.path(), next); err != nil {
		return storeError("cannot save OAuth state")
	}
	s.state, s.seen = next, nil
	_ = s.reload() // records the saved file; on failure the next read retries
	return nil
}

// clone copies the state deeply enough that a failed update leaves the
// cached state untouched.
func (st oauthState) clone() oauthState {
	out := oauthState{Version: st.Version, Clients: map[string]*oauthClient{}, Installations: map[string]*installation{}, Tokens: map[string]*oauthToken{}, Owner: st.Owner}
	out.Owner.Credentials = append([]webauthn.Credential(nil), st.Owner.Credentials...)
	for id, c := range st.Clients {
		cp := *c
		cp.RedirectURIs = append([]string(nil), c.RedirectURIs...)
		cp.Legacy = maps.Clone(c.Legacy)
		out.Clients[id] = &cp
	}
	for id, in := range st.Installations {
		cp := *in
		cp.Grants = maps.Clone(in.Grants)
		out.Installations[id] = &cp
	}
	for h, t := range st.Tokens {
		cp := *t
		out.Tokens[h] = &cp
	}
	return out
}

// prune drops expired tokens and whatever no longer has an owner. An
// installation of a document client is its token family, so it goes once it
// has no tokens and no use for pendingTTL; a DCR installation lasts as long
// as its client, as the client's approval did in version 1.
func (st *oauthState) prune(now time.Time) {
	for id, in := range st.Installations {
		if st.Clients[in.Client] == nil {
			delete(st.Installations, id)
		}
	}
	live := map[string]bool{}
	for h, t := range st.Tokens {
		if !now.Before(t.Expires) || st.Clients[t.Client] == nil || st.Installations[t.Installation] == nil {
			delete(st.Tokens, h)
			continue
		}
		live[t.Installation] = true
	}
	for id, in := range st.Installations {
		if st.Clients[in.Client].Document && !live[id] && !now.Before(in.LastUsed.Add(pendingTTL)) {
			delete(st.Installations, id)
		}
	}
	for id, c := range st.Clients {
		if !c.approved(st) && !now.Before(c.Registered.Add(pendingTTL)) {
			delete(st.Clients, id)
		}
	}
	if !now.Before(st.Owner.EnrollmentExpires) {
		st.Owner.Enrollment, st.Owner.EnrollmentExpires, st.Owner.EnrollmentReplaces = "", time.Time{}, false
	}
}

// evictPending makes room for one more pending client by removing the oldest
// pending ones, so registrations that are never approved can't fill the cap
// for good. A client with any grant is never touched. With maxPending at least
// registerLimit, a new registration survives at least registerWindow, longer
// than its consent request lasts.
func (st *oauthState) evictPending(now time.Time) {
	var pending []*oauthClient
	for _, c := range st.Clients {
		if !c.approved(st) && now.Before(c.Registered.Add(pendingTTL)) {
			pending = append(pending, c)
		}
	}
	if len(pending) < maxPending {
		return
	}
	slices.SortFunc(pending, func(x, y *oauthClient) int {
		return cmp.Or(x.Registered.Compare(y.Registered), strings.Compare(x.ID, y.ID))
	})
	for _, c := range pending[:len(pending)-maxPending+1] {
		delete(st.Clients, c.ID)
	}
}

// revokeFamily deletes every token issued from one authorization.
func (st *oauthState) revokeFamily(family string) {
	for h, t := range st.Tokens {
		if t.Family == family {
			delete(st.Tokens, h)
		}
	}
}

// approved reports whether the client has any installation.
func (c *oauthClient) approved(st *oauthState) bool {
	return len(st.installationsOf(c.ID)) > 0
}

// revokeClient deletes a client and its installations, and returns how many
// installations it had.
func (st *oauthState) revokeClient(id string) int {
	n := st.revokeInstallations(func(in *installation) bool { return in.Client == id })
	delete(st.Clients, id)
	return n
}

// revokeInstallations deletes the matching installations and returns how
// many there were. The prune before every save deletes their tokens.
func (st *oauthState) revokeInstallations(match func(*installation) bool) int {
	n := 0
	for id, in := range st.Installations {
		if match(in) {
			delete(st.Installations, id)
			n++
		}
	}
	return n
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
