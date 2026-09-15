package memorymcp

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/memory"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Characterization, not a rendering fix: both wire representations are required
// for existing clients. A model-facing selector must preserve the full envelope.
func TestPresentationWireCompatibility(t *testing.T) {
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
	st, ct := sdk.NewInMemoryTransports()
	server, err := New(a).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := sdk.NewClient(&sdk.Implementation{Name: "presentation-test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, tc := range []struct {
		name, args string
		readOnly   bool
		wantOK     bool
	}{
		{"memory_remember", `{"kind":"fact","summary":"Fictional project","body":"Silver Heron; previously Copper Finch. A < B & C. Café.\nQuoted: \"yes\".","basis":"user-direction","reason":"Synthetic fixture"}`, false, true},
		{"memory_recall", `{}`, false, true},
		{"memory_recall", `{"limit":-1}`, false, false},
		{"memory_scopes", `{}`, false, true},
		{"foundling_list", `{}`, false, true},
		{"memory_remember", `{}`, true, false},
	} {
		a.ReadOnly = tc.readOnly
		r, err := client.CallTool(ctx, &sdk.CallToolParams{Name: tc.name, Arguments: json.RawMessage(tc.args)})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Content) != 1 {
			t.Fatalf("%s: lost compatibility text", tc.name)
		}
		content, ok := r.Content[0].(*sdk.TextContent)
		if !ok {
			t.Fatalf("%s: unexpected content type", tc.name)
		}
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(content.Text), &raw); err != nil {
			t.Fatal(err)
		}
		structured, err := json.Marshal(r.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		// Canonicalize nested object order; the SDK decodes structured content
		// into maps. UseNumber prevents rounding historical JSON numbers.
		canonical := func(data []byte) []byte {
			t.Helper()
			var value any
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				t.Fatal(err)
			}
			out, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			return out
		}
		left := canonical(raw)
		right := canonical(structured)
		if !bytes.Equal(left, right) {
			t.Fatalf("%s: text/structured envelopes differ", tc.name)
		}
		var envelope api.Envelope
		if err := json.Unmarshal(structured, &envelope); err != nil {
			t.Fatal(err)
		}
		if r.IsError == envelope.OK {
			t.Fatalf("%s: inconsistent error state", tc.name)
		}
		if envelope.OK != tc.wantOK {
			t.Fatalf("%s (%s): ok=%t, want %t", tc.name, tc.args, envelope.OK, tc.wantOK)
		}
		wrapper, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(wrapper) <= 2*len(structured) {
			t.Fatalf("%s: duplication reproduction changed", tc.name)
		}
		t.Logf("%s: serialized wrapper=%d bytes; one envelope=%d bytes (not native tokens)", tc.name, len(wrapper), len(structured))
	}
}
