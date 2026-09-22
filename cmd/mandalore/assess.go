package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/readiness"
)

func runAssessment(ctx context.Context, args []string, out io.Writer) int {
	f := flag.NewFlagSet("connection assess", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var in readiness.Input
	f.StringVar(&in.Harness, "harness", "", "Native harness: codex, pi or claude-code (required)")
	f.StringVar(&in.StateDir, "state-dir", "", "Installation state directory")
	f.StringVar(&in.NativeHome, "native-home", "", "Selected native profile directory")
	f.StringVar(&in.NativeBinary, "native-binary", "", "Selected native executable; inspected, not executed")
	f.StringVar(&in.Binding, "binding", "", "Machine-local signet binding")
	f.StringVar(&in.ConnectionRoot, "connection-root", "", "Explicit retained generation; none selected by default")
	f.BoolVar(&in.IncludePrompt, "prompt", false, "Include sanitized follow-up guidance; never execute it")
	_ = f.Bool("read-only", false, "Assessment is always non-executing and read-only")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(out, "Invalid assessment flags or missing values; use --help.")
	}
	if f.NArg() != 0 {
		return bad(out, "Unexpected assessment arguments.")
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return bad(out, "Cannot encode assessment selection.")
	}
	return emit(out, api.New(nil, true).Call(ctx, "connection_assess", raw))
}
