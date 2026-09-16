# Solution Design: privacy and lifetime contract

- **Status:** Final
- **Issue:** #79
- **Planning PR:** #82
- **Repository basis:** e78e08329335f753adce821979541bf77d879d91
- **Execution envelope:** implementation

## Decision

Promote #16 O1 into a single user-facing privacy guide and focused regression
tests of existing behavior. No runtime policy or schema changes.
Approved discovery: [PR78](https://github.com/acoz-labs/mandalore/pull/78),
reviewed head b0e6880bc5f1e6f2085e26a093625ba69e2a0abb.

## Needs Attention

None blocking implementation. Pending runtime acceptance/release elsewhere in
the roadmap is not resolved by this docs/test delivery.

## Decision Spotlight

Sensitivity labels remain descriptive. Ordinary confirmed learning stays enabled.
No-save prohibits mutation, not disclosure of existing content. Corrections do
not erase history. Export #80 and withdrawal #81 remain deferred designs.

## Plan Map

[Context](01-context.md), [Decision](02-decision.md), [Design](03-design.md),
[Verification](04-verification.md), [Handoff](05-handoff.md).

## Final Gate

ADR 0003 authorizes a distinct exact-head contributor self-review. Required
checks precede merge. No independent acceptance, personal activation or release
is implied by this implementation-only envelope.
