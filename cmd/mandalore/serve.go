package main

import (
	"context"
	"flag"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/server"
)

func serverConfig(name string, args []string, out io.Writer) (server.Config, string, int, bool) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	path := f.String("config", "", "Server configuration file")
	slug := f.String("signet", "", "Signet slug")
	if f.Parse(args) != nil || f.NArg() != 0 || *path == "" || (*slug != "" && name != "server reconcile") {
		return server.Config{}, "", bad(out, "Use --config FILE (and --signet SLUG for reconcile); see --help."), false
	}
	c, err := server.LoadConfig(*path)
	if err != nil {
		return c, "", bad(out, err.Error()), false
	}
	return c, *slug, 0, true
}

// runServe serves until interrupted. Logs go to errout; stdout carries only
// a machine-readable failure.
func runServe(ctx context.Context, args []string, out, errout io.Writer) int {
	c, _, code, ok := serverConfig("serve", args, out)
	if !ok {
		return code
	}
	s, err := server.New(c, errout)
	if err != nil {
		return emit(out, api.Failure("server.invalid", err.Error(), false))
	}
	if err := s.Run(ctx); err != nil {
		return emit(out, api.Failure("server.failed", "Server stopped unexpectedly; inspect the listen address.", false))
	}
	return 0
}

func runServer(ctx context.Context, args []string, out io.Writer) int {
	if len(args) == 0 {
		return bad(out, "Use server status|reconcile --config FILE.")
	}
	switch args[0] {
	case "status":
		c, _, code, ok := serverConfig("server status", args[1:], out)
		if !ok {
			return code
		}
		statuses, err := server.Status(ctx, c)
		if err != nil {
			return emit(out, api.Failure("server.invalid", err.Error(), false))
		}
		return emit(out, api.Success(statuses))
	case "reconcile":
		c, slug, code, ok := serverConfig("server reconcile", args[1:], out)
		if !ok {
			return code
		}
		if slug == "" && len(c.Signets) == 1 {
			slug = c.Signets[0].Slug
		}
		if slug == "" {
			return bad(out, "Several signets are configured; choose one with --signet SLUG.")
		}
		status, err := server.Reconcile(ctx, c, slug)
		if err != nil {
			return emit(out, api.Failure("reconcile.refused", "Reconcile refused: "+err.Error()+". Nothing was merged or discarded; see docs/server.md.", false))
		}
		status.Notice += " Restart the server to serve this signet."
		return emit(out, api.Success(map[string]any{"signet": slug, "status": status}))
	}
	return bad(out, "Use server status|reconcile --config FILE.")
}
