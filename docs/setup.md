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

**The Armorer · Inspect connection** labels pass, fail and not-tested independently. It inspects structure
and native inventory; it does not prove login, hook trust, live MCP, remote
freshness or active context. **The Armorer · Repair connection** asks for a retained connection root
(shown by inspection), previews a fresh generation and requires approval. Unknown
edits or missing ownership evidence are refused. Partial native failures show
completed phase and retained paths; inspect before retrying, not assumed rollback.

`--state-dir DIR`, `--native-home DIR` and `--native-binary FILE` select an explicit
installation/profile. `--binary FILE` sets the local artifact offered by the menu.
Codex is required only for native journeys, not opening the menu or memory setup.
Exact-candidate acceptance is recorded in [the v1.0.0 release record](releases/1.0.0.md).
No assistant launcher, capability framework or live predecessor import is added.

## CLI installation and updates

**CLI · Install, update or select a retained runtime** offers the latest published
stable release, a specific version, an explicit local candidate, or a retained
manifest SHA-256. An unavailable release is reported without changing the machine.
The first published version is [1.0.0](releases/1.0.0.md). This journey is also available directly
as `mandalore release install`, including `--plain` and optional `--prefix DIR`.

The preview shows source and content identities, compatibility, destination,
previous runtime and effects. Confirmation defaults to No. Applying activates only
an owned CLI launcher and retains older runtimes; it changes no memory or native
connections. A launcher outside PATH is shown as a full command, not silently added
to shell settings. Missing receipt fields are displayed as None.

After CLI success, optionally choose one Codex connection. Its preview executes
the verified new runtime to prepare **its own embedded plugin**, then asks for a
second confirmation before native activation. CLI success and any later native
failure are reported separately. The default leaves connections unchanged; a
successful update requires a fresh native session. Use this release journey when
updating runtime and plugin together. The older local-artifact Codex journey above
still previews the running toolkit's embedded plugin without executing the selected
artifact; its `--binary` flag is not a release-package selector.

If activation was interrupted, inspect the retained phase and pending record and
reapply its exact reviewed plan through the [CLI recovery interface](interface.md#apply-a-reviewed-cli-installation).
The menu does not guess a replacement recovery plan or delete ambiguous state.
The guarded publication workflow is implemented under #11; it is not permission
to publish. Retained-candidate nomination, independent acceptance and explicit
release prerequisites remain separate from local installation.

## Foundling journeys

**Foundlings · Manage historical references** uses the selected signet binding.
References do not become current memory merely by being registered or searched.
The submenu pins that signet and its incorporation authorship until Back/exit.
If another process replaces the binding file while a confirmation is open, the
reviewed operation cannot jump to a different bank. Return to the main menu and
re-enter Foundlings to deliberately load a changed binding. Memory and reference
files are still read fresh; this pins selection, not a cached knowledge snapshot.

- **List** checks a page of registrations and local availability without displaying
  document bodies. Use Next for further pages; missing paths are not absent knowledge.
- **Register** asks for a display name/description, local text directory or existing
  standalone Git checkout, portable identity, absolute local path and reason. The
  preview distinguishes portable source/pin from the local-only path and shows
  eligible files/bytes and exclusions. One default-No confirmation registers and
  connects; a failed connection retains and identifies the completed registration.
- **Connect** selects an existing active reference and verifies a local directory
  against its existing pin. It changes only the clone-local connection. A changed
  pin is refused here, not silently adopted as part of moving to another machine.
- **Inspect/search** shows availability and recovery advice. Search and relative-file
  reads display bounded, unreviewed excerpts, file hashes and continuation offsets.
  These actions never promote, journal or synchronize the reference text.
- **Update pin**, inside inspection, previews new content plus the previous revision
  and pin. A reason and default-No confirmation append a superseding registration
  and reconnect the path. Unchanged pins direct the user to Connect instead.
- **Disconnect** previews the exact active registration and requires a reason plus
  default-No confirmation. Original files, connections, historical registrations
  and previously promoted knowledge remain intact.

Conflicting registration heads and invalid local configuration are displayed, not
automatically repaired. Use typed registration/history operations for deliberate
conflict reconciliation. Git sources must already exist locally; the menu does not
clone, fetch, execute source scripts or provision credentials. See
[foundlings](foundlings.md) for eligibility, provenance and authority limits.

## Agent-ready equivalents

The plugin's **The Armorer** skill provides conversational administration: ask
"check my Mandalore setup", "connect these old notes", or "update Mandalore".
It uses structured CLI operations, not menu navigation. **This Is the Way** stays
focused on ordinary recall and learning; administrative guidance loads on demand.

Managed installations generate a skill-local `references/connection.json` with
the retained runtime, binding and native installation paths. Those machine-local
values are ownership-checked projection, not public package or signet content.
The skill works without a global PATH entry and does not assume MCP process
exports are available in the agent shell. Explicit target/override differences
must be resolved before changes. Public unconnected plugins have no such file;
an agent needs a supplied runtime or one on PATH and the missing setup choices.
The skill cannot bootstrap itself before the plugin is installed.

`mandalore connection armorer` is the read-only inspection command;
`connection doctor` remains a compatibility alias. The typed operation is still
`connection_doctor`. A diagnostic request does not authorize repair or sync.
Requested setup/update/repair follows the existing preview/apply contracts;
native authentication and hook trust remain separate, and a structural pass
does not establish live memory or remote health. Shell access is required for
administration, even when memory MCP works.

Use `mandalore operations` for schemas and [the interface](interface.md) for
`signet`, `memory`, `foundling`, `connection`, `release` and `call` commands. Agents should use structured
operations rather than menu keys. Installation operations remain CLI-only; they
do not enlarge the bound memory MCP tool set. Complete JSON receipts are available
there while the menu shows human summaries.

Exit status is 0 on ordinary exit, 1 if an operation/output failed during the
invocation, 2 for invalid flags and 130 for process-context cancellation.
Exiting or cancelling never undoes an earlier completed step.
