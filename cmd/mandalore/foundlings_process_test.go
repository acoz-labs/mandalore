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
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCompiledFoundlingCLIAndMCP(t *testing.T) {
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
	search := invoke(nil, 0, "foundling", "search", "--binding", bind, "--foundling-id", id, "--query", "Copper Finch")
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
	cmd := exec.CommandContext(ctx, binary, "mcp", "--binding", bind, "--harness", "synthetic-foundling-mcp")
	cmd.Dir = cwd
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	client, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-foundling", Version: "1"}, nil).Connect(ctx, &sdk.CommandTransport{Command: cmd, TerminateDuration: time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	tools, err := client.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 21 {
		t.Fatal(tools, err)
	}
	inputBytes, outputBytes, descriptionBytes := 0, 0, 0
	for _, tool := range tools.Tools {
		inputJSON, _ := json.Marshal(tool.InputSchema)
		outputJSON, _ := json.Marshal(tool.OutputSchema)
		inputBytes += len(inputJSON)
		outputBytes += len(outputJSON)
		descriptionBytes += len(tool.Description)
		if tool.Name == "foundling_register" || tool.Name == "foundling_connect" {
			t.Fatal("CLI-only setup advertised through MCP")
		}
	}
	t.Logf("MCP discovery: %d tools; input schemas %d bytes; output schemas %d bytes; descriptions %d bytes (not token counts or transport overhead)", len(tools.Tools), inputBytes, outputBytes, descriptionBytes)
	if forbidden, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "foundling_register", Arguments: map[string]any{}}); err == nil && (forbidden == nil || !forbidden.IsError) {
		t.Fatal("CLI-only setup callable through MCP", forbidden)
	}
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "foundling_search", Arguments: map[string]any{"foundling_id": id, "query": "Copper Finch"}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	data, _ = json.Marshal(result.StructuredContent)
	var fromMCP api.Envelope
	if err := json.Unmarshal(data, &fromMCP); err != nil || !reflect.DeepEqual(search, fromMCP) {
		t.Fatal("CLI/MCP search mismatch", err)
	}
	for _, tc := range []struct {
		name string
		args map[string]any
		want api.Envelope
	}{
		{"foundling_search", map[string]any{"foundling_id": id, "query": "PaginationMarker"}, page},
		{"foundling_search", map[string]any{"foundling_id": id, "query": "PaginationMarker", "registration_revision_id": revision, "offset": 3, "excerpt_bytes": 1024, "budget_bytes": 32768, "limit": 10}, continued},
		{"foundling_read", map[string]any{"foundling_id": id, "registration_revision_id": revision, "relative_locator": "page-00.md"}, initialRead},
	} {
		wire, err := client.CallTool(ctx, &sdk.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil || wire.IsError {
			t.Fatalf("progressive stdio request: %v %+v", err, wire)
		}
		encoded, _ := json.Marshal(wire.StructuredContent)
		var actual api.Envelope
		if err := json.Unmarshal(encoded, &actual); err != nil || !reflect.DeepEqual(actual, tc.want) {
			t.Fatalf("progressive CLI/MCP mismatch: %s: %v", tc.name, err)
		}
	}
	wireFailure, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "foundling_search", Arguments: map[string]any{"foundling_id": id, "query": "EscapedMarker", "budget_bytes": 2048}})
	if err != nil || !wireFailure.IsError {
		t.Fatalf("stdio budget error: %v %+v", err, wireFailure)
	}
	data, _ = json.Marshal(wireFailure.StructuredContent)
	var failureFromMCP api.Envelope
	if err := json.Unmarshal(data, &failureFromMCP); err != nil || !reflect.DeepEqual(budgetFailure, failureFromMCP) {
		t.Fatal("CLI/MCP budget error mismatch", err)
	}
	if !reflect.DeepEqual(before, treeDigest(t, root)) {
		t.Fatal("MCP read saved memory")
	}
	promotion := foundlings.PromotionInput{FoundlingID: id, RegistrationID: revision, Locator: "notes.md", ContentSHA256: excerpt.Origin.ContentSHA256, Write: memory.Write{Kind: "decision", Summary: "Project name", Body: "Silver Heron is current; Copper Finch was historical.", Basis: "import", Reason: "Adapt to current direction"}}
	result, err = client.CallTool(ctx, &sdk.CallToolParams{Name: "foundling_promote", Arguments: promotion})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	if err := client.Close(); err != nil || diagnostics.Len() != 0 {
		t.Fatal(err, diagnostics.String())
	}
	readOnlyCmd := exec.CommandContext(ctx, binary, "mcp", "--binding", bind, "--read-only")
	readOnlyCmd.Dir = cwd
	readOnlyCmd.Stderr = &diagnostics
	readOnlyClient, err := sdk.NewClient(&sdk.Implementation{Name: "synthetic-readonly", Version: "1"}, nil).Connect(ctx, &sdk.CommandTransport{Command: readOnlyCmd, TerminateDuration: time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = readOnlyClient.Close() })
	readOnlyBefore := treeDigest(t, root)
	for _, tc := range []struct {
		name string
		args map[string]any
		want api.Envelope
	}{
		{"foundling_search", map[string]any{"foundling_id": id, "query": "PaginationMarker"}, page},
		{"foundling_search", map[string]any{"foundling_id": id, "query": "PaginationMarker", "registration_revision_id": revision, "offset": 3, "excerpt_bytes": 1024, "budget_bytes": 32768, "limit": 10}, continued},
		{"foundling_read", map[string]any{"foundling_id": id, "registration_revision_id": revision, "relative_locator": "page-00.md"}, initialRead},
	} {
		wire, err := readOnlyClient.CallTool(ctx, &sdk.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil || wire.IsError {
			t.Fatalf("read-only retrieval refused: %s: %v %+v", tc.name, err, wire)
		}
		encoded, _ := json.Marshal(wire.StructuredContent)
		var actual api.Envelope
		if err := json.Unmarshal(encoded, &actual); err != nil || !reflect.DeepEqual(actual, tc.want) {
			t.Fatalf("read-only retrieval changed result: %s: %v", tc.name, err)
		}
	}
	denied, err := readOnlyClient.CallTool(ctx, &sdk.CallToolParams{Name: "foundling_promote", Arguments: promotion})
	if err != nil || !denied.IsError {
		t.Fatal("read-only MCP promoted reference", denied, err)
	}
	data, _ = json.Marshal(denied.StructuredContent)
	var deniedEnvelope api.Envelope
	if err := json.Unmarshal(data, &deniedEnvelope); err != nil || deniedEnvelope.Error == nil || deniedEnvelope.Error.Code != "operation.read_only" {
		t.Fatal(deniedEnvelope, err)
	}
	if err := readOnlyClient.Close(); err != nil || diagnostics.Len() != 0 {
		t.Fatal(err, diagnostics.String())
	}
	if !reflect.DeepEqual(readOnlyBefore, treeDigest(t, root)) {
		t.Fatal("read-only retrieval/refused promotion changed signet")
	}
	if !reflect.DeepEqual(sourceBefore, treeDigest(t, reference)) {
		t.Fatal("promotion changed reference source")
	}
	remembered := invoke(nil, 0, "memory", "recall", "--binding", bind, "--query", "Silver Heron")
	data, _ = json.Marshal(remembered)
	if !strings.Contains(string(data), "Silver Heron") {
		t.Fatal(string(data))
	}
	refused := invoke(promotion, 2, "foundling", "promote", "--binding", bind, "--read-only")
	if refused.Error == nil || refused.Error.Code != "operation.read_only" {
		t.Fatal(refused)
	}
	invoke(api.FoundlingDisconnectInput{FoundlingID: id, RegistrationID: revision, Reason: "Disconnect while preserving history"}, 0, "foundling", "disconnect", "--binding", bind)
	changed := invoke(map[string]any{"foundling_id": id, "query": "Copper"}, 1, "call", "foundling_search", "--binding", bind)
	if changed.Error == nil || changed.Error.Code != "foundling.changed" {
		t.Fatal(changed)
	}
}
