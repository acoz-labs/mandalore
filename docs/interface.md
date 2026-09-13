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

Tests cover strict schemas, binding isolation/replacement, CLI/MCP equivalence,
compiled executable create/correction/history, unrelated cwd, explicit binding
precedence, no-write hashes, real stdio, EOF and interruption. Full race tests,
vet and four platform builds run in `bin/ci`. Cross-builds are not native runtime
acceptance. These tests use synthetic local data and an SDK client, not a model
conversation or installed Codex plugin. Those remain #6/#10.
