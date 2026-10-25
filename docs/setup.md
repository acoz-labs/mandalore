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

## CLI installation and updates

**CLI · Install, update or select a retained runtime** offers the latest published
stable release, a specific version, an explicit local candidate, or a retained
manifest SHA-256. An unavailable release is reported without changing the machine.
The latest published version is [2.0.3](releases/2.0.3.md). This journey is also available directly
as `mandalore release install`, including `--plain` and optional `--prefix DIR`.

The preview shows source and content identities, compatibility, destination,
previous runtime and effects. Confirmation defaults to No. Applying activates only
an owned CLI launcher and retains older runtimes; it changes no memory. A launcher outside PATH is shown as a full command, not silently added
to shell settings. Missing receipt fields are displayed as None.

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

Agents connect to memory through the 2.0 server, not this menu: install the
[Claude plugin](../plugins/claude/README.md) and point it at the server. Its
**The Armorer** skill diagnoses that connection without a shell.

Use `mandalore operations` for schemas and [the interface](interface.md) for
`signet`, `memory`, `foundling`, `release` and `call` commands. Agents should use structured
operations rather than menu keys. Installation operations remain CLI-only; they
do not enlarge the bound memory MCP tool set. Complete JSON receipts are available
there while the menu shows human summaries.

Exit status is 0 on ordinary exit, 1 if an operation/output failed during the
invocation, 2 for invalid flags and 130 for process-context cancellation.
Exiting or cancelling never undoes an earlier completed step.

## Foundling references

The foundling menu distinguishes legacy historical references from canon signet
references. Verify the source identity and bootstrap Git pin before registering
canon, and check reader compatibility. The server refreshes canon on a
schedule; the [foundling guide](foundlings.md) explains its snapshots and honest
stale-cache behavior.
