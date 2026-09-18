package foundlings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestConfiguredRootsIncludesDisconnectedWithoutReadingSources(t *testing.T) {
	m, r, root := connectedFixture(t)
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := m.memory.WriteFoundling(memory.FoundlingWrite{FoundlingID: r.FoundlingID, Name: r.Name, Description: r.Description, Source: r.Source, Pin: r.Pin, State: "disconnected", Supersedes: []string{r.ID}, Reason: "Disconnect synthetic source"}); err != nil {
		t.Fatal(err)
	}
	// A missing source must still be reported as a protected coordinate. The
	// caller decides whether its destination can be safely checked against it.
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(root+"-moved", root)
	before := fileTree(t, m.memory.Root())
	a, err := m.ConfiguredRoots(context.Background())
	if err != nil || !reflect.DeepEqual(a.Roots, []string{root}) || len(a.SHA256) != 64 {
		t.Fatal(a, err)
	}
	b, err := m.ConfiguredRoots(context.Background())
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal(a, b, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("configuration read wrote state")
	}
}

func TestConfiguredRootsAbsentStateDoesNotCreateIt(t *testing.T) {
	m, _, _ := connectedFixture(t)
	before := fileTree(t, m.memory.Root())
	v, err := m.ConfiguredRoots(context.Background())
	if err != nil || len(v.Roots) != 0 || len(v.SHA256) != 64 {
		t.Fatal(v, err)
	}
	if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
		t.Fatal("configuration read created state")
	}
}

func TestConfiguredRootsPinsRawMetadataAndBoundsEnumeration(t *testing.T) {
	m, r, root := connectedFixture(t)
	if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
		t.Fatal(err)
	}
	a, err := m.ConfiguredRoots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.memory.Root(), connectionPath(r.FoundlingID))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := m.ConfiguredRoots(context.Background())
	if err != nil || a.SHA256 == b.SHA256 || !reflect.DeepEqual(a.Roots, b.Roots) {
		t.Fatal(a, b, err)
	}
	for i := 0; i < 256; i++ {
		if err := os.WriteFile(filepath.Join(filepath.Dir(path), fmt.Sprintf("foundling-extra-%03d.json", i)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.ConfiguredRoots(context.Background()); err == nil {
		t.Fatal("unbounded connection enumeration accepted")
	}
}

func TestConfiguredRootsRejectsUnknownAndRedirectedMetadata(t *testing.T) {
	for _, kind := range []string{"unknown", "directory", "symlink", "tampered", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			m, r, root := connectedFixture(t)
			if _, err := m.Connect(context.Background(), r.FoundlingID, r.ID, root, ""); err != nil {
				t.Fatal(err)
			}
			base := filepath.Join(m.memory.Root(), ".mandalore", "foundlings")
			switch kind {
			case "unknown":
				writeFixture(t, base, "unknown.txt", "preserve")
			case "directory":
				if err := os.Mkdir(filepath.Join(base, "nested"), 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(base, r.FoundlingID+".json"), filepath.Join(base, "foundling-alias.json")); err != nil {
					t.Fatal(err)
				}
			case "tampered":
				writeFixture(t, base, r.FoundlingID+".json", `{ "schema_version": 1 }`)
			case "oversized":
				if err := os.WriteFile(filepath.Join(base, r.FoundlingID+".json"), make([]byte, 16385), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := fileTree(t, m.memory.Root())
			if _, err := m.ConfiguredRoots(context.Background()); err == nil {
				t.Fatal("unsafe metadata accepted")
			}
			if !reflect.DeepEqual(before, fileTree(t, m.memory.Root())) {
				t.Fatal("failure changed metadata")
			}
		})
	}
	m, _, _ := connectedFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.ConfiguredRoots(ctx); err == nil {
		t.Fatal("cancelled read accepted")
	}
}
