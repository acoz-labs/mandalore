# Runbook

Mandalore 2.0 replaces the 1.x Codex, Pi, Claude Code and Hermes connections
with the [server](server.md) and one [Claude plugin](../plugins/claude/README.md).
Retire 1.x clients with the [cutover](migration-2.0.md). A release does not
authorize a live signet schema migration or establish that an operator has
upgraded.

[Mandalore 1.6.1](releases/1.6.1.md) fixes canon refresh timeouts for moderately
sized references. Read its rollback caveat before you select the retained 1.6.0
runtime.

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

See the [format contract](signet-format.md) for data compatibility boundaries.

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

## Canon reference failure

For canon, inspect `foundling_status` or `mandalore server status`.
Keep network/authentication failure, stale verified snapshots, missing cache,
source identity/format failure and registration conflicts distinct. Fix actual
server access or explicit registration problems, then run
`mandalore server canon refresh`; do not
repeat semantic saves to retry transport or manually edit receipts/cache pins.
An unavailable reference is not proof that source knowledge is absent. Do not
fall back to source journals/history to recreate withdrawn records.
