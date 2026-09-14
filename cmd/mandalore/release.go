package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/distribution"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

func runRelease(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	if len(args) > 0 && args[0] == "apply" {
		return runReleaseApply(ctx, args[1:], input, out)
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 0 || (args[0] != "inspect" && args[0] != "plan") {
		return bad(out, "Choose release inspect, plan or apply; interactive installation and promotion are still under development.")
	}
	f := flag.NewFlagSet("release "+args[0], flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var in api.ReleaseInspectInput
	f.StringVar(&in.Version, "version", "", "Explicit published version; omitted selects latest stable")
	f.StringVar(&in.Candidate, "candidate", "", "Local candidate directory; no network")
	var selection distribution.InstallOptions
	if args[0] == "plan" {
		f.StringVar(&selection.Prefix, "prefix", "", "Required user-owned installation prefix")
		f.StringVar(&selection.Retained, "retained", "", "Retained manifest SHA-256 for compatible rollback; no network")
	}
	readOnly := f.Bool("read-only", false, "Keep this task read-only")
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(out, "Invalid release flags or missing values; use --help.")
	}
	if f.NArg() != 0 {
		return bad(out, "Unexpected release arguments.")
	}
	var value any = in
	if args[0] == "plan" {
		selection.Version, selection.Candidate = in.Version, in.Candidate
		value = selection
	}
	raw, _ := json.Marshal(value)
	return emit(out, api.New(nil, *readOnly).Call(ctx, "release_"+args[0], raw))
}

func runReleaseApply(ctx context.Context, args []string, input io.Reader, out io.Writer) int {
	f := flag.NewFlagSet("release apply", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	readOnly := f.Bool("read-only", false, "Reject mutations before reading a plan")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(out, "Invalid release apply flags; use --help.")
	}
	if f.NArg() != 0 {
		return bad(out, "Release apply reads its reviewed plan from stdin.")
	}
	if *readOnly {
		return emit(out, api.Failure("operation.read_only", "Mutations are disabled for this task.", false))
	}
	if ctx.Err() != nil {
		return emit(out, api.Failure("operation.cancelled", "Cancelled before installation work.", false))
	}
	// Human CLI accepts its own pretty-printed envelope (bounded at 64 KiB).
	// The extracted raw plan still obeys the typed operation's 32 KiB budget.
	raw, err := io.ReadAll(io.LimitReader(input, 2*distribution.MaxInstallPlanBytes+1))
	if err != nil {
		return bad(out, "Could not read the reviewed installation plan.")
	}
	raw, err = unwrapReleasePlan(raw)
	if err != nil {
		return bad(out, "Expected a valid plan or successful installation-plan envelope.")
	}
	if _, err := distribution.ParseInstallPlan(raw); err != nil {
		return bad(out, "Installation plan is invalid or exceeds its input budget.")
	}
	return emit(out, api.New(nil, false).Call(ctx, "release_apply", raw))
}

func unwrapReleasePlan(raw []byte) ([]byte, error) {
	var object map[string]any
	if err := strictjson.Decode(raw, &object, 2*distribution.MaxInstallPlanBytes); err != nil {
		return nil, err
	}
	if _, wrapped := object["protocol_version"]; wrapped {
		var envelope struct {
			Protocol int                      `json:"protocol_version"`
			OK       bool                     `json:"ok"`
			Result   distribution.InstallPlan `json:"result"`
		}
		if err := strictjson.Decode(raw, &envelope, 2*distribution.MaxInstallPlanBytes); err != nil || !envelope.OK || envelope.Protocol != api.ProtocolVersion {
			return nil, strictjson.ErrInvalid
		}
		raw, _ = json.Marshal(envelope.Result)
	} else {
		// Re-encode the typed object, not map[string]any: directory device/inode
		// identifiers can exceed float64's exact integer range on supported hosts.
		var plan distribution.InstallPlan
		if err := strictjson.Decode(raw, &plan, 2*distribution.MaxInstallPlanBytes); err != nil {
			return nil, err
		}
		raw, _ = json.Marshal(plan)
	}
	return raw, nil
}
