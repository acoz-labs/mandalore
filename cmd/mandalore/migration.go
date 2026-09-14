package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/migration"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func runMigration(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	if len(args) == 0 {
		return bad(out, "Choose migration preflight or apply; use --help.")
	}
	if args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	sub := args[0]
	if sub != "preflight" && sub != "apply" {
		return bad(out, "Unknown migration command; use --help.")
	}
	f := flag.NewFlagSet("migration "+sub, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	readOnly := f.Bool("read-only", false, "Reject mutations")
	var options migration.Options
	var stopped bool
	if sub == "preflight" {
		f.StringVar(&options.Source, "source", "", "Explicit legacy memory-only bank directory")
		f.StringVar(&options.Output, "output", "", "New output bundle; parent must exist")
		f.StringVar(&options.DeviceLabel, "device-label", "", "Explicit conversion machine label")
		f.StringVar(&options.Actor, "actor", "", "Explicit conversion actor")
		f.StringVar(&options.Binding, "legacy-binding", "", "Optional old machine binding to inspect, never copy")
		f.StringVar(&options.NativeHome, "native-home", "", "Optional explicit native profile for inventory")
		f.StringVar(&options.NativeBinary, "native-binary", "", "Explicit trusted native binary for inventory")
	} else {
		f.BoolVar(&stopped, "writers-stopped", false, "Acknowledge all relevant writers have been stopped")
	}
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(out, "Invalid migration flags or missing values; use --help.")
	}
	if f.NArg() != 0 {
		return bad(out, "Unexpected migration arguments.")
	}
	if *readOnly && sub == "apply" {
		return emit(out, api.Failure("operation.read_only", "Mutations are disabled for this task.", false))
	}
	var value any = options
	if sub == "apply" {
		if !stopped {
			return bad(out, "Stop all relevant writers, then supply --writers-stopped with the reviewed preflight plan.")
		}
		raw, err := io.ReadAll(io.LimitReader(input, api.MaxInputBytes+1))
		if err != nil {
			return bad(out, "Cannot read migration plan.")
		}
		var object map[string]any
		if strictjson.Decode(raw, &object, api.MaxInputBytes) != nil {
			return bad(out, "Expected a valid plan object or successful migration-preflight envelope.")
		}
		if _, ok := object["protocol_version"]; ok {
			var envelope struct {
				Protocol int            `json:"protocol_version"`
				OK       bool           `json:"ok"`
				Result   migration.Plan `json:"result"`
			}
			if strictjson.Decode(raw, &envelope, api.MaxInputBytes) != nil || !envelope.OK || envelope.Protocol != api.ProtocolVersion {
				return bad(out, "Expected a successful migration-preflight envelope.")
			}
			value = migration.ApplyInput{Plan: envelope.Result, WritersStopped: true}
		} else {
			var plan migration.Plan
			if strictjson.Decode(raw, &plan, api.MaxInputBytes) != nil {
				return bad(out, "Expected a reviewed migration plan.")
			}
			value = migration.ApplyInput{Plan: plan, WritersStopped: true}
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return bad(out, "Cannot encode migration input.")
	}
	return emit(out, api.New(nil, *readOnly).Call(ctx, "migration_"+sub, raw))
}
