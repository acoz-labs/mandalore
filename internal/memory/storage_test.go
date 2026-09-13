package memory

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestWriterContentionAndRecovery(t *testing.T) {
	s := fixtureStore(t)
	f, err := os.OpenFile(filepath.Join(s.Root, ".mandalore/write.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	other, err := Open(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Put(revision("revision-busy")); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatal("writer did not report contention", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := other.Put(revision("revision-after-lock")); err != nil {
		t.Fatal(err)
	}
}

func TestAtomicDirectoryPublicationNeverReplacesTarget(t *testing.T) {
	root := t.TempDir()
	from, to := filepath.Join(root, "staging"), filepath.Join(root, "target")
	for _, dir := range []string{from, to} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := renameNewDirectory(from, to); err == nil {
		t.Fatal("replaced an existing empty directory")
	}
	for _, dir := range []string{from, to} {
		if _, err := os.Stat(dir); err != nil {
			t.Fatal("publication failure lost data", err)
		}
	}
}

func TestOrphanSourceAndDeviceValidation(t *testing.T) {
	for _, dir := range []string{"memory/sources", "provenance/devices"} {
		t.Run(dir, func(t *testing.T) {
			s := fixtureStore(t)
			if err := os.WriteFile(filepath.Join(s.Root, dir, "invalid.json"), []byte(`{"schema_version":99}`), 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.Validate(); err == nil {
				t.Fatal("validation ignored malformed unreferenced data")
			}
		})
	}
}

func TestRevisionPathMustBeCanonical(t *testing.T) {
	s := fixtureStore(t)
	r := revision("revision-path")
	if err := s.Put(r); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(s.Root, "memory/records/extra"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(s.Root, "memory/records", r.RecordID), filepath.Join(s.Root, "memory/records/extra", r.RecordID)); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err == nil {
		t.Fatal("accepted noncanonical revision location")
	}
}
