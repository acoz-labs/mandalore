# Decision

Add memory_remember_and_sync and memory_journal_append_and_sync through the shared
CLI/MCP catalog. Both are non-idempotent mutations and explicitly network-capable.
Existing memory_remember and memory_journal_append remain byte-compatible in their
input schemas/defaults and retain local-only network annotations.

Use combined operations for ordinary useful learning only when synchronization
is allowed; use old local-only tools under no-sync. Read-only/no-save prohibits
both. Keep standalone sync for existing pending delivery, explicit consolidation
without new facts, and later authorized recovery. No implicit Git initialization,
credential work, background service, cached authority or transcript parsing.

No owned rendered UI change: this is an agent-facing typed operation and guidance
change, verified behaviorally rather than through graphical design or screenshots.
