package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func inlineGit(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("synthetic Git fixture: %v: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func inlineInventory(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".mandalore" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out[rel] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSaveAndSyncRejectsBeforePublication(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"memory_remember_and_sync", `"record":{"kind":"fact","summary":"Route","body":"Birch Loop","basis":"user-direction","reason":"Confirmed"}`},
		{"memory_journal_append_and_sync", `"entry":{"kind":"test","summary":"Confirmed test outcome."}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, suffix := range []string{`,"timeout_seconds":0`, `,"timeout_seconds":31`, `,"timeout_seconds":"3"`, `,"timeout_seconds":1.5`, `,"unknown":true`} {
				a := fixture(t)
				before := inlineInventory(t, a.service.Root())
				out := a.Call(context.Background(), tc.name, []byte("{"+tc.body+suffix+"}"))
				if out.OK || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred {
					t.Fatalf("invalid input: %+v", out)
				}
				if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
					t.Fatal("invalid input published memory")
				}
			}
			a := fixture(t)
			before := inlineInventory(t, a.service.Root())
			a.ReadOnly = true
			out := a.Call(context.Background(), tc.name, []byte("{"+tc.body+"}"))
			if out.OK || out.Error.Code != "operation.read_only" {
				t.Fatal(out)
			}
			a.ReadOnly = false
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			out = a.Call(ctx, tc.name, []byte("{"+tc.body+"}"))
			if out.OK || out.Error.Code != "operation.cancelled" || out.Error.WriteMayHaveOccurred {
				t.Fatal(out)
			}
			if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
				t.Fatal("prohibited/cancelled call changed memory")
			}
		})
	}
}

func TestSaveAndSyncDeliveryFailureKeepsSavedIdentity(t *testing.T) {
	for _, setup := range []string{"missing-git", "no-origin", "unavailable-origin"} {
		t.Run(setup, func(t *testing.T) {
			a := fixture(t)
			if setup != "missing-git" {
				if out := a.Call(context.Background(), "memory_git_init", []byte(`{}`)); !out.OK {
					t.Fatal(out)
				}
			}
			if setup == "unavailable-origin" {
				inlineGit(t, "-C", a.service.Root(), "remote", "add", "origin", filepath.Join(t.TempDir(), "absent.git"))
			}
			out := a.Call(context.Background(), "memory_remember_and_sync", []byte(`{"record":{"kind":"fact","summary":"Route","body":"Birch Loop","basis":"user-direction","reason":"Confirmed"}}`))
			if !out.OK {
				t.Fatalf("delivery hid successful save: %+v", out)
			}
			got := out.Result.(SaveAndSyncResult)
			if !got.Saved.DurableLocally || got.Saved.ID == "" || got.Saved.RecordID == "" {
				t.Fatal(got)
			}
			if _, err := os.Stat(filepath.Join(a.service.Root(), "memory", "records", got.Saved.RecordID, got.Saved.ID+".json")); err != nil {
				t.Fatal(err)
			}
			if setup == "missing-git" {
				if got.Delivery.OK || got.Delivery.Error == nil {
					t.Fatal(got)
				}
				if _, err := os.Stat(filepath.Join(a.service.Root(), ".git")); !os.IsNotExist(err) {
					t.Fatalf("implicit Git setup: %v", err)
				}
			} else {
				if !got.Delivery.OK || got.Delivery.Result == nil || got.Delivery.Result.Delivered {
					t.Fatal(got)
				}
				want := "pending"
				if setup == "no-origin" {
					want = "local-only"
				}
				if got.Delivery.Result.State != want {
					t.Fatal(got)
				}
			}
		})
	}
}

func TestSaveAndSyncCancellationBetweenStages(t *testing.T) {
	a := fixture(t)
	inlineRemote(t, a)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var published Receipt
	out, err := saveAndSync(ctx, a.service, nil, func() (Receipt, error) {
		var err error
		published, err = appendJournal(ctx, a.service, JournalWrite{Kind: "test", Summary: "Durable before cancellation."})
		cancel()
		return published, err
	})
	if err != nil || out.Saved.ID != published.ID || !out.Saved.DurableLocally || out.Delivery.OK || out.Delivery.Error == nil || out.Delivery.Error.Code != "operation.cancelled" {
		t.Fatalf("cancellation lost saved receipt: %+v, %v", out, err)
	}
	if out.Delivery.Result != nil && out.Delivery.Result.Delivered {
		t.Fatal("cancelled delivery claimed success")
	}
}

func inlineRemote(t *testing.T, a *API) string {
	t.Helper()
	if out := a.Call(context.Background(), "memory_git_init", []byte(`{}`)); !out.OK {
		t.Fatal(out)
	}
	remote := filepath.Join(t.TempDir(), "origin.git")
	inlineGit(t, "init", "--bare", "--initial-branch=main", remote)
	inlineGit(t, "-C", a.service.Root(), "remote", "add", "origin", remote)
	if out := a.Call(context.Background(), "memory_sync", []byte(`{}`)); !out.OK {
		t.Fatal(out)
	}
	return remote
}

func TestSaveAndSyncPublishesBothKinds(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"memory_remember_and_sync", `{"record":{"kind":"fact","summary":"Fictional route","body":"Birch Loop","basis":"user-direction","reason":"Confirmed synthetic request"}}`},
		{"memory_journal_append_and_sync", `{"entry":{"kind":"test","summary":"Confirmed synthetic inline journal outcome."}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture(t)
			remote := inlineRemote(t, a)
			before := inlineGit(t, "--git-dir="+remote, "rev-parse", "main")
			out := a.Call(context.Background(), tc.name, []byte(tc.input))
			if !out.OK {
				t.Fatalf("combined save failed: %+v", out.Error)
			}
			var got struct {
				Saved    Receipt `json:"saved"`
				Delivery struct {
					OK     bool               `json:"ok"`
					Result *signetsync.Status `json:"result"`
					Error  *MemoryError       `json:"error"`
				} `json:"delivery"`
			}
			data, err := json.Marshal(out.Result)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !got.Saved.DurableLocally || got.Saved.ID == "" || got.Saved.SignetID != a.service.ID() {
				t.Fatalf("missing durable saved identity: %s", data)
			}
			if !got.Delivery.OK || got.Delivery.Error != nil || got.Delivery.Result == nil || !got.Delivery.Result.Delivered || got.Delivery.Result.State != "synchronized" {
				t.Fatalf("delivery not established: %s", data)
			}
			after := inlineGit(t, "--git-dir="+remote, "rev-parse", "main")
			if after == before || after != got.Delivery.Result.Head || after != inlineGit(t, "-C", a.service.Root(), "rev-parse", "HEAD") {
				t.Fatalf("receipt/local/remote mismatch: %s", data)
			}
			if tc.name == "memory_remember_and_sync" && got.Saved.RecordID == "" {
				t.Fatalf("missing knowledge record: %s", data)
			}
		})
	}
}

func TestSaveAndSyncCatalogPreservesLocalOnlyTools(t *testing.T) {
	expected := map[string]bool{
		"memory_remember": false, "memory_journal_append": false,
		"memory_remember_and_sync": true, "memory_journal_append_and_sync": true,
	}
	for _, op := range Catalog() {
		if network, ok := expected[op.Name]; ok {
			if op.Network != network || op.ReadOnly || op.Idempotent || !op.RequiresBinding || op.CLIOnly {
				t.Fatalf("incorrect operation boundary for %s", op.Name)
			}
			delete(expected, op.Name)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing operations: %v", expected)
	}
}
