# Solution Design: Prepare presentation before the first result

- **Status:** Final
- **Issue:** #56
- **Planning PR:** #91
- **Repository basis:** 6f9bb5e4d3b79a1c5cf4947b428dcd5b31da0a41
- **Execution envelope:** implementation

## Decision

Return the failed first-error scenario to design as required by PR #58.
Preserve its Mandalore-only, compatible, best-effort selector design; make
preparing that selector an explicit prerequisite of the first raw code-mode
Mandalore call, including errors. Do not change the wire protocol or harness.

## Needs Attention

Native adherence is not guaranteed by wording. Retain the failing candidate
observation and test this sequencing hypothesis before merge. Acceptance,
publication and personal activation remain separate, unchanged gates.

## Decision Spotlight

- Put the conditional prerequisite near the skill entry point, before recall
  instructions can lead to a first call. Keep the detailed selector conditional.
- Do not retry a memory call to improve presentation, especially after a write.
- Preserve conservative fallback and full error/provenance evidence. Reduced
  duplicate payload bytes do not guarantee lower total context or latency.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Exact-head engineering self-review is authorized by ADR 0003. Required checks
remain mandatory; neither self-review nor this plan is product acceptance.
