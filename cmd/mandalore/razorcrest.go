package main

import (
	"context"
	"flag"
	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/razorcrest"
	"github.com/acoz-labs/mandalore/internal/strictjson"
	"io"
	"os"
)

func runRazorCrest(ctx context.Context, args []string, out io.Writer) int {
	f := flag.NewFlagSet("razor-crest", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	config := f.String("config", "", "Explicit service configuration")
	if len(args) == 0 || args[0] != "serve" {
		return bad(out, "Use razor-crest serve --config FILE.")
	}
	if f.Parse(args[1:]) != nil || f.NArg() != 0 || *config == "" {
		return bad(out, "Explicit Razor Crest config required.")
	}
	file, e := os.Open(*config)
	if e != nil {
		return bad(out, "Service configuration unavailable.")
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, 65537))
	if e != nil {
		return bad(out, "Service configuration unavailable.")
	}
	var c razorcrest.Config
	if strictjson.Decode(raw, &c, 65536) != nil {
		return bad(out, "Invalid service configuration.")
	}
	service, e := razorcrest.New(c)
	if e != nil {
		return bad(out, "Invalid service configuration or binding; inspect private configuration.")
	}
	if e = service.Run(ctx); e != nil {
		return emit(out, api.Failure("service.failed", "Razor Crest stopped unexpectedly.", false))
	}
	return 0
}
