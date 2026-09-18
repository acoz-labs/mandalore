# Solution Design: 1.2.0 retained candidate

- Status: Draft
- Issue: #124
- Execution envelope: implementation
- Repository basis: 947022b1c365c47305e4e2bef1319b8bdaf40ee0

## Decision

Prepare version1.2.0 and an issue-mapped acceptance guide for the completed
recovery/update and memory-control batch. Reuse existing build, verification,
nomination, acceptance and publication tooling; no new release mechanism.

## Needs Attention

The new artifact needs actual human acceptance and separate publication authority.
Earlier candidate-specific owner exceptions do not extend automatically.
Native testing must use retained candidate bytes, not engineering rebuilds.

## Decision Spotlight

Format1 remains supported/default. Format2 is explicit per clone. The old1.1
updater refuses the new manifest; use a verified new executable or reviewed new
bootstrap, without modifying immutable old assets. Installation is not migration.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Exact-head engineering self-review under ADR0003 and passing checks precede
implementation/merge. Product acceptance and publication remain separate.
