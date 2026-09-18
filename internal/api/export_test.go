package api

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/exportreport"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestExportCatalogAndReadOnlyBeforeDecode(t *testing.T) {
	count := 0
	for _, op := range Catalog() {
		if op.Name == "export_preview" || op.Name == "export_apply" {
			count++
			if !op.CLIOnly || op.RequiresBinding || op.Network || op.ReadOnly != (op.Name == "export_preview") {
				t.Fatal("wrong export authority", op.Name)
			}
		}
	}
	if count != 2 {
		t.Fatal("missing export operations")
	}
	out := New(nil, true).Call(context.Background(), "export_apply", []byte("invalid"))
	if out.OK || out.Error.Code != "operation.read_only" || out.Error.WriteMayHaveOccurred {
		t.Fatal(out)
	}
}

func TestExportTypedRoundTripAndReceipt(t *testing.T) {
	root := t.TempDir()
	s, err := memory.Create(filepath.Join(root, "bank"), "Synthetic", "device-test", "Test")
	if err != nil {
		t.Fatal(err)
	}
	bpath := filepath.Join(root, "config", "binding.json")
	if _, err := binding.Bind(s.Root, bpath, "Test", "Test"); err != nil {
		t.Fatal(err)
	}
	svc, err := binding.Open(bpath, "test")
	if err != nil {
		t.Fatal(err)
	}
	r, err := svc.Remember(memory.Write{Kind: "fact", Summary: "Synthetic", Body: "Selected", Basis: "observation", Reason: "Test"})
	if err != nil {
		t.Fatal(err)
	}
	in := exportreport.Request{BindingPath: bpath, Destination: filepath.Join(root, "report"), Selection: exportreport.Selection{RecordIDs: []string{r.RecordID}}}
	data, _ := json.Marshal(in)
	preview := New(nil, true).Call(context.Background(), "export_preview", data)
	if !preview.OK {
		t.Fatal(preview.Error)
	}
	plan, ok := preview.Result.(exportreport.Plan)
	if !ok {
		t.Fatal("untyped preview")
	}
	data, _ = json.Marshal(plan)
	applied := New(nil, false).Call(context.Background(), "export_apply", data)
	if !applied.OK {
		t.Fatal(applied.Error)
	}
	if _, err := os.Stat(filepath.Join(in.Destination, "report.json")); err != nil {
		t.Fatal(err)
	}
	replay := New(nil, false).Call(context.Background(), "export_apply", data)
	if replay.OK || replay.Error.Code != "export.failed" || replay.Error.WriteMayHaveOccurred || replay.Error.ExportResult == nil {
		t.Fatal("missing prewrite refusal", replay)
	}
}

func TestExportFailurePreservesPartialAndCancelledReceipts(t *testing.T) {
	for _, cause := range []error{errors.New("export failed"), context.Canceled} {
		r := exportreport.Receipt{Phase: "writing", StagingDirectory: "/tmp/synthetic/stage", Destination: "/tmp/synthetic/report", BytesWritten: 12}
		out := New(nil, false).failure(Operation{Name: "export_apply"}, &exportFailure{err: cause, result: &r})
		if out.OK || !out.Error.WriteMayHaveOccurred || !out.Error.InspectBeforeRetry || out.Error.ExportResult != &r {
			t.Fatal("lost partial receipt", out)
		}
		if cause == context.Canceled && out.Error.Code != "operation.cancelled" {
			t.Fatal("lost cancellation", out)
		}
	}
}
