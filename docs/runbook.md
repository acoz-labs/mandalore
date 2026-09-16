# Runbook

The Codex-first [v1.0.0 release](releases/1.0.0.md) is published. Subsequent
development builds, including the Pi work, are not accepted production upgrades.
See the [setup guide](setup.md), [Codex guide](../plugins/codex/README.md) and
[Pi development guide](../plugins/pi/README.md) for the respective boundaries.
Archiving My Friday neither migrates nor removes existing pinned installations.

## Project operations

- Run `bin/ci`; distinguish host, container and native evidence.
- If Actions cannot schedule, inspect `CI_RUNNER`; never route public code to
  a private runner.
- If Projects permission is unavailable, issues remain the backlog. Report board
  work as incomplete and do not expand account authorization silently.
- Scope lives in [product](product.md); current work is in [roadmap](roadmap.md).
- Public reports use synthetic fixtures, never private memory, machine names,
  workstation paths, credentials or raw session transcripts.

## Storage recovery and remaining integration work

### Implemented library behavior

- A busy writer fails without taking over another process's lock. Retry after
  the owner finishes; never remove the lock path while another writer is active.
- Invalid source/revision graphs and serialized size limits are checked before
  sourced publication. Rejected input is not a successful save.
- A genuine disk error after source publication can leave valid orphan evidence
  but no new revision. Inspect source and history before retrying; retries are
  not promised exactly-once. Do not delete orphan evidence automatically.
- Unknown/mixed formats, changed signet identity, noncanonical record paths and
  symlinks fail closed. Preserve data and use an explicit repair/migration path;
  do not rename legacy manifests to bypass validation.
- A fresh Git clone can be read without ignored machine-local state. The first
  write creates its local lock directory; reads do not repair or journal.

These library checks are separate from the installation doctor/repair CLI.
See the [format contract](signet-format.md) for data compatibility boundaries;
connection repair does not rewrite signet records or migrate old schemas.

The development `memory inspect --binding FILE` command validates structure
read-only; it is not a full installation doctor. The [interface contract](interface.md)
describes binding selection, stable errors, retry ambiguity and local receipts.
It has no automatic repair, remote sync or implicit native installation.

Explicit `memory git-init`, `checkpoint`, `sync` and read-only `sync-status` now
provide [the synchronization backend](synchronization.md). If sync is pending,
inspect its phase/head and native origin/auth configuration. If conflicted,
preserve both histories; do not force, reset, remove another writer's lock, or
rewrite evidence to hide it. A malformed local status receipt is not repaired by
inspection. No-save/read-only tasks must not trigger synchronization.

### Legacy conversion recovery

Use [migration preflight/apply](migration.md) only for the supported memory-only
bank format. A refused assistant repository belongs to foundlings, not an in-place
manifest rename. Review source digest, conflicts, exclusions and writer observations;
stop all relevant writers explicitly. An available local lock is not global quiescence.

An apply failure may retain a new staging directory or a published bundle. Inspect
`migration_result.phase`, `published` and recovery paths. A prepared receipt in a
staging directory is not proof of publication. Source data and existing output
are never deleted. Before activation, continue with the original writer/source
for rollback; after new writes, preserve and reconcile both histories. Binding,
Git configuration, native handoff and synchronization remain separate operations.

### Native integration and installation recovery

Current-source builds report verified quota refusals as `release.rate_limited`.
If retry timing is available, wait until every reported lower bound has passed;
availability is not guaranteed. If timing is unavailable, wait before explicitly
retrying rather than looping. Ordinary refusal is not automatically a quota
problem. The installer never borrows native GitHub credentials, sleeps or retries
for you. The original published v1.0.0 has the older generic refusal message.
Bootstrap failures before the CLI starts also retain their separate generic
download error. See [the typed quota contract](interface.md#read-only-release-inspection)
for current-source metadata; inspect the receipt regardless of HTTP status.

For a partial **CLI** activation, preserve the pending record and retained runtime.
The failure screen shows the pending path and retry command. Inspect the record,
resolve the reported cause, then use the original Mandalore executable to run
`release apply < PREFIX/lib/mandalore/pending.json`. This retries the exact recorded
plan with fresh filesystem/byte checks; it does not take over foreign files or
change a memory connection. See [the release recovery contract](interface.md#apply-a-reviewed-cli-installation).

Use `mandalore menu` or the typed `connection` commands to preview, apply,
inspect and repair. Read [setup](setup.md) for exact paths and side effects and
[the connection interface](interface.md) for machine-readable receipts.
Creation, binding and Git initialization can partially succeed; the menu reports
completed steps. Preserve them and inspect before retrying. New banks have no
remote until native Git is explicitly configured.

The installer retains source/runtime copies, validates known ownership, and
refuses edited or foreign registrations. An interrupted replacement may leave
no active registration; inspect the completed phase and previous/target roots.
Repair requires an intact ownership receipt and compatible retained runtime and
binding, then stages a fresh generation. It is not arbitrary in-place repair.
Native cache replacement is possible even when old source/runtime copies remain.
Do not remove a lock simply because it exists or assume rollback happened.

Local-artifact update does not replace the running shell command or configure
PATH; it pins the approved runtime for the native connection. The separate
`release install` journey previews official/local CLI selection and owned launcher
activation, then optionally hands one connection to the verified new runtime.
Exact-candidate acceptance and public release remain separately required.

The development Codex plugin has read-only local hooks and a shared MCP
connection. Read-only describes the hooks themselves, not the whole session:
ordinary confirmed learning remains enabled unless the current user task
prohibits it. When diagnosing a refusal to save, distinguish an instruction
interpretation from unavailable tools or a verified storage error. Do not fix a
prompt ambiguity by bypassing a real user no-save instruction. Restart after
correcting loaded instructions; old context is not evidence for the new wording.

Inspect and trust the exact new hooks through native review, preserving other
hooks. Do not overwrite trust state or use a blanket bypass as installation.
The user may delegate routine menu choices without authorizing private-data
migration, provider account changes or public release.

The Armorer (`connection armorer`, with `connection doctor` retained as an alias)
separates structural checks from login, MCP startup, hook trust, remote
freshness and active-session context. Repair touches only known managed state
and preserves edits/old copies, reporting partial failures.

Local saves do not establish remote synchronization. Offline changes stay local;
Git or semantic conflicts remain visible. Never force-push or reset user memory
to hide a conflict. Inspect after ambiguous write failure before retrying: the
predecessor did not guarantee idempotent writes.

Changing plugin/runtime/connection may need a fresh session. Updated files do
not automatically refresh context already read. An interrupted turn cannot
guarantee persistence of unfinished work.

### Pi-specific diagnosis

Select Pi in the Armorer menu or use `connection armorer --harness pi` with the
actual native profile/executable/state paths. Inspect `error.pi_connection_report`
on failure. Structural success does not establish a loaded connection, model
login or remote access in an already-open session.

For unavailable attachment, inspect the exact retained generation and binding.
Pi guards binding bytes and signet identity on every call; replacement is not
silently adopted at restart. Do not remove guards or fall back to a cwd bank.
Edited or ambiguous packages require investigation. Missing owned files can be
repaired into a fresh generation using the intact retained runtime, preserving
old generations, binding and access mode. Native login remains Pi's task.

Apply failures retain `pi_connection_result` with an attempt path, phase and
uncertain-native-effects flag. The old registration may already be removed.
Reinspect and preview recovery; do not assume rollback or repeat stale plans.
Native registration leaves unrelated profile resources under Pi's ownership.

Tool completion is not proof of delivery. A combined save may be durable while
nested delivery is pending, cancelled or conflicted. Keep the saved ID, inspect
effects and do not repeat the save to retry delivery. Explicit synchronization
can deliver the existing record. A delivered semantic conflict remains unresolved
knowledge, not permission to select the newest timestamp.

Pi reads current local memory each turn without saving or synchronizing. An
externally superseded fact can appear next turn without restarting; previously
read reasoning is not changed mid-turn. Reload or restart is still needed for
changed extension/skill/connection code. Hooks do not promise an exit checkpoint.
