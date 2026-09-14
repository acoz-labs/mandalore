# Architecture

## Current state

`internal/memory` implements the signet engine as a library: immutable records,
sources/devices, journals, supersession, scoped lexical recall, bounded service
responses, and data-only foundling registration/citation validation. See the
[format contract](signet-format.md) and [extraction inventory](memory-extraction.md).
`internal/api`, `internal/binding`, `internal/mcp` and `cmd/mandalore` now provide
the [shared local interface](interface.md). `internal/sync` now supplies explicit
[Git checkpoints and reconciliation](synchronization.md). `internal/codex` and
`plugins/codex` supply the development native plugin, read-only lifecycle context
and memory skill. `internal/install` supplies explicit native connection plans,
retained runtime/package publication and doctor/repair; `internal/console` supplies
presentation-only terminal prompts and reports. The [guided menu](setup.md)
delegates to shared operations rather than owning a second implementation.
Later harness plugins remain integration targets. [Native engineering evidence](codex-native-evidence.md) is
separate from independent product acceptance; no release is implied.

| Location | Responsibility |
| --- | --- |
| `cmd/mandalore` | Human CLI/menu and machine-readable commands |
| `internal/memory` | Records, journals, scoped retrieval, provenance, supersession |
| `internal/api`, `internal/strictjson` | Shared operations, bounded strict schemas, receipts and errors |
| `internal/sync` | Local checkpoints and recoverable Git synchronization |
| `internal/binding` | Local signet selection and originating-device identity |
| `internal/mcp` | Typed stdio MCP access over the same memory engine |
| `internal/codex` | Bounded read-only native event adapter; no transcript access or synchronization |
| `internal/install` | Native connections, runtime pinning, doctor/update/repair |
| `internal/console` | Presentation-only terminal prompts, navigation and bounded-width reports |
| `plugins/codex` | First native plugin and `this-is-the-way` skill |
| `plugins/pi` | Reserved second integration after Codex acceptance |
| `plugins/claude-code` | Reserved third integration after Pi acceptance |

## Ownership and trust

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
doctor does not establish authentication, hook trust, live MCP or fresh context.

The host agent supplies semantic interpretation. No separate inference service
or embedding subscription is required. Memory is evidence, not executable
permission. Explicit no-write boundaries apply to hooks, CLI and MCP alike.

The engine has no assistant identity, capability registry, provider adapter,
subprocess execution, networking or synchronization dependency. `net/url` is used
only to parse portable reference identities. A regression check rejects direct
process/network or other product-package imports. Machine-local state is ignored
by Git and not necessary for reads; a writer creates its local lock directory.

Corrections preserve history through supersession edges. Concurrent heads remain
conflicts; a successful Git merge does not prove semantic agreement. Reads must
use current local state, not an unbounded stale process cache. Start with scoped
lexical retrieval, byte budgets, pagination and history access; measure before
adding indexes or vectors.

Git remotes and harness executables are explicit dependencies. Local persistence
works offline; cross-machine freshness requires successful sync. The initial MCP
transport is local stdio. Remote hosting/authentication is deferred.

See [product](product.md), [migration](migration.md), and [delivery](deployment.md).
