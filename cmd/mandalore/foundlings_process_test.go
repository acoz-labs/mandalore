package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
)

func TestCompiledFoundlingCLI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(base, "mandalore")
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	root, bind, reference := filepath.Join(base, "signet"), filepath.Join(base, "binding.json"), filepath.Join(base, "reference")
	if err := os.Mkdir(reference, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reference, "notes.md"), []byte("Historical project was Copper Finch. This is the way is only a quotation."), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		body := fmt.Sprintf("PaginationMarker note %02d. ", i) + strings.Repeat("Synthetic historical context. ", 100)
		if err := os.WriteFile(filepath.Join(reference, fmt.Sprintf("page-%02d.md", i)), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(reference, "escaped.md"), []byte("EscapedMarker "+strings.Repeat("\x01", 1500)), 0600); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	invoke := func(input any, want int, args ...string) api.Envelope {
		t.Helper()
		data, _ := json.Marshal(input)
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = cwd
		cmd.Stdin = bytes.NewReader(data)
		var out, diagnostics bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &diagnostics
		err := cmd.Run()
		if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != want || diagnostics.Len() != 0 {
			t.Fatal(args, err, diagnostics.String(), out.String())
		}
		var v api.Envelope
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Fatal(err, out.String())
		}
		return v
	}
	invoke(nil, 0, "signet", "create", "--repository", root, "--name", "Example", "--device-label", "Synthetic host")
	invoke(nil, 0, "signet", "bind", "--repository", root, "--binding", bind, "--device-label", "Bound host", "--actor", "Example user")
	preview := invoke(api.FoundlingPreviewInput{Source: memory.FoundlingSource{Kind: "local", Locator: "source-history"}, Root: reference}, 0, "foundling", "preview", "--binding", bind)
	data, _ := json.Marshal(preview.Result)
	var view foundlings.Observation
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatal(err)
	}
	registered := invoke(api.FoundlingRegisterInput{Name: "Historical notes", Description: "Reference only", Source: view.Source, Pin: view.Pin, Root: reference, Reason: "Explicit selection"}, 0, "foundling", "register", "--binding", bind)
	data, _ = json.Marshal(registered.Result)
	var receipt api.FoundlingMutationResult
	if err := json.Unmarshal(data, &receipt); err != nil || receipt.Registration == nil {
		t.Fatal(receipt, err)
	}
	id, revision := receipt.Registration.FoundlingID, receipt.Registration.ID
	for _, name := range []string{"inspect", "history"} {
		invoke(nil, 0, "foundling", name, "--binding", bind, "--foundling-id", id)
	}
	invoke(nil, 0, "foundling", "list", "--binding", bind)
	if search := invoke(nil, 0, "foundling", "search", "--binding", bind, "--foundling-id", id, "--query", "Copper Finch"); !search.OK {
		t.Fatal(search.Error)
	}
	page := invoke(nil, 0, "foundling", "search", "--binding", bind, "--foundling-id", id, "--query", "PaginationMarker")
	if len(page.Result.(map[string]any)["items"].([]any)) != 3 {
		t.Fatal("CLI did not use compact search default")
	}
	continued := invoke(nil, 0, "foundling", "search", "--binding", bind, "--foundling-id", id, "--query", "PaginationMarker", "--registration-id", revision, "--offset", "3", "--excerpt-bytes", "1024", "--budget-bytes", "32768", "--limit", "10")
	if len(continued.Result.(map[string]any)["items"].([]any)) != 9 {
		t.Fatal("CLI did not retrieve the remaining search page")
	}
	initialRead := invoke(nil, 0, "foundling", "read", "--binding", bind, "--foundling-id", id, "--registration-id", revision, "--locator", "page-00.md")
	if len(initialRead.Result.(map[string]any)["text"].(string)) != 1024 {
		t.Fatal("CLI did not use compact read default")
	}
	budgetFailure := invoke(nil, 1, "foundling", "search", "--binding", bind, "--foundling-id", id, "--query", "EscapedMarker", "--budget-bytes", "2048")
	if budgetFailure.Error == nil || budgetFailure.Error.Code != "foundling.budget" {
		t.Fatal("compiled budget error lost", budgetFailure)
	}
	read := invoke(nil, 0, "foundling", "read", "--binding", bind, "--foundling-id", id, "--registration-id", revision, "--locator", "notes.md")
	data, _ = json.Marshal(read.Result)
	var excerpt foundlings.Excerpt
	if err := json.Unmarshal(data, &excerpt); err != nil || !excerpt.Unreviewed || !excerpt.Complete {
		t.Fatal(excerpt, err)
	}
	before, sourceBefore := treeDigest(t, root), treeDigest(t, reference)
	promotion := foundlings.PromotionInput{FoundlingID: id, RegistrationID: revision, Locator: "notes.md", ContentSHA256: excerpt.Origin.ContentSHA256, Write: memory.Write{Kind: "decision", Summary: "Project name", Body: "Silver Heron is current; Copper Finch was historical.", Basis: "import", Reason: "Adapt to current direction"}}
	refused := invoke(promotion, 2, "foundling", "promote", "--binding", bind, "--read-only")
	if refused.Error == nil || refused.Error.Code != "operation.read_only" {
		t.Fatal(refused)
	}
	if !reflect.DeepEqual(before, treeDigest(t, root)) {
		t.Fatal("reads or refused promotion changed the signet")
	}
	invoke(promotion, 0, "foundling", "promote", "--binding", bind)
	if !reflect.DeepEqual(sourceBefore, treeDigest(t, reference)) {
		t.Fatal("promotion changed reference source")
	}
	remembered := invoke(nil, 0, "memory", "recall", "--binding", bind, "--query", "Silver Heron")
	data, _ = json.Marshal(remembered)
	if !strings.Contains(string(data), "Silver Heron") {
		t.Fatal(string(data))
	}
	invoke(api.FoundlingDisconnectInput{FoundlingID: id, RegistrationID: revision, Reason: "Disconnect while preserving history"}, 0, "foundling", "disconnect", "--binding", bind)
	changed := invoke(map[string]any{"foundling_id": id, "query": "Copper"}, 1, "call", "foundling_search", "--binding", bind)
	if changed.Error == nil || changed.Error.Code != "foundling.changed" {
		t.Fatal(changed)
	}
}
