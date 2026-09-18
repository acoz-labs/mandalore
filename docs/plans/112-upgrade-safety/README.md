# Solution Design: safe deferred native upgrades

- **Status:** Final
- **Issue:** #112
- **Planning PR:** #114
- **Repository basis:** c9e216fa2c9aec443a74a1b8696993780a6c63b2
- **Execution envelope:** implementation

## Decision

Enforce a stopped-session handoff before replacing native Codex connections.
Verified identical replay must not reinstall. Full design and compatibility
decisions are in [Technical design](03-design.md).

## Decision Spotlight

- Defer unsafe replacement by default; explicit acknowledgement is not inferred
  from an idle agent, a plan, a previous receipt or successful inventory.
- Protect legacy sessions by preventing cache deletion, rather than promising
  retroactive hook relocation or full hot reload.
- Guard delegated older runtimes as well as this runtime's direct apply.
- No new process scanner, persistent consent, credentials or background service.

## Needs Attention

No unresolved design blockers. Product acceptance, public release and live
installation remain outside this envelope. Native and rendered evidence is
required during implementation before readiness.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Technical design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

ADR0003 permits exact-head self-review. Required checks must pass before merge;
native and rendered implementation evidence remains a later gate, not a planning
claim. No other roadmap work is included.
