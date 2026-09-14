# Solution Design: retrieval quality and context efficiency

- **Status:** Draft
- **Issue:** #9
- **Planning PR:** Pending
- **Repository basis:** b283e7cc8890d0c2f1da33c649614d25da09aba4
- **Execution envelope:** implementation

## Decision

Add a reproducible synthetic evaluation and measurement suite for the current
scoped lexical memory implementation, then make only evidenced bounded fixes.
Keep measurement separate from product correctness and independent acceptance.
Do not introduce an index, inference service or alternative retrieval dependency.

## Decision Spotlight

- Quality assertions protect current versus historical/conflicting evidence and
  exact scopes; a faster incorrect result is not an improvement.
- Measure fresh service/process reads and warm repetitions separately. Neither
  is an OS cold-cache measurement; do not flush a user's filesystem cache.
- Report bytes, allocations, latency distributions and native call counts in
  their actual units. A requested context budget is a ceiling, not consumed tokens.
- Any guidance tuning must preserve progressive disclosure, no-save behavior,
  provenance access and useful retry after actual change; do not optimize by
  hiding conflict or source-freshness evidence.
- Derived indexing remains a follow-up decision with explicit triggers and
  freshness requirements, not an implementation assumption.

## Needs Attention

Native model behavior is variable and shares inherited tools. Aggregate session
token counters are not isolated Mandalore overhead. Host timing is not a universal
SLA; independent immutable-candidate acceptance remains #10/#11.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)
