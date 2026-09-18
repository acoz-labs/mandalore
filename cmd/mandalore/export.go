package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/exportreport"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func runExport(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(out, help); err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 0 || (args[0] != "preview" && args[0] != "apply") {
		return bad(out, "Choose export preview or apply; input is a JSON request or reviewed preview.")
	}
	sub := args[0]
	f := flag.NewFlagSet("export "+sub, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	readOnly := f.Bool("read-only", false, "Reject writes before reading input")
	var bindingPath string
	if sub == "preview" {
		f.StringVar(&bindingPath, "binding", "", "Explicit binding, if absent from the JSON request")
	}
	if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
		return bad(out, "Invalid export flags or arguments; use export --help.")
	}
	if *readOnly && sub == "apply" {
		return emit(out, api.Failure("operation.read_only", "Mutations are disabled for this task.", false))
	}
	raw, err := io.ReadAll(io.LimitReader(input, api.MaxInputBytes+1))
	if err != nil {
		return bad(out, "Cannot read export input.")
	}
	if sub == "preview" {
		var request exportreport.Request
		if strictjson.Decode(raw, &request, api.MaxInputBytes) != nil {
			return bad(out, "Expected a strict export request object.")
		}
		if bindingPath != "" {
			if request.BindingPath != "" && request.BindingPath != bindingPath {
				return bad(out, "Binding flag and request disagree; select one explicit binding.")
			}
			request.BindingPath = bindingPath
		}
		raw, err = json.Marshal(request)
	} else {
		var object map[string]any
		if strictjson.Decode(raw, &object, api.MaxInputBytes) != nil {
			return bad(out, "Expected the complete reviewed export preview or successful preview envelope.")
		}
		if _, ok := object["protocol_version"]; ok {
			var envelope struct {
				Protocol int               `json:"protocol_version"`
				OK       bool              `json:"ok"`
				Result   exportreport.Plan `json:"result"`
			}
			if strictjson.Decode(raw, &envelope, api.MaxInputBytes) != nil || !envelope.OK || envelope.Protocol != api.ProtocolVersion {
				return bad(out, "Expected a successful export preview envelope.")
			}
			raw, err = json.Marshal(envelope.Result)
		}
	}
	if err != nil {
		return bad(out, "Cannot encode export input.")
	}
	return emit(out, api.New(nil, *readOnly).Call(ctx, "export_"+sub, raw))
}
