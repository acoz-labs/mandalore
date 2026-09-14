# Retained-candidate synthetic migration checkpoint

Contributor verification for #8/#10, not independent acceptance or live migration.
The actual retained macOS ARM64 executable ran in the designated Herdr pane:
source `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`, binary
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
It was not rebuilt, and earlier implementation evidence was not relabeled.

## Fixture and observed results

The committed synthetic `internal/migration/testdata/bank-v1` fixture was copied
to a fresh owned directory. It contains 10 files, 11 directories, two devices,
two sources, two revisions in one record, one prior journal and one source-change
observation. Its content identity is
`67cde56aa83bf99c03576922d17a2d910ebdb838308c8d5e13b1ab4e21de7ad1`.

1. Preflight left every source file hash and directory path unchanged and created
   no output bundle. Native inventory was explicitly not tested in this run; an
   absent legacy lock was not presented as proof of idle writers elsewhere.
2. Apply without the writer-stopped acknowledgement returned exit 2. Read-only
   apply also returned exit 2 with `operation.read_only`. Neither published.
3. The driver deliberately changed only the synthetic README after preview.
   Apply refused the stale plan at preflight, reporting no publication/write;
   the changed source was preserved without repair. Its exact original bytes
   were restored before the successful conversion.
4. Explicit acknowledged apply published the new bundle. Every source file and
   directory still matched the original baseline; `bundle/original` matched that
   same snapshot, including empty directories. No real memory was involved.
5. Replaying the successful plan refused the existing output without overwriting
   it; full bundle file/directory fingerprints stayed unchanged.
6. An explicit assistant-format fixture was refused by preflight without output
   or alteration to its `agent.json`. No manifest rename was used to bypass it.
7. A separate bind enrolled `Example Writer` with a new device, distinct from the
   conversion audit device and both historical devices. No Git initialization,
   sync or native activation occurred.
8. Recall returned Silver Heron. History retained Copper Finch, the exact
   superseding revision, original dates and desktop/Codex versus laptop/Pi
   authorship. Both bank-wide scopes became signet scopes while preserving the
   opaque `bank-migration-demo` ID. The raw history retained
   `9007199254740993123456789.123456789` exactly; no floating-point reserialization
   was used to publish that history.
9. Journal retained the original event and a distinct conversion event authored
   by `Example Migrator`/CLI. Read-only recall/history/journal left the converted
   signet unchanged. The unrelated native-test bank also retained all baseline
   file hashes.

## Evidence and limits

`history.json`, `recall.json` and `journal.json` are unmodified synthetic CLI
outputs. Other JSON files are explicitly selected projections of actual receipts
that omit local paths/bindings; their fields are not reconstructed operation
results. The source/original fingerprints contain only fixture-relative paths.

This verifies the bounded public memory-only conversion and refusal cases on the
retained macOS ARM64 artifact. It does not prove arbitrary legacy-format support,
Git-history transfer, off-device backup, cross-machine writer quiescence, every
filesystem crash boundary, duplicate-plugin inventory or native handoff. Those
must not be inferred from this successful data conversion. Existing implementation
tests retain broader synthetic fault coverage, not independent candidate acceptance.
