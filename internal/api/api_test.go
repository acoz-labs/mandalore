package api

import (
	"context"
	"encoding/json"
	"github.com/acoz-labs/mandalore/internal/memory"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func fixture(t *testing.T) *API {
	t.Helper()
	s, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Example", "device-test", "Test")
	if err != nil {
		t.Fatal(err)
	}
	service, err := memory.OpenService(s.Root, memory.Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return New(service, false)
}

func TestCorruptStoreIsNotAnInputError(t *testing.T) {
	a := fixture(t)
	out := a.Call(context.Background(), "memory_remember", []byte(`{"kind":"fact","summary":"Example","body":"Copper Finch","basis":"user-direction","reason":"Confirmed"}`))
	if !out.OK {
		t.Fatal(out)
	}
	r := out.Result.(Receipt)
	path := filepath.Join(a.service.Root(), "memory", "records", r.RecordID, r.ID+".json")
	if err := os.WriteFile(path, []byte(`{"body":"PRIVATE-CANARY"}`), 0600); err != nil {
		t.Fatal(err)
	}
	out = a.Call(context.Background(), "memory_recall", []byte(`{}`))
	if out.OK || out.Error.Code != "store.invalid" || out.Error.Retryable || out.Error.WriteMayHaveOccurred {
		t.Fatal(out)
	}
}

func TestCatalogAndSharedCalls(t *testing.T) {
	a := fixture(t)
	catalog := Catalog()
	if len(catalog) != 36 {
		t.Fatalf("expected 36 implemented operations, got %d", len(catalog))
	}
	for _, op := range catalog {
		if op.InputSchema == nil || op.OutputSchema == nil {
			t.Fatal("missing typed discovery", op.Name)
		}
	}
	out := a.Call(context.Background(), "memory_remember", []byte(`{"kind":"fact","summary":"Example","body":"Copper Finch","basis":"user-direction","reason":"Confirmed"}`))
	if !out.OK {
		t.Fatal(out.Error)
	}
	receipt := out.Result.(Receipt)
	if !receipt.DurableLocally || receipt.Synchronization != "not-requested" || receipt.ID == "" {
		t.Fatal(receipt)
	}
	recalled := a.Call(context.Background(), "memory_recall", []byte(`{"query":"Copper"}`))
	if !recalled.OK {
		t.Fatal(recalled.Error)
	}
	data, err := json.Marshal(recalled)
	if err != nil || len(data) > 9000 {
		t.Fatal(err)
	}
}

func TestRejectedCallsAndReadOnly(t *testing.T) {
	a := fixture(t)
	for _, input := range []string{`{"query":"one","query":"two"}`, `{"unknown":"PRIVATE-CANARY"}`, `{"limit":0}`, `{"limit":-1}`} {
		out := a.Call(context.Background(), "memory_recall", []byte(input))
		if out.OK || out.Error.Code != "input.invalid" {
			t.Fatal(out)
		}
	}
	a.ReadOnly = true
	out := a.Call(context.Background(), "memory_remember", []byte(`{}`))
	if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
		t.Fatal(out)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out = a.Call(ctx, "memory_recall", []byte(`{}`))
	if out.OK || out.Error.Code != "operation.cancelled" {
		t.Fatal(out)
	}
	if out = a.Call(context.Background(), "not_a_tool", []byte(`{}`)); out.OK || out.Error.Code != "operation.unknown" {
		t.Fatal(out)
	}
}

func TestWriteContentionAndIOHaveDifferentRetryAdvice(t *testing.T) {
	a := fixture(t)
	input := []byte(`{"kind":"fact","summary":"Example","body":"Evidence","basis":"user-direction","reason":"Confirmed"}`)
	lock, err := os.OpenFile(filepath.Join(a.service.Root(), ".mandalore", "write.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	out := a.Call(context.Background(), "memory_remember", input)
	if out.OK || out.Error.Code != "store.busy" || !out.Error.Retryable || out.Error.WriteMayHaveOccurred || ExitCode(out) != 3 {
		t.Fatal(out)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(a.service.Root(), "memory", "sources"), filepath.Join(t.TempDir(), "sources")); err != nil {
		t.Fatal(err)
	}
	out = a.Call(context.Background(), "memory_remember", input)
	if out.OK || out.Error.Code != "operation.io" || out.Error.Retryable || !out.Error.WriteMayHaveOccurred || !out.Error.InspectBeforeRetry {
		t.Fatal(out)
	}
}
