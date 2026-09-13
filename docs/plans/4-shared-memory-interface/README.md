# Solution Design: shared memory interface

- **Status:** Final
- **Issue:** #4
- **Planning PR:** #21
- **Repository basis:** dc6add6ef854f39de900cc0599bffacd8f598e11
- **Execution envelope:** implementation

## Decision

Expose the merged memory library through an agent-ready CLI and local stdio MCP
server with shared typed operations, defaults, receipts and error semantics.
Keep setup administration explicit and separate from a running server's fixed
signet binding. This is not the interactive installer or native plugin.

## Decision Spotlight

- Explicit machine-local binding outside the Git-backed signet; never infer a
  bank from cwd. Default binding selection may use MANDALORE_BINDING or the
  platform user-config directory, with explicit flags taking precedence.
- A binding pins signet ID, root, enrolled device and attribution actor. Replacing
  the bank fails rather than silently following the path to different memory.
- One operation catalog for machine-readable discovery, CLI and MCP adapters.
  The existing agent supplies interpretation; there is no new model service.
- A server is bound to one signet for its lifetime. Create/bind administration is
  available as noninteractive CLI commands, not cross-bank MCP mutation tools.
- Mutation receipts mean local publication, not remote delivery. Cancellation
  before mutation prevents writes; cancellation/I/O ambiguity after publication
  must not produce a fictional rollback or invite blind duplicate retries.
- Sync remains a #5 dependency of complete #4 acceptance. Do not advertise a fake
  sync operation or claim this first interface slice completes that criterion.
- CLI text/JSON flows are self-reviewed here; the interactive TUI is #7.

## Plan Map

- [Context and user flows](01-context.md)
- [Decision](02-decision.md)
- [Technical design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Use the owner-authorized MVP self-review policy. Record the exact reviewed head
in the planning PR, resolve findings and retain CI evidence before implementation.
No independent product acceptance, live installation or release is implied.
