# Guided local setup

Run `mandalore menu`. An interactive terminal supports arrows or j/k, Enter/l
to select, Esc/h to go back, and gg/G for first/last. Text fields use ordinary
typing, not Vim command mode; Tab inserts the displayed default for editing.
Opening the menu changes nothing. Mutation confirmations default to No.

Use `--plain` for numbered prompts, pipes or assistive tooling. Non-terminal
and dumb-terminal streams select plain mode automatically; `MANDALORE_PLAIN=1`
also selects it. `NO_COLOR` removes color without disabling navigation. Status
labels remain explicit. In plain mode answers must end with a newline: EOF,
including an unfinished confirmation, never approves changes. `:back` cancels
a journey. Oversized input stops without interpreting subsequent bytes as choices.

## Signet journeys

A signet is your private memory bank, not an agent or portable session.

- **Create** asks for an absolute directory, a neutral name, an explicit machine
  label, writer attribution and a machine-local binding outside the bank. Labels
  enter memory; do not put secrets in them. No hostname or account is inferred.
  Existing binding collisions are checked before creating the bank.
- **Connect** enrolls a new device in an existing local Mandalore signet. Clone
  the intended private repository with native Git first. Connection preserves
  its identity and does not initialize Git, fetch or push.
- **Inspect** checks structure and local Git/receipt state without network or
  memory writes. A prior delivery receipt does not prove current remote freshness.
- **Synchronize** previews checkpoint/fetch/integrate/push effects before approval.
  Inspect delivery state; pending, offline or conflicting results are not delivery.

Creation, binding and Git initialization are separate durable steps. If a later
step fails, the menu reports partial setup and retains completed work. Inspect
before retrying: there is no automatic rollback or overwrite. New banks are
local-only. The menu does not provision accounts, private remote repositories,
credentials or Git origins. Configure native Git separately.

Binding selection is `--binding FILE`, then `MANDALORE_BINDING`, then the platform
config default. A newly created/connected binding becomes selected for this menu.
Use explicit bindings for additional banks; no cwd discovery or shell edits occur.

## Codex journeys

**Connect/update** asks for a trusted local Mandalore executable and previews its
hash, the preparing toolkit's embedded plugin identity, bound signet, native
executable/profile and retained installation paths. Applying executes selected
binaries. A digest identifies bytes, not publisher trust. The menu and JSON CLI
use the same revalidated plan. It does not launch agents or accept hook trust.

The managed connection pins a retained runtime and binding, so exports are not
needed. Explicit environment overrides still win; doctor reports conflicts.
Old managed source/runtime copies are retained, but native cache can be replaced.
Unmanaged or edited registrations are not silently removed. Start a fresh native
session and review exact hooks afterward. No native credentials are copied.

**Doctor** labels pass, fail and not-tested independently. It inspects structure
and native inventory; it does not prove login, hook trust, live MCP, remote
freshness or active context. **Repair** asks for a retained connection root
(shown by doctor), previews a fresh generation and requires approval. Unknown
edits or missing ownership evidence are refused. Partial native failures show
completed phase and retained paths; inspect before retrying, not assumed rollback.

`--state-dir DIR`, `--native-home DIR` and `--native-binary FILE` select an explicit
installation/profile. `--binary FILE` sets the local artifact offered by the menu.
Codex is required only for native journeys, not opening the menu or memory setup.
Published update discovery remains #11; exact-candidate acceptance remains #10.
No assistant launcher, capability framework or live predecessor import is added.

## Agent-ready equivalents

Use `mandalore operations` for schemas and [the interface](interface.md) for
`signet`, `memory`, `connection` and `call` commands. Agents should use structured
operations rather than menu keys. Installation operations remain CLI-only; they
do not enlarge the bound memory MCP tool set. Complete JSON receipts are available
there while the menu shows human summaries.

Exit status is 0 on ordinary exit, 1 if an operation/output failed during the
invocation, 2 for invalid flags and 130 for process-context cancellation.
Exiting or cancelling never undoes an earlier completed step.
