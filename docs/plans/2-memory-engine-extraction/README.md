# Solution Design: memory-only engine extraction

- **Status:** Draft
- **Issue:** #2
- **Planning PR:** #17
- **Repository basis:** 0c904334b55534a3255af613c32de3759dabdc98
- **Execution envelope:** implementation

## Decision

Selectively extract the tested memory engine and define its signet contract
with #3. Preserve semantic behavior and tests, not the assistant package tree.
This first implementation delivers a tested library and format documentation;
it does not install a native plugin, migrate a user's memory or publish a release.

## Needs Attention

Independent maintainer review and product approval on the final exact planning
head are pending. The foundation and planning branch are published, and the
template-derived public project board is configured with all successor issues.
Do not mark this plan Final or issues Ready until the actual gate is satisfied.

## Decision Spotlight

- Fresh source history; selectively preserve upstream license/attribution.
- Memory format branding must be explicit, not an uncontrolled text replacement.
  New signets use the proposed contract below; existing banks need #8 import.
- Immutable evidence, explicit supersession and visible conflicts remain core.
- No legacy capability, account, hook-script or assistant execution framework.
- No live migration, native configuration change or release in this envelope.
- No new vector/index service: preserve scoped lexical behavior first.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

This is a contributor-authored draft, not an independently approved plan.
Record the planning PR and final full head after review, resolve findings,
obtain product authority and merge the plan before beginning implementation.
