# Hermes integration

Released in [Mandalore 1.6.0](releases/1.6.0.md), with independent retained-artifact
acceptance on Hermes 0.21.5 for macOS arm64 CLI and persistent-service paths.
The release record distinguishes native observations, model behavior and
platform limits. An existing installation needs a separate reviewed upgrade.

## Setup and inspection

Use Mandalore 1.6.0 or a later release with Hermes support; older binaries do
not acquire the adapter through configuration alone. Select a
healthy Hermes executable/profile and an explicit machine-local signet binding.
The guided connection menu includes Hermes. Agent-friendly commands provide
the equivalent `connection plan --harness hermes`, `connection apply --harness
hermes`, `connection armorer --harness hermes` and `connection repair --harness
hermes` paths. Consult each command's help for profile and retained-root flags.

Preview with `--session-sync` to explicitly enable foreground refresh and
delivery, or choose `--memory-read-only` for an enforced read-only connection;
these modes cannot be combined. Apply consumes the complete prepared plan on
stdin. Stop selected Hermes sessions/services before activation, then restart
them to load the new generation; Hermes apply does not use the
`--sessions-stopped` flag. Connection update uses a new reviewed plan, preserving
retained generations for recovery. Inspection reports structure and native
selection separately from untested live model behavior.

The adapter includes native `this-is-the-way` and `the-armorer` skills. The thin
launcher's `hermes` selection sets `HERMES_HOME` to the chosen profile; it does
not replace Hermes authentication or install its dependencies. Readiness can
assess Hermes explicitly, but readiness inspection is not acceptance of a
running service. See [setup](setup.md) and [development](development.md) for the
shared preview, candidate and verification boundaries.

For a selected default or custom root, native administration and the thin
launcher additionally pass `--profile default` so Hermes's sticky
`active_profile` selection cannot redirect the command. A selected native named
profile under `profiles/<name>` uses Hermes's direct `HERMES_HOME` pin. The
connection always targets the selected profile rather than changing the user's
global active-profile choice.

## Memory coexistence

Hermes keeps its built-in memory, learned procedures, conversation management,
native tools and model authentication. Mandalore provides portable knowledge
through one explicitly selected signet. The integration is a general Hermes
plugin, not a replacement memory provider.

The agent uses its existing model to choose useful semantic updates during
ordinary work and at meaningful completion checkpoints. Confirmed preferences,
decisions, reusable procedures and project conventions become records; useful
work outcomes can become concise journals. Temporary assumptions and
conversation scaffolding may remain local to Hermes. Nothing requires a second
model or extraction account.

A checkpoint is guidance to consider learning, not a promise that every turn
produces a record. The adapter must not scrape transcripts, mirror native memory
files, automatically publish all learned skills, or copy signet records into
Hermes memory. Explicit do-not-remember and no-journal directions apply. Secrets
and raw transcripts are not learning inputs for automatic publication.

The signet is also an input: relevant knowledge saved by another harness can
inform the next Hermes turn. This is semantic continuity, not native-thread
transfer or a file synchronization protocol. Historical memory remains evidence,
not instructions. Corrections and conflicting heads retain their provenance;
withdrawn knowledge must not be recreated from stale native memory or journals.

## Native lifecycle mapping

The adapter uses Hermes's general-plugin tool registration and
`pre_llm_call` context hook. The hook receives a native session identity, a turn
identity and the current user message. It awaits the shared engine's
bounded foreground refresh before constructing that turn's memory context.
The first observed turn for each native session maps to `startup` when Hermes
reports a first turn, and otherwise to `resume`; later turns map to `turn`.
These boundaries drive shared-engine synchronization and canon-reference
snapshot creation without relying on the agent selecting a synchronization
tool. Raw native session IDs are passed to the engine, which adds its harness
namespace where required by canon-reference tools.

In the inspected Hermes implementation, `pre_llm_call` runs at the user-turn
boundary, rather than for every model/tool-loop iteration. Its returned context
is appended to the user-message context, not installed as a system prompt.
Memory evidence must therefore be clearly framed as untrusted evidence, with
integration guidance supplied through the appropriate native plugin surface.
Persistence-disabled internal forks skip this hook; they must not be represented
as independently refreshed sessions. Image or file messages still receive
refresh and orientation using an empty retrieval query; the adapter does not
inspect attachments or conversation history to construct that query.

Checkpoint guidance is registered as a native system-prompt section. The adapter
does not add a completion-time model call or extract memory in a finalization
hook. Session finalization and reset only discard adapter session bookkeeping.

The CLI can switch to a cached conversation without emitting a fresh session
start. Its native `pre_command` observer therefore invalidates bookkeeping for
the departing session before `resume`, `sessions`, `branch` or `new`. Only the
bounded session identity supplied by the CLI is used; command arguments and
conversation history are not inspected. This also covers the numbered picker
opened by a bare `/resume`. A failed, canceled or listing-only command can
conservatively cause a resume refresh on the next turn in the same session.
Canonical command names cover native aliases.

Desktop reconnects that reattach to a still-live backend session continue that
session and retain its canon-reference snapshot. They still receive normal
per-turn signet refresh. Actual backend session teardown emits finalization;
the next observed turn after teardown creates a startup or resume boundary.
Merely switching visible chat tabs does not establish a new native session.

Semantic memory tools call the shared engine, which owns validation, record
identity, scope isolation, corrections and post-write delivery. Hermes hooks do
not independently reimplement those semantics. A save receipt distinguishes
local durability from delivery; successful Git transport does not resolve
semantic disagreement. Pending delivery can retry at a later foreground
boundary. There is no background delivery guarantee when Hermes is closed.

## Profiles, failure and disable

Each integration is bound to a selected Hermes profile and retained Mandalore
runtime/binding. Concurrent sessions must use their own native session and turn
identities. A process-global current session or ambient working directory must
not choose a signet. Profile changes and plugin reloads must not let stale work
inject context into a different session.

Session bookkeeping is bounded. A concurrent callback for the same session is
refused rather than waiting behind an in-flight refresh; different sessions keep
separate lifecycle state. Integration installation requires a healthy native
Hermes profile and runtime. An arbitrary `HERMES_HOME` directory may lack
Hermes-managed package/dependency state even when the Mandalore files are valid;
creating an empty directory alone does not establish native readiness.

Hermes bounds hot-path plugin hooks and may skip a callback after timeout while
its worker is still running. The inspected host defaults to a 30-second callback
budget; enabled-session adapter context calls use a 9-second subprocess deadline
with a bounded termination grace period. An operator-configured lower host
budget can abandon the callback before its subprocess deadline. Abandonment
does not immediately cancel the Python worker. The adapter bounds its own work,
but a missing hook result cannot prove a successful refresh. Expose failure or
pending receipts honestly and inspect possible
effects before retrying a write. A hook timeout must not trigger duplicate saves
or weaken read-only access.

Disable the native Mandalore integration for the selected profile and start a
fresh session to establish that its tools, hooks, skills and context are absent.
An already running service may require a restart to unload native registrations.
Disabling does not erase previously read context, native Hermes memory, or
signet records. Conversational requests to stop synchronization are not a
transport control while the integration remains enabled.

## Verification for upgrades

Acceptance must exercise the retained artifact through a supported Hermes
runtime, including the actual CLI and persistent-service paths being claimed.
Synthetic scenarios should prove startup/resume and turn context, semantic
saves, offline recovery, concurrent-session isolation, profile isolation,
read-only restrictions, native disable, and safe installation/update/rollback.

For cross-harness continuity, save a confirmed synthetic fact through Hermes,
recall it in a fresh supported harness session, then publish a correction there
and recall it from a new Hermes turn. Verify the record and transport receipts,
not merely a plausible model answer. Hermes's native memory should remain
available throughout. Follow [delivery](delivery.md) for independent review,
exact-candidate acceptance and same-byte publication evidence.
