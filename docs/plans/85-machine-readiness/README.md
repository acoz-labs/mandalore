# Solution Design: inspect memory integration readiness

- **Status:** Draft
- **Issue:** #85
- **Planning PR:** #86
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

No unresolved product or technical choice remains. Product-design self-review
of head `2d8fbe0f241e134bd2d3c4eb15729c485270639a` is recorded in PR #86.
The completed solution still requires exact-final-head engineering review and
passing checks before this draft becomes Final. Native rendered implementation
evidence and new-candidate product acceptance are later gates, not blockers to
the selected implementation envelope.

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

Product direction, bounded static inspection, shared schemas and verification
are specified. Record exact-head solution self-review under ADR 0003, then mark
Final and review that status-change head before implementation/merge. Required
checks must pass before merge. The envelope is implementation only, not live
activation, Pi acceptance or release.
