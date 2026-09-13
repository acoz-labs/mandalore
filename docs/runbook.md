# Runbook

No Mandalore runtime is installed or released by this foundation. Do not offer
an install command until distribution acceptance passes. Archiving My Friday
neither migrates nor removes existing pinned installations.

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

These library checks do not implement a doctor/repair CLI yet. See the
[format contract](signet-format.md) for actual data and compatibility boundaries.

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

### Remaining integration behavior

Doctor separates structural checks from login, MCP startup, hook trust, remote
freshness and active-session context. Repair touches only known managed state
and preserves edits/old copies, reporting partial failures.

Local saves do not establish remote synchronization. Offline changes stay local;
Git or semantic conflicts remain visible. Never force-push or reset user memory
to hide a conflict. Inspect after ambiguous write failure before retrying: the
predecessor did not guarantee idempotent writes.

Changing plugin/runtime/connection may need a fresh session. Updated files do
not automatically refresh context already read. An interrupted turn cannot
guarantee persistence of unfinished work.
