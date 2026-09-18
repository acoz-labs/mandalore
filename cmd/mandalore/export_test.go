package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/exportreport"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestExportWrapperReadOnlyBeforeInputAndStrictFlags(t *testing.T) {
	for _, args := range [][]string{{"export", "apply", "--read-only"}, {"export", "apply", "--binding", "/unrelated"}, {"export", "preview", "unexpected"}, {"call", "export_preview", "--binding", "/ignored"}} {
		var out bytes.Buffer
		code := run(context.Background(), args, strings.NewReader("invalid"), &out, &out)
		if code != 2 {
			t.Fatal(args, code, out.String())
		}
		if len(args) > 2 && args[2] == "--read-only" && !strings.Contains(out.String(), "operation.read_only") {
			t.Fatal("decoded before denial", out.String())
		}
	}
}

func TestExportWrapperAcceptsReviewedEnvelopeAndBindingFlag(t *testing.T) {
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
	in := exportreport.Request{Destination: filepath.Join(root, "report"), Selection: exportreport.Selection{RecordIDs: []string{r.RecordID}}}
	raw, _ := json.Marshal(in)
	var preview, applied bytes.Buffer
	if code := run(context.Background(), []string{"export", "preview", "--binding", bpath}, bytes.NewReader(raw), &preview, &preview); code != 0 {
		t.Fatal(code, preview.String())
	}
	if code := run(context.Background(), []string{"export", "apply"}, bytes.NewReader(preview.Bytes()), &applied, &applied); code != 0 {
		t.Fatal(code, applied.String())
	}
	if _, err := os.Stat(filepath.Join(in.Destination, "report.json")); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{`{"protocol_version":1,"ok":false,"result":{}}`, `{"protocol_version":1,"ok":true,"result":{},"extra":1}`, `{"version":1,"version":2}`} {
		var out bytes.Buffer
		if code := run(context.Background(), []string{"export", "apply"}, strings.NewReader(data), &out, &out); code != 2 {
			t.Fatal("invalid envelope accepted", code, out.String())
		}
	}
}

func TestExportWrapperAndTypedPreviewParity(t *testing.T) {
	// Invalid explicit paths still exercise common typed failure semantics,
	// without any native profile, binding or environment fallback.
	input := `{"binding_path":"/nonexistent-export-test/binding.json","destination":"/nonexistent-export-test/report","selection":{"record_ids":["record-test"]}}`
	var wrapper, typed bytes.Buffer
	a := run(context.Background(), []string{"export", "preview"}, strings.NewReader(input), &wrapper, &wrapper)
	b := run(context.Background(), []string{"call", "export_preview"}, strings.NewReader(input), &typed, &typed)
	if a != b || wrapper.String() != typed.String() {
		t.Fatal(a, b, wrapper.String(), typed.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(wrapper.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["ok"] != false {
		t.Fatal("invalid path accepted")
	}
}
