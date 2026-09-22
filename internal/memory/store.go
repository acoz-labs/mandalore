// Package memory implements versioned, harness-neutral signet storage.
package memory

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const FormatVersion = 1

var ErrWriterBusy = errors.New("signet writer busy; retry the operation")
var ErrIdentityChanged = errors.New("store identity changed; reopen the intended signet")

//go:embed schemas/*.json
var schemas embed.FS
var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{2,127}$`)

type Signet struct {
	Version int    `json:"schema_version"`
	ID      string `json:"id"`
	Name    string `json:"name"`
}
type Device struct {
	Version int    `json:"schema_version"`
	ID      string `json:"id"`
	Label   string `json:"label"`
}
type Source struct {
	ExternalOrigin *ExternalOrigin `json:"external_origin,omitempty"`
	Version        int             `json:"schema_version"`
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	Summary        string          `json:"summary"`
	DeviceID       string          `json:"device_id"`
	RecordedAt     string          `json:"recorded_at"`
}

type Store struct {
	Root   string
	Signet Signet
}

var directories = []string{".mandalore", "memory", "memory/records", "memory/events", "memory/sources", "provenance", "provenance/devices", "foundlings", "foundlings/registrations"}

func NewID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(b)
}

func Create(root, name, deviceID, label string) (*Store, error) {
	if !textWithin(name, 256) || strings.ContainsAny(name, "\r\n") {
		return nil, errors.New("signet name must be one nonempty line, at most 256 bytes")
	}
	if !identifier.MatchString(deviceID) || !textWithin(label, 256) {
		return nil, errors.New("valid device ID and label required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(abs); !os.IsNotExist(err) {
		return nil, errors.New("target already exists or cannot be inspected")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return nil, err
	}
	abs = filepath.Join(parent, filepath.Base(abs))
	staging, err := os.MkdirTemp(parent, ".mandalore-create-")
	if err != nil {
		return nil, err
	}
	// This operation owns only this newly created temporary tree.
	defer os.RemoveAll(staging)
	s := &Store{Root: staging, Signet: Signet{Version: FormatVersion, ID: NewID("signet"), Name: name}}
	for _, dir := range directories {
		if err := os.MkdirAll(filepath.Join(staging, dir), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(staging, dir, ".gitkeep"), nil, 0600); err != nil {
			return nil, err
		}
	}
	if err := writeNewJSON(filepath.Join(staging, "signet.json"), s.Signet); err != nil {
		return nil, err
	}
	if err := s.AddDevice(Device{Version: 1, ID: deviceID, Label: label}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(staging, ".gitignore"), []byte(".mandalore/\n.DS_Store\n"), 0600); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if err := renameNewDirectory(staging, abs); err != nil {
		return nil, err
	}
	s.Root = abs
	return s, nil
}

func Open(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("signet root must be a real directory")
	}
	for _, marker := range []string{"agent.json", "bank.json"} {
		if _, err := os.Lstat(filepath.Join(abs, marker)); !os.IsNotExist(err) {
			return nil, errors.New("legacy or mixed format requires explicit migration")
		}
	}
	s := &Store{Root: abs}
	if err := readJSON(filepath.Join(abs, "signet.json"), &s.Signet); err != nil {
		return nil, err
	}
	if err := validateSignetMetadata(s.Signet); err != nil {
		return nil, err
	}
	return s, nil
}

func readJSON(path string, out any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}
	if info.Size() > 4<<20 {
		return fmt.Errorf("record exceeds 4 MiB: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	d.UseNumber() // Opaque extension numbers must survive history/recall unchanged.
	if err = d.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON in %s", path)
	}
	return nil
}

func encodeJSON(value any) ([]byte, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	if len(b) > 4<<20 {
		return nil, errors.New("record exceeds 4 MiB")
	}
	return b, nil
}

func writeNewJSON(path string, value any) error {
	b, err := encodeJSON(value)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".write-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Link(tmp, path); err != nil {
		return fmt.Errorf("immutable file already exists or cannot be created: %w", err)
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (s *Store) withLock(fn func() error) error {
	return s.WithExclusiveLock(fn)
}

// WithExclusiveLock coordinates external storage adapters with memory writers.
// The callback must not invoke another locking memory mutation. It does not
// authorize changing the signet identity or bypassing data validation.
func (s *Store) WithExclusiveLock(fn func() error) error {
	return s.withExclusiveLock(false, fn)
}

// WithFormatUpgradeLock permits inspection/recovery of a validated pending
// transition under the ordinary writer lock. The callback must verify its exact
// source/HEAD/binding pins; this is not permission to skip malformed evidence,
// reinterpret historical digests, downgrade or activate an unreviewed plan.
func (s *Store) WithFormatUpgradeLock(fn func() error) error {
	return s.withExclusiveLock(true, fn)
}

func (s *Store) withExclusiveLock(allowPending bool, fn func() error) error {
	if err := s.checkDirectories(); err != nil {
		return err
	}
	local := filepath.Join(s.Root, ".mandalore")
	if err := os.Mkdir(local, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	if info, err := os.Lstat(local); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("invalid machine-local state directory")
	}
	path := filepath.Join(s.Root, ".mandalore", "write.lock")
	// Git subprocesses must not inherit this lock if the runtime is killed.
	// Set close-on-exec atomically, avoiding a concurrent process-start race.
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return ErrWriterBusy
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if err := s.checkDirectories(); err != nil {
		return err
	}
	if err := s.validateUpgradeState(); err != nil && !(allowPending && errors.Is(err, ErrUpgradePending)) {
		return err
	}
	return fn()
}
func (s *Store) checkDirectories() error {
	if err := s.checkDirectoryLayout(); err != nil {
		return err
	}
	if s.Signet.Version == 2 {
		return s.validateUpgradeState()
	}
	return nil
}

// Layout-only validation lets bounded snapshot readers own all body reads and
// their aggregate budget; it must not pre-read every upgrade receipt unbounded.
func (s *Store) checkDirectoryLayout() error {
	current, err := Open(s.Root)
	if err != nil {
		return err
	}
	if current.Signet.ID != s.Signet.ID {
		return ErrIdentityChanged
	}
	if current.Signet.Version != s.Signet.Version {
		return errors.New("signet format changed; reopen the connection before continuing")
	}
	for _, dir := range directories {
		info, err := os.Lstat(filepath.Join(s.Root, dir))
		// Local state is ignored by Git and is not required for a read.
		if dir == ".mandalore" && os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("invalid signet directory: %s", dir)
		}
	}
	for _, dir := range []string{"memory", "provenance", "foundlings"} {
		if err := filepath.WalkDir(filepath.Join(s.Root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 || (!d.IsDir() && !d.Type().IsRegular()) {
				return errors.New("managed data tree contains a symlink or nonregular entry")
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) AddDevice(d Device) error {
	if d.Version != 1 || !identifier.MatchString(d.ID) || strings.TrimSpace(d.Label) == "" {
		return errors.New("invalid device record")
	}
	return s.withLock(func() error { return writeNewJSON(filepath.Join(s.Root, "provenance/devices", d.ID+".json"), d) })
}
func (s *Store) AddSource(source Source) error {
	if err := s.validateSource(source); err != nil {
		return err
	}
	return s.withLock(func() error { return writeNewJSON(filepath.Join(s.Root, "memory/sources", source.ID+".json"), source) })
}

func (s *Store) validateSource(source Source) error {
	return s.validateSourceWithDevice(source, s.deviceExists)
}

func (s *Store) validateSourceWithDevice(source Source, device func(string) error) error {
	if err := validateSourceMetadata(source, device); err != nil {
		return err
	}
	return s.validateOrigin(source.ExternalOrigin)
}

func validateSourceMetadata(source Source, device func(string) error) error {
	if source.Version != 1 || !identifier.MatchString(source.ID) || strings.TrimSpace(source.Summary) == "" || strings.TrimSpace(source.Kind) == "" {
		return errors.New("invalid source record")
	}
	if _, err := time.Parse(time.RFC3339Nano, source.RecordedAt); err != nil {
		return err
	}
	return device(source.DeviceID)
}

func (s *Store) readSource(id string, out *Source) error {
	if !identifier.MatchString(id) {
		return errors.New("invalid source ID")
	}
	if err := readJSON(filepath.Join(s.Root, "memory/sources", id+".json"), out); err != nil {
		return err
	}
	if out.ID != id {
		return errors.New("source ID mismatch")
	}
	return s.validateSource(*out)
}
func (s *Store) deviceExists(id string) error {
	if err := ValidateDeviceID(id); err != nil {
		return err
	}
	var d Device
	if err := readJSON(filepath.Join(s.Root, "provenance/devices", id+".json"), &d); err != nil {
		return fmt.Errorf("unknown device %s: %w", id, err)
	}
	return validateDeviceMetadata(d, id)
}

func memorySchema() (*jsonschema.Schema, error) {
	b, err := schemas.ReadFile("schemas/memory-revision.schema.json")
	if err != nil {
		return nil, err
	}
	var value any
	if err = json.Unmarshal(b, &value); err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	if err = compiler.AddResource("memory.json", value); err != nil {
		return nil, err
	}
	return compiler.Compile("memory.json")
}

func (s *Store) revisions() ([]Revision, error) {
	if err := s.checkDirectories(); err != nil {
		return nil, err
	}
	records := []Revision{}
	err := filepath.WalkDir(filepath.Join(s.Root, "memory/records"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in memory: %s", path)
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == ".gitkeep" {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			return fmt.Errorf("unexpected memory file: %s", path)
		}
		var r Revision
		if err := readJSON(path, &r); err != nil {
			return err
		}
		if !identifier.MatchString(r.RecordID) || !identifier.MatchString(r.ID) || path != filepath.Join(s.Root, "memory/records", r.RecordID, r.ID+".json") {
			return errors.New("memory ID/path mismatch")
		}
		records = append(records, r)
		return nil
	})
	return records, err
}

func (s *Store) Validate() error {
	records, err := s.revisions()
	if err != nil {
		return err
	}
	if err := s.validateGraph(records); err != nil {
		return err
	}
	if err := s.validateRootFiles(); err != nil {
		return err
	}
	for _, dir := range []string{"memory/sources", "provenance/devices"} {
		entries, err := os.ReadDir(filepath.Join(s.Root, dir))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == ".gitkeep" && entry.Type().IsRegular() {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".json") || !entry.Type().IsRegular() {
				return errors.New("unexpected provenance entry")
			}
			id := strings.TrimSuffix(entry.Name(), ".json")
			if dir == "memory/sources" {
				var source Source
				if err := s.readSource(id, &source); err != nil {
					return err
				}
			} else if err := s.deviceExists(id); err != nil {
				return err
			}
		}
	}
	if _, err := s.FoundlingRegistrations(); err != nil {
		return err
	}
	_, err = s.Journal("", 1)
	return err
}
