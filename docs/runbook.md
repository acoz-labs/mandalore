# Runbook

The [v1.1.0 release](releases/1.1.0.md) is published with Codex and Pi support.
Subsequent development builds, including format2 withdrawal, are not accepted
production upgrades.
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

For privacy incidents, start with [memory lifetime and scoped incident guidance](privacy.md#accidental-secret-or-sensitive-content-save).
Rotate/revoke an exposed credential first without printing it into a new report.
Corrections and foundling disconnection preserve historical evidence. A local
deletion does not erase Git, remote, backup or native-session copies; append-only
sync is not a redaction transport. Review exact cleanup targets, backup exposure
and recovery authority separately. Do not force-push, prune history or remove
evidence as a routine repair. Current-source export and withdrawal preserve source
history; neither is erasure. Normal authorized learning remains enabled.

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
inspection. Legacy no-save/read-only tasks must not trigger synchronization.
Enabled-session transport follows its reviewed policy independently of content
selection; disable the full integration to stop its activity.

### Format2 activation and visibility recovery

First install a trusted compatible runtime on each intended machine. The released
1.1.0 updater cannot validate a new manifest declaring formats1/2; use the reviewed
new-release bootstrap or verified new executable, not a modified old manifest.
This installs tooling only and does not authorize bank migration.

Stop writers to the selected clone, including the current agent if it uses that
bank. Do not claim stopped writers merely because a local lock is available.
With an explicit binding in JSON, call `signet_upgrade_preview` and review its
identity, clean Git checkpoint, manifest and inventory pins. Then call
`signet_upgrade_apply` with that exact plan and `stopped_writers: true`. See the
[cataloged inputs](interface.md) instead of editing `signet.json` manually.

Activation preserves original records, journals, signet identity and Git history.
Its receipt reports evidence publication, activation and local durability
separately; no checkpoint or delivery is performed. If interrupted, inspect the
partial `upgrade_result` and retained preparation. Explicit
`signet_upgrade_recover` continues the same plan/evidence identity and cannot
start a new upgrade. Changed binding/source/HEAD pins refuse recovery. Preserve
the files for diagnosis; do not delete the receipt, downgrade the manifest or
blindly repeat apply.

After successful activation, start fresh compatible sessions. Synchronization is
a separate authorized action. A format1 clone seeing an upgraded remote reports
`upgrade-required`; review and upgrade that clone locally before retrying sync.
Independent upgrade receipts can converge. No operation upgrades every offline
copy or retracts content already read into a native session.

For an explicit withdrawal or restoration, inspect `memory_visibility_history`
and submit exact current content/visibility heads. `memory.stale_heads` requires
fresh inspection, not guessed retries. On a failed mutation, inspect
`visibility_result`, the event ID and history before doing anything again: a
cancelled call may already have published. A correction is not restoration;
concurrent visibility decisions remain withheld until explicitly reconciled.
Never use a new record or foundling promotion to bypass a withdrawal.

Retention review uses CLI-only `retention_preview` with explicit policy and
selection. It neither writes nor applies anything, and its metadata may still be
sensitive. There is no retention cleanup/TTL command. Journals and external
sources must be considered separately, not inferred from record selection.

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

For initial or new-machine checks in current-source builds, start with
`connection assess --harness codex|pi` or **The Armorer → Assess this machine**.
It works with missing setup and performs no native execution, network or product
writes. Read support, setup and scenario evidence independently; completed is not
an overall healthy verdict. Use the reported next step rather than treating
unknown/historical evidence as a repair instruction.

For partial findings, correct invalid explicit paths or inspect the named metadata
problem, then reassess. Select a retained root explicitly if its identity matters;
do not guess the latest generation. A matching receipt/runtime does not prove
active native registration, authentication, memory graph validity or remote
freshness. Optional follow-up guidance contains no repair authority. See the
[bounded assessment contract](interface.md#non-executing-machine-assessment).

For deeper native inspection, choose it separately after reviewing the selected
program/profile and possible native logs/cache writes. Menu confirmation defaults
to No. CLI `connection armorer` / `doctor` retain their existing executing
behavior; `--read-only` does not suppress native log/cache effects. Failed native
inspection does not repair, retry or change the earlier static assessment.

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
The menu prints its recovery command as one unstyled logical line, separately
from wrapped receipt fields. Select the whole command; do not copy wrapped path
fields or insert newlines where your terminal visually wraps. Terminal clipboard
behavior varies. Automation should use the machine-readable pending path and
its own argument-safe invocation. Unsafe display characters suppress the command
rather than silently changing its meaning. The command is guidance, not permission
to retry before inspecting the failure.
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

Enabled native sessions use a pinned policy for automatic refresh before context
and delivery after semantic writes. Existing legacy hooks remain local-only;
read-only connections keep their restrictions. Distinguish content-level requests
not to remember from disabling the complete integration. A model acknowledgment
of “no sync” does not disconnect an enabled runtime; use the native disable
boundary and start a fresh session. Existing model context remains independent.

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

### Deferred Codex upgrades

Native marketplace replacement can delete cached hook scripts and skills used
by existing sessions, even while an old MCP process remains alive. A healthy
MCP alone is therefore not proof that an in-session update is safe.

When apply reports phase `deferred`, keep using the old connection or finish and
exit every session using the selected profile. An idle/paused agent still counts.
From a separate shell, apply the reviewed plan with
`mandalore connection apply --sessions-stopped < plan.json`. For an approved
repair, add `--sessions-stopped` to `connection repair --apply`. The acknowledgement
is an operator assertion for this invocation, not automatic session detection or
permission inherited from a previous install. Then restart and review native
hook trust/MCP startup. No hot reload is promised.

In the menu, the separate handoff question defaults to defer. Declining it does
not undo an already-completed CLI update. CLI update and plugin activation are
separate outcomes. Agents running inside the affected profile should leave the
standalone command for the user, not acknowledge their own session as stopped.

A later failure with `registration-removed`, `marketplace-registered` or another
partial phase is not the same as a safe initial deferral. Inspect native inventory
and retained receipts before retrying; neither automatic rollback nor cache
resurrection is attempted. Ownership, stale-plan and edited-file refusals still
apply even with stopped-session acknowledgement.

## Export failure and partial output

Export preview is read-only; it does not synchronize to establish remote freshness.
If freshness matters, perform separately authorized synchronization before choosing
the source. Never export a user's bank merely to test installation health.

On `export.failed` or cancellation, inspect `error.export_result` when present.
It reports phase, staging/destination/report paths, bytes written, publication and
durability separately. A stage or published report may remain even though the
operation failed. Paths name the locations selected at the time of the attempt;
external directory moves can make those locations stale. Preserve evidence and
inspect filesystem identity before any manually authorized cleanup.

Mandalore never deletes partial exports, rolls them back or retries automatically.
A published-but-not-durable receipt is not a claim of successful persistence.
A retained stage is unconfirmed output, not an accepted report. Do not publish it
without review. A retry requires a fresh destination and newly reviewed preview;
the same plan cannot replace an existing artifact. Source/binding changes require
new preview, not editing hashes in the old one. No provider errors or source text
are needed in a public diagnostic issue.

Malformed or unavailable configured-reference paths require explicit inspection,
even for disconnected references; do not bypass the guard by deleting local
configuration. Export neither repairs references nor reads their contents for
destination validation. Prior reports, Git history, backups and native/provider
copies remain outside this operation's effects. An unrestricted filesystem owner
can still alter produced files; these guards are not a sandbox against that owner.

## Pi-specific diagnosis

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

Pi assembles current memory context each turn. An enabled session first attempts
refresh; legacy connections read local records without synchronization. An
externally superseded fact can appear next turn after successful refresh;
previously read reasoning is not changed mid-turn. Reload or restart is still needed for
changed extension/skill/connection code. Hooks do not promise an exit checkpoint.
