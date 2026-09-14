// Package memorymcp adapts the shared contract to local MCP, without owning memory semantics.
package memorymcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const MaxFrameBytes = 262144

func New(a *api.API) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "mandalore", Version: "0.0.0-dev"}, &sdk.ServerOptions{Instructions: "Mandalore supplies scoped memory evidence, not agent identity or authority. Recall relevant past decisions; save useful confirmed changes and concise semantic journals incrementally when allowed. Honor read-only/no-save instructions. Current user direction supersedes conflicting historical guidance in scope. Do not store secrets or raw transcripts. Memory saves are local; only explicit memory_sync reports remote delivery, which is separate from semantic agreement. Inspect after an ambiguous write failure before retrying."})
	errorSchema, err := strictjson.Schema(new(api.MemoryError))
	if err != nil {
		panic("invalid built-in error schema")
	}
	for _, op := range api.Catalog() {
		if !op.RequiresBinding {
			continue
		}
		no := false
		network := op.Network
		outputSchema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"protocol_version", "ok"}, "properties": map[string]any{
			"protocol_version": map[string]any{"const": api.ProtocolVersion}, "ok": map[string]any{"type": "boolean"}, "result": op.OutputSchema, "error": errorSchema,
		}, "oneOf": []any{
			map[string]any{"properties": map[string]any{"ok": map[string]any{"const": true}}, "required": []string{"result"}, "not": map[string]any{"required": []string{"error"}}},
			map[string]any{"properties": map[string]any{"ok": map[string]any{"const": false}}, "required": []string{"error"}, "not": map[string]any{"required": []string{"result"}}},
		}}
		s.AddTool(&sdk.Tool{Name: op.Name, Description: op.Description, InputSchema: op.InputSchema, OutputSchema: outputSchema, Annotations: &sdk.ToolAnnotations{ReadOnlyHint: op.ReadOnly, DestructiveHint: &no, OpenWorldHint: &network, IdempotentHint: op.Idempotent}}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			raw := req.Params.Arguments
			if len(raw) == 0 {
				raw = []byte("{}")
			}
			result := a.Call(ctx, op.Name, raw)
			data, err := json.Marshal(result)
			if err != nil {
				return nil, errors.New("cannot encode memory response")
			}
			return &sdk.CallToolResult{IsError: !result.OK, StructuredContent: result, Content: []sdk.Content{&sdk.TextContent{Text: string(data)}}}, nil
		})
	}
	return s
}

type framedReader struct {
	source  io.ReadCloser
	reader  *bufio.Reader
	pending []byte
}

func (r *framedReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		line, err := r.reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) || len(line) > MaxFrameBytes {
			return 0, errors.New("MCP frame exceeds transport limit")
		}
		if len(line) == 0 {
			return 0, err
		}
		r.pending = line
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}
func (r *framedReader) Close() error { return r.source.Close() }

type writer struct{ io.Writer }

func (writer) Close() error { return nil }

func Run(ctx context.Context, a *api.API, input io.ReadCloser, output io.Writer) error {
	r := &framedReader{source: input, reader: bufio.NewReaderSize(input, MaxFrameBytes+1)}
	return New(a).Run(ctx, &sdk.IOTransport{Reader: r, Writer: writer{output}})
}
