package memory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) upgradeRecords() ([]UpgradeRecord, error) {
	dir := filepath.Join(s.Root, "provenance/upgrades")
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return []UpgradeRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("invalid upgrade evidence directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	records := []UpgradeRecord{}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			return nil, errors.New("nonregular upgrade evidence")
		}
		if e.Name() == ".gitkeep" {
			info, err := e.Info()
			if err != nil {
				return nil, err
			}
			if info.Size() != 0 {
				return nil, errors.New("nonempty upgrade placeholder")
			}
			continue
		}
		var u UpgradeRecord
		if err := readJSON(filepath.Join(dir, e.Name()), &u); err != nil {
			return nil, err
		}
		if !strings.HasSuffix(e.Name(), ".json") || !identifier.MatchString(u.ID) || e.Name() != u.ID+".json" {
			return nil, errors.New("upgrade evidence ID/path mismatch")
		}
		records = append(records, u)
	}
	return records, nil
}

func (s *Store) validateUpgradeState() error {
	records, err := s.upgradeRecords()
	if err != nil {
		return err
	}
	return ValidateUpgradeRecords(s.Signet, records, memoizeDeviceValidation(s.deviceExists))
}
