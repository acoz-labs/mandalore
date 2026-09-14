package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/codex"
	memorymcp "github.com/acoz-labs/mandalore/internal/mcp"
	"github.com/acoz-labs/mandalore/internal/memory"
)

var version = "0.0.0-dev"

const help = `Mandalore — durable memory across tools

  mandalore menu [--plain]                       Guided setup, inspection and recovery
  mandalore operations                          JSON schemas and implemented operations
  mandalore version                             Runtime/protocol version
  mandalore signet create --repository DIR --name NAME --device-label LABEL
  mandalore signet bind --repository DIR --binding FILE --device-label LABEL --actor NAME
  mandalore memory recall --binding FILE [--query TEXT] [--scope-kind KIND --scope-id ID]
  mandalore memory scopes|history|journal|inspect --binding FILE [options]
  mandalore memory remember|journal-append --binding FILE < input.json
  mandalore memory git-init|checkpoint|sync-status --binding FILE
  mandalore memory sync --binding FILE [--timeout-seconds 10]
  mandalore call OPERATION --binding FILE < input.json
  mandalore mcp --binding FILE [--harness NAME] [--read-only]
  mandalore codex-memory-hook [--binding FILE]   Read-only native lifecycle JSON
  mandalore connection plan [--binary FILE] [--binding FILE] [profile options]
  mandalore connection apply < approved-plan.json
  mandalore connection doctor [profile options]
  mandalore connection repair --connection-root DIR [--apply]

Profile options: --state-dir DIR, --native-home DIR, --native-binary FILE.
Connection plan/doctor/repair preview do not activate a connection.

Memory options: --limit N, --offset N (scopes/history), --record-id ID (history),
--budget-bytes N (recall), --query TEXT (recall/journal).
Common options: --binding FILE, --harness NAME, --read-only, --help.
Binding selection: explicit file, then MANDALORE_BINDING, then platform config.
No cwd-based bank discovery. Memory saves are local; explicit sync reports delivery.
This development build supports local artifact updates, not published update discovery.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() { <-ctx.Done(); _ = os.Stdin.Close() }()
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func emit(out io.Writer, v api.Envelope) int {
	if err := json.NewEncoder(out).Encode(v); err != nil {
		return 1
	}
	return api.ExitCode(v)
}
func bad(out io.Writer, message string) int {
	return emit(out, api.Failure("input.invalid", message, false))
}

func run(ctx context.Context, args []string, input io.Reader, out, errout io.Writer) int {
	if len(args) > 0 && args[0] == "menu" {
		return runMenu(ctx, args[1:], input, out)
	}
	if len(args) > 0 && args[0] == "connection" {
		return runConnection(ctx, args[1:], input, out)
	}
	if len(args) > 0 && args[0] == "codex-memory-hook" {
		f := flag.NewFlagSet("codex-memory-hook", flag.ContinueOnError)
		f.SetOutput(io.Discard)
		path := f.String("binding", "", "Machine-local binding file")
		if err := f.Parse(args[1:]); err != nil || f.NArg() != 0 {
			_, err := io.WriteString(out, "{\"systemMessage\":\"Mandalore hook configuration is invalid; no memory was changed.\"}\n")
			if err != nil {
				return 1
			}
			return 0
		}
		if err := codex.Run(*path, input, out); err != nil {
			return 1
		}
		return 0
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(out, help)
		if err != nil {
			return 1
		}
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		if len(args) != 1 {
			return bad(out, "version takes no arguments")
		}
		return emit(out, api.Success(map[string]any{"name": "mandalore", "version": version, "protocol_version": api.ProtocolVersion,
			"codex_hook_protocol": 1, "os": runtime.GOOS, "arch": runtime.GOARCH}))
	}
	if args[0] == "operations" {
		if len(args) != 1 {
			return bad(out, "operations takes no arguments")
		}
		return emit(out, api.Success(map[string]any{"operations": api.Catalog(), "max_input_bytes": api.MaxInputBytes, "max_output_bytes": api.MaxOutputBytes, "notice": "Result schemas describe the semantic result within the common protocol envelope; setup operations are CLI-only. Mutations are not exactly-once; inspect ambiguous outcomes before retrying."}))
	}
	var name string
	var rest []string
	human := false
	switch args[0] {
	case "memory", "signet", "call":
		if len(args) < 2 {
			return bad(out, "missing operation; use --help")
		}
		if args[1] == "--help" || args[1] == "-h" {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		name = args[1]
		rest = args[2:]
		if args[0] != "call" {
			name = args[0] + "_" + strings.ReplaceAll(name, "-", "_")
			human = true
		}
	case "mcp":
		name = "mcp"
		rest = args[1:]
	default:
		return emit(out, api.Failure("operation.unknown", "Unknown command; use --help.", false))
	}
	var selected *api.Operation
	for _, op := range api.Catalog() {
		if op.Name == name {
			selected = &op
			break
		}
	}
	if selected == nil && args[0] != "mcp" {
		return emit(out, api.Failure("operation.unknown", "Unknown operation; inspect the operation catalog.", false))
	}
	failureOut := out
	if name == "mcp" {
		failureOut = errout
	}
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	path := f.String("binding", "", "Machine-local binding file")
	harnessDefault := "cli"
	if name == "mcp" {
		harnessDefault = "mcp"
	}
	harness := f.String("harness", harnessDefault, "Attribution harness label")
	readOnly := f.Bool("read-only", false, "Reject mutations")
	var repository, label, actor, displayName, query, kind, scopeID, recordID string
	var limit, offset, budget, timeout int
	if human {
		switch name {
		case "signet_create", "signet_bind":
			f.StringVar(&repository, "repository", "", "Signet path")
			f.StringVar(&label, "device-label", "", "Explicit machine label")
			if name == "signet_create" {
				f.StringVar(&displayName, "name", "", "Signet name")
			} else {
				f.StringVar(&actor, "actor", "", "Attribution actor")
			}
		case "memory_recall":
			f.StringVar(&query, "query", "", "Search query")
			f.StringVar(&kind, "scope-kind", "", "Scope kind")
			f.StringVar(&scopeID, "scope-id", "", "Stable scope ID")
			f.IntVar(&limit, "limit", 5, "Result limit")
			f.IntVar(&budget, "budget-bytes", 8192, "Semantic result budget")
		case "memory_scopes", "memory_history":
			f.IntVar(&limit, "limit", 5, "Page limit")
			f.IntVar(&offset, "offset", 0, "Page offset")
			if name == "memory_history" {
				f.StringVar(&recordID, "record-id", "", "Stable record ID")
			}
		case "memory_journal":
			f.StringVar(&query, "query", "", "Query")
			f.IntVar(&limit, "limit", 5, "Result limit")
		case "memory_sync":
			f.IntVar(&timeout, "timeout-seconds", 10, "Sync budget, 1–30 seconds")
		}
	}
	if err := f.Parse(rest); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err := io.WriteString(out, help)
			if err != nil {
				return 1
			}
			return 0
		}
		return bad(failureOut, "Invalid flags or missing values; use --help.")
	}
	if f.NArg() != 0 {
		return bad(failureOut, "Unexpected positional arguments.")
	}
	if (kind == "") != (scopeID == "") {
		return bad(failureOut, "scope-kind and scope-id must be supplied together.")
	}
	if ctx.Err() != nil {
		return emit(failureOut, api.Failure("operation.cancelled", "Operation cancelled before execution.", false))
	}
	if selected != nil && *readOnly && !selected.ReadOnly {
		return emit(out, api.Failure("operation.read_only", "Mutations are disabled for this connection.", false))
	}
	var service *memory.Service
	if name == "mcp" || selected.RequiresBinding {
		var err error
		if *path == "" {
			*path, err = binding.DefaultPath()
		}
		if err == nil {
			service, err = binding.Open(*path, *harness)
		}
		if err != nil {
			return emit(failureOut, api.Failure("binding.invalid", "Cannot open the selected binding/signet; inspect the path, version, enrolled device and pinned identity.", false))
		}
	}
	a := api.New(service, *readOnly)
	if name == "mcp" {
		reader, ok := input.(io.ReadCloser)
		if !ok {
			reader = io.NopCloser(input)
		}
		err := memorymcp.Run(ctx, a, reader, out)
		if ctx.Err() != nil {
			return emit(errout, api.Failure("operation.cancelled", "MCP server stopped.", false))
		}
		if err != nil {
			return emit(errout, api.Failure("operation.io", "MCP transport failed; inspect the connection.", false))
		}
		return 0
	}
	var raw []byte
	var value any
	if human {
		switch name {
		case "signet_create":
			value = api.CreateInput{Repository: repository, Name: displayName, DeviceLabel: label}
		case "signet_bind":
			value = api.BindInput{Repository: repository, Binding: *path, DeviceLabel: label, Actor: actor}
		case "memory_recall":
			v := api.RecallInput{Query: query, Limit: &limit, BudgetBytes: &budget}
			if kind != "" {
				v.Scope = &memory.Scope{Kind: kind, ID: scopeID}
			}
			value = v
		case "memory_scopes":
			value = api.PageInput{Offset: offset, Limit: &limit}
		case "memory_history":
			value = api.HistoryInput{RecordID: recordID, Offset: offset, Limit: &limit}
		case "memory_journal":
			value = api.JournalInput{Query: query, Limit: &limit}
		case "memory_sync":
			value = api.SyncInput{TimeoutSeconds: &timeout}
		case "memory_inspect", "memory_git_init", "memory_checkpoint", "memory_sync_status":
			value = struct{}{}
		}
	}
	if value != nil {
		raw, _ = json.Marshal(value)
	} else {
		var err error
		raw, err = io.ReadAll(io.LimitReader(input, api.MaxInputBytes+1))
		if ctx.Err() != nil {
			return emit(out, api.Failure("operation.cancelled", "Input cancelled before execution.", false))
		}
		if err != nil {
			return bad(out, "Could not read structured input.")
		}
	}
	return emit(out, a.Call(ctx, name, raw))
}
