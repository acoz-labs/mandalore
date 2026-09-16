# Solution Design: 1.1.0 candidate preparation

- **Status:** Draft
- **Issue:** #88
- **Planning PR:** Pending
- **Repository basis:** 55b1e89c6e8f9c18d5fa6814274df3e87d3db240
- **Execution envelope:** through-staging

## Decision

Use the existing artifact workflow for a cumulative 1.1.0 candidate containing
the completed roadmap outcomes. Make only version and acceptance-handoff
documentation changes; do not add product behavior or change release authority.
Artifact verification/nomination replaces staging; no staging service exists.

## Decision Spotlight

1. Use 1.1.0 for additive APIs/Pi support, not another candidate labeled 1.0.0.
2. Build once after the reviewed version change; test/accept/promote those bytes.
   No production release or live personal installation is authorized here.
3. Lifecycle O1/O2, export/withdrawal and Claude remain unfinished. Independent
   implementations can be accepted without claiming the entire roadmap is done.

## Needs Attention

Human acceptance and publication remain later gates. The MVP owner exception
does not cover this candidate. No blocking engineering design unknown remains.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

ADR 0003 engineering self-review binds the exact planning head and required CI.
This applies to bounded release integration of the approved roadmap, not a new
acceptance policy. Keep #88 and included delivery issues open after merge.
