# Save and delivery by installed mode

For an enabled-session policy, ordinary semantic save tools persist once and
software attempts bounded delivery automatically. Inspect the original result or
partial error receipt plus top-level `session_sync`; memory_context also includes
its synchronization attempt. Local durability is not proof of remote delivery or
semantic agreement. Combined tools remain compatible and make no duplicate attempt.
Do not resubmit a save to retry delivery; inspect after ambiguous responses.
Do-not-remember/no-journal governs content selection, while already-saved content
continues to synchronize until the actual native session disable boundary.

The instructions below apply only to legacy connections without enabled-session
policy; do not use them to reinterpret an active session's authorization.

# Save and delivery are separate outcomes

When both are allowed, `memory_remember_and_sync` accepts the normal knowledge
fields inside `record`; `memory_journal_append_and_sync` accepts journal fields
inside `entry`. Both have optional `timeout_seconds` (delivery only, default 3).
The server saves locally, then makes at most one bounded sync attempt.

Outer `ok: true` confirms a save, not remote delivery. Keep `saved.id` and any
`saved.record_id`; inspect the complete `delivery` envelope. A successful delivery
requires its result to report `delivered: true`; pending, local-only, conflict and
errors retain their normal meanings. Do not resubmit the save because delivery
failed. If the response is lost, inspect before retrying: the save may exist.

No additional sync is needed just because a combined call returned. For several
useful saves together, local-only calls followed by a final combined save can
coalesce delivery. Do not add a journal or fact just to cause an attempt. When no
new save is warranted, use the existing standalone sync if authorized.

Under no-sync, use the local-only tools. Under read-only/no-save, use neither.
If an older connected runtime lacks companions, use its existing local save and
one authorized short-budget sync; do not install or switch runtimes automatically.
These choices express current task intent, not permission for future hooks.
