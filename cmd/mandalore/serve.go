package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/server"
)

type serverFlags struct {
	signet, client, scope string
	installation          string
	replace               bool
}

// serverConfig parses --config and the extra flags a command allows.
func serverConfig(name string, args []string, out io.Writer, allowed ...string) (server.Config, serverFlags, int, bool) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var v serverFlags
	path := f.String("config", "", "Server configuration file")
	f.StringVar(&v.signet, "signet", "", "Signet slug")
	f.StringVar(&v.client, "client", "", "OAuth client ID")
	f.StringVar(&v.installation, "installation", "", "OAuth client installation ID")
	f.StringVar(&v.scope, "scope", "", "read or read-write")
	f.BoolVar(&v.replace, "replace", false, "Make the new passkey the only one")
	ok := f.Parse(args) == nil && f.NArg() == 0 && *path != ""
	f.Visit(func(fl *flag.Flag) {
		if fl.Name != "config" && !slices.Contains(allowed, fl.Name) {
			ok = false
		}
	})
	if !ok {
		return server.Config{}, v, bad(out, "Use "+name+" --config FILE with only its documented flags; see --help."), false
	}
	c, err := server.LoadConfig(*path)
	if err != nil {
		return c, v, bad(out, err.Error()), false
	}
	return c, v, 0, true
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

const serverUsage = "Use server status|reconcile|canon refresh|clients list|clients revoke|clients set-scope|owner enroll|install-launchd --config FILE, server uninstall-launchd, or server migrate."

func runServer(ctx context.Context, args []string, input io.Reader, out, errout io.Writer) int {
	if len(args) == 0 {
		return bad(out, serverUsage)
	}
	switch args[0] {
	case "migrate":
		return runMigrate(ctx, args[1:], out)
	case "install-launchd", "uninstall-launchd":
		return runLaunchd(ctx, args[0] == "install-launchd", args[1:], input, out, errout)
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
		c, f, code, ok := serverConfig("server reconcile", args[1:], out, "signet")
		if !ok {
			return code
		}
		slug := f.signet
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
	case "canon":
		if len(args) > 1 && args[1] == "refresh" {
			c, f, code, ok := serverConfig("server canon refresh", args[2:], out, "signet")
			if !ok {
				return code
			}
			refreshed, err := server.RefreshCanon(ctx, c, f.signet)
			if err != nil {
				return emit(out, api.Failure("canon.refresh_failed", err.Error(), false))
			}
			return emit(out, api.Success(refreshed))
		}
	case "clients":
		if len(args) > 1 {
			return runClients(args[1], args[2:], out)
		}
	case "owner":
		if len(args) > 1 && args[1] == "enroll" {
			c, f, code, ok := serverConfig("server owner enroll", args[2:], out, "replace")
			if !ok {
				return code
			}
			link, expires, err := server.EnrollOwner(c, f.replace)
			if err != nil {
				return emit(out, api.Failure("server.invalid", err.Error(), false))
			}
			return emit(out, api.Success(map[string]any{"link": link, "expires": expires.UTC().Format(time.RFC3339),
				"notice": "Open this one-time link in the browser that will hold the owner passkey. It works once, for 10 minutes."}))
		}
	}
	return bad(out, serverUsage)
}

func runClients(command string, args []string, out io.Writer) int {
	switch command {
	case "list":
		c, _, code, ok := serverConfig("server clients list", args, out)
		if !ok {
			return code
		}
		clients, err := server.Clients(c)
		if err != nil {
			return emit(out, api.Failure("server.invalid", err.Error(), false))
		}
		return emit(out, api.Success(clients))
	case "revoke":
		c, f, code, ok := serverConfig("server clients revoke", args, out, "client", "installation")
		if !ok || (f.client == "") == (f.installation == "") {
			return orUsage(out, code, ok, "Use server clients revoke --config FILE --client ID or --installation ID.")
		}
		if f.installation != "" {
			client, err := server.RevokeInstallation(c, f.installation)
			if err != nil {
				return emit(out, api.Failure("client.refused", err.Error(), false))
			}
			return emit(out, api.Success(map[string]any{"installation": f.installation, "client": client, "revoked": true,
				"notice": "Revoked this installation only. Other installations of the client keep working."}))
		}
		n, err := server.RevokeClient(c, f.client)
		if err != nil {
			return emit(out, api.Failure("client.refused", err.Error(), false))
		}
		return emit(out, api.Success(map[string]any{"client": f.client, "revoked": true, "installations": n,
			"notice": fmt.Sprintf("Revoked the client and every installation of it (%d). To revoke one, use --installation ID from server clients list.", n)}))
	case "set-scope":
		c, f, code, ok := serverConfig("server clients set-scope", args, out, "client", "installation", "scope", "signet")
		if !ok || (f.client == "") == (f.installation == "") || f.scope == "" {
			return orUsage(out, code, ok, "Use server clients set-scope --config FILE --client ID or --installation ID --scope read|read-write [--signet SLUG].")
		}
		if f.installation != "" {
			if err := server.SetInstallationScope(c, f.installation, f.signet, server.Scope(f.scope)); err != nil {
				return emit(out, api.Failure("client.refused", err.Error(), false))
			}
			return emit(out, api.Success(map[string]any{"installation": f.installation, "scope": f.scope, "signet": f.signet}))
		}
		if err := server.SetClientScope(c, f.client, f.signet, server.Scope(f.scope)); err != nil {
			return emit(out, api.Failure("client.refused", err.Error(), false))
		}
		return emit(out, api.Success(map[string]any{"client": f.client, "scope": f.scope, "signet": f.signet,
			"notice": "Changed every installation of the client. To change one, use --installation ID from server clients list."}))
	}
	return bad(out, serverUsage)
}

// orUsage returns a parse failure already emitted, or emits the usage.
func orUsage(out io.Writer, code int, parsed bool, usage string) int {
	if !parsed {
		return code
	}
	return bad(out, usage)
}
