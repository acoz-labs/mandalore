# Revised 1.1.0 candidate — engineering checkpoint

This separate evidence branch records actual retained-artifact tests, not human
acceptance, release authority or a personal installation update. The original
criteria and remaining matrix in the [candidate guide](../../../releases/1.1.0-candidate.md)
remain applicable. Published v1.0.0 is unchanged.

## Exact candidate

- Source: `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`, the reviewed PR #92 merge.
- [Build](https://github.com/acoz-labs/mandalore/actions/runs/35060663575), attempt 1;
  retained artifact `10432790497`, 70,380,945 bytes.
- Archive SHA-256: `417b331ea598bb312744900eccd67e96ca458cd3fe7bf7d27e47c7b1c61b380b`.
- Manifest SHA-256: `ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`.
- [Transport](transport.json) and [verification](verified.json) capture provenance
  and every payload checked before extraction/execution. Observed expiry:
  2026-12-15T05:43:33Z. Reverify availability/provenance before promotion.
- [Nomination](https://github.com/acoz-labs/mandalore/actions/runs/35060900748)
  independently verified the same archive. [Marker receipt](nomination.json)
  binds all eight release-bearing issues to this identity; none is accepted.
- [Package inspection](packages.json) confirms actual macOS arm64 runtime 1.1.0,
  source above, protocol/format 1, and embedded Codex/Pi package identities.

The earlier `6f9bb5e` candidate remains retained as counterexample evidence. Its
first-error rendering failure motivated PR #91/#92; neither that failure nor its
other passing results are relabeled as results for this candidate.

## Packaged rendering and retrieval

[Rendering receipt](rendering.json) captures four fresh Codex 0.154.0,
gpt-6-astra/high conversations using this verified runtime and packaged skill:
two error-first runs, ordinary current/history recall, and historical/current
reference consultation. Existing synthetic fixtures were read-only; complete
inventories (paths, bytes, types, modes and mtimes) remained unchanged.

All four read the selector reference before their first memory response. Every
printed memory envelope matched its complete wire envelope, including errors
and provenance, with zero duplicated outer wrappers. The unchanged selector
also passed 21 synthetic checks. These observations do not guarantee every model
or client will follow the guidance or reduce total conversation cost.

| Case | All tool-output bytes | Tool-code bytes | Final-request input tokens |
| --- | ---: | ---: | ---: |
| Error A | 14,008 | 1,871 | 19,155 |
| Error B | 14,514 | 1,962 | 19,391 |
| Ordinary recall/history | 46,746 | 1,711 | 26,571 |
| Historical/current distinction | 69,970 | 2,848 | 32,599 |

Those output totals include non-memory tools. The receipt separately records
selected memory-envelope bytes and aggregate usage; aggregate native usage is
not a Mandalore-only context measurement. Prior observations varied, so this is
not a blanket whole-session savings claim.

The historical answer was correct and distinguished an unapproved old proposal
from adopted current decisions. Pins, source identity, locator, unreviewed label,
truncation and complete relevant evidence were preserved. **The no-overlap
subcheck failed:** after a 512-byte search preview with `next_offset: 512`, the
agent requested the whole 1,332-byte file at offset 0, repeating 512 bytes. This
is a #57 efficiency finding, separate from #56 wrapper fidelity. It is not a
clean progressive-continuation pass. Do not erase it by rerunning for a better
answer; assess it alongside the original #57 quality/cost matrix. Only the local
path in the published answer's citation target was normalized to `operations.md`;
the recorded measurements use original outputs.

## Delivery boundary checks

[Delivery receipt](delivery.json) records five retained-CLI scenarios with fresh
synthetic signets and local bare origins:

- Both combined operations rejected invalid inputs and enforced read-only mode
  without changing fixture inventories.
- A missing Git repository retained the saved record and reported delivery
  failure without implicit initialization.
- No origin reported local-only; an unavailable origin reported pending. Both
  preserved durable local receipts.
- Explicit synthetic remote recovery delivered already-saved content without
  changing semantic content; actual local and remote heads matched.
- Concurrent revisions delivered with one visible conflict and no authoritative
  current winner. The journal receipt remained durable; neither branch was
  silently selected or discarded.

These are real CLI checks, not native model judgment or hosted authentication.
Fresh [native callback and lifecycle checks](lifecycle.md) add cancellation,
pre-publication writer contention and ambiguous Git push acknowledgement.
These are not model-selected recovery or rendered Escape-key tests.

## Initial cross-harness continuity

[Native receipt](initial-native.json) verifies fresh Pi recall of Thursday in
synthetic bank A and Saturday in B for the same project scope. Neither prompt
named Mandalore. Both read-only runs preserved complete fixture inventories.
An ordinary confirmed change, without a remember command or special cue, was
saved and delivered by Pi: Friday superseded Thursday on the same record, with
Pi authorship and stable device attribution. Bank B remained unchanged.

The actual operation sequence was `memory_sync`, `memory_recall`, then
`memory_remember_and_sync`. There was one pre-recall freshness attempt and no
redundant post-save synchronization. The existing cross-machine freshness
guidance permits such a pre-read check; this observation is **not** the minimum
two-operation learning case. Actual local/remote heads matched the combined
receipt at `fdcd7266ac06bd38ae1ac783c357e6739ec07c0e`, with no semantic conflicts.

Fresh Codex in another project directory recalled Friday and the former Thursday
via scopes, recall and history, preserving the complete synthetic fixture. These
cross-harness runs verify semantics, not Codex model-visible wrapper deduplication;
the separate four rendering runs above supply that evidence. Native versions are
Codex 0.154.0 and Pi 0.85.1, model gpt-6-astra/high, driver Node 24.1.0. Existing
provider login was inherited in place, not copied into test profiles. Local bare
origins do not establish hosted authentication or cross-machine networking.

[Reverse-continuity receipt](continuity.json) records four more native scenarios:

- Pi answered a language question containing the quoted phrase without any
  memory operations; complete fixture inventory remained unchanged.
- Codex corrected Friday to Monday using one local-only `memory_remember`.
  It retained the same record and supersession chain, made no sync/journal call,
  and preserved both Git heads, journal inventory and the unrelated bank.
- Fresh Pi recalled Monday replacing Friday, read-only, with scopes independent
  of the project directory and no mutation to either bank or remote.
- A direct consolidation cue with no new facts used two recalls and one sync,
  delivering the pending correction without adding/changing records, journals,
  foundlings or provenance. The other bank was unchanged. Actual local/remote
  heads and receipt matched `5497d608fb52289bbe614d1377555b7befcb85d0`.

The three revisions retain fixture → Pi → Codex authorship on one stable device.
These are individual observed outcomes, not a statistical reliability claim or
deterministic lifecycle synchronization. Explicit restrictions belonged to test
requests; ordinary usage is not put into a default read-only mode.

## Remaining gates and evidence boundary

The [lifecycle checkpoint](lifecycle.md) adds native startup/reload/new/resume/fork,
a live same-session model refresh, 18 delivery/cancellation calls and seven
contention/acknowledgement-boundary calls. Its manifest independently binds
actual runtime copies, package identities and final local/remote heads.

The [readiness terminal checkpoint](readiness/README.md) adds seven actual retained-
binary recordings and eleven static cases, including both harnesses, narrow/plain/
no-color, missing/malformed/unsupported/stale states, default-No, native handoffs,
EOF/Ctrl+C and terminal restoration. It retains an alias-path diagnostic limitation
and a controller error separately from the clean canonical-path observations.

The [quota checkpoint](quota/README.md) adds eight passing fresh source-bound
recordings for quota/refusal/partial/default-No/Back presentation and fixture
request-count/retry regressions. This uses test executables compiled from the
exact clean source, not the retained release executable. An initial wrong-cwd
bootstrap test failure is retained separately, and #77's narrow-command wrapping
limitation remains visible.

The [upgrade/recovery checkpoint](upgrade/README.md) adds actual published
1.0-to-retained-1.1 CLI and Codex connection updates, a real permission-induced
partial activation, same-plan recovery, no-write replay and compatible retained
rollback/re-upgrade. Two fresh retained-candidate recordings cover rollback
default-No and narrow plain approval with the native handoff declined. All 32
protected fixture entries and old generations survived unchanged; structural
native checks do not imply fresh-session authentication/MCP behavior.

This is a partial engineering checkpoint. The full candidate matrix still
requires additional lifecycle, retrieval-quality, rendering/failure,
native connection failure/recovery and the remaining original candidate-guide
matrix. Earlier-head evidence is a baseline, not exact-candidate acceptance.
The 512-byte repeated preview remains an explicit observed limitation.

No raw sessions, private machine paths, authentication or personal memories are
published. No other-platform native, screen-reader or alternate-locale claims
are made. Engineering self-review does not supply a human verdict or extend
the earlier MVP-only acceptance exception. No tag, public release or live
activation has occurred.
