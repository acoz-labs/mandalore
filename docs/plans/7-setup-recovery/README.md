# Solution Design: everyday setup and recovery

- **Status:** Draft
- **Issue:** #7
- **Planning PR:** Pending
- **Repository basis:** 5c55bbdd8b50b21d610274f9d7479417ca967671
- **Execution envelope:** implementation

## Decision

Provide a small native-terminal menu over the existing memory API and explicit
machine-local installation commands. Embed the Codex package in the CLI; stage
content-addressed runtime/plugin copies and use native plugin management.
Preserve the user's agent, signet and native authentication ownership.

## Decision Spotlight

- Opening a menu, preview and structural doctor do not mutate the signet or
  connection. Every menu mutation has an effects preview and default-No apply.
- Install/update/repair share one validated connection plan, not three divergent
  scripts. Agent-ready CLI JSON uses the same functions without a TUI.
- Machine labels are explicit user input, not automatic hostname publication.
- One native profile has one default Mandalore connection. Other banks remain
  explicitly selectable; installing a second default replaces only a proven
  managed registration after preview, not unrelated plugins or native resources.
- Update accepts an explicitly selected trusted local artifact first. Published
  release discovery/integrity is #11, not an invented latest-release contract.
- Doctor reports structural facts separately from untested authentication, hook
  trust, live MCP readiness, remote freshness and already-loaded model context.

## Needs Attention

Independent immutable-candidate acceptance and public release remain #10/#11.
The currently installed development marketplace is not a managed installation;
replacement requires its own explicit ownership decision during native tests.
No personal signet is migrated or selected by this plan.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Record the actual planning PR and an exact-head self-review under AGENTS.md and
decision 0001 before implementation. Engineering review is not independent
product acceptance.
