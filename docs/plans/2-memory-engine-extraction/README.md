# Solution Design: memory-only engine extraction

- **Status:** Final
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

Engineering self-review is authorized by the product owner for the MVP effort.
Record the exact reviewed head and findings in #17. Hosted CI remains unresolved
under #19; local checks are not hosted evidence. Implementation can be stacked
on the self-reviewed plan while CI/merge remains pending.

## Decision Spotlight

- Fresh source history; selectively preserve upstream license/attribution.
- Memory format branding must be explicit, not an uncontrolled text replacement.
  New signets use the proposed contract below; existing banks need #8 import.
- Immutable evidence, explicit supersession and visible conflicts remain core.
- Define foundling registration and external-origin citation data now (#12),
  preserving original provenance separately from incorporation. This slice does
  not fetch historical sources or implement the foundling management workflow.
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

This is a contributor-authored, self-reviewed plan under the repository's
[MVP review authorization](../../decisions/0001-mvp-self-review.md), not independent
approval. Resolve findings and record the final full reviewed head in #17 before
implementation. CI and eventual merge remain separately tracked evidence.
