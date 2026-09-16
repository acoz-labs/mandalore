# Solution Design: corrected-candidate owner eligibility

- **Status:** Final
- **Issue:** #105
- **Planning PR:** #106
- **Repository basis:** 51aee17afec015ba2ad44584f8190b4bb6d901a8
- **Execution envelope:** implementation

## Decision

Add one literal source/artifact pair and its eleven explicit issue IDs to the existing eligibility predicate. Preserve every other control.

## Needs Attention

None for implementation. Human verdicts and release authority remain separate.

## Decision Spotlight

The owner's explicit authorization applies only to the retained corrected pair,
not descendants or all 1.1.0 builds. Preserve old scopes separately. This control
change does not rebuild the product or alter its implementation set.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Exact-head engineering self-review under ADR0003, required CI and reviewed merge;
no product acceptance can be inferred from these gates.
