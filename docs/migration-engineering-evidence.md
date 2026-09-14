# Memory-only migration engineering evidence

This is contributor engineering verification for #8 / PR #30, not independent
product acceptance, live-user migration or release authority.

## Exact tested implementation

- Implementation source: `c1b6d9f0f4458c2cfda16d1d23e7bc6c29e9ff30`.
- Committed macOS arm64 executable SHA-256:
  `9905fadf22fd5284f860f5c8d7fcb9dce0540512612508ddbb43e2c2858a586a`.
- Go 1.26.4, host-local route described in [development](development.md).
- Native inventory: Codex 0.153.4, explicitly selected empty synthetic profile;
  native binary SHA-256
  `b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.
- Hosted implementation CI:
  [34800587289](https://github.com/acoz-labs/mandalore/actions/runs/34800587289), passed.
  Its Ubuntu runner executed the CLI, migration/filesystem and synchronization
  Go tests natively. This is Linux engineering evidence, not a Linux Codex session
  or independent product acceptance.
- Actual compiled CLI and all build/test commands ran in the owner's designated
  sibling Herdr pane. No live memory or auth fixtures were copied.

The final PR reconciliation separately binds its final head. Documentation and
temporary-plan removal after the source above do not change the tested runtime
code. Exact immutable-candidate acceptance remains #10/#11.

## Synthetic scenario results

The committed [legacy bank fixture](../internal/migration/testdata/bank-v1/README.md)
contains two original devices, two sources, two revisions with one supersession,
one journal event and one Git-index change observation. Its ten portable files,
eleven directories and 2823 bytes produced source digest
`67cde56aa83bf99c03576922d17a2d910ebdb838308c8d5e13b1ab4e21de7ad1`.
That digest frames the sorted file digest and portable directory inventory; it
is not a Git commit or a claim about excluded runtime state.

Observed with the committed executable:

- Preflight completed without altering the source; original fixture comparison
  passed. The native adapter listed the empty selected profile, reported no
  potential memory writers and explicitly left active sessions/other machines
  unverified. Native-owned logging is not represented as Mandalore configuration.
- Explicit apply published a new bundle. The source still matched the committed
  fixture; `original/` matched the source recursively, including empty directories.
- Separate explicit binding enrolled a new writer device, distinct from both
  historical devices and the conversion audit device. No Git initialization,
  remote, synchronization or native plugin switch was performed.
- Recall returned Silver Heron. History retained Copper Finch and its superseding
  rename, original dates and desktop/Codex versus laptop/Pi authorship. Both bank-wide
  scopes became `signet` with the original opaque bank ID.
- History output retained the exact extension number
  `9007199254740993123456789.123456789`; body prose containing legacy names remained
  unchanged. This was verified in the raw CLI JSON, not a float-decoding viewer.
- Reapplying the plan refused the existing bundle at preflight, returned
  `published:false` / `write_may_have_occurred:false` for that attempted operation,
  and did not overwrite the previous successful conversion.

Local receipts and binaries are retained in a disposable synthetic test directory.
They are not public acceptance artifacts or a proposed user installation location.
No rendered UI changed: this slice uses existing JSON CLI/admin patterns, not a
new menu or plugin surface. Rendered UI evidence is therefore not applicable.

## Automated coverage and review findings

Failing-first tests cover snapshot validation with no filesystem fallback; current
graph/source/device/journal rules; preserved branches and orphan evidence; exact
JSON numbers; project/account/task byte preservation; required/unknown/duplicate
fields; unsupported/mixed formats; source/path/size/count bounds; FIFOs and
symlinks; old-binding identity; shared/exclusive lock behavior; acknowledgement,
cancellation and read-only refusal; source changes before/during apply; retained
staging on I/O/cancellation; no-replace publication races; and published output
retained when the final parent sync fails. Human and typed administration share
the same contract; migration details are excluded from bound MCP tool schemas.

Review found and corrected:

- A typed JSON wrapper probe rejected the CLI's own successful plan envelope.
  The shape probe now uses a generic object followed by strict typed decoding.
- Empty directories were initially absent from the snapshot fingerprint. They
  are now fingerprinted and retained in the original snapshot.
- The existing engine reader rounded opaque extension numbers during retrieval.
  It now uses JSON numbers, with an independent history regression test.
- A misplaced root `.gitkeep` initially passed preflight but failed on-disk
  validation. Preflight now refuses it before creating output.
- Full-directory enumeration could allocate before enforcing the source census
  bound. The converter now reads directory entries in bounded batches.
- A failure-injection test compared macOS path aliases rather than canonical
  paths; correcting the fixture made the intended parent-sync failure observable.

Full local `bin/ci` passed: formatting, module verification, race-enabled tests,
vet, public-content/workflow checks and darwin/linux amd64/arm64 builds. Existing
memory and real local-Git synchronization regressions were included after the
shared-validator and number-reader changes.

## Deliberate limits

Local file/lock observations cannot prove session quiescence on another machine,
and installed/enabled plugins are not proof that a session targets this bank.
The issue criterion and docs state that distinction; explicit writer-stop
acknowledgement remains required. The converter is not a defense against an
administrator concurrently rewriting its owned staging tree or filesystem.

The original snapshot excludes Git history/configuration, local runtime state,
OS metadata and external binding/authentication. It is not an off-device backup.
Prepared receipts do not prove publication when retained in staging. No automatic
rollback after new writes is provided; preserve and reconcile both histories.

Native Linux Codex-session behavior, real-memory adoption, native writer
handoff, independent product acceptance and release are not established here.
Foundling consultation/promotion remains separate work, required before adopting
arbitrary historical assistant memory.
