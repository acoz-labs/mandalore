package memory

import (
	"errors"
	"fmt"
	"strings"
)

// Snapshot is a decoded, closed set of portable memory objects. Importers own
// bounded strict decoding, canonical file paths and preservation of original
// bytes. This validation API neither reads nor writes the filesystem. Foundling
// registrations are deliberately outside the supported legacy import format.
type Snapshot struct {
	Signet    Signet
	Devices   []Device
	Sources   []Source
	Revisions []Revision
	Journal   []JournalEntry
}

func ValidateSnapshot(snapshot Snapshot) error {
	s := snapshot.Signet
	if s.Version != FormatVersion || !identifier.MatchString(s.ID) || !textWithin(s.Name, 256) || strings.ContainsAny(s.Name, "\r\n") {
		return errors.New("invalid snapshot signet")
	}
	devices := map[string]bool{}
	for _, d := range snapshot.Devices {
		if d.Version != 1 || !identifier.MatchString(d.ID) || strings.TrimSpace(d.Label) == "" || devices[d.ID] {
			return errors.New("invalid or duplicate snapshot device")
		}
		devices[d.ID] = true
	}
	device := func(id string) error {
		if !devices[id] {
			return fmt.Errorf("unknown snapshot device %s", id)
		}
		return nil
	}
	sources := map[string]bool{}
	for _, source := range snapshot.Sources {
		if sources[source.ID] || source.ExternalOrigin != nil {
			return errors.New("duplicate or unsupported snapshot source")
		}
		if err := validateSourceMetadata(source, device); err != nil {
			return err
		}
		sources[source.ID] = true
	}
	if err := validateRevisionGraph(snapshot.Revisions, s.ID, device, func(id string) error {
		if !sources[id] {
			return fmt.Errorf("unknown snapshot source %s", id)
		}
		return nil
	}); err != nil {
		return err
	}
	entries := map[string]bool{}
	for _, entry := range snapshot.Journal {
		if entries[entry.ID] {
			return errors.New("duplicate snapshot journal entry")
		}
		if err := validateJournalEntry(entry, device); err != nil {
			return err
		}
		entries[entry.ID] = true
	}
	return nil
}
