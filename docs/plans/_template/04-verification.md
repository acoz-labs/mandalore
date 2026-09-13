# Verification And Release Design

## Test Strategy

Map acceptance criteria and design decisions to failing-first tests at the
lowest useful level. Name likely test files, fixtures, commands, and manual
evidence without duplicating the same matrix at every layer.

Cover happy paths, validation, denial, state transitions, failure/recovery,
concurrency/idempotency, migration/compatibility, accessibility, observability,
and rollback in proportion to risk.

## Red/Green Sequence

Order characterization and implementation tests so each slice begins with a
useful failure for the intended missing behavior.

## Acceptance Evidence

Distinguish deterministic automated evidence, rendered/browser evidence, and
deployed or external-system evidence. Name the candidate identity that must be
preserved through acceptance.

For meaningful UI work, classify the rendered impact using
`docs/operations/ui-acceptance.md`. Define the smallest useful scenario matrix,
the functional and accessibility checks, the visual-regression strategy for
stable high-risk screens, the durable pull-request artifacts bound to the exact
head, and the fresh evidence an independent acceptor must capture from the exact
candidate.
Do not treat a new screenshot or an updated baseline as its own product-design
approval.

## Rollout

Define configuration, migration preflight, feature activation, staging,
acceptance, production promotion, and verification within the selected
execution envelope.

## Rollback And Recovery

Describe application, data, configuration, and external-system rollback. State
what must be retained and which operations are deliberately not automatic.

## Release Prerequisites

List non-blocking implementation prerequisites separately from decisions that
would prevent the plan becoming final.

## Production Readiness Preflight

For release-bearing work, verify the complete path before the final plan gate:
required secret slots exist (never record values), workflow injection is named,
deploy and verification commands are executable, activation is explicit,
rollback is tested or bounded, and the exact-candidate production receipt is
defined. Write `Not applicable` only when the execution envelope cannot touch
production.
