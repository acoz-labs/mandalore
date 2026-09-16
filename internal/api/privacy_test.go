package api

import (
	"context"
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/memory"
)

type privacyFileState struct {
	Mode fs.FileMode
	Hash [32]byte
}

// Includes Git, operational files and empty directories; excludes access times
// and OS caches. Test-owned fixtures contain no symlinks or special files.
func privacyInventory(t *testing.T, root string) map[string]privacyFileState {
	t.Helper()
	out := map[string]privacyFileState{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		state := privacyFileState{Mode: info.Mode()}
		if !entry.IsDir() {
			if !info.Mode().IsRegular() {
				t.Fatal("unexpected fixture file type")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			state.Hash = sha256.Sum256(data)
		}
		out[rel] = state
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPrivacyReadOnlyDeniesMutationNotDisclosure(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		name := "without-git"
		if initialized {
			name = "with-git"
		}
		t.Run(name, func(t *testing.T) {
			a := fixture(t)
			ctx := context.Background()
			const marker = "SYNTHETIC-RESTRICTED-NOT-A-SECRET"
			if _, err := a.service.Remember(memory.Write{Kind: "fact", Summary: "Privacy example", Body: marker, Basis: "observation", Reason: "Fixture", Sensitivity: "restricted"}); err != nil {
				t.Fatal(err)
			}
			if _, err := a.service.AppendJournal("fixture", marker); err != nil {
				t.Fatal(err)
			}
			if initialized {
				if out := a.Call(ctx, "memory_git_init", []byte(`{}`)); !out.OK {
					t.Fatal(out.Error)
				}
			}
			readOnly := New(a.service, true)
			before := privacyInventory(t, a.service.Root())
			const record = `{"kind":"fact","summary":"Denied","body":"Synthetic","basis":"observation","reason":"Fixture"}`
			const journal = `{"kind":"fixture","summary":"Denied synthetic entry"}`
			for _, call := range []struct{ name, input string }{
				{"memory_remember", record},
				{"memory_journal_append", journal},
				{"memory_remember_and_sync", `{"record":` + record + `}`},
				{"memory_journal_append_and_sync", `{"entry":` + journal + `}`},
				{"memory_git_init", `{}`}, {"memory_checkpoint", `{}`}, {"memory_sync", `{}`},
			} {
				out := readOnly.Call(ctx, call.name, []byte(call.input))
				if out.OK || out.Error == nil || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
					t.Errorf("%s did not reject before mutation: %+v", call.name, out.Error)
				}
			}
			for _, operation := range []string{"memory_recall", "memory_journal"} {
				out := readOnly.Call(ctx, operation, []byte(`{}`))
				if !out.OK {
					t.Fatal("read-only unexpectedly denied read", operation, out.Error)
				}
			}
			text, warning := contextText(t, readOnly, "Privacy example")
			if warning != "" || !strings.Contains(text, marker) {
				t.Fatal("restricted content absent from native context", warning)
			}
			if !reflect.DeepEqual(before, privacyInventory(t, a.service.Root())) {
				t.Fatal("read-only changed files, directories, content or modes")
			}
		})
	}
}
