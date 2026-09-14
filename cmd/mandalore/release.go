package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/distribution"
)

func runRelease(ctx context.Context, args []string, out io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 0 || (args[0] != "inspect" && args[0] != "plan") {
		return bad(out, "Choose release inspect or plan; activation and promotion are still under development.")
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
	var input any = in
	if args[0] == "plan" {
		selection.Version, selection.Candidate = in.Version, in.Candidate
		input = selection
	}
	raw, _ := json.Marshal(input)
	return emit(out, api.New(nil, *readOnly).Call(ctx, "release_"+args[0], raw))
}
