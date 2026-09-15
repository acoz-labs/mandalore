package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/memory"
)

func inlineCall(t *testing.T, a *API, name string, input any) Envelope {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	out := a.Call(context.Background(), name, raw)
	if !out.OK {
		t.Fatalf("%s: %+v", name, out.Error)
	}
	return out
}

func TestSaveAndSyncSaveFailureNeverDelivers(t *testing.T) {
	a := fixture(t)
	remote := inlineRemote(t, a)
	before := inlineInventory(t, a.service.Root())
	head := inlineGit(t, "--git-dir="+remote, "rev-parse", "main")
	for _, name := range []string{"memory_remember_and_sync", "memory_journal_append_and_sync"} {
		out := a.Call(context.Background(), name, []byte(`{}`))
		if out.OK || out.Error.Code != "input.invalid" || out.Error.WriteMayHaveOccurred {
			t.Fatal(out)
		}
	}
	if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) || head != inlineGit(t, "--git-dir="+remote, "rev-parse", "main") {
		t.Fatal("failed save changed content/delivery")
	}
}

func TestSaveAndSyncWriterContentionAtEachStage(t *testing.T) {
	for _, stage := range []string{"save", "delivery"} {
		t.Run(stage, func(t *testing.T) {
			a := fixture(t)
			remote := inlineRemote(t, a)
			head := inlineGit(t, "--git-dir="+remote, "rev-parse", "main")
			lock, err := os.OpenFile(filepath.Join(a.service.Root(), ".mandalore", "write.lock"), os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			take := func() {
				if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "save" {
				take()
			}
			out, err := saveAndSync(context.Background(), a.service, nil, func() (Receipt, error) {
				r, err := appendJournal(context.Background(), a.service, JournalWrite{Kind: "test", Summary: "Contention boundary."})
				if err == nil && stage == "delivery" {
					take()
				}
				return r, err
			})
			if stage == "save" {
				if err == nil || out.Saved.DurableLocally {
					t.Fatal(out, err)
				}
			} else if err != nil || !out.Saved.DurableLocally || out.Delivery.Error == nil || out.Delivery.Error.Code != "store.busy" || out.Delivery.OK {
				t.Fatal(out, err)
			}
			if head != inlineGit(t, "--git-dir="+remote, "rev-parse", "main") {
				t.Fatal("contention caused remote delivery")
			}
			if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
				t.Fatal(err)
			}
			if stage == "delivery" {
				before := inlineInventory(t, a.service.Root())
				inlineCall(t, a, "memory_sync", struct{}{})
				if !reflect.DeepEqual(before, inlineInventory(t, a.service.Root())) {
					t.Fatal("recovery duplicated the save")
				}
			}
		})
	}
}

func TestSaveAndSyncCancellationDuringFetchKeepsReceipt(t *testing.T) {
	testSaveAndSyncCancellationDuringFetch(t, "normal")
}

func TestSaveAndSyncCancellationWaitsForCompleteFetchMarker(t *testing.T) {
	testSaveAndSyncCancellationDuringFetch(t, "slow-marker")
}

func testSaveAndSyncCancellationDuringFetch(t *testing.T, startup string) {
	t.Helper()
	a, _, marker := cancellationFixture(t, startup)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Envelope, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- a.Call(ctx, "memory_journal_append_and_sync", []byte(`{"entry":{"kind":"test","summary":"Saved before fetch cancellation."},"timeout_seconds":30}`))
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("combined worker retained")
		}
	})
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
wait:
	for {
		select {
		case out := <-done:
			t.Fatalf("returned before fetch: %+v", out)
		case <-deadline.C:
			t.Fatal("fetch did not start")
		case <-ticker.C:
			// Redirection creates the marker before printf writes its PID. Wait
			// for the complete line, or cancellation can destroy our evidence.
			if data, err := os.ReadFile(marker); err == nil && strings.HasSuffix(string(data), "\n") && len(strings.TrimSpace(string(data))) > 0 {
				break wait
			} else if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
	}
	cancel()
	select {
	case out := <-done:
		if !out.OK {
			t.Fatal("cancellation hid save", out)
		}
		got := out.Result.(SaveAndSyncResult)
		if !got.Saved.DurableLocally || got.Saved.ID == "" || got.Delivery.OK || got.Delivery.Error == nil || got.Delivery.Error.Code != "operation.cancelled" || got.Delivery.Error.SyncStatus == nil || got.Delivery.Error.SyncStatus.Phase != "fetch" {
			t.Fatal(got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fetch did not cancel")
	}
	assertStoppedGit(t, marker)
	if err := a.service.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAndSyncRetainsConcurrentSemanticHeads(t *testing.T) {
	a := fixture(t)
	remote := inlineRemote(t, a)
	base := memory.Write{Kind: "decision", Summary: "Route", Body: "Old route", Basis: "user-direction", Reason: "Initial choice"}
	r := inlineCall(t, a, "memory_remember", base).Result.(Receipt)
	inlineCall(t, a, "memory_sync", struct{}{})
	clone := filepath.Join(t.TempDir(), "clone")
	inlineGit(t, "clone", remote, clone)
	s, err := memory.OpenService(clone, memory.Authorship{DeviceID: "device-test", Actor: "Other test writer", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	b := New(s, false)
	base.RecordID, base.Supersedes, base.Body = r.RecordID, []string{r.ID}, "Left route"
	inlineCall(t, a, "memory_remember", base)
	base.Body = "Right route"
	inlineCall(t, b, "memory_remember_and_sync", RememberAndSyncInput{Record: base})
	got := inlineCall(t, a, "memory_journal_append_and_sync", JournalAndSyncInput{Entry: JournalWrite{Kind: "test", Summary: "Concurrent choices remain unresolved."}}).Result.(SaveAndSyncResult)
	if !got.Saved.DurableLocally || !got.Delivery.OK || got.Delivery.Result == nil || !got.Delivery.Result.Delivered || got.Delivery.Result.State != "conflicted" || got.Delivery.Result.SemanticConflicts != 1 {
		t.Fatal(got)
	}
	packet, err := a.service.Recall("", nil, 5, 8192)
	if err != nil || len(packet.Conflicts) != 1 {
		t.Fatal(packet, err)
	}
}

func TestSaveAndSyncBindingIsolationAndReplacement(t *testing.T) {
	a, other := fixture(t), fixture(t)
	remote := inlineRemote(t, a)
	beforeOther := inlineInventory(t, other.service.Root())
	head := inlineGit(t, "--git-dir="+remote, "rev-parse", "main")
	out, err := saveAndSync(context.Background(), a.service, nil, func() (Receipt, error) {
		r, err := appendJournal(context.Background(), a.service, JournalWrite{Kind: "test", Summary: "Saved before identity replacement."})
		if err != nil {
			return r, err
		}
		replacement, err := os.ReadFile(filepath.Join(other.service.Root(), "signet.json"))
		if err != nil {
			return r, err
		}
		return r, os.WriteFile(filepath.Join(a.service.Root(), "signet.json"), replacement, 0600)
	})
	if err != nil || !out.Saved.DurableLocally || out.Saved.SignetID != a.service.ID() || out.Delivery.Error == nil || out.Delivery.Error.Code != "binding.invalid" {
		t.Fatal(out, err)
	}
	if head != inlineGit(t, "--git-dir="+remote, "rev-parse", "main") || !reflect.DeepEqual(beforeOther, inlineInventory(t, other.service.Root())) {
		t.Fatal("switched bank or delivered replacement")
	}
}
