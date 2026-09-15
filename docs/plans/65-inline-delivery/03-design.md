# Design

Inputs contain record (the existing memory.Write) or entry (existing journal
kind/summary), plus optional timeout_seconds default 3, range 1–30. Validate the
complete shape and timeout before calling the existing save implementation.
Cancellation before execution or failed save never starts synchronization.

Return saved (the existing durable local receipt) and delivery (typed existing
sync success/error information). After publication, a failed/busy/cancelled sync
still returns the saved identity. Outer success means a save completed, not that
remote delivery did; inspect delivery's exact state/head/delivered/error fields.
Preserve sanitized error codes, phase and inspect-before-retry semantics. Do not
retry the save or delivery automatically. Lost outer transport remains ambiguous.

Reuse shared save helpers and existing API/synchronizer error mapping instead of
duplicating Git or classifying raw command output. Synchronization reacquires the
existing writer lock and may include concurrent/prior valid local work, as normal
sync already does. No transaction, rollback or exclusivity across both operations
is claimed. Local save time is separate from the bounded delivery attempt.

Expose both operations via generated catalog/MCP schemas and the ordinary memory
CLI dispatch. Update MCP orientation's existing sync-only delivery claim. Keep
skill entrypoint guidance concise and route delivery details progressively.
