# 0004. Compatible memory withdrawal with same-signet continuity

Date: 2026-09-18

## Status

Selected by the owner; implementation and engineering verification in #81 / PR123.
This decision is not candidate acceptance, public release or live migration authority.

## Context

Memory may cease to be appropriate current guidance without becoming a false fact
or requiring destruction. Withdrawal must travel with the bank, preserve history,
and not silently disappear when an older client reads it. Existing append-only Git
synchronization also needs to recognize a genuine format transition without
loosening unrelated evidence protections.

## Considered options

- Extension flags or sensitivity labels: rejected because old readers can ignore
  the semantics and still disclose the content as current guidance.
- Local suppression: rejected because it does not travel across machines.
- Fabricated corrections: rejected because visibility is not factual revision.
- Deletion/history rewriting: rejected as a different, unauthorized promise.
- Import into a second bank: not selected; the owner chose continuity of signet
  identity, repository, existing records, journals and Git history.

## Decision

Use explicit opt-in format2 in the same signet/repository. Old clients refuse it.
Existing evidence bytes remain intact; new revisions acknowledge the visibility
heads observed by their writer. Separate append-only visibility events record
withdraw/restore, reviewed content heads, parent visibility heads, reason and
authorship. Validate the combined causal graph before disclosure.

Withdrawn, visibility-conflicted and causally unreviewed content are withheld from
ordinary guidance. Restoration is explicit and does not resolve content conflicts.
Neither timestamps, corrections nor successful Git delivery restore a record.
Journals and external reference documents remain independent evidence.

Local upgrade requires reviewed source/binding/checkpoint pins and stopped writers.
Retained preparation supports recovery of the same evidence identity. Activation,
checkpoint and delivery remain separate. Each clone opts in locally; sync accepts
only the strictly proven manifest transition while preserving original evidence.

Retention remains read-only, explicitly selected metadata review under a supplied
policy. No TTL, destructive apply, automatic withdrawal or erasure is included.

## Consequences

Same-bank continuity avoids splitting historical knowledge, but requires compatible
clients and per-clone coordination. The released 1.1.0 updater does not accept a
formats1/2 manifest; a reviewed new bootstrap or verified new executable is needed
before the separate bank upgrade. Do not modify immutable old release manifests.

Offline old copies and already-returned native context cannot be revoked. Explicit
history remains inspectable. These limits are part of the product contract, not
defects hidden by calling withdrawal deletion. Downgrade is not routine recovery.

## Verification and provenance

- Owner explicitly approved the same-repository opt-in recommendation after its
  compatibility and preservation tradeoffs were explained on September 18.
- Discovery PR121, solution plan PR122, implementation PR123; issue81 retains the
  lifecycle links and product acceptance/release gate.
- [Format](../signet-format.md), [synchronization](../synchronization.md),
  [privacy](../privacy.md) and [runbook](../runbook.md) define the durable contract.
- [Engineering evidence](../evidence/withdrawal/README.md) distinguishes compiled
  CLI/native model observations from component tests and product acceptance.
