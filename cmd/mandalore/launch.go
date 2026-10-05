package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/launch"
	"github.com/acoz-labs/mandalore/internal/strictjson"
)

const launchHelp = `Usage:
  mandalore launch configure NAME [--config FILE] --confirm-profile < entry.json
  mandalore launch list [--config FILE]
  mandalore launch schema
  mandalore launch NAME [--config FILE] [--agent codex|pi|claude-code|hermes] [--preview] [-- native arguments]

Configuration is local JSON, outside every signet. Each entry has binding,
default_agent and agents (keyed by harness). Each agent requires native_home,
native_binary, state_dir and connection_root, all canonical absolute paths.
Configure validates existing connections and pins binding identity. It claims
profiles for one signet/agent permanently; --confirm-profile confirms existing
native history belongs to that signet. Authentication remains native-owned.
Missing setup: use mandalore connection plan/apply with explicit profile paths.
List exposes local configuration; preview validates bytes without running the
agent, fetching references, or changing setup. Canon refresh belongs to hooks.
Native profile/config overrides and external session paths are refused. Resume
uses only the selected profile (Codex/Claude UUID or native profile chooser;
Pi --resume/--continue). Use a dedicated new profile for a different signet.
No aliases, shell files, credentials or portable signet content are changed.
`

func runLaunch(ctx context.Context, args []string, input io.Reader, out, errout io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, e := io.WriteString(out, launchHelp)
		if e != nil {
			return 1
		}
		return 0
	}
	sub := args[0]
	args = args[1:]
	if sub == "schema" {
		if len(args) != 0 {
			return bad(out, "launch schema accepts no arguments")
		}
		return emit(out, api.Success(launch.EntrySchema()))
	}
	name := sub
	if sub == "configure" {
		if len(args) == 0 {
			return bad(out, "configure requires a name and --confirm-profile; use launch --help")
		}
		name, args = args[0], args[1:]
	}
	f := flag.NewFlagSet("launch", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	configPath := f.String("config", "", "Machine-local launch configuration")
	var agent *string
	var preview, confirmed *bool
	if sub == "configure" {
		confirmed = f.Bool("confirm-profile", false, "Confirm dedicated profile history belongs to the selected signet")
	} else if sub != "list" {
		agent = f.String("agent", "", "Invocation-only agent override")
		preview = f.Bool("preview", false, "Inspect without executing or changing setup")
	}
	// Native arguments are accepted only after an explicit -- boundary.
	options, native := args, []string{}
	for i, a := range args {
		if a == "--" {
			options, native = args[:i], args[i+1:]
			break
		}
	}
	if f.Parse(options) != nil || f.NArg() != 0 || ((sub == "configure" || sub == "list") && len(native) > 0) {
		return bad(out, "Invalid launch arguments; use launch --help and put native arguments after --")
	}
	var e error
	if *configPath == "" {
		*configPath, e = launch.DefaultPath()
		if e != nil {
			return bad(out, "Cannot locate local configuration; provide --config FILE")
		}
	}
	if sub == "configure" {
		raw, e := io.ReadAll(io.LimitReader(input, 256<<10+1))
		if e != nil {
			return bad(out, "Cannot read launch entry")
		}
		var entry launch.Entry
		if strictjson.Decode(raw, &entry, 256<<10) != nil {
			return bad(out, "Expected a strict launch entry; inspect launch schema")
		}
		config, e := launch.Configure(*configPath, name, entry, *confirmed)
		if e != nil {
			return bad(out, e.Error())
		}
		return emit(out, api.Success(config.Entries[name]))
	}
	config, e := launch.Load(*configPath)
	if e != nil {
		return bad(out, e.Error())
	}
	if sub == "list" {
		return emit(out, api.Success(config))
	}
	plan, e := launch.Resolve(config, name, *agent, native)
	if e != nil {
		return bad(out, e.Error())
	}
	if *preview {
		return emit(out, api.Success(plan))
	}
	fmt.Fprintf(errout, "%s · %s · signet: %s (%s)\nNative hooks report synchronization and reference freshness.\n", plan.Name, plan.Agent, plan.SignetID, plan.Access)
	if e = plan.Execute(ctx); e != nil {
		return bad(errout, e.Error())
	}
	return 0
}
