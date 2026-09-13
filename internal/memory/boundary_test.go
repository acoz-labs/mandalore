package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSignetContainsOnlyMemoryData(t *testing.T) {
	s := fixtureStore(t)
	for _, path := range []string{"signet.json", "memory/records", "memory/events", "memory/sources", "provenance/devices", "foundlings/registrations", ".mandalore"} {
		if _, err := os.Stat(filepath.Join(s.Root, path)); err != nil {
			t.Fatal(path, err)
		}
	}
	for _, path := range []string{"agent.json", "bank.json", "capabilities", "instructions", "integrations", "provenance/changes", ".my-friday"} {
		if _, err := os.Lstat(filepath.Join(s.Root, path)); !os.IsNotExist(err) {
			t.Fatal("unexpected assistant state", path, err)
		}
	}
}
func TestLegacyMarkersRefusedWithoutWriting(t *testing.T) {
	for _, marker := range []string{"agent.json", "bank.json"} {
		t.Run(marker, func(t *testing.T) {
			s := fixtureStore(t)
			if err := os.WriteFile(filepath.Join(s.Root, marker), []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(s.Root); err == nil {
				t.Fatal("accepted a mixed-format store")
			}
			if err := s.Put(revision("revision-mixed")); err == nil {
				t.Fatal("old handle bypassed legacy guard")
			}
		})
	}
}
