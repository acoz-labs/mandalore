package memory

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Read the complete event graph on every operation. There is no process-local
// visibility cache that could outlive synchronization or a concurrent decision.
func (s *Store) visibilityEvents() ([]VisibilityEvent, error) {
	if err := s.checkDirectories(); err != nil {
		return nil, err
	}
	root := filepath.Join(s.Root, "memory/visibility")
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return []VisibilityEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("invalid visibility directory")
	}
	events := []VisibilityEvent{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in visibility evidence")
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("nonregular visibility evidence")
		}
		if d.Name() == ".gitkeep" {
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Size() != 0 {
				return errors.New("nonempty visibility placeholder")
			}
			return nil
		}
		var e VisibilityEvent
		if err := readJSON(path, &e); err != nil {
			return err
		}
		if !identifier.MatchString(e.RecordID) || !identifier.MatchString(e.ID) || path != filepath.Join(root, e.RecordID, e.ID+".json") {
			return errors.New("visibility ID/path mismatch")
		}
		events = append(events, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.Signet.Version == 1 && len(events) != 0 {
		return nil, errors.New("visibility evidence requires upgraded signet format")
	}
	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	return events, nil
}

func (s *Store) visibilityStates(records []Revision) (map[string]VisibilityState, error) {
	events, err := s.visibilityEvents()
	if err != nil {
		return nil, err
	}
	return ResolveVisibility(records, events, memoizeDeviceValidation(s.deviceExists))
}
