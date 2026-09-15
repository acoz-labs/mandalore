# CLI and local MCP

The development CLI and stdio server use one typed operation dispatcher. They
do not supply a model, replace native agent identity, discover memory from cwd,
or synchronize implicitly. CLI-only connection operations install the native
plugin after explicit approval; they are not memory MCP tools. The
[guided menu](setup.md) delegates to these same operations. Explicit Git work is
described in [synchronization](synchronization.md). Exact-candidate/release
acceptance and publication are recorded for [v1.0.0](releases/1.0.0.md).
No live migration is implied.

## Save with bounded delivery

`memory_remember` and `memory_journal_append` remain local-only, with unchanged
inputs and annotations. New network-capable companions are
`memory_remember_and_sync` and `memory_journal_append_and_sync`. CLI equivalents:

```sh
mandalore memory remember-and-sync --binding /example/binding.json < record-and-delivery.json
mandalore memory journal-append-and-sync --binding /example/binding.json < entry-and-delivery.json
```

Inputs nest the existing knowledge fields under `record` or journal fields under
`entry`. Optional `timeout_seconds` bounds the delivery stage only: default 3,
range 1–30. The operation catalog supplies authoritative schemas. Invalid shape,
timeout or memory content is refused before publication; failed saves do not sync.
Both companions are non-idempotent mutations. Read-only connections refuse them.

On a successful local save the result contains `saved` and a typed `delivery`
envelope. Outer `ok: true` (and CLI exit zero) means the save completed, **not**
that delivery succeeded. `saved` retains signet/record/event identity and local
durability; its synchronization field summarizes the delivery state. Inspect
`delivery.result.delivered`, state, exact heads and conflicts, or `delivery.error`.
`delivery.ok: true` can still mean pending or local-only. A conflict can be
delivered without being semantically resolved. A failed delivery retains the
successful save and sanitized error/phase evidence rather than turning it into
an apparently failed write. Never repeat a save just to retry delivery.

No transaction spans both stages. Another writer can act between them; ordinary
sync validates and may deliver other valid pending work too. A cancelled/lost
response may hide a save or completed push: inspect before retrying. No Git setup,
credential enrollment, rollback, tight retry or background worker is implicit.
Use local-only tools for no-sync tasks, and neither type under no-save/read-only.
See [delivery semantics](synchronization.md#save-triggered-delivery).

## MCP result presentation

MCP returns the same complete envelope in `structuredContent` and a JSON text
block for client compatibility. `isError` agrees with the envelope's `ok` flag.
Keep both wire representations. In code-mode orchestration, the memory skill
guides the agent to print one equivalent envelope while preserving errors,
provenance, conflicts and continuation metadata. Text-only, distinct content,
unknown metadata and uncertain equivalence fall back intact. This is scoped
presentation guidance, not a global renderer or a change to memory semantics.
Do not retry an operation merely to render its receipt differently.

The [synthetic presentation checks](evidence/mcp-rendering/README.md) distinguish
wire compatibility, selector behavior and observed native engineering runs from
still-required exact-candidate product acceptance.
Published v1.0.0 has not been modified by this development change.

`mandalore version` is read-only and does not need a signet binding. It reports
the runtime version, source commit (empty for an unstamped development build),
actual Go version, OS/architecture, protocol and hook compatibility, supported
signet read/write schema versions, and the actual embedded plugin version/digest.
The plugin digest uses the same content-map encoding as native connection plans.
These declarations support verification; they are not publisher authentication or
proof that the binary has passed independent acceptance. See
[candidate builds](development.md#local-distribution-candidates).

## Read-only release inspection

```sh
mandalore release inspect --candidate /example/candidate --read-only
mandalore release inspect --read-only
```

The first command checks all files in an explicitly selected local candidate,
without executing them or contacting a provider. The second checks the official
latest stable release when one exists. Use `--version VERSION` for an explicit
published version, including a prerelease. `--candidate` and `--version` are
mutually exclusive. An absent selected release reports `release.unavailable`,
not a fabricated update. Public [v1.0.0](releases/1.0.0.md) is available.

Agents can use the equivalent typed `release_inspect` operation with
`{"candidate":"/example/candidate"}` or `{"version":"1.0.0"}` as structured
stdin. It is advertised as CLI-only, unbound, read-only and potentially networked;
it is not listed or callable through memory MCP and adds no memory-tool schema.
An invalid or absent memory binding does not affect release inspection.

The result distinguishes local byte/content verification from published-release
inspection. Published inspection checks the official immutable release ID, exact
tag commit, manifest/checksum bytes and GitHub asset IDs/sizes/digests. It does
not claim that every executable was downloaded or run. Asset downloads recheck
the pinned release ID and exact inspection before accepting matching bytes;
`latest` is not reselected silently. None of these checks installs a runtime,
activates a native connection, writes memory or establishes independent acceptance.

Only public HTTPS GitHub API/release hosts are used, with bounded redirects,
headers, responses and timeouts. No GitHub CLI login, provider token or cookies
are borrowed. Unavailable, incompatible, mutable, corrupt or rate-limited releases
fail visibly; local candidate inspection remains available offline.

## Read-only CLI installation planning

```sh
mandalore release plan --candidate /example/candidate --prefix /example/tools --read-only
mandalore release plan --version 1.0.0 --prefix /example/tools --read-only
```

`release_plan` is the equivalent typed, CLI-only operation. Its input is
`{"candidate":"/example/candidate","prefix":"/example/tools"}`. Prefix is
required for this machine-readable preview; select at most one candidate directory,
published version or retained manifest SHA-256 (`--retained SHA256`). Omitting all
three selects the latest stable published release and pins its exact release ID,
version, manifest and asset identities in the plan. No binary is downloaded or run
by published planning. Local planning verifies the complete candidate bytes.
Neither interface requires a signet binding or adds a tool to memory MCP.

The plan names the current machine's exact target, executable and plugin digests,
launcher, retained runtime, observed directory identities and current receipt/target.
It describes CLI-only effects: memory connections, signets, credentials and shell
settings remain unchanged. Local byte verification is not publisher authentication;
choosing a local source for later execution requires trusting that source.

The selected prefix may resolve an explicit alias, such as macOS `/tmp`. Managed
descendants must be real directories owned by the current user, not writable by
other users. Only `PREFIX/bin/mandalore` and `PREFIX/lib/mandalore` are in scope;
the rest of the prefix is not owned by Mandalore. A pre-existing launcher must
match a valid ownership receipt and the verified retained target. Regular files,
foreign/dangling links, redirected managed directories, edited retained bytes,
unrecognized state and pending activation are refused without writing anything.

Retained selection verifies a compatible manifest and platform binary beneath
`PREFIX/lib/mandalore/releases/sha256-MANIFEST/OS_ARCH/`. A serialized plan is not
proof that its observations remain current or authorization to change connections.
Activation is a separate explicit operation described below. The interactive
equivalent is [the guided release journey](#guided-cli-installation-and-native-handoff).

## Apply a reviewed CLI installation

```sh
mandalore release plan --candidate /example/candidate --prefix /example/tools > reviewed-plan.json
mandalore release apply < reviewed-plan.json
```

Review the exact source, destinations and effects before applying. Human CLI apply
accepts its own successful plan envelope or a raw plan object; typed
`release_apply` accepts the raw object. `--read-only` refuses apply before reading
stdin. Plan input is bounded at 32 KiB (64 KiB for the human envelope); planning
refuses a result too large for the typed input budget. These administrative
operations and their error details stay out of the bound memory MCP.

Apply rechecks the source and destination, stages and hashes the selected bytes,
and verifies the actual executable's Go build settings and native `version`
response against the manifest. The version probe runs with a minimal environment,
no inherited home, memory binding or provider credentials, a 15-second deadline
and a 16 KiB output limit. This is execution of the explicitly trusted selected
source, not a sandbox or independent publisher approval. Staged bytes are checked
again after the probe. Published downloads revalidate the pinned release ID and
asset identity; an unavailable/corrupt payload does not activate anything.

Installation writers coordinate with a persistent file lock in their owned state.
An initial state tree is assembled privately and published without replacing an
existing directory. Updates retain content-keyed runtime directories and switch
only the verified launcher. Files and affected directories are synchronized around
atomic publication. Existing runtimes are not removed, and neither PATH nor native
memory connections are changed. These guards coordinate installers and reject
observed foreign paths; they are not a security boundary against a malicious
process running as the same user.

Results expose the phase, exact plan digest, launcher/runtime, previous runtime,
pending record if present, destination changes and unchanged connections. Errors
include `release_result`; successful cleanup is required for `installed: true`.
Do not infer success solely from a launcher that already points at the new binary.
CLI success also does not imply any native connection was updated.

An interrupted activation retains `PREFIX/lib/mandalore/pending.json`, with the
exact reviewed plan, previous receipt bytes and observed directory identities.
After inspecting that record and resolving the reported problem, the human CLI
can read it directly using the original Mandalore executable:

```sh
mandalore release apply < /example/tools/lib/mandalore/pending.json
```

This extracts only its exact digest-matching reviewed plan; apply independently
re-reads the current pending record and performs the existing ownership checks.
Do not use a partially activated launcher or edit the record to force a retry.
Pending input is bounded at 256 KiB; the extracted plan still obeys the 32 KiB
typed-operation limit. No extra memory MCP operation or parser dependency is added.

Reapplying that **same plan** can finish only if the retained bytes, directories,
launcher and receipt still match an expected old/new state. Recovery refuses
foreign or conflicting files, missing/corrupt runtime bytes and a different plan.
It does not guess ownership or roll back memory. A completed exact-plan replay
is recognized by the final receipt and verified runtime and performs no writes.
If staging stopped before a pending record was published, inspect a fresh plan;
new directories or a retained target can make the original preview stale.

To roll back the CLI, select the older retained **manifest SHA-256**, review the
new plan and apply it through the same interface:

```sh
mandalore release plan --retained MANIFEST_SHA256 --prefix /example/tools > rollback-plan.json
mandalore release apply < rollback-plan.json
```

Only compatible declared protocol/schema and the current platform are accepted.
The newer runtime stays retained. No signet data is rewritten, and a CLI rollback
does not by itself roll back a separately installed native connection.

## Guided CLI installation and native handoff

```sh
mandalore release install --candidate /example/candidate --prefix /example/tools
mandalore release install --version 1.0.0 --prefix /example/tools --plain
```

This uses the same `release_plan` and `release_apply` contracts, with a readable
preview and default-No confirmation. The main menu also offers published versions,
explicit local candidates and retained runtimes. Without `--prefix`, it asks for a
user-owned destination, defaulting to the user's `.local` directory. It never edits
PATH or shell configuration. `--read-only` refuses the journey before reading input;
use inspect/plan instead. EOF, Back and cancellation stop subsequent steps without
undoing an installation already completed.

After CLI verification, keeping native connections unchanged is the default. An
optional Codex handoff asks for the selected binding, native executable/profile
and installation state. Flags `--binding`, `--native-binary`, `--native-home` and
`--state-dir` prefill these choices; CLI-only installation does not require them.
Preparing that handoff **executes the explicitly trusted installed runtime** via
its existing `call connection_plan --read-only` operation. The parent verifies its
bounded JSON plan against the selected paths, executable/binding hashes and release
plugin identity. It never substitutes the parent's embedded plugin.

A second default-No confirmation precedes that same runtime's
`call connection_apply`. Native authentication is inherited normally, not copied;
raw failed output is suppressed. A typed partial receipt remains visible even
when the subprocess exits unsuccessfully. CLI installation success remains distinct
from native failure. Start a fresh native session after a successful connection
update; existing threads do not reload their plugin context automatically.

Agents do not need to drive menu keys: use the typed release operations, then invoke
the verified installed executable's existing connection plan/apply operations with
explicit options. This adds no memory MCP tools. An interrupted CLI activation is
still recovered by reapplying its saved exact plan as described above, not by
selecting a different release in the menu.

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
`memory scopes` to discover stored IDs before selecting a project/account/task
when the relevant ID has not already been discovered. Reusing a known ID routes
a fresh recall, not a cached answer. Recall defaults to five hits and an 8192-byte
result budget; increase these only when needed. See [retrieval](retrieval.md) for
lexical limits, completeness/conflict semantics and measured scale behavior.

## Machine-ready contract

The plugin's on-demand **The Armorer** skill drives these administrative
operations through shell access. Human inspection is
`mandalore connection armorer [profile options]`; `connection doctor` is a
compatible alias with identical flags, JSON reports and exit status. The typed
name remains `connection_doctor` to preserve existing automation. Repair remains
`connection_repair_plan` followed by `connection_apply`, not an effect of
inspection. No administrative tools are added to memory MCP.

`mandalore operations` lists versioned input/result JSON schemas, read/write and
idempotency annotations, `requires_binding` and `cli_only` visibility.
`mandalore call OPERATION --binding FILE < input.json`
uses the same decoder and methods as the human commands and MCP tools.

| Operation | Human command | MCP |
| --- | --- | --- |
| `signet_create`, `signet_bind` | `signet create`, `signet bind` | Not exposed |
| `migration_preflight`, `migration_apply` | `migration preflight`, `migration apply` | Not exposed |
| `memory_recall`, `memory_scopes`, `memory_history` | `memory recall`, `scopes`, `history` | Bound signet only |
| `memory_journal`, `memory_inspect` | `memory journal`, `inspect` | Bound signet only |
| `memory_remember`, `memory_journal_append` | `memory remember`, `journal-append` | Bound signet only |
| `memory_git_init`, `memory_checkpoint`, `memory_sync`, `memory_sync_status` | `memory git-init`, `checkpoint`, `sync`, `sync-status` | Bound signet only |
| `foundling_list`, `foundling_inspect`, `foundling_search`, `foundling_read`, `foundling_promote` | `foundling list`, `inspect`, `search`, `read`, `promote` | Bound signet only |
| `foundling_preview`, `foundling_register`, `foundling_connect`, `foundling_disconnect`, `foundling_history` | `foundling preview`, `register`, `connect`, `disconnect`, `history` | Not exposed; still require a selected binding |

Every operation returns `{protocol_version, ok, result}` or
`{protocol_version, ok, error}`. The current protocol is 1. The catalog describes
the semantic result schema; MCP advertises its complete envelope schema and
returns the same envelope in structured content and a text fallback. Tool errors
also set MCP `isError`. MCP protocol/framing errors remain SDK transport errors,
not operation results. Startup diagnostics use stderr, never protocol stdout.

Migration administration uses explicit source/output/device-label/actor inputs;
it never loads the normal binding default. `migration_preflight` accepts optional
`legacy_binding`, `native_home` and `native_binary`. `migration_apply` accepts
`{"plan": <preflight-result>, "writers_stopped": true}`. The human apply command
accepts either the raw plan or its successful preflight envelope on stdin, plus
`--writers-stopped`. Both routes call the same dispatcher; `--read-only` refuses
apply before reading input. `migration.failed` (exit 1) may include a
`migration_result` with phase, published state and retained staging/output paths.
This administration-only error detail is not added to MCP's memory-tool schemas.
See [migration](migration.md) for supported data, snapshot limits and handoff.

Foundling administration separates portable registrations from ignored clone-local
paths. Preview takes `source: {kind, locator}` and an absolute `local_root`.
Register takes `name`, `description`, `source`, the exact preview `pin`, and
`reason`; optional `local_root` connects after registration. Without a local root,
registration is metadata-only, not proof of source availability. Updates also
require `foundling_id` and explicit predecessor registration IDs in `supersedes`.
Replacing a local connection requires its `expected_connection_id`.

Registration/connection is not one atomic transaction. A failed setup can return
`error.foundling_result` with a phase, compact completed registration receipt and
connection publication/durability result. Inspect those before retrying; do not
create a duplicate registration because its connection failed. These administrative
error details are excluded from MCP's memory-tool schemas.

`foundling connect` takes `foundling_id`, `registration_revision_id`, `local_root`
and an optional `expected_connection_id`. `foundling disconnect` takes the foundling
ID, exact single active registration revision and a reason; it appends history,
not deletion. Resolve conflicts explicitly through registration updates first.
Concurrent metadata writes can produce visible conflicting heads; a receipt is
evidence of the appended revision, not an enduring availability guarantee.

The human read commands support `--foundling-id`, `--query` (search), `--limit`
(list/history/search), `--offset` (list/history/search/read), `--registration-id`
(search/read), `--excerpt-bytes` and `--budget-bytes` (search), and `--locator`,
`--limit-bytes` (read). Setup mutations, preview and promotion accept JSON on stdin;
use the catalog for exact schemas. Search defaults to three results (range 1–10),
512-byte previews (128–1024) and an 8192-byte serialized-result budget (2048–32768).
Read defaults to 1024 content bytes (range 1–8192).
Search queries require 1–16 nonempty literal terms, at most 1024 UTF-8 bytes;
matching is case-insensitive substring matching without stemming. Invalid queries
return `input.invalid`, not a suggestion to inspect or repair the source.
Search-result offsets are document ranks; continuing above zero requires the exact
returned registration revision and the same query. Each excerpt's offsets instead
select UTF-8 bytes within its file. New pagination fields are additive; existing
explicit read sizes remain supported. A `foundling.budget` error asks for a larger
supported result budget for that same page, not source repair or a blind retry.
Promotion supplies exact foundling/registration/locator/file-SHA fields and a normal `write`, without its
`external_origin`: verified provenance is generated. Optional original author/date
are distinct from current incorporation authorship. Read-only rejects every mutation.
See [foundlings](foundlings.md) for source limits, reference authority and examples.

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
| `foundling.budget` | 1 | First remaining search item cannot fit; explicitly increase the result budget, preserving query/registration/offset; not evidence of absence |
| `foundling.changed`, `foundling.unavailable`, `foundling.connection`, `foundling.failed` | 1 | Inspect the selected source/registration/connection and any partial setup result; no automatic repair or promotion |

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
