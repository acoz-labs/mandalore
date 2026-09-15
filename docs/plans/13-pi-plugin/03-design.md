# Technical Design

## Component And Behavior Flow

`setup/Armorer -> reviewed Pi connection plan -> explicit apply -> retained runtime
and package -> native Pi install -> fresh ordinary Pi session -> bound tools`

At attachment the extension validates its generated local context, runtime
identity/protocol and binding, then loads only the bound operation catalog.
Tool names match shared operation names. Detect collisions before registration;
refuse this attachment rather than overwrite another extension's tools. Never
replace the native active-tool selection. No fallback to cwd, PATH or a different
binding after a configured connection fails.

Each invocation runs the retained executable with an argument array, explicit
binding/digest, `--harness pi`, and enforced `--read-only` when configured. Pass
JSON on stdin, not through shell interpolation. Preserve the complete successful
or error envelope as one native tool text result; metadata must not repeat its
body. A complete nonzero-exit envelope remains evidence, not a generic exception
that discards a partial save. Missing/invalid output after dispatch is ambiguous
for mutations: report possible effects and inspect before retrying. No retries.
Use sequential execution for Mandalore tools to preserve in-turn intent ordering;
cross-session concurrency remains the shared engine's responsibility. Pi's native
transport success is not memory success: the envelope's `ok`, saved and delivery
fields remain authoritative. Do not invent an unsupported native `isError` field.

## State And Data Model

Reuse signet schema and device enrollment unchanged. A machine-local generated
connection records schema version, harness, runtime absolute path/digest,
package digest/version, binding path/digest, signet ID, native profile/binary and
managed generation root. It contains no tokens and stays outside the signet.
The generation receipt binds the exact generated package/config bytes to the
reviewed plan; a connection update creates a fresh retained generation.

Pin binding identity at connection apply, not from arbitrary current bytes on
every restart. A changed binding requires an explicit new connection plan;
repair is for proven owned assets, not permission to switch signets. Guarded
binding open validates the supplied SHA-256 and signet ID against the same bytes
it decodes. The memory service continues checking root identity on operations.
This protects accidental/stale redirection, not a hostile same-user filesystem.

## Interfaces And Contracts

- Add optional `--binding-sha256` and `--signet-id` common guards, strictly
  validated and used together by Pi. Existing unguarded callers retain their
  behavior; no new bank-discovery mode. Guard failure precedes writes/network.
- Add a CLI-only read-only context operation taking the bounded current prompt.
  Extract shared orientation/recall assembly from the Codex hook without changing
  its existing output contract. Context uses a UTF-8-safe 2048-byte query, at most
  three bank-wide recall hits/4096 result bytes, five routing scopes, and a total
  encoded packet at most 16383 bytes. Foundlings are never auto-scanned.
- Add unbound CLI-only `pi_package_inspect` for protocol/package identity. Do
  not extend generic `version` fields or the format-1 manifest. Stamping in the
  builder's isolated export includes Pi before compilation; no new external
  release asset or runtime npm install. Deterministic package-byte identity is
  verified when materializing from the selected executable's embedded payload.
- Add `pi_connection_plan`, `pi_connection_apply`, `pi_connection_doctor`,
  `pi_connection_repair_plan`, and corresponding `connection --harness pi`
  routing. Existing omitted-harness behavior remains Codex. Typed plans bind
  paths, source/native/binding/package identities, selected profile inventory,
  generation and effects. Preview reads only; apply executes selected trusted
  binaries. Doctor may inspect the native version but does not authenticate,
  synchronize or assert loaded model context.
- Catalog input/output is bounded (1 MiB catalog ceiling); individual tool stdin
  is at most shared `MaxInputBytes` (32768) and stdout at most `MaxOutputBytes`
  (65536). Reject invalid framing, excess output, exit/envelope contradictions
  and unsupported protocol. Raw stderr is bounded/discarded, never placed in the
  model context. Startup/metadata and local-context subprocesses have five-second
  deadlines; native installation commands have 30-second deadlines;
  memory calls have a 45-second outer deadline so the shared 30-second maximum
  sync budget can finish and emit evidence. Abort initiates process-group
  cancellation with bounded escalation/reaping, not a detached background write.

### Native lifecycle and model context

| Native event | Mandalore behavior |
| --- | --- |
| `session_start` (startup/reload/new/resume/fork) | Validate explicit connection; register tools once per extension lifecycle; refresh attachment diagnostics. |
| `before_agent_start` | Obtain fresh bounded local packet and append it to the event's existing system prompt for that turn. Label memory as untrusted evidence. |
| `resources_discover` | Use package-native skill discovery; no extra copied profile or transcript search. |
| `input` | No text-triggered writes, sync or authorization parser. |
| compaction events, `agent_end`, `agent_settled` | No saves/checkpoints/sync; lifecycle #55 remains separate. |
| `session_shutdown` | Cancel/reap owned subprocesses; no exit-save promise. |

Unavailable local context emits a compact native warning and honest unavailable
orientation; ordinary Pi remains usable. Memory tool calls return their actual
failure. Do not tell the model memory is attached successfully when validation
failed. Rendered warnings must work without an interactive UI as well.

The Pi `this-is-the-way` skill teaches recall, confirmed incremental learning,
supersession, scoped lookup and distinct local/delivery receipts. The direct cue
is additive semantic intent; a quoted or recalled phrase is not a trigger.
Conversational no-save instructions constrain the host agent; only configured
read-only mode is an enforced runtime boundary. Explain this distinction.
Native `/skill:this-is-the-way` is also available without introducing a launcher.
The Armorer uses CLI-only operations and generated local context, not memory tools
or guesses based on the project directory. Keep neutral foundling/delivery rules
consistent with Codex through shared packaging or tested equality; do not copy
Codex-only code-mode instructions into Pi.

### Installation, update and recovery UX

Classify as `in-pattern-visual-change`: existing console list, preview/effects,
default-No confirmation, partial receipt and next-action patterns. Add a harness
choice to connect and connection inspection/recovery; show the selected harness,
profile, signet and retained runtime clearly. Cancelling must leave native state
unchanged. Typed operations offer the same outcome without menu keystrokes.

Apply stages exclusive owned bytes before `pi install ABSOLUTE_PACKAGE_PATH`,
with `PI_CODING_AGENT_DIR` set for the selected profile. Native settings inventory
is read as bounded structured data, resolving relative package paths; preserve
unrelated keys/packages and resource filters. No hand edits to settings or native
credentials. Refuse ambiguous existing Mandalore registrations, foreign files,
changed plan inputs and incompatible native contracts. Package generation lives
outside native profiles and signets. Reject unsafe overlaps/symlinks.

Updates remove only an exact proven old registration via native Pi, then install
the new one. Retain both generated copies and phase evidence. A failure between
these steps is an inactive connection, not success or a reason to overwrite the
profile. Recovery re-inspects native inventory and ownership; an interrupted
command is not assumed to have failed without effects. Serialize Mandalore's own
updates; revalidate inventory before/after native calls and report interference.
Other tools may still edit the profile; no global native settings lock is claimed.
Never remove an unknown package or silently retry a mutation.

## Authorization And Data Exposure

The host user chooses the explicit connection and grants ordinary native tool
authority. Go enforces schema, binding, read-only, locking and size constraints.
Admin operations require an explicit setup/repair/update request and reviewed
effects; a diagnosis is not an apply. Native auth is inherited, not inspected or
copied. Runtime Git uses existing configured transport exactly as today; no
credential broker. Public evidence records synthetic IDs and semantic summaries,
not personal paths, transcripts or bank contents.

## Failure, Recovery, And Observability

Differentiate unavailable attachment, stale binding, unsupported Pi contract,
busy storage, validation failure, cancelled/ambiguous write, saved-local/pending
delivery and installed-but-not-loaded. Structural Armorer success proves neither
authentication nor active-session context. Require a fresh session or native
reload after a connection update; do not claim hot reloading of existing tools.
Retain bounded local generation receipts; do not create a raw transcript log.

## Design Traceability

Native package/installer cover ordinary Pi setup and recovery; shared operations
cover cross-harness memory/history/sync; guarded binding and separate profiles
cover isolation; `pi` authorship covers provenance; read-only per-turn packets and
skills cover passive recall/learning/cue/boundaries. The verification matrix
tests each claim at its actual layer, including native model behavior.
