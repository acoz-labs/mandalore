# Solution Design: efficient anonymous release operations

- **Status:** Draft
- **Issue:** #74
- **Planning PR:** Pending
- **Repository basis:** 6e008087a25daf5a0db6686f1d2669ae5930677f
- **Execution envelope:** implementation

## Decision

Retain verified manifest/checksum bytes in a private, operation-owned inspection
result. Reuse them for staging and full verification, never across commands or
from serialized input. Preserve source refreshes before and after staging and
the independent safe-download interface. Add bounded typed quota advice at the
HTTP boundary and preserve operation receipts in API/menu errors.

Approved discovery: [exact reviewed head](https://github.com/acoz-labs/mandalore/blob/ba87c86a6b16e95aa6684a1b853ad39c53e48bd3/docs/discovery/54-release-quota/README.md),
merged in #73. Product-design shaping is recorded on #74 under ADR 0003.

## Needs Attention

Exact-head engineering review and required CI precede merge. Native synthetic
rendered evidence is required for implementation. No public release, personal
installation or nominated-candidate acceptance is authorized by this plan.

## Decision Spotlight

- No authenticated fallback, persisted cache, automatic retry or sleeping.
- Private original bytes, not reserialized JSON, preserve manifest identity.
- Retry timing is advisory and bounded; unknown timing stays unknown. A normal
  403 is not necessarily quota exhaustion.
- Shell bootstrap keeps its current trust boundary and two asset downloads;
  its pre-runtime failures are not falsely described as typed quota errors.
- The final source check and post-probe byte verification remain mandatory.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

ADR 0003 authorizes a separate recorded exact-head engineering self-review in
place of independent engineering approval. Record the planning PR and final
head, resolve blocking findings, and pass required checks before merge. A
self-reviewed planning head permits stacked implementation. This is not product
acceptance and does not expand the implementation execution envelope.
