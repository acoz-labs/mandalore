# Architecture

## Current state

`main` is 2.0 development. `internal/server` is the single-writer
[server](server.md): it owns each configured signet, serves MCP over Streamable
HTTP with per-client authorship, and backs each signet up by push-only Git. The
1.x stdio MCP server (`internal/mcp`), native hook adapters (`internal/codex`,
`internal/claudecode`), session transport (`internal/sessionsync`), their context
assembler (`internal/memorycontext`) and Razor Crest (`internal/razorcrest`) left
the build in #167. The paragraph below describes the remaining 1.x packages;
installers and plugins leave in #163.

`internal/memory` implements the signet engine as a library: immutable records,
sources/devices, journals, supersession, scoped lexical recall, bounded service
responses, and immutable foundling registration/routing/citation validation. See the
[format contract](signet-format.md) and [extraction inventory](memory-extraction.md).
`internal/api`, `internal/binding` and `cmd/mandalore` provide
the [shared local interface](interface.md). `internal/sync` now supplies explicit
[Git checkpoints and reconciliation](synchronization.md). `plugins/claude` is the
one 2.0 plugin: an HTTP MCP server URL, a `memory_context` hook and two skills, with
no runtime. `internal/console` supplies presentation-only terminal prompts and
reports. The [guided menu](setup.md)
delegates to shared operations rather than owning a second implementation.
`internal/distribution` supplies strict release identity, pinned builds, candidate
transport, owned CLI activation/recovery and verified same-byte publication.
Maintainer commands and workflows coordinate nomination, acceptance guards and
release-ledger finalization outside the memory MCP surface.
The 1.x Codex, Pi, Claude Code and Hermes adapters and their installer left the
build in 2.0 (#163); [Codex](codex-native-evidence.md) and
[Pi engineering evidence](evidence/pi/README.md) remain as historical records.
`internal/cutover` keeps the 1.x connection discovery and disable that
`server migrate` needs to retire old clients.

| Location | Responsibility |
| --- | --- |
| `cmd/mandalore` | Human CLI/menu and machine-readable commands |
| `internal/memory` | Records, journals, scoped retrieval, provenance, supersession |
| `internal/formatupgrade` | Explicit pinned in-place format transition and retained recovery, without checkpoint or delivery |
| `internal/exportreport` | Bounded source snapshots, scoped derived reports and no-replace publication |
| `internal/retention` | Read-only explicit policy/selection metadata review; no expiry or apply |
| `internal/api`, `internal/strictjson` | Shared operations, bounded strict schemas, receipts and errors |
| `internal/sync` | Local checkpoints, recoverable Git synchronization, push-only backup and fast-forward reconcile |
| `internal/binding` | Local signet selection and originating-device identity |
| `internal/foundlings` | Bounded local/Git reference observation, clone-local connections and verified selective promotion |
| `internal/server` | Single-writer Streamable HTTP MCP server: config, authenticators, scoped tools, `memory_context`, per-client authorship, push-only backup, status and reconcile |
| `internal/cutover` | 1.x clone retirement: finding and disabling 1.x native connections through each harness's own commands |
| `internal/distribution` | Versioned assets, provenance and byte verification, owned CLI installation and phased publication |
| `cmd/build-artifacts`, `cmd/candidate-transport`, `cmd/promote-candidate`, `cmd/release-publication` | Repository-maintainer build/verification/publication entrypoints; not memory tools |
| `internal/migration` | Read-only legacy census, explicit out-of-place conversion and retained recovery receipts |
| `internal/console` | Presentation-only terminal prompts, navigation and bounded-width reports |
| `plugins/claude` | The Claude plugin: server URL, `memory_context` hook, `this-is-the-way` and `the-armorer` |

## Ownership and trust

The private signet owns knowledge/history, not native sessions or credentials.
Local bindings own clone paths and device enrollment. The public repository owns code, schemas, templates
and synthetic tests. Do not retain an assistant-framework dependency merely to
avoid extracting memory primitives.

A release plan pins a manifest, platform binary and observed prefix state;
apply retains content-addressed bytes and replaces only a verified owned launcher.
An incomplete activation retains its exact pending plan for identity-checked
recovery. CLI rollback never migrates memory schemas.

The maintainer path builds product bytes once, retains and verifies the exact
Actions archive, derives a candidate identity, and requires independent acceptance
before promotion. The promoter stages those same bytes, checks both release guards,
uploads only absent matching draft assets and verifies every remote asset before
and after publication. Only fresh public verification permits ledger completion.
Control tooling can compile at its separately trusted revision; accepted product
payloads cannot be rebuilt during promotion. Partial or conflicting state is
retained, not overwritten or labeled successful. See [delivery](deployment.md)
and [the maintainer contracts](development.md#guarded-retained-candidate-promotion).

The host agent supplies semantic interpretation. No separate inference service
or embedding subscription is required. Memory is evidence, not executable
permission. Explicit no-write boundaries apply to hooks, CLI and MCP alike.

Foundlings keep source adapters outside the pure memory library. They never clone,
fetch, execute source tooling or turn historical prose into native instructions.
Portable registrations carry identity/pins; ignored clone-local connections carry
paths. Source verification precedes bounded retrieval and runs under the memory
writer lock immediately before selective promotion. The host agent compares
current knowledge and user direction and supplies an adapted lesson and explicit
supersession; storage generates verified provenance, not semantic endorsement.
The skill loads detailed foundling guidance only for relevant consultation.
See [foundlings](foundlings.md) for limits, partial receipts and authority boundaries.

Migration is CLI-only administration, not a model-facing memory mutation. The
converter performs bounded strict decoding and canonical-path validation, retains
raw JSON values, and reuses `memory.ValidateSnapshot` for closed-set provenance
and graph checks without filesystem access. That validation shares the normal
engine's rules; it is not a second memory implementation. Only the supported
legacy memory-only format is accepted. Canonical on-disk validation runs again
before no-replace publication. Native inventory reuses the bounded installer
adapter and distinguishes potential writers from active sessions.

The engine has no assistant identity, capability registry, provider adapter,
subprocess execution, networking or synchronization dependency. `net/url` is used
only to parse portable reference identities. A regression check rejects direct
process/network or other product-package imports. Machine-local state is ignored
by Git and not necessary for reads; a writer creates its local lock directory.

Format2 visibility is a causal graph alongside content supersession, not an
extension interpreted only by a plugin. The shared engine resolves it before
ordinary recall/context/export. Withdraw/restore compare both current content
and visibility heads under the writer lock; new content revisions acknowledge
observed visibility without implicitly restoring a withdrawn record. Concurrent
visibility heads and unreviewed content are withheld. Explicit history preserves
evidence; independent journals and reference documents are not silently removed.

Format upgrade is CLI-only administration with explicit stopped writers and a
reviewed source/binding/Git checkpoint. Prepared local state supports recovery of
the same receipt identity; a canonical manifest transition is the only protected
edit allowed by sync, which validates its ancestor proof and evidence preservation.
Each clone opts in separately. Old readers refuse format2; offline old copies and
already-read context are not revoked. Retention preview reuses bounded report
snapshots but emits policy/relationship metadata only, without a lock or apply.

Corrections preserve history through supersession edges. Concurrent heads remain
conflicts; a successful Git merge does not prove semantic agreement. Reads must
use current local state, not an unbounded stale process cache; only the server,
as sole writer, keeps the validated graph between its own writes. Start with scoped
lexical retrieval, byte budgets, pagination and history access. Lexical tokens
ignore terminal prose periods but preserve internal dots/hyphens; absent synonyms
remain a limitation. A graph validation reuses successful device checks only
within that validation, never across later service operations. Full source and
graph checks still run. See [retrieval](retrieval.md) for quality/freshness tests,
measured history-growth costs and the evidence required before adding an index.

Git remotes and harness executables are explicit dependencies. Local persistence
works offline; cross-machine freshness requires successful sync. In 2.0 MCP is
served only by the server over Streamable HTTP, behind its built-in OAuth 2.1
authorization server with owner passkey approval (see [server](server.md#oauth)).

See [product](product.md), [migration](migration.md), and [delivery](deployment.md).

## Foreground synchronization

A session policy pins exact binding bytes, signet identity and retained runtime.
The session-aware API is separate from the default administrative dispatcher;
local-only commands and legacy read-only connections do not inherit permission
from an enabled session elsewhere. Generated native connections also pin policy
bytes so remembered content cannot retarget or change transport authority.

Coordination is local to a physical checkout. Separate clones of the same signet
must each refresh; concurrent harnesses on one checkout serialize attempts. A
coalesced result is reusable only when its completed head still covers local
state. Native session/event identity, bounded startup adjacency and a consumed
one-shot marker avoid duplicate startup/first-prompt work without treating
identical prompt text as an event ID. Pi adds a per-start nonce to its turn counter.

The semantic operation saves first. Its original receipt/error is preserved and
optional `session_sync` metadata records the subsequent bounded transport outcome.
Combined tools route through the same local primitive and coordinator once.
Pending data retries at the next foreground boundary, not a daemon or assumed
exit callback. No transport operation extracts facts from transcripts or resolves
semantic conflicts automatically.
