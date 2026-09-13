# Implementation Handoff

## Change Tier And Smallest Complete Outcome

Classify the change and state the smallest user-visible or system-visible result
that must ship coherently.

## Dependency Order And Reviewable Slices

List failing-first implementation slices, likely ownership paths, dependency
order, and an objective exit condition for each slice.

## Acceptance Traceability

Map each acceptance group to its implementation slice and required automated,
rendered, staging, or manual evidence. Link to `04-verification.md` for detailed
cases rather than restating them.

## Documentation Promotion

Nominate the smallest likely durable destinations by purpose: system overview,
capability architecture, contract, security, ADR, development, deployment,
runbook, or user/admin guide. These are implementation inputs; reconciliation
must update them from the behavior that actually ships.

## Pull Request And Review Contract

State branch/PR shape, required checks, design or security review, evidence,
and reconciliation expectations. The implementation PR must promote durable
knowledge and remove this temporary plan before leaving draft.

## Explicit Non-Goals And YAGNI Boundary

Name tempting abstractions, dependencies, mechanisms, and adjacent outcomes
that the implementation must refuse.

## Exceptions That Reopen Design

Stop and return to Solution Design only for a material product-contract change,
contradictory repository or external evidence, a new trust/data boundary,
unplanned irreversible risk, or work beyond the approved execution envelope.
