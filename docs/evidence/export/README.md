# Scoped export engineering evidence

Issue #80, implementation PR #120. This is synthetic engineering evidence, not
product acceptance or release authorization. No personal signet, account policy,
credential or provider transcript is included.

## Native compiled CLI

On September 18, 2026 the retained driver ran in the designated Herdr lab shell
on macOS arm64 with Node 24.1.0 against source
`b50ed43dba87829da9231306a3189e6ba3c675b1`. The executable was built with the
repository's pinned Go 1.26.4 toolchain, not installed into a user connection.

- Binary SHA256: `22a15a0218e56645a3d3608bf21ec8d0a7a7c67eab0322b133e005b2d15f90b9`
- Driver SHA256: `651da427c03edec891b15f523c5ee0bba8552fd62e818a9fee9e4befdf30283d`
- [Driver](native-driver.mjs) and [retained result](native-result.json).

The earlier source `c3fb593eea3eb74a4ab148a478653eeb7616501b` also passed the
same synthetic scenarios. The final listed run repeats them after adding the
per-chunk cancellation/fault seam and expanded omission tests. Local fixtures
were retained; public results omit workstation and temporary paths.

The driver creates a fresh synthetic signet/binding, records a rename and an
unrelated record/journal, and checks:

- Typed/wrapper preview parity, metadata-only output and current-head selection.
- Reviewed-envelope apply, exact report hash, 0700 directory/0600 report modes.
- No unrelated record, implicit journal or superseded name in default output.
- Explicit history/journal opt-ins and all-field omission leaving only fixed
  item structure, status and report-local ordinals.
- Read-only denial, bank-overlap refusal, no-overwrite replay and a destination
  created after preview remaining untouched.
- Full synthetic bank inventory invariance: bytes, modes and modification times,
  including ignored local state. No Git initialization or network is invoked.

## Deterministic regression coverage

`internal/exportreport` tests exercise strict bounded exact-byte snapshots,
source digest changes, historical-origin validation, selected current/history/
future/conflict behavior, explicit journal/item omissions and invalid selection.
The disclosure audit includes source/origin authors and times, actor/device/model/
session data, IDs/scopes, reasons and opaque extension content. Every effective
omission removes its output group; dependencies close alternate citation paths.

Preview/apply tests cover source/binding drift, canonical and ancestor aliases,
configured disconnected references, missing/unknown reference metadata, existing
or dangling-link destinations, parent/stage replacement, no-replace publication,
cancellation and retained output. A report larger than a write chunk is interrupted
after 32 KiB; the receipt's byte count agrees with the retained partial file.
`internal/foundlings` tests bound configuration enumeration and validate exact
raw metadata without reading original reference contents.

API and CLI tests exercise typed discovery, strict envelopes, binding disagreement,
read-only denial before decoding, partial/cancelled receipts and successful wrapper
execution. MCP regressions retain eighteen everyday tools and exclude export
commands/receipt schemas. Existing memory, synchronization and native regressions
remain part of full CI.

## Limits

Linux no-replace code is cross-built by full CI; this native run is macOS arm64,
not native acceptance of every target. Tests are not a sandbox against an
unrestricted filesystem owner, secure-erasure evidence, automatic secret detection
or permission to publish a report. Source double reads detect cooperative drift;
they are not a transaction spanning arbitrary outside processes. Preview metadata
is sensitive even when report fields are omitted. Review/acceptance/release gates
remain separate.
