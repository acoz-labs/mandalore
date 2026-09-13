# Solution Design: {{TITLE}}

- **Status:** Draft
- **Issue:** #{{ISSUE_NUMBER}}
- **Planning PR:** Pending
- **Repository basis:** {{REPOSITORY_BASIS}}
- **Execution envelope:** Pending

Allowed execution envelopes are `implementation`, `through-staging`, and
`through-production`. The final approval authorizes only the named envelope and
the repository's existing release policy.

## Decision

State the selected solution and why it is the smallest coherent path.

## Needs Attention

List release prerequisites, residual risks, and exceptional decisions. Write
`None` when no item remains.

## Decision Spotlight

List the non-obvious, consequential choices that shape the shipped product,
including UX defaults, data ownership, automation, permissions, privacy, and
trust-boundary behavior. For each, state the selected default and the reason.
Write `None` only when the plan contains no such choice.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

The plan may become `Final` only when the issue is still the approved product
outcome, the complete pack has received independent maintainer review, no
blocking unknown remains, the planning pull request number and full repository
basis are recorded, and the execution envelope is explicit.
