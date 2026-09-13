# Handoff

1. Inspect the pinned binding/MCP source and installed SDK schema/transport code.
2. Add failing binding/input/equivalence tests; implement machine-local binding
   and shared operation types/discovery/errors.
3. Implement CLI, local stdio MCP and signal handling; run protocol/equivalence
   tests and actual binary flows in the designated testing pane.
4. Promote the actual interface, binding, errors, limits and examples into durable
   documentation. Reconcile the exact PR head and remove this temporary pack.

Keep the implementation in feature/shared-memory-interface with Refs #4 and
related #3 provenance/binding evidence. Self-review follows the recorded MVP
authorization. Do not silently include sync, native plugins or the TUI: coordinate
their tracked slices and keep #4 open until all of its criteria are verified.

Durable destinations: docs/architecture.md, docs/interface.md, docs/signet-format.md,
docs/development.md and docs/runbook.md; generated discovery/schema data should
be authoritative rather than duplicated hand-maintained examples.

Reopen design for unexpected SDK input looseness, mandatory assistant dependency,
unbounded/cancellation-unsafe I/O, implicit bank selection, live migration or a
new external service/credential requirement.
