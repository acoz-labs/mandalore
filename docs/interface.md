# CLI and local MCP

The development CLI and stdio server use one typed operation dispatcher. They
do not supply a model, replace native agent identity, install a plugin, discover
memory from the working directory, or synchronize implicitly. Real explicit Git
operations are described in [synchronization](synchronization.md); native lifecycle
and release acceptance remain outstanding. No live migration is implied.

## Try a synthetic signet

Build with the pinned toolchain:

```sh
mise exec go@1.26.4 -- go build -o ./mandalore ./cmd/mandalore
./mandalore --help
./mandalore operations
```

Use new, explicit paths of your choosing; the following are placeholders, not
existing configuration:

```sh
./mandalore signet create --repository /example/signet --name Example --device-label Test
./mandalore signet bind --repository /example/signet --binding /example/local/binding.json --device-label Test --actor Example
./mandalore memory inspect --binding /example/local/binding.json
./mandalore memory remember --binding /example/local/binding.json < confirmed-memory.json
./mandalore memory recall --binding /example/local/binding.json --query project
./mandalore mcp --binding /example/local/binding.json --harness example-agent
```

`confirmed-memory.json` can contain:

```json
{"kind":"fact","summary":"Fictional project","body":"The project is Copper Finch.","basis":"user-direction","reason":"The user confirmed this name."}
```

Record prose enters through structured stdin, not shell interpolation. Do not
store secrets or raw transcripts. A correction supplies the returned `record_id`
and the predecessor revision ID in `supersedes`, plus a new body and reason.
`memory history --record-id ID` preserves original and correction provenance.
Omitting scope means only this signet's global scope, not every project; use
`memory scopes` to discover stored IDs before selecting a project/account/task.

## Machine-ready contract

`mandalore operations` lists versioned input/result JSON schemas, read/write and
idempotency annotations. `mandalore call OPERATION --binding FILE < input.json`
uses the same decoder and methods as the human commands and MCP tools.

| Operation | Human command | MCP |
| --- | --- | --- |
| `signet_create`, `signet_bind` | `signet create`, `signet bind` | Not exposed |
| `memory_recall`, `memory_scopes`, `memory_history` | `memory recall`, `scopes`, `history` | Bound signet only |
| `memory_journal`, `memory_inspect` | `memory journal`, `inspect` | Bound signet only |
| `memory_remember`, `memory_journal_append` | `memory remember`, `journal-append` | Bound signet only |
| `memory_git_init`, `memory_checkpoint`, `memory_sync`, `memory_sync_status` | `memory git-init`, `checkpoint`, `sync`, `sync-status` | Bound signet only |

Every operation returns `{protocol_version, ok, result}` or
`{protocol_version, ok, error}`. The current protocol is 1. The catalog describes
the semantic result schema; MCP advertises its complete envelope schema and
returns the same envelope in structured content and a text fallback. Tool errors
also set MCP `isError`. MCP protocol/framing errors remain SDK transport errors,
not operation results. Startup diagnostics use stderr, never protocol stdout.

Strict object input rejects duplicate, unknown or incorrectly cased fields,
wrong types, missing required fields, trailing objects and excessive nesting.
Arguments are at most 32 KiB. Semantic envelopes are at most 64 KiB; recall's
default result budget is 8192 bytes (range 1024–32768). Default page/result limit
is 5 (range 1–50); zero is invalid, not an implicit default. MCP newline input
frames have a separate 256 KiB limit. Schemas and duplicated text/structured
content add protocol overhead outside the semantic result budget.

`--read-only` refuses mutations before reading their stdin or touching a binding.
Reads, discovery and startup do not journal, repair, enroll or sync. A running
MCP server retains one binding; changing its local file requires restarting the
server. Reads load current local records; no persistent model context is refreshed
merely by updating those records.

## Binding and provenance

Selection is explicit `--binding`, then absolute `MANDALORE_BINDING`, then
`mandalore/binding.json` under Go's platform user-config directory. No cwd search
occurs. On macOS the default base is the user's Library/Application Support;
on Linux it is XDG_CONFIG_HOME or the user's .config directory. Keep environment
selection absolute so changing project directory cannot select a different bank.

A version-1 binding pins the canonical signet path/ID, enrolled device ID and
actor. It must be a regular bounded JSON file outside the signet, including through
symlink ancestors. Creation refuses existing destinations. Reads reject unknown
fields/versions and changed bank identity. Labels and actors are user-supplied;
hostnames are not discovered. Harness attribution comes from `--harness` (default
`cli`, or `mcp` for the server), not from automatic native-session detection.

Create and bind are deliberately separate. Create writes a bootstrap device;
each new binding enrolls a fresh opaque device ID, including another binding on
the same machine. Thus a label is descriptive, not a global hardware identity.
Old authorship remains intact. Binding publication failure may leave an unused
device record; preserve it and inspect instead of deleting history. Creating a
signet does **not** initialize Git in this slice; its receipt says so explicitly.

## Failure and retry

| Code | CLI exit | Meaning |
| --- | --- | --- |
| `input.invalid`, `operation.unknown`, `operation.read_only` | 2 | Correct input/selection or honor the read-only boundary |
| `store.busy` | 3 | Another writer owns the lock; retry after it finishes |
| `operation.cancelled` | 130 | Cancelled before execution, or an interrupted read/server |
| `binding.invalid`, `store.invalid` | 1 | Inspect selected configuration/data; no automatic repair |
| `operation.io`, `output.invalid` | 1 | Inspect I/O/output failure and any possible partial write |
| `sync.failed` | 1 | Inspect the returned sync phase/head and local/remote history before retrying |

Errors include `retryable`, `write_may_have_occurred` and `inspect_before_retry`.
Stopped synchronization may also include `sync_status`; its wrapped cancellation
does not erase evidence of an earlier checkpoint or possible delivery.
Sensitive input values and raw filesystem errors are not reflected into error
messages. Conservative write ambiguity is intentional: publication followed by
an I/O or response failure is not a proven rollback. A lost process/transport
response may provide no envelope at all; inspect history/journal before retrying.

Successful mutation receipts identify immutable records and say
`durable_locally: true`, `synchronization: "not-requested"`. They do not promise
remote delivery or exactly-once writes. Cancellation is checked at operation
boundaries; synchronous filesystem work already underway is not transactionally
rolled back or instantaneously interruptible. A published write retains its
receipt when available. SIGINT/SIGTERM stop the process; clean MCP stdin EOF
closes the server normally.

## Verification boundary

`mandalore codex-memory-hook [--binding FILE]` is a separate, read-only native
adapter, not a shared memory operation. It consumes native event JSON and emits
Codex hook context/warnings rather than the CLI envelope. Invalid configuration
or unavailable memory produces a nonblocking warning. See the
[Codex integration](../plugins/codex/README.md) for budgets, installation and
native verification boundaries. It does not belong in the MCP tool catalog.

Tests cover strict schemas, binding isolation/replacement, CLI/MCP equivalence,
compiled executable create/correction/history, unrelated cwd, explicit binding
precedence, no-write hashes, real stdio, EOF and interruption. Full race tests,
vet and four platform builds run in `bin/ci`. Cross-builds are not native runtime
acceptance. These tests use synthetic local data and an SDK client, not a model
conversation or installed Codex plugin. Those remain #6/#10.

## Development connection management

The CLI embeds the public Codex package. `connection plan` previews explicit
runtime, binding and native profile selection without executing binaries or
writing files. `connection apply` accepts that successful JSON envelope or its
raw plan object on stdin and revalidates inputs before activation. A plan
identifies the runtime, native executable, binding and embedded package bytes;
the selected runtime must also pass machine/protocol and read-only hook probes.
Checksums establish identity, not publisher trust.

`connection doctor` returns structural checks and explicit `not-tested` entries
for native login, hook trust, live MCP, remote freshness and active-session
context. An unhealthy report has a nonzero exit and `error.connection_report`.
`connection repair --connection-root DIR` only previews a fresh generation;
`--apply` explicitly applies it. Edited/unknown state and changed bindings are
preserved and require inspection, not blind overwrites.

Profile flags are `--state-dir`, `--native-home` and `--native-binary`. Defaults
are the platform config directory's `mandalore/installation`, existing
`CODEX_HOME` (otherwise the native home directory's `.codex`) and Codex on PATH.
Plan additionally accepts `--binary` (default current CLI) and `--binding`
(the ordinary explicit/environment/platform binding selection). These paths
must be separate from Git-backed memory. No shell configuration is written.
Generated local defaults still yield to explicit `MANDALORE_BIN/BINDING` values;
doctor flags an override that selects a different connection.

The equivalent catalog operations are `connection_plan`, `connection_apply`,
`connection_doctor` and `connection_repair_plan`. All are CLI-only, not bound
memory MCP tools. The shared API takes raw typed objects; the human apply command
also accepts its own plan envelope for convenient preview/apply workflows.
Read-only mode refuses apply. Connection failures use `connection.failed`
(exit 1), retain `connection_result` when application started, and identify the
last completed phase. Cancellation can retain that receipt too. Do not infer
rollback from a nonzero exit or repeat an ambiguous native mutation blindly.
Installation result/report schemas are intentionally absent from bound memory
MCP error schemas, keeping machine-administration details out of agent context.

Runtime and package copies are retained. Native registration changes use native
Codex commands, not handwritten profile edits; unknown marketplace ownership or
active legacy/duplicate memory plugins prevent activation. A partial generation
is not overwritten. Repair requires an intact ownership receipt and an intact
runtime copy or explicitly selected trusted replacement. Native trust review and
a fresh session remain separate from installation success.
Native marketplace removal may delete Codex's installed cache. Recovery relies
on retained managed source/runtime copies, not cache retention; edited cache
files are refused before replacement so that native cleanup cannot erase them.

This is in-progress #7 engineering, not release or immutable-candidate
acceptance. The interactive menu and published-release update discovery are
still pending. Native #6 behavior evidence is recorded separately in
[the Codex receipt](codex-native-evidence.md).
