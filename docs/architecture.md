# Architecture

## Current state

`internal/memory` implements the signet engine as a library: immutable records,
sources/devices, journals, supersession, scoped lexical recall, bounded service
responses, and immutable foundling registration/routing/citation validation. See the
[format contract](signet-format.md) and [extraction inventory](memory-extraction.md).
`internal/api`, `internal/binding`, `internal/mcp` and `cmd/mandalore` now provide
the [shared local interface](interface.md). `internal/sync` now supplies explicit
[Git checkpoints and reconciliation](synchronization.md). `internal/codex` and
`plugins/codex` supply the development native plugin, read-only lifecycle context
and memory skill. `internal/install` supplies explicit native connection plans,
retained runtime/package publication and doctor/repair; `internal/console` supplies
presentation-only terminal prompts and reports. The [guided menu](setup.md)
delegates to shared operations rather than owning a second implementation.
`internal/distribution` supplies strict release identity, pinned builds, candidate
transport, owned CLI activation/recovery and verified same-byte publication.
Maintainer commands and workflows coordinate nomination, acceptance guards and
release-ledger finalization outside the memory MCP surface.
`plugins/pi` supplies a second native adapter over guarded, short-lived CLI calls;
`internal/memorycontext` shares bounded local evidence assembly across harnesses.
Claude Code remains a later target after Pi acceptance. [Codex](codex-native-evidence.md)
and [Pi engineering evidence](evidence/pi/README.md) are separate from product
acceptance. Published v1.0.0 is Codex-first; Pi development code is not an accepted
production upgrade.

| Location | Responsibility |
| --- | --- |
| `cmd/mandalore` | Human CLI/menu and machine-readable commands |
| `internal/memory` | Records, journals, scoped retrieval, provenance, supersession |
| `internal/api`, `internal/strictjson` | Shared operations, bounded strict schemas, receipts and errors |
| `internal/sync` | Local checkpoints and recoverable Git synchronization |
| `internal/binding` | Local signet selection and originating-device identity |
| `internal/foundlings` | Bounded local/Git reference observation, clone-local connections and verified selective promotion |
| `internal/mcp` | Typed stdio MCP access over the same memory engine |
| `internal/codex` | Bounded read-only native event adapter; no transcript access or synchronization |
| `internal/memorycontext` | Shared bounded local orientation, bank-wide evidence and scope routing |
| `internal/install` | Native connections, runtime pinning, doctor/update/repair |
| `internal/readiness` | Non-executing setup observations, support declarations, exact scenario evidence and bounded follow-up guidance |
| `internal/distribution` | Versioned assets, provenance and byte verification, owned CLI installation and phased publication |
| `cmd/build-artifacts`, `cmd/candidate-transport`, `cmd/promote-candidate`, `cmd/release-publication` | Repository-maintainer build/verification/publication entrypoints; not memory tools |
| `internal/migration` | Read-only legacy census, explicit out-of-place conversion and retained recovery receipts |
| `internal/console` | Presentation-only terminal prompts, navigation and bounded-width reports |
| `plugins/codex` | First native plugin and `this-is-the-way` skill |
| `plugins/pi` | Embedded native extension, guarded CLI transport, memory/Armorer skills and tests |
| `plugins/claude-code` | Reserved third integration after Pi acceptance |

## Ownership and trust

Machine readiness is CLI-only administration. `internal/readiness` owns one
assessor used by the menu and typed/human CLI. Its reviewed format-1 catalog
separates intended targets, dependencies and immutable engineering/accepted
scenario evidence. No online registry or caller-supplied evidence is consulted.
No executable self-hash is embedded: measured disk bytes, process metadata and
retained-runtime identities stay separate. Missing scenario identities are not
wildcards; historical evidence is not a diagnosis of broken software.

The assessor follows a fixed read set: executable fingerprints, directory presence,
binding/manifest/enrolled-device identity, and optionally one explicitly selected
owned receipt/runtime. It reuses pure memory validators and installer receipt
projections, not full store validation or native Doctor/Prepare/Apply routines.
Ownership/selection checks precede following retained paths. Managed metadata
redirects and special files are refused; selected executable aliases can resolve
to a regular target. Bounded streamed reads check cancellation and changes in
identity, size, mode and mtime. This is not loaded-image attestation, a filesystem
lease or protection from a hostile local user; OS access-time/cache bookkeeping
can occur. No product state, credential, remembered content or network is involved.

Assessment has no persistent state or recovery transaction. Optional fixed-text
guidance is an assessment-time snapshot and no new authority. Native inspection
is an explicit subsequent operation with separately disclosed log/cache effects.
See the [assessment interface](interface.md#non-executing-machine-assessment).

The private signet owns knowledge/history, not native sessions or credentials.
Local bindings own clone paths and device enrollment. Generated connections own
only declared managed files. The public repository owns code, schemas, templates
and synthetic tests. Do not retain an assistant-framework dependency merely to
avoid extracting memory primitives.

Installation is CLI-only administration, not another bound memory tool. Plans
identify the preparing toolkit's embedded plugin and selected trusted runtime
separately; applying revalidates bytes, paths and binding identity before execution.
Retained content-addressed runtimes and versioned package generations avoid
in-place overwrite. Native commands own registration/cache, not hand-edited native
configuration. Ownership receipts and filesystem integrity protect recovery;
unknown edits or partial publication are preserved for inspection. A structural
doctor does not establish authentication, hook trust, live tools or fresh context.

Pi owns native tools, authentication, resources and sessions. Mandalore registers
the bound memory catalog once, preserves native tool selection and appends fresh
bounded evidence to the existing system prompt per turn. It does not accumulate
persistent attachment messages, scan transcripts or write at startup, compaction
or shutdown. Shutdown cancels/reaps owned calls. Confirmed learning and delivery
remain semantic tool use; lifecycle synchronization is a separate unresolved
workstream, not implemented by these context hooks.

Pi's generated connection pins runtime/package hashes, binding bytes and signet
identity. Go validates the same bytes it hashes, preventing later binding changes
from silently redirecting calls. Native tools return one full envelope with only
operation/ok metadata, retaining nested save/delivery outcomes after cancellation.
There is no JavaScript memory engine, MCP proxy or credential broker.

Pi registration uses native install/remove commands against the selected profile.
Inventory preserves unrelated packages and resource filters; foreign, ambiguous
or edited state is refused. The menu executes the selected runtime's own planner
and apply operation. Repair uses the owned generation's intact runtime and keeps
its binding/access mode. Replacement is phased rather than atomic: interruption
can leave the connection inactive with retained receipts.

CLI distribution and native connection ownership are separate. A release plan
pins a manifest, platform binary, embedded plugin and observed prefix state;
apply retains content-addressed bytes and replaces only a verified owned launcher.
An incomplete activation retains its exact pending plan for identity-checked
recovery. Selecting a native update delegates fixed typed operations to the
verified new runtime so an older installer's plugin cannot substitute for it.
Neither CLI rollback nor native repair migrates memory schemas.

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

Corrections preserve history through supersession edges. Concurrent heads remain
conflicts; a successful Git merge does not prove semantic agreement. Reads must
use current local state, not an unbounded stale process cache. Start with scoped
lexical retrieval, byte budgets, pagination and history access. Lexical tokens
ignore terminal prose periods but preserve internal dots/hyphens; absent synonyms
remain a limitation. A graph validation reuses successful device checks only
within that validation, never across later service operations. Full source and
graph checks still run. See [retrieval](retrieval.md) for quality/freshness tests,
measured history-growth costs and the evidence required before adding an index.

Git remotes and harness executables are explicit dependencies. Local persistence
works offline; cross-machine freshness requires successful sync. The initial MCP
transport is local stdio. Remote hosting/authentication is deferred.

See [product](product.md), [migration](migration.md), and [delivery](deployment.md).
