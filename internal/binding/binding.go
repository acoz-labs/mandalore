package binding

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

// Binding is machine-local. Each clone is enrolled independently; moving the
// bank preserves its ID, while replacing it with another bank requires rebind.
type Binding struct {
	Version  int    `json:"schema_version"`
	SignetID string `json:"signet_id"`
	Root     string `json:"root"`
	DeviceID string `json:"device_id"`
	Actor    string `json:"actor"`
}

func DefaultPath() (string, error) {
	if path := os.Getenv("MANDALORE_BINDING"); path != "" {
		if !filepath.IsAbs(path) {
			return "", errors.New("MANDALORE_BINDING must be an absolute path")
		}
		return path, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "mandalore", "binding.json"), nil
}

// prospectivePath resolves the existing ancestor before any directories are
// created, including when a symlink points a proposed local path into the bank.
func prospectivePath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) || filepath.Dir(path) == path {
		return "", err
	}
	if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("binding paths cannot traverse unresolved symlinks; select a real directory")
	}
	parent, err := prospectivePath(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

// ValidateBindingDestination preflights a new machine-local connection before
// a wizard creates its bank. Bind repeats it before writing and publishes
// the connection without replacing an existing file.
func ValidateBindingDestination(root, path string) error {
	if err := validateLocation(root, path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return errors.New("binding already exists or cannot be inspected; use a new path, preserving the existing binding")
	}
	return nil
}

func validateLocation(root, path string) error {
	if root == "" || path == "" {
		return errors.New("bank and binding paths required")
	}
	root, err := prospectivePath(root)
	if err != nil {
		return err
	}
	path, err = prospectivePath(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return errors.New("machine binding must be outside the Git-backed bank")
	}
	return nil
}

func Bind(root, path, label, actor string) (Binding, error) {
	var b Binding
	s, err := memory.Open(root)
	if err != nil {
		return b, err
	}

	if !textWithin(label, 256) || !textWithin(actor, 256) || path == "" {
		return b, errors.New("binding path, machine label, and actor required")
	}
	path, err = prospectivePath(path)
	if err != nil {
		return b, err
	}
	if err := ValidateBindingDestination(s.Root, path); err != nil {
		return b, err
	}
	if err := s.Validate(); err != nil {
		return b, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return b, err
	}
	b = Binding{Version: 1, SignetID: s.Signet.ID, Root: s.Root, DeviceID: memory.NewID("device"), Actor: actor}
	f, err := os.CreateTemp(filepath.Dir(path), ".memory-binding-*")
	if err != nil {
		return Binding{}, err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := json.NewEncoder(f).Encode(b); err != nil {
		return Binding{}, err
	}
	if err := f.Sync(); err != nil {
		return Binding{}, err
	}
	if err := f.Close(); err != nil {
		return Binding{}, err
	}
	if err := s.AddDevice(memory.Device{Version: 1, ID: b.DeviceID, Label: label}); err != nil {
		return Binding{}, err
	}
	// A failed publication may leave an unused device record. It must not
	// overwrite another binding, nor be reported as successful enrollment.
	if err := os.Link(f.Name(), path); err != nil {
		return Binding{}, err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return Binding{}, err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return Binding{}, err
	}
	return b, nil
}

func Open(path, harness string) (*memory.Service, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 16384 {
		return nil, errors.New("binding must be a regular JSON file under 16 KiB")
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
	var b Binding
	if err := strictjson.Decode(data, &b, 16384); err != nil {
		return nil, err
	}
	if b.Version != 1 || !filepath.IsAbs(b.Root) || !textWithin(harness, 64) {
		return nil, errors.New("invalid binding version, root, or harness")
	}
	if err := validateLocation(b.Root, path); err != nil {
		return nil, err
	}
	s, err := memory.OpenService(b.Root, memory.Authorship{DeviceID: b.DeviceID, Actor: b.Actor, Harness: harness})
	if err != nil {
		return nil, err
	}
	if s.ID() != b.SignetID {
		return nil, errors.New("bound bank identity changed; select the intended bank explicitly")
	}
	return s, nil
}

func textWithin(value string, max int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= max && !strings.ContainsRune(value, '\x00')
}
