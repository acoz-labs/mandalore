# Solution Design: native Codex memory

- **Status:** Final
- **Issue:** #6
- **Planning PR:** #25
- **Repository basis:** a0f53601abebb2dd46fab2b15f1ca426d14a391f
- **Execution envelope:** implementation

## Decision

Ship one Codex compatibility plugin, one implicit memory skill, and a bounded
read-only native hook adapter over the existing signet binding and memory API.
Use the supported scaffold layout; do not introduce an assistant launcher.

## Decision Spotlight

- Hooks read local state only. They never synchronize, save, parse transcripts,
  classify user authorization, or invoke another model. The current task decides
  whether the skill may sync and learn.
- Direct user consolidation intent is interpreted in conversation, not detected
  by a substring in arbitrary event or retrieved text.
- Use a machine-local absolute runtime override and the existing local binding;
  no source checkout or private configuration travels inside the plugin.
- Keep compatibility packaging for this first pinned Codex integration. The new
  portable manifest is not required to share the harness-independent MCP engine.

## Needs Attention

Native hook trust is a user-controlled gate. Existing memory integrations must
be inspected before activating another; no removal or live bank migration is
authorized. Physical-machine and immutable-candidate acceptance remain #10.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Exact-head self-review is authorized by AGENTS.md and decision 0001. Record the
review and checks without representing engineering self-review as independent
product acceptance.
