# Verification

Use failing-first synthetic fixtures derived from the pinned public format;
do not use an actual user's bank. Verify shared API and compiled CLI in the
designated Herdr pane under pinned Go 1.26.4.

- Exact schema/namespace inventory: bank/signet/binding/lock/plugin/env mappings.
- Refuse assistant, mixed/unknown versions, bad IDs/paths/fields, oversized trees,
  symlinks/devices, overlap, existing output and malformed plans without mutation.
- Preflight creates no files/directories/locks, does not execute old binaries or
  change Git/native state, and labels untested session/network boundaries.
- Busy lock and changed source/binding/plan refusal; absent lock is not falsely
  called quiescent. Native inventory distinguishes enabled integration from active
  writer and reports potential duplicate legacy/current connections.
- Preserve all record/revision/source/device IDs, original attribution/dates,
  journals, orphan evidence, branched supersession and explicit conflict state.
  Only the supported manifest/scope mapping changes; other scopes stay intact.
- Original snapshot bytes and declared exclusions match the source inventory;
  operational provenance is retained outside current signet knowledge.
- Faults before/during publication, no overwrite on retry, returned staging or
  published paths, source unchanged and usable for rollback. No implicit binding,
  Git initialization, native switch, synchronization, credential or shell edits.
- Read/recall/history the converted bank and compare semantics. Verify explicit
  new binding uses a new device while historical authors remain original.

Run full bin/ci (race tests, vet, four builds, public-content checks) and actual
compiled migration against a synthetic old bank in the sibling pane. Exercise
the native memory inventory read-only against an explicit test profile. Changes
to engine validation require all existing integrity/graph/retrieval/sync tests.

## Production Readiness Preflight

The execution envelope is implementation, not live migration or release. No real
memory, credentials, native writer replacement or private Git hosting changes
are authorized by this plan. #10/#11 must associate migration evidence with the
exact immutable candidate. Archive status is not backup or runtime retirement.
Owner live-data adoption remains an explicit later operation; synthetic success
does not authorize it. A post-activation rollback may require reconciliation,
not deleting newer history or switching blindly to a stale source.
