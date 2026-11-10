package memory

import (
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyTree copies a committed fixture into a fresh directory.
func copyTree(t *testing.T, from string) string {
	t.Helper()
	to := filepath.Join(t.TempDir(), "signet")
	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0700)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), b, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return to
}

func hashTree(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	for _, dir := range []string{"signet.json", "memory", "provenance", "foundlings"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(path)
			rel, _ := filepath.Rel(root, path)
			out[filepath.ToSlash(rel)] = sha256.Sum256(b)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// #246 requirement 7: data written by 2.1.2 (testdata/compat, rebuilt by its
// generate.sh) stays valid byte for byte, and topics add files only under
// memory/scopes.
func TestReleasedFormatsStayValidWithTopics(t *testing.T) {
	for _, format := range []string{"format1", "format2"} {
		t.Run(format, func(t *testing.T) {
			root := copyTree(t, filepath.Join("testdata/compat", format))
			before := hashTree(t, root)
			read, err := OpenReadOnlyService(root)
			if err != nil {
				t.Fatal(err)
			}
			if got := map[string]int{"format1": 1, "format2": 2}[format]; read.store.Signet.Version != got {
				t.Fatalf("fixture is format %d", read.store.Signet.Version)
			}
			copper := Scope{Kind: "project", ID: "copper-finch"}
			p, err := read.Recall("", &copper, 10, 8192)
			if err != nil || len(p.Conflicts) != 1 {
				t.Fatalf("fixture conflict not read: %+v %v", p, err)
			}
			journal, err := read.Journal("", nil, 10)
			if err != nil || len(journal) != 1 || journal[0].Scope != nil {
				t.Fatalf("fixture journal: %+v %v", journal, err)
			}
			registrations, err := read.store.FoundlingRegistrations()
			if err != nil || len(registrations) != 2 {
				t.Fatalf("fixture registrations: %d %v", len(registrations), err)
			}
			devices, err := read.store.DeviceIDs()
			if err != nil || len(devices) != 2 {
				t.Fatal(devices, err)
			}
			s, err := OpenService(root, Authorship{DeviceID: devices[0], Actor: "Synthetic actor", Harness: "test"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.NameScopeIdempotent("compat-name-request-01", ScopeLabelWrite{Scope: copper, Name: "Copper Finch", Description: "Synthetic topic over released data.", Reason: "r"}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.AppendJournalIdempotent("compat-journal-request", "outcome", "Scoped entry over released data", &copper); err != nil {
				t.Fatal(err)
			}
			prime, err := s.Prime(copper, 8192)
			if err != nil || len(prime.Labels) != 1 || len(prime.Conflicts) != 1 || len(prime.Journal) != 1 {
				t.Fatalf("prime over released data: %+v %v", prime, err)
			}
			if journal, err = s.Journal("", nil, 10); err != nil || len(journal) != 2 {
				t.Fatalf("unscoped entry no longer returned: %+v %v", journal, err)
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
			after := hashTree(t, root)
			for path, sum := range before {
				if after[path] != sum {
					t.Fatalf("%s changed", path)
				}
			}
			added := 0
			for path := range after {
				if _, ok := before[path]; !ok {
					added++
					if !strings.HasPrefix(path, "memory/scopes/") {
						t.Fatalf("new file outside memory/scopes: %s", path)
					}
				}
			}
			if added != 2 {
				t.Fatalf("%d new files", added)
			}
		})
	}
}
