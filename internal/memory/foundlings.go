package memory

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// FoundlingSource is a portable identity, never a machine-local checkout path.
type FoundlingSource struct {
	Kind    string `json:"kind"`
	Locator string `json:"locator"`
}

type SourcePin struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type FoundlingRegistration struct {
	Version      int             `json:"schema_version"`
	ID           string          `json:"id"`
	FoundlingID  string          `json:"foundling_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Source       FoundlingSource `json:"source"`
	Pin          SourcePin       `json:"pin"`
	State        string          `json:"state"`
	RecordedAt   string          `json:"recorded_at"`
	Authorship   Authorship      `json:"authorship"`
	Supersedes   []string        `json:"supersedes"`
	ChangeReason string          `json:"change_reason"`
}

// ExternalOrigin describes historical attribution separately from incorporation.
// A syntactically valid citation is NOT proof that a source was fetched/verified.
type ExternalOrigin struct {
	FoundlingID            string          `json:"foundling_id"`
	RegistrationRevisionID string          `json:"registration_revision_id"`
	SourceIdentity         FoundlingSource `json:"source_identity"`
	SourcePin              SourcePin       `json:"source_pin"`
	RelativeLocator        string          `json:"relative_locator"`
	ContentSHA256          string          `json:"content_sha256"`
	OriginalRecordedAt     string          `json:"original_recorded_at,omitempty"`
	OriginalAuthor         string          `json:"original_author,omitempty"`
}

var hexDigest = regexp.MustCompile(`^[0-9a-f]+$`)
var sshIdentity = regexp.MustCompile(`^(?:[a-zA-Z0-9._-]+@)?[a-zA-Z0-9][a-zA-Z0-9.-]*:[^:]+$`)

// ValidateFoundlingIdentity checks portable metadata only, never local source
// availability. Adapters use the same rules before connecting reference files.
func ValidateFoundlingIdentity(source FoundlingSource, pin SourcePin) error {
	return validateFoundlingSource(source, pin)
}

func relativeLocator(value string) bool {
	if !textWithin(value, 2048) || strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\\\r\n\t") || path.Clean(value) != value || value == "." {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." || part == "." || part == "" {
			return false
		}
	}
	return true
}

func validateFoundlingSource(source FoundlingSource, pin SourcePin) error {
	if !textWithin(source.Locator, 2048) || strings.ContainsAny(source.Locator, "\r\n\t\\$?#") {
		return errors.New("invalid portable source locator")
	}
	switch source.Kind {
	case "local":
		if !identifier.MatchString(source.Locator) || pin.Algorithm != "sha256" {
			return errors.New("local source requires an opaque ID and sha256 pin")
		}
	case "git":
		if pin.Algorithm != "git-sha1" && pin.Algorithm != "git-sha256" {
			return errors.New("Git source requires an immutable Git object pin")
		}
		if strings.Contains(source.Locator, "://") {
			u, err := url.Parse(source.Locator)
			if err != nil || (u.Scheme != "https" && u.Scheme != "ssh") || u.Hostname() == "" || u.Opaque != "" || !relativeLocator(strings.TrimPrefix(u.Path, "/")) || u.RawQuery != "" || u.Fragment != "" {
				return errors.New("invalid HTTPS/SSH Git locator")
			}
			if u.User != nil {
				_, password := u.User.Password()
				if u.Scheme == "https" || password || u.User.Username() == "" {
					return errors.New("credentials are not portable source metadata")
				}
			}
		} else {
			parts := strings.SplitN(source.Locator, ":", 2)
			if !sshIdentity.MatchString(source.Locator) || len(parts) != 2 || !relativeLocator(parts[1]) {
				return errors.New("invalid SCP-style Git locator")
			}
		}
	default:
		return errors.New("source kind must be git or local")
	}
	length := 64
	if pin.Algorithm == "git-sha1" {
		length = 40
	}
	if len(pin.Value) != length || !hexDigest.MatchString(pin.Value) {
		return errors.New("invalid immutable source pin")
	}
	return nil
}

func (s *Store) validateRegistrations(items []FoundlingRegistration) error {
	byID := map[string]FoundlingRegistration{}
	roots := map[string]int{}
	for _, r := range items {
		if r.Version != 1 || !identifier.MatchString(r.ID) || !identifier.MatchString(r.FoundlingID) || !textWithin(r.Name, 256) || !textWithin(r.Description, 4096) || !textWithin(r.ChangeReason, 1024) || len(r.Supersedes) > 32 || (r.State != "active" && r.State != "disconnected") {
			return errors.New("invalid foundling registration")
		}
		if _, err := time.Parse(time.RFC3339Nano, r.RecordedAt); err != nil {
			return errors.New("invalid registration time")
		}
		if err := s.ValidateAuthorship(r.Authorship); err != nil {
			return err
		}
		if err := validateFoundlingSource(r.Source, r.Pin); err != nil {
			return err
		}
		if _, exists := byID[r.ID]; exists {
			return errors.New("duplicate registration revision")
		}
		byID[r.ID] = r
		if len(r.Supersedes) == 0 {
			roots[r.FoundlingID]++
		}
	}
	for _, r := range items {
		if roots[r.FoundlingID] != 1 {
			return errors.New("foundling must have exactly one registration root")
		}
		seen := map[string]bool{}
		for _, id := range r.Supersedes {
			parent, exists := byID[id]
			if !exists || seen[id] || parent.FoundlingID != r.FoundlingID {
				return errors.New("invalid registration predecessor")
			}
			seen[id] = true
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return errors.New("registration supersession cycle")
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, parent := range byID[id].Supersedes {
			if err := visit(parent); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

// FoundlingRegistrations returns all immutable revisions, not an arbitrarily
// selected current head. Multiple unsuperseded revisions are a visible conflict.
func (s *Store) FoundlingRegistrations() ([]FoundlingRegistration, error) {
	if err := s.checkDirectories(); err != nil {
		return nil, err
	}
	items := []FoundlingRegistration{}
	err := filepath.WalkDir(filepath.Join(s.Root, "foundlings/registrations"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".gitkeep" {
			return nil
		}
		var r FoundlingRegistration
		if err := readJSON(p, &r); err != nil {
			return err
		}
		if !identifier.MatchString(r.FoundlingID) || !identifier.MatchString(r.ID) || p != filepath.Join(s.Root, "foundlings/registrations", r.FoundlingID, r.ID+".json") {
			return errors.New("registration ID/path mismatch")
		}
		items = append(items, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.validateRegistrations(items); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Store) PutFoundlingRegistration(r FoundlingRegistration) error {
	if r.Supersedes == nil {
		r.Supersedes = []string{}
	}
	return s.withLock(func() error {
		items, err := s.FoundlingRegistrations()
		if err != nil {
			return err
		}
		if err := s.validateRegistrations(append(items, r)); err != nil {
			return err
		}
		dir := filepath.Join(s.Root, "foundlings/registrations", r.FoundlingID)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		return writeNewJSON(filepath.Join(dir, r.ID+".json"), r)
	})
}

func (s *Store) validateOrigin(origin *ExternalOrigin) error {
	if origin == nil {
		return nil
	}
	if !identifier.MatchString(origin.FoundlingID) || !identifier.MatchString(origin.RegistrationRevisionID) || !relativeLocator(origin.RelativeLocator) || strings.Contains(origin.RelativeLocator, ":") || len(origin.ContentSHA256) != 64 || !hexDigest.MatchString(origin.ContentSHA256) {
		return errors.New("invalid external-source citation")
	}
	if origin.OriginalAuthor != "" && !textWithin(origin.OriginalAuthor, 256) {
		return errors.New("invalid original author")
	}
	if origin.OriginalRecordedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, origin.OriginalRecordedAt); err != nil {
			return errors.New("invalid original source time")
		}
	}
	items, err := s.FoundlingRegistrations()
	if err != nil {
		return err
	}
	for _, r := range items {
		if r.ID == origin.RegistrationRevisionID && r.FoundlingID == origin.FoundlingID && r.Source == origin.SourceIdentity && r.Pin == origin.SourcePin {
			return nil
		}
	}
	return errors.New("citation does not match an existing immutable registration")
}
