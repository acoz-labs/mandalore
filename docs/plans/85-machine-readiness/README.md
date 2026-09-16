# Solution Design: inspect memory integration readiness

- **Status:** Draft
- **Issue:** #85
- **Planning PR:** Pending
- **Repository basis:** b28224827e0749ac4b27cd2fee807526ccf9ecc6
- **Execution envelope:** implementation

## Decision

Add a bounded non-executing assessment shared by the CLI and Armorer menu.
Separate support contracts, observed setup and scenario-specific evidence.
Offer a sanitized follow-up prompt and a separately selected existing native
inspection. Do not change ordinary memory tools or silently repair a machine.

Discovery authority is [PR #84 at reviewed head 12418fc](https://github.com/acoz-labs/mandalore/blob/12418fc7dd9f75a565d0ef2b3fb921d11b646088/docs/discovery/15-machine-readiness/README.md).
Exact-head engineering self-review follows ADR 0003; product acceptance and
publication are separate. The shipped 1.0.0 artifacts remain immutable.

## Needs Attention

Before final review, resolve static receipt selection and component identity
matching without executing native binaries. Confirm bounded file access and
cancellation seams against the actual code; do not reuse a full binding open
that traverses remembered content. Finish the rendered product-design review
before implementing the menu. No implementation is authorized by this draft.

## Decision Spotlight

- The default is non-executing, not an implicit invocation of existing Armorer.
  The discovery measured native Codex inventory creating temporary directories.
- Supported, configured and verified are separate; unknown is not failure.
- A generated prompt is guidance for a subsequent authorized task, not consent
  to install software, access credentials, synchronize or rewrite native state.
- Exact-artifact evidence is separate from a build's own metadata; no circular
  self-hash declaration or source/version-only compatibility certification.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

This draft requires completed product/solution design, resolved bounded static
inspection seams, a recorded PR number, exact-head ADR 0003 review and passing
checks. The envelope is implementation only, not live activation or release.
