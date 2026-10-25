package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/acoz-labs/mandalore/internal/api"
	"github.com/acoz-labs/mandalore/internal/binding"
	"github.com/acoz-labs/mandalore/internal/foundlings"
	"github.com/acoz-labs/mandalore/internal/memory"
)

var version = "0.0.0-dev"
var sourceCommit = "" // Stamped only by the pinned exact-source release builder.

const help = `Mandalore — durable memory across tools

  mandalore serve --config FILE                  Single-writer MCP server for configured signets
  mandalore server status --config FILE          Per-signet head, backup, remote and canon references
  mandalore server reconcile --config FILE [--signet SLUG]
                                                 Fetch, validate and fast-forward a remote-ahead signet
  mandalore server owner enroll --config FILE [--replace]
                                                 Print a one-time, 10-minute owner passkey enrollment link
  mandalore server clients list --config FILE    Approved and pending OAuth clients with their grants
  mandalore server clients revoke --config FILE --client ID
  mandalore server clients set-scope --config FILE --client ID --scope read|read-write [--signet SLUG]
  mandalore server canon refresh --config FILE [--signet SLUG]
                                                 Refresh configured canon references now
  mandalore server migrate preflight --clone PATH [--format json|text]
                                                 Read-only 1.x clone check: exit 0 clean, 1 blocked, 2 error
  mandalore server migrate init --config FILE --remote URL --slug S --root DIR
                                 [--preflight FILE]... [--device-lost ID --reason TEXT]...
                                                 Fresh server clone; needs every enrolled device preflighted or lost
  mandalore server migrate retire --clone PATH --server URL [--native HARNESS=HOME,BINARY]...
                                                 Disable 1.x connections, write RETIRED.md, make the clone read-only
  mandalore server migrate verify --config FILE --slug S
                                                 Server stopped: head equals remote, canary write and withdraw push
  mandalore server install-launchd --config FILE [--label LABEL]
                                                 macOS: print the KeepAlive agent plist, ask, then write and load it
  mandalore server uninstall-launchd [--label LABEL]
                                                 macOS: ask, then unload the agent and remove its plist
  mandalore menu [--plain]                       Guided setup, inspection and recovery
  mandalore operations                          JSON schemas and implemented operations
  mandalore version                             Runtime/protocol version
  mandalore release inspect [--version VERSION | --candidate DIR]
  mandalore release plan [--version VERSION | --candidate DIR | --retained SHA256] --prefix DIR
  mandalore release apply < reviewed-plan.json
  mandalore release apply < PREFIX/lib/mandalore/pending.json
  mandalore release install [--version VERSION | --candidate DIR | --retained SHA256] [--prefix DIR] [--plain]
  mandalore signet create --repository DIR --name NAME --device-label LABEL
  mandalore signet bind --repository DIR --binding FILE --device-label LABEL --actor NAME
  mandalore memory recall --binding FILE [--query TEXT] [--scope-kind KIND --scope-id ID]
  mandalore memory scopes|history|journal|inspect --binding FILE [options]
  mandalore memory visibility-history --binding FILE --record-id ID [--offset N] [--limit N]
  mandalore memory withheld --binding FILE [--query TEXT] [--scope-kind KIND --scope-id ID] [--offset N] [--limit N]
  mandalore memory withdraw|restore --binding FILE < reviewed-heads.json
  mandalore call signet_upgrade_preview|signet_upgrade_apply|signet_upgrade_recover < input.json
  mandalore call retention_preview --read-only < explicit-policy-selection.json
  mandalore memory remember|journal-append --binding FILE < input.json
  mandalore memory git-init|checkpoint --binding FILE
  mandalore foundling list|inspect|history|search|read --binding FILE [options]
  mandalore foundling preview|register|connect|disconnect|promote --binding FILE < input.json
  mandalore foundling status|canon-scopes|canon-recall|canon-heads|canon-promote --binding FILE < input.json
  mandalore foundling refresh --binding FILE < session.json
  mandalore call OPERATION --binding FILE < input.json
  mandalore migration preflight --source DIR --output NEW-DIR --device-label LABEL --actor NAME
  mandalore migration apply --writers-stopped < reviewed-preflight.json
  mandalore export preview [--binding PATH] < selection.json
  mandalore export apply [--read-only] < reviewed-preview.json

Migration preflight accepts optional --legacy-binding FILE and explicit
--native-home DIR --native-binary FILE for native inventory. No implicit defaults.
Migration does not activate a writer or copy Git history/configuration.

Memory options: --limit N, --offset N (scopes/history), --record-id ID (history),
--budget-bytes N (recall), --query TEXT (recall/journal).
Foundling options: --foundling-id ID, --query TEXT (search), --limit N (list/history/search),
--offset N (list/history/search/read), --registration-id ID (search/read),
--excerpt-bytes N --budget-bytes N (search), --locator PATH --limit-bytes N (read).
Search offsets select document ranks; read offsets select UTF-8 bytes.
Foundling setup is CLI-only. Legacy uses pinned historical text; canon uses server-refreshed signet evidence.
Canon operations take structured JSON; session_id is optional and ignored. The server
refreshes canon (mandalore server canon refresh refreshes now); canon reads never fetch.
Common options: --binding FILE, --harness NAME, --read-only, --help.
Optional connection guards: --binding-sha256 SHA256 --signet-id ID (both required).
Binding selection: explicit file, then MANDALORE_BINDING, then platform config.
No cwd-based bank discovery. Local saves and delivery receipts are distinct.
Release inspection/planning are read-only; apply changes only the selected CLI installation.
Release install/menu use default-No previews.
Publication uses separate maintainer acceptance/release gates; local builds are not releases.
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
	if len(args) > 0 && args[0] == "serve" {
		return runServe(ctx, args[1:], out, errout)
	}
	if len(args) > 0 && args[0] == "server" {
		return runServer(ctx, args[1:], input, out, errout)
	}
	if len(args) > 0 && args[0] == "export" {
		return runExport(ctx, args[1:], input, out)
	}
	if len(args) > 0 && args[0] == "release" {
		return runRelease(ctx, args[1:], input, out)
	}
	if len(args) > 0 && args[0] == "migration" {
		return runMigration(ctx, args[1:], input, out)
	}
	if len(args) > 0 && args[0] == "menu" {
		return runMenu(ctx, args[1:], input, out)
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
		info, err := runtimeMetadata()
		if err != nil {
			return emit(out, api.Failure("runtime.invalid", "Cannot inspect embedded runtime metadata.", false))
		}
		return emit(out, api.Success(info))
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
	case "memory", "signet", "foundling", "call":
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
	if selected == nil {
		return emit(out, api.Failure("operation.unknown", "Unknown operation; inspect the operation catalog.", false))
	}
	failureOut := out
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	path := f.String("binding", "", "Machine-local binding file")
	harness := f.String("harness", "cli", "Attribution harness label")
	readOnly := f.Bool("read-only", false, "Reject mutations")
	bindingSHA := f.String("binding-sha256", "", "Expected binding bytes SHA-256; requires signet-id")
	signetID := f.String("signet-id", "", "Expected signet identity; requires binding-sha256")
	var repository, label, actor, displayName, query, kind, scopeID, recordID string
	var foundlingID, registrationID, locator string
	var limit, offset, budget, excerptBytes int
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
		case "memory_scopes", "memory_history", "memory_visibility_history":
			f.IntVar(&limit, "limit", 5, "Page limit")
			f.IntVar(&offset, "offset", 0, "Page offset")
			if name == "memory_history" || name == "memory_visibility_history" {
				f.StringVar(&recordID, "record-id", "", "Stable record ID")
			}
		case "memory_journal":
			f.StringVar(&query, "query", "", "Query")
			f.IntVar(&limit, "limit", 5, "Result limit")
		case "memory_withheld":
			f.StringVar(&query, "query", "", "Current-head summary/ID query for explicit historical inspection")
			f.StringVar(&kind, "scope-kind", "", "Scope kind")
			f.StringVar(&scopeID, "scope-id", "", "Stable scope ID")
			f.IntVar(&limit, "limit", 5, "Page limit")
			f.IntVar(&offset, "offset", 0, "Page offset")
		case "foundling_list", "foundling_history":
			f.IntVar(&limit, "limit", 5, "Page limit")
			f.IntVar(&offset, "offset", 0, "Page offset")
			if name == "foundling_history" {
				f.StringVar(&foundlingID, "foundling-id", "", "Explicit reference ID")
			}
		case "foundling_inspect", "foundling_search", "foundling_read":
			f.StringVar(&foundlingID, "foundling-id", "", "Explicit reference ID")
			if name == "foundling_search" {
				f.StringVar(&query, "query", "", "Literal search terms")
				f.IntVar(&limit, "limit", foundlings.DefaultSearchLimit, "Result limit")
				f.StringVar(&registrationID, "registration-id", "", "Exact registration for continuation")
				f.IntVar(&offset, "offset", 0, "Document-rank offset")
				f.IntVar(&excerptBytes, "excerpt-bytes", foundlings.DefaultSearchExcerptBytes, "Preview content bytes")
				f.IntVar(&budget, "budget-bytes", foundlings.DefaultSearchBudgetBytes, "Serialized search-result bytes")
			}
			if name == "foundling_read" {
				f.StringVar(&registrationID, "registration-id", "", "Exact registration revision")
				f.StringVar(&locator, "locator", "", "Relative document locator")
				f.IntVar(&offset, "offset", 0, "UTF-8 byte offset")
				f.IntVar(&budget, "limit-bytes", foundlings.DefaultReadBytes, "Excerpt content bytes")
			}
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
	if (name == "export_preview" || name == "export_apply") && *path != "" {
		return bad(failureOut, "Typed export operations take the binding inside their JSON input, not --binding.")
	}
	if strings.HasPrefix(name, "signet_upgrade_") && *path != "" {
		return bad(failureOut, "Upgrade operations take their explicit binding inside reviewed JSON input, not --binding.")
	}
	if name == "retention_preview" && *path != "" {
		return bad(failureOut, "Retention preview takes its explicit binding inside JSON input, not --binding.")
	}
	guard := binding.Guard{SHA256: *bindingSHA, SignetID: *signetID}
	guardRequested := false
	f.Visit(func(option *flag.Flag) {
		if option.Name == "binding-sha256" || option.Name == "signet-id" {
			guardRequested = true
		}
	})
	if guardRequested && guard == (binding.Guard{}) {
		return bad(failureOut, "Connection guard values must not be empty.")
	}
	if err := guard.Validate(); err != nil {
		return bad(failureOut, "Connection guards require a lowercase SHA-256 and a valid signet ID together.")
	}
	if guard != (binding.Guard{}) && !selected.RequiresBinding {
		return bad(failureOut, "Connection guards apply only to bound operations.")
	}
	if (kind == "") != (scopeID == "") {
		return bad(failureOut, "scope-kind and scope-id must be supplied together.")
	}
	if ctx.Err() != nil {
		return emit(failureOut, api.Failure("operation.cancelled", "Operation cancelled before execution.", false))
	}
	if *readOnly && !selected.ReadOnly {
		return emit(out, api.Failure("operation.read_only", "Mutations are disabled for this connection.", false))
	}
	var service *memory.Service
	if selected.RequiresBinding {
		var err error
		if *path == "" {
			*path, err = binding.DefaultPath()
		}
		if err == nil {
			service, err = binding.OpenGuarded(*path, *harness, guard)
		}
		if err != nil {
			return emit(failureOut, api.Failure("binding.invalid", "Cannot open the selected binding/signet; inspect the path, version, enrolled device and pinned identity.", false))
		}
	}
	a := api.New(service, *readOnly)
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
		case "memory_withheld":
			v := api.WithheldInput{Query: query, Offset: offset, Limit: &limit}
			if kind != "" {
				v.Scope = &memory.Scope{Kind: kind, ID: scopeID}
			}
			value = v
		case "memory_history", "memory_visibility_history":
			value = api.HistoryInput{RecordID: recordID, Offset: offset, Limit: &limit}
		case "memory_journal":
			value = api.JournalInput{Query: query, Limit: &limit}
		case "memory_inspect", "memory_git_init", "memory_checkpoint":
			value = struct{}{}
		case "foundling_list":
			value = api.PageInput{Offset: offset, Limit: &limit}
		case "foundling_history":
			value = api.FoundlingHistoryInput{FoundlingID: foundlingID, Offset: offset, Limit: &limit}
		case "foundling_inspect":
			value = api.FoundlingSelector{FoundlingID: foundlingID}
		case "foundling_search":
			value = api.FoundlingSearchInput{FoundlingID: foundlingID, RegistrationID: registrationID, Query: query, Limit: &limit, Offset: offset, ExcerptBytes: &excerptBytes, BudgetBytes: &budget}
		case "foundling_read":
			value = api.FoundlingReadInput{FoundlingID: foundlingID, RegistrationID: registrationID, Locator: locator, Offset: offset, Limit: &budget}
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
