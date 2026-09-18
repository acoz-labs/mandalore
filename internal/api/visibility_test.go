package api

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/formatupgrade"
	"github.com/acoz-labs/mandalore/internal/memory"
	signetsync "github.com/acoz-labs/mandalore/internal/sync"
)

func upgradedVisibilityAPI(t *testing.T) (*API, memory.Revision) {
	t.Helper()
	a := fixture(t)
	r, err := a.service.Remember(memory.Write{Kind: "fact", Summary: "Synthetic record", Body: "WITHDRAW_CANARY", Basis: "observation", Reason: "Fixture"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "binding.json")
	if _, err := binding.Bind(a.service.Root(), path, "Test", "Test"); err != nil {
		t.Fatal(err)
	}
	sy, err := signetsync.Open(a.service.Root(), a.service.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sy.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	p := inlineCall(t, New(nil, true), "signet_upgrade_preview", formatupgrade.Request{BindingPath: path}).Result.(formatupgrade.Plan)
	request := formatupgrade.ApplyRequest{Plan: p, StoppedWriters: true}
	activated := inlineCall(t, New(nil, false), "signet_upgrade_apply", request).Result.(formatupgrade.Receipt)
	if !activated.DurableLocally || !activated.Activated || activated.Checkpointed || activated.Delivered {
		t.Fatal(activated)
	}
	recovered := inlineCall(t, New(nil, false), "signet_upgrade_recover", request).Result.(formatupgrade.Receipt)
	if !recovered.DurableLocally || recovered.EvidenceID != activated.EvidenceID {
		t.Fatal(recovered)
	}
	s, err := binding.Open(path, "test")
	if err != nil {
		t.Fatal(err)
	}
	return New(s, false), r
}

func TestVisibilityAPIAuthorityAndTypedRoundTrip(t *testing.T) {
	for _, name := range []string{"memory_withdraw", "memory_restore"} {
		out := New(nil, true).Call(context.Background(), name, []byte("invalid"))
		if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
			t.Fatal(out)
		}
	}
	a, r := upgradedVisibilityAPI(t)
	h := inlineCall(t, a, "memory_visibility_history", HistoryInput{RecordID: r.RecordID}).Result.(memory.VisibilityHistory)
	in := memory.VisibilityWrite{RecordID: r.RecordID, ContentHeads: h.State.ContentHeads, VisibilityHeads: h.State.VisibilityHeads, Reason: "Explicit withdrawal"}
	w := inlineCall(t, a, "memory_withdraw", in).Result.(memory.VisibilityReceipt)
	if !w.DurableLocally || w.State.State != "withdrawn" || w.Synchronization != "not-requested" {
		t.Fatal(w)
	}
	data, _ := json.Marshal(in)
	stale := a.Call(context.Background(), "memory_restore", data)
	if stale.OK || stale.Error.Code != "memory.stale_heads" || stale.Error.WriteMayHaveOccurred || stale.Error.VisibilityResult == nil {
		t.Fatal(stale)
	}
	h = inlineCall(t, a, "memory_visibility_history", HistoryInput{RecordID: r.RecordID}).Result.(memory.VisibilityHistory)
	in.ContentHeads, in.VisibilityHeads = h.State.ContentHeads, h.State.VisibilityHeads
	restored := inlineCall(t, a, "memory_restore", in).Result.(memory.VisibilityReceipt)
	if !restored.DurableLocally || restored.State.State != "visible" {
		t.Fatal(restored)
	}
}

func TestVisibilityFailuresRetainReceiptsAndCancellation(t *testing.T) {
	r := memory.VisibilityReceipt{SignetID: "signet-test", RecordID: "record-test", EventID: "visibility-test", WriteMayHaveOccurred: true, Synchronization: "not-requested"}
	out := New(nil, false).failure(Operation{Name: "memory_withdraw"}, &visibilityFailure{err: context.Canceled, result: r})
	if out.OK || out.Error.Code != "operation.cancelled" || !out.Error.InspectBeforeRetry || !out.Error.WriteMayHaveOccurred || out.Error.VisibilityResult.EventID != r.EventID {
		t.Fatal(out)
	}
	a := fixture(t)
	data := []byte(`{"record_id":"record-test","expected_content_heads":["revision-test"],"expected_visibility_heads":[],"reason":"Explicit decision"}`)
	out = a.Call(context.Background(), "memory_withdraw", data)
	if out.OK || out.Error.Code != "memory.upgrade_required" || out.Error.WriteMayHaveOccurred {
		t.Fatal(out)
	}
}

func TestUpgradeAPICatalogAndPartialReceipts(t *testing.T) {
	count := 0
	for _, op := range Catalog() {
		if op.Name != "signet_upgrade_preview" && op.Name != "signet_upgrade_apply" && op.Name != "signet_upgrade_recover" {
			continue
		}
		count++
		if !op.CLIOnly || op.RequiresBinding || op.Network || op.ReadOnly != (op.Name == "signet_upgrade_preview") {
			t.Fatal("upgrade exposed as ordinary memory action", op.Name)
		}
		if !op.ReadOnly {
			out := New(nil, true).Call(context.Background(), op.Name, []byte("invalid"))
			if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
				t.Fatal(out)
			}
		}
	}
	if count != 3 {
		t.Fatal("upgrade operations missing")
	}
	r := formatupgrade.Receipt{Phase: "manifest-replaced", StagingDirectory: "/tmp/synthetic/prepared", Activated: true, EvidencePublished: true}
	out := New(nil, false).failure(Operation{Name: "signet_upgrade_apply"}, &upgradeFailure{err: context.Canceled, result: &r})
	if out.OK || out.Error.Code != "operation.cancelled" || !out.Error.WriteMayHaveOccurred || !out.Error.InspectBeforeRetry || out.Error.UpgradeResult != &r {
		t.Fatal("partial activation lost", out)
	}
}
