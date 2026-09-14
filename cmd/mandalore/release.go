package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
)

func runRelease(ctx context.Context, args []string, out io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 0 || args[0] != "inspect" {
		return bad(out, "Choose release inspect; installation and promotion are still under development.")
	}
	f := flag.NewFlagSet("release inspect", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var in api.ReleaseInspectInput
	f.StringVar(&in.Version, "version", "", "Explicit published version; omitted selects latest stable")
	f.StringVar(&in.Candidate, "candidate", "", "Local candidate directory; no network")
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
	raw, _ := json.Marshal(in)
	return emit(out, api.New(nil, *readOnly).Call(ctx, "release_inspect", raw))
}
