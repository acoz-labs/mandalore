package memorymcp

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/memory"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"path/filepath"
	"testing"
	"time"
)

func TestMCPUsesSharedContractAndRejectsDuplicates(t *testing.T) {
	s, err := memory.Create(filepath.Join(t.TempDir(), "signet"), "Example", "device-test", "Test")
	if err != nil {
		t.Fatal(err)
	}
	service, err := memory.OpenService(s.Root, memory.Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"})
	if err != nil {
		t.Fatal(err)
	}
	a := api.New(service, false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	server, err := New(a).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	list, err := client.ListTools(ctx, nil)
	if err != nil || len(list.Tools) != 18 {
		t.Fatal(list, err)
	}
	for _, tool := range list.Tools {
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatal("missing schema")
		}
		if tool.Name == "signet_create" || tool.Name == "release_inspect" || tool.Name == "release_plan" || tool.Name == "release_apply" || tool.Name == "foundling_register" || tool.Name == "foundling_connect" || tool.Name == "foundling_disconnect" || tool.Name == "foundling_preview" || tool.Name == "foundling_history" {
			t.Fatal("cross-bank admin exposed")
		}
		network := tool.Name == "memory_sync" || tool.Name == "memory_remember_and_sync" || tool.Name == "memory_journal_append_and_sync"
		if tool.Annotations == nil || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint != network {
			t.Fatal("incorrect network annotation", tool.Name)
		}
		data, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(`"connection_result"`)) || bytes.Contains(data, []byte(`"connection_report"`)) || bytes.Contains(data, []byte(`"migration_result"`)) || bytes.Contains(data, []byte(`"foundling_result"`)) || bytes.Contains(data, []byte(`"release_result"`)) {
			t.Fatal("installation-only schemas consume memory-tool context", tool.Name, len(data))
		}
		if tool.Name == "memory_checkpoint" {
			t.Logf("memory_checkpoint output schema: %d bytes", len(data))
		}
		var schema jsonschema.Schema
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		resolved, err := schema.Resolve(nil)
		if err != nil {
			t.Fatalf("unresolvable %s output schema: %v", tool.Name, err)
		}
		input := []byte(`{}`)
		if tool.Name == "memory_remember" {
			input = []byte(`{"kind":"fact","summary":"Schema fixture","body":"Example","basis":"user-direction","reason":"Confirmed"}`)
		}
		if tool.Name == "memory_journal_append" {
			input = []byte(`{"kind":"session","summary":"Schema fixture"}`)
		}
		if tool.Name == "memory_remember_and_sync" {
			input = []byte(`{"record":{"kind":"fact","summary":"Combined schema fixture","body":"Birch Loop","basis":"user-direction","reason":"Confirmed"}}`)
		}
		if tool.Name == "memory_journal_append_and_sync" {
			input = []byte(`{"entry":{"kind":"session","summary":"Combined schema fixture"}}`)
		}
		out := a.Call(ctx, tool.Name, input)
		data, _ = json.Marshal(out)
		var generic any
		if err := json.Unmarshal(data, &generic); err != nil {
			t.Fatal(err)
		}
		if err := resolved.Validate(generic); err != nil {
			t.Fatalf("%s result violates schema: %v", tool.Name, err)
		}
		if tool.Name == "memory_remember_and_sync" || tool.Name == "memory_journal_append_and_sync" {
			wire, err := client.CallTool(ctx, &sdk.CallToolParams{Name: tool.Name, Arguments: json.RawMessage(input)})
			if err != nil || wire.IsError {
				t.Fatalf("combined SDK call: %v %+v", err, wire)
			}
			b, err := json.Marshal(wire.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				OK     bool                  `json:"ok"`
				Result api.SaveAndSyncResult `json:"result"`
			}
			if err := json.Unmarshal(b, &decoded); err != nil || !decoded.OK || !decoded.Result.Saved.DurableLocally || decoded.Result.Saved.ID == "" {
				t.Fatalf("lost combined SDK receipt: %s %v", b, err)
			}
			if err := json.Unmarshal(b, &generic); err != nil {
				t.Fatal(err)
			}
			if err := resolved.Validate(generic); err != nil {
				t.Fatalf("combined wire schema: %v", err)
			}
		}
	}
	if _, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "release_inspect", Arguments: json.RawMessage(`{"candidate":"/synthetic/unavailable"}`)}); err == nil {
		t.Fatal("CLI-only release operation was callable through memory MCP")
	}
	if _, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "release_plan", Arguments: json.RawMessage(`{"candidate":"/synthetic/unavailable","prefix":"/synthetic/prefix"}`)}); err == nil {
		t.Fatal("CLI-only release planning was callable through memory MCP")
	}
	if _, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "release_apply", Arguments: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("CLI-only release activation was callable through memory MCP")
	}
	for _, raw := range []string{`{"query":"one","query":"two"}`, `{"unknown":"PRIVATE-CANARY"}`, `{"limit":-1}`, `{}`} {
		result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_recall", Arguments: json.RawMessage(raw)})
		if err != nil {
			t.Fatal(err)
		}
		direct := a.Call(ctx, "memory_recall", []byte(raw))
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := json.Marshal(direct)
		if err != nil {
			t.Fatal(err)
		}
		var lhs, rhs any
		json.Unmarshal(data, &lhs)
		json.Unmarshal(expected, &rhs)
		canonicalL, _ := json.Marshal(lhs)
		canonicalR, _ := json.Marshal(rhs)
		if string(canonicalL) != string(canonicalR) || result.IsError == direct.OK {
			t.Fatalf("different MCP/CLI contract: %s / %s", canonicalL, canonicalR)
		}
	}
	write := json.RawMessage(`{"kind":"fact","summary":"Project","body":"Copper Finch","basis":"user-direction","reason":"Confirmed"}`)
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_remember", Arguments: write})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		OK     bool        `json:"ok"`
		Result api.Receipt `json:"result"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil || !receipt.OK || !receipt.Result.DurableLocally || receipt.Result.Synchronization != "not-requested" {
		t.Fatal(receipt, err)
	}
	result, err = client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_history", Arguments: map[string]any{"record_id": receipt.Result.RecordID}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	a.ReadOnly = true
	result, err = client.CallTool(ctx, &sdk.CallToolParams{Name: "memory_remember", Arguments: write})
	if err != nil || !result.IsError {
		t.Fatal("read-only mutation accepted", result, err)
	}
	data, err = json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var refused api.Envelope
	if err := json.Unmarshal(data, &refused); err != nil || refused.Error == nil || refused.Error.Code != "operation.read_only" {
		t.Fatal(refused, err)
	}
}
