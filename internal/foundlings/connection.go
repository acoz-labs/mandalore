package foundlings

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

type Manager struct {
	memory         *memory.Service
	canonTransport func(string) string
	clock          func() time.Time
	// A configured Manager serves only the listed canon references and reads
	// legacy references from configured roots instead of local connections.
	configured bool
	canon      []string
	legacy     map[string]string
}

func New(service *memory.Service) *Manager { return &Manager{memory: service} }

// Config is a server's foundling configuration for one signet.
type Config struct {
	Canon  []string          // canon foundling IDs to refresh and read
	Legacy map[string]string // legacy foundling ID → readable local root
	// Transport rewrites a source locator before fetching and permits file://
	// Git. Tests only; production leaves it nil.
	Transport func(string) string
}

func NewConfigured(service *memory.Service, c Config) *Manager {
	return &Manager{memory: service, canonTransport: c.Transport, configured: true, canon: c.Canon, legacy: c.Legacy}
}

// Memory is the signet service the Manager reads and promotes into.
func (m *Manager) Memory() *memory.Service { return m.memory }

func (m *Manager) now() time.Time {
	if m.clock != nil {
		return m.clock()
	}
	return time.Now()
}

// Preview checks an explicitly selected source without writing a registration
// or local connection and without allowing overlap through path aliases.
func (m *Manager) Preview(ctx context.Context, source memory.FoundlingSource, path string) (Observation, error) {
	if _, err := m.store(); err != nil {
		return Observation{}, err
	}
	root, err := canonicalDirectory(path)
	if err != nil {
		return Observation{}, err
	}
	if overlapping(root, m.memory.Root()) {
		return Observation{}, errors.New("reference source cannot overlap the selected signet")
	}
	return Observe(ctx, source, root)
}

// Connection is ignored, clone-local configuration. It is never registration
// evidence, a credential source, or permission to execute the reference.
type Connection struct {
	Version        int                    `json:"schema_version"`
	ID             string                 `json:"id"`
	SignetID       string                 `json:"signet_id"`
	FoundlingID    string                 `json:"foundling_id"`
	RegistrationID string                 `json:"registration_revision_id"`
	Source         memory.FoundlingSource `json:"source"`
	Pin            memory.SourcePin       `json:"pin"`
	Root           string                 `json:"local_root"`
}

type Inspection struct {
	State        string                  `json:"state"`
	Registration memory.FoundlingSummary `json:"registration"`
	Connection   *Connection             `json:"connection,omitempty"`
	Observation  *Observation            `json:"observation,omitempty"`
	Notice       string                  `json:"notice"`
}

type ConnectResult struct {
	Connected  bool       `json:"connected"`
	Durable    bool       `json:"durable"`
	Connection Connection `json:"connection"`
}

var connectionID = regexp.MustCompile(`^connection-[a-f0-9]{64}$`)
var registrationID = regexp.MustCompile(`^[a-z][a-z0-9-]{2,127}$`)

// ValidID reports whether id has the form of a foundling ID.
func ValidID(id string) bool { return registrationID.MatchString(id) }

var ErrConnection = errors.New("invalid or changed local reference connection; inspect and preserve existing configuration")

func connectionPath(id string) string { return ".mandalore/foundlings/" + id + ".json" }

func connectionFingerprint(c Connection) string {
	c.ID = ""
	data, _ := json.Marshal(c)
	return "connection-" + hash(data)
}

func (m *Manager) store() (*memory.Store, error) {
	s, err := memory.Open(m.memory.Root())
	if err != nil {
		return nil, err
	}
	if s.Signet.ID != m.memory.ID() {
		return nil, memory.ErrIdentityChanged
	}
	return s, nil
}

func (m *Manager) localRoot(create bool) (*os.Root, error) {
	if _, err := m.store(); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(m.memory.Root())
	if err != nil {
		return nil, err
	}
	for _, name := range []string{".mandalore", ".mandalore/foundlings"} {
		st, err := root.Lstat(name)
		if os.IsNotExist(err) && create {
			if err = root.Mkdir(name, 0700); err == nil || os.IsExist(err) {
				st, err = root.Lstat(name)
			}
		}
		if err != nil {
			root.Close()
			return nil, err
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, ErrConnection
		}
	}
	return root, nil
}

func (m *Manager) load(id string) (*Connection, error) {
	if m.configured {
		return m.configuredConnection(id)
	}
	root, err := m.localRoot(false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	name := connectionPath(id)
	st, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 16384 {
		return nil, ErrConnection
	}
	data, err := readRegular(root, name)
	if err != nil {
		return nil, ErrConnection
	}
	return decodeConnection(data, id, m.memory.ID())
}

// configuredConnection stands in for a local connection with the configured
// root and the current registration; reads still verify the source's pin.
func (m *Manager) configuredConnection(id string) (*Connection, error) {
	path, ok := m.legacy[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	r, err := m.memory.Foundling(id)
	if err != nil {
		return nil, err
	}
	if r.Mode == "canon" || r.State != "active" || len(r.HeadIDs) != 1 || r.Source == nil || r.Pin == nil {
		return nil, ErrChanged
	}
	if root, err := canonicalDirectory(path); err == nil {
		path = root
	}
	c := Connection{Version: 1, SignetID: m.memory.ID(), FoundlingID: id, RegistrationID: r.HeadIDs[0], Source: *r.Source, Pin: *r.Pin, Root: path}
	c.ID = connectionFingerprint(c)
	return &c, nil
}

func decodeConnection(data []byte, id, signetID string) (*Connection, error) {
	var c Connection
	if err := strictjson.Decode(data, &c, 16384); err != nil {
		return nil, ErrConnection
	}
	if c.Version != 1 || !connectionID.MatchString(c.ID) || c.ID != connectionFingerprint(c) || c.SignetID != signetID || c.FoundlingID != id || !registrationID.MatchString(id) || !registrationID.MatchString(c.RegistrationID) || memory.ValidateFoundlingIdentity(c.Source, c.Pin) != nil || !filepath.IsAbs(c.Root) || filepath.Clean(c.Root) != c.Root || len(c.Root) > 4096 || !utf8.ValidString(c.Root) || strings.IndexFunc(c.Root, unicode.IsControl) >= 0 {
		return nil, ErrConnection
	}
	return &c, nil
}

func overlapping(a, b string) bool {
	// Compare filesystem identity, not just path spelling: native bindings may
	// name a real root through an ancestor alias (or a case-insensitive spelling).
	a, err := canonicalDirectory(a)
	if err != nil {
		return true
	}
	b, err = canonicalDirectory(b)
	if err != nil {
		return true
	}
	contains := func(parent, child string) bool {
		ancestor, err := os.Stat(parent)
		if err != nil {
			return true
		}
		for {
			current, err := os.Stat(child)
			if err != nil || os.SameFile(ancestor, current) {
				return true
			}
			next := filepath.Dir(child)
			if next == child {
				return false
			}
			child = next
		}
	}
	return contains(a, b) || contains(b, a)
}

func matches(c *Connection, r memory.FoundlingSummary) bool {
	return r.State == "active" && len(r.HeadIDs) == 1 && c.RegistrationID == r.HeadIDs[0] && r.Source != nil && c.Source == *r.Source && r.Pin != nil && c.Pin == *r.Pin
}

func (m *Manager) Inspect(ctx context.Context, id string) (Inspection, error) {
	if err := ctx.Err(); err != nil {
		return Inspection{}, err
	}
	if _, err := m.store(); err != nil {
		return Inspection{}, err
	}
	r, err := m.memory.Foundling(id)
	if err != nil {
		return Inspection{}, err
	}
	v := Inspection{Registration: r, State: r.State, Notice: "Reference metadata is not current guidance. Inspection never repairs, synchronizes or promotes content."}
	if r.State != "active" {
		return v, nil
	}
	if r.Mode == "canon" {
		v.State = "server_refreshed"
		v.Notice = "Canon tracks the registered branch; the server refreshes it. Inspect foundling_status for its commit and freshness; registration metadata is untrusted evidence."
		return v, nil
	}
	c, err := m.load(id)
	if os.IsNotExist(err) {
		v.State = "unconnected"
		return v, nil
	}
	if err != nil {
		v.State = "invalid_connection"
		return v, nil
	}
	v.Connection = c
	if !matches(c, r) {
		v.State = "changed"
		return v, nil
	}
	root, err := canonicalDirectory(c.Root)
	if err != nil {
		v.State = "unavailable"
		return v, nil
	}
	if root != c.Root || overlapping(root, m.memory.Root()) {
		v.State = "invalid_connection"
		return v, nil
	}
	observed, err := Observe(ctx, c.Source, root)
	if ctx.Err() != nil {
		return Inspection{}, ctx.Err()
	}
	if err != nil {
		v.State = "unavailable"
		if errors.Is(err, ErrChanged) {
			v.State = "changed"
		}
		return v, nil
	}
	v.Observation = &observed
	v.State = "available"
	if observed.Pin != c.Pin {
		v.State = "changed"
	}
	// Do not return availability if a cooperative writer changed the registration
	// or local connection during source verification.
	current, err := m.memory.Foundling(id)
	if err != nil {
		return Inspection{}, err
	}
	now, err := m.load(id)
	if err != nil || *now != *c || !matches(c, current) {
		v.State = "changed"
	}
	return v, nil
}

// Connect explicitly creates or replaces a clone-local path. The caller supplies
// both the selected registration revision and the expected prior connection ID.
// A nonempty result with an error reports publication before a durability failure.
func (m *Manager) Connect(ctx context.Context, id, revision, sourceRoot, expected string) (ConnectResult, error) {
	return m.connect(ctx, id, revision, sourceRoot, expected, syncLocalDirectory)
}

func (m *Manager) connect(ctx context.Context, id, revision, sourceRoot, expected string, syncDir func(*os.Root, string) error) (ConnectResult, error) {
	var result ConnectResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	r, err := m.memory.Foundling(id)
	if err != nil {
		return result, err
	}
	if r.Mode == "canon" || r.State != "active" || len(r.HeadIDs) != 1 || r.HeadIDs[0] != revision {
		return result, ErrChanged
	}
	if m.configured {
		return result, errors.New("legacy roots come from the server configuration")
	}
	root, err := canonicalDirectory(sourceRoot)
	if err != nil {
		return result, err
	}
	if overlapping(root, m.memory.Root()) {
		return result, errors.New("reference source cannot overlap the selected signet or its local state")
	}
	view, err := Observe(ctx, *r.Source, root)
	if err != nil {
		return result, err
	}
	if view.Pin != *r.Pin {
		return result, ErrChanged
	}
	s, err := m.store()
	if err != nil {
		return result, err
	}
	err = s.WithExclusiveLock(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		current, err := m.memory.Foundling(id)
		if err != nil {
			return err
		}
		c := Connection{Version: 1, SignetID: m.memory.ID(), FoundlingID: id, RegistrationID: revision, Source: *r.Source, Pin: *r.Pin, Root: root}
		c.ID = connectionFingerprint(c)
		if !matches(&c, current) {
			return ErrChanged
		}
		previous, err := m.load(id)
		if os.IsNotExist(err) {
			if expected != "" {
				return ErrConnection
			}
		} else if err != nil {
			return ErrConnection
		} else if expected == "" || previous.ID != expected {
			return ErrConnection
		}
		confirmed, err := Observe(ctx, c.Source, c.Root)
		if err != nil {
			return err
		}
		if confirmed != view {
			return ErrChanged
		}
		local, err := m.localRoot(true)
		if err != nil {
			return err
		}
		defer local.Close()
		tmp := ".mandalore/foundlings/." + memory.NewID("stage")
		f, err := local.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		defer local.Remove(tmp)
		defer f.Close()
		if err = json.NewEncoder(f).Encode(c); err != nil {
			return err
		}
		if err = f.Sync(); err != nil {
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		latest, latestErr := m.load(id)
		if previous == nil {
			if !os.IsNotExist(latestErr) {
				return ErrConnection
			}
		} else if latestErr != nil || *latest != *previous {
			return ErrConnection
		}
		// Cooperating registration/sync/connection writers share the signet lock.
		// New connections use no-replace publication; replacements require the
		// exact expected identity. Unknown configuration is never silently repaired.
		if previous == nil {
			err = local.Link(tmp, connectionPath(id))
		} else {
			err = local.Rename(tmp, connectionPath(id))
		}
		if err != nil {
			return err
		}
		result = ConnectResult{Connected: true, Connection: c}
		for _, name := range []string{".mandalore/foundlings", ".mandalore", "."} {
			if err := syncDir(local, name); err != nil {
				return err
			}
		}
		result.Durable = true
		return nil
	})
	return result, err
}

func syncLocalDirectory(root *os.Root, name string) error {
	dir, err := root.Open(name)
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr := dir.Close()
	if err != nil {
		return err
	}
	return closeErr
}
