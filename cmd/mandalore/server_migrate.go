package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/cutover"
	"github.com/acoz-labs/mandalore/internal/install"
	"github.com/acoz-labs/mandalore/internal/server"
)

const migrateUsage = "Use server migrate preflight --clone PATH [--format json|text], init --config FILE --remote URL --slug S --root DIR [--preflight FILE]... [--device-lost ID --reason TEXT]..., retire --clone PATH --server URL [--native HARNESS=HOME,BINARY]..., or verify --config FILE --slug S."

// disableLegacy is the retire step that runs native commands; tests stub it.
var disableLegacy cutover.Disabler = install.DisableLegacyConnection

func runMigrate(ctx context.Context, args []string, out io.Writer) int {
	if len(args) == 0 {
		return bad(out, migrateUsage)
	}
	f := flag.NewFlagSet("server migrate "+args[0], flag.ContinueOnError)
	f.SetOutput(io.Discard)
	clone := f.String("clone", "", "1.x clone")
	format := f.String("format", "json", "json or text")
	config := f.String("config", "", "Server configuration file")
	remote := f.String("remote", "", "Signet Git remote")
	slug := f.String("slug", "", "Signet slug")
	root := f.String("root", "", "New server checkout")
	serverURL := f.String("server", "", "2.0 server URL")
	var preflights []string
	var lost []server.LostDevice
	var profiles []cutover.NativeProfile
	f.Func("preflight", "Preflight receipt file", func(v string) error { preflights = append(preflights, v); return nil })
	f.Func("device-lost", "Enrolled device marked lost", func(v string) error {
		lost = append(lost, server.LostDevice{ID: v})
		return nil
	})
	f.Func("reason", "Why the preceding --device-lost is lost", func(v string) error {
		if len(lost) == 0 || lost[len(lost)-1].Reason != "" {
			return errors.New("--reason follows its --device-lost")
		}
		lost[len(lost)-1].Reason = v
		return nil
	})
	f.Func("native", "HARNESS=HOME,BINARY", func(v string) error {
		harness, paths, _ := strings.Cut(v, "=")
		home, binary, _ := strings.Cut(paths, ",")
		if !slices.Contains(install.LegacyHarnesses, harness) || !filepath.IsAbs(home) || !filepath.IsAbs(binary) {
			return errors.New("--native takes HARNESS=HOME,BINARY with absolute paths")
		}
		profiles = append(profiles, cutover.NativeProfile{Harness: harness, Home: home, Binary: binary})
		return nil
	})
	allowed := map[string][]string{
		"preflight": {"clone", "format"},
		"init":      {"config", "remote", "slug", "root", "preflight", "device-lost", "reason"},
		"retire":    {"clone", "server", "native"},
		"verify":    {"config", "slug"},
	}[args[0]]
	ok := allowed != nil && f.Parse(args[1:]) == nil && f.NArg() == 0
	f.Visit(func(fl *flag.Flag) { ok = ok && slices.Contains(allowed, fl.Name) })
	if !ok {
		return bad(out, migrateUsage)
	}
	switch args[0] {
	case "preflight":
		if *clone == "" || (*format != "json" && *format != "text") {
			return bad(out, migrateUsage)
		}
		return runPreflight(ctx, *clone, *format, out)
	case "init":
		if *config == "" || *remote == "" || *slug == "" || *root == "" || slices.ContainsFunc(lost, func(l server.LostDevice) bool { return l.Reason == "" }) {
			return bad(out, "Use server migrate init --config FILE --remote URL --slug S --root DIR, with a --reason after each --device-lost.")
		}
		m, err := server.Init(ctx, server.MigrateInit{ConfigPath: *config, Remote: *remote, Slug: *slug, Root: *root, Preflights: preflights, Lost: lost})
		if err != nil {
			return emit(out, api.Failure("migrate.refused", "Init refused: "+err.Error()+".", false))
		}
		return emit(out, api.Success(m))
	case "retire":
		if *clone == "" || *serverURL == "" {
			return bad(out, "Use server migrate retire --clone PATH --server URL [--native HARNESS=HOME,BINARY]...")
		}
		r, err := cutover.Retire(ctx, *clone, *serverURL, profiles, disableLegacy)
		if err != nil {
			return emit(out, api.Failure("migrate.refused", "Retire stopped: "+err.Error()+".", false))
		}
		return emit(out, api.Success(r))
	case "verify":
		if *config == "" || *slug == "" {
			return bad(out, "Use server migrate verify --config FILE --slug S.")
		}
		c, err := server.LoadConfig(*config)
		if err != nil {
			return bad(out, err.Error())
		}
		v, err := server.Verify(ctx, c, *slug)
		if err != nil {
			return emit(out, api.Failure("migrate.unverified", "Verify failed: "+err.Error()+".", false))
		}
		return emit(out, api.Success(v))
	}
	return bad(out, migrateUsage)
}

// runPreflight exits 0 when the clone is clean, 1 when blocked and 2 when it
// cannot be inspected. The JSON output is the receipt init accepts.
func runPreflight(ctx context.Context, clone, format string, out io.Writer) int {
	path, err := binding.DefaultPath()
	if err != nil {
		path = ""
	}
	r, err := cutover.Preflight(ctx, clone, path)
	if err != nil {
		if format == "text" {
			fmt.Fprintf(out, "Preflight error: %s.\n", err)
		} else {
			_ = json.NewEncoder(out).Encode(api.Failure("preflight.error", "Preflight could not inspect the clone: "+err.Error()+".", false))
		}
		return 2
	}
	if format == "text" {
		device := r.DeviceID
		if device == "" {
			device = "unknown (no binding names this clone)"
		}
		verdict := "clean: safe to retire"
		if !r.Clean {
			verdict = "blocked"
		}
		fmt.Fprintf(out, "Signet %s, device %s: %s\n", r.SignetID, device, verdict)
		for _, b := range r.Blockers {
			fmt.Fprintf(out, "  - %s\n", b)
		}
	} else if err := json.NewEncoder(out).Encode(api.Success(r)); err != nil {
		return 2
	}
	if !r.Clean {
		return 1
	}
	return 0
}
