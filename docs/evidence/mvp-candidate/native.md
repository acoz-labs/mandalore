# Native candidate learning and recovery checkpoint

These are fresh contributor-operated Codex tests of source
`5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
They supplement, rather than relabel, the preceding CLI/MCP checkpoint and older
candidate evidence. This is not independent product acceptance or a release.

## Actual setup and execution

The retained candidate's macOS ARM64 executable was hash-checked and applied
through its normal connection plan/apply flow. The existing local marketplace
identifier was validated using the plugin helper, and the managed generation
provided its deterministic version suffix; candidate bytes were not edited with
a timestamp cache-buster. Installed plugin validation and doctor passed.

New connection generation:
`a0e6484fcab76aa5917a19e88f54ae96ddab07b64dd2d513fc7475c94fbe9221`.
Runtime SHA-256:
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
Previous generation `d3d0de91e763012c2626630002fd77b9894d32a7c29ad88e882be4d7e14b7cec`
and runtime remained retained; every previous synthetic-bank file hash matched.
The native cache was normally replaced by the managed update. No authentication
was copied/enrolled, shell configuration edited or real bank migrated.

The fresh synthetic signet is `signet-095bdee00fc7e0b3352bfe88af2c3d6d`, with
explicit device label `Test Mac A`, actor `Example User` and device
`device-4a1ab3cd984d27760d738c3f5dd64876`. A local bare Git fixture was explicitly
prepared for delivery tests; it is not a second machine. Native attribution says
`codex`; unknown model/session fields in stored memory remain null, not invented.

All interactive launches ran in the designated pane using Codex 0.153.4, binary
SHA-256 `b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`,
existing native authentication/resources and the existing gpt-6-astra/high model.
The native banner confirmed YOLO mode. The 0.154.0 update offer was skipped; the
two empty synthetic project directories were explicitly trusted. No replacement
native home or Mandalore runtime/binding environment exports were used.

The native `/hooks` UI showed both Mandalore hooks active and trusted for the new
managed source, with their five-second timeout. The unrelated user SessionStart
hook stayed active and unchanged. No hook-trust bypass flag or trust toggle was
used. Structural doctor does not by itself establish these native observations.

## Learning, correction and explicit cue

An ordinary prompt said the fictional project was Velvet Comet and that project
status updates should use two short sentences. It did not say remember or use the
consolidation cue. The agent loaded the installed memory skill, saved the project
and preference, and synchronized. It did not create a journal on that first turn.

A normal follow-up renamed the same project to Nimbus Finch. The agent recalled
the existing scope `velvet-comet`, corrected the same entity record with the exact
predecessor, appended a short rename journal and synchronized. It did not create
a competing project or derive a new scope from the display name.

A third prompt added decision-first explanations followed by one brief reason,
directly said “This is the way,” and requested consolidation/journaling. The agent
preserved the existing facts, saved the new preference, appended one semantic
consolidation journal and synchronized. There were three records, two revisions
of the project record and two journal events at the final checkpoint. Local and
bare-remote heads both matched `614b26ad07ac17779af438daadc9f81629435db7`.

## Compaction and read-only continuity

A requested baseline command recorded all bank-file SHA-256 values outside the
bank. Native `/compact` completed and emitted one compaction event (56.435 seconds).
The following read-only question asked for current/former project name and both
format preferences. Two MCP recalls, one project and one bank-wide, returned the
correct answer. Their semantic text envelopes were 879 and 1,296 UTF-8 bytes.

All bank files, including local Git/receipt files, matched the baseline through
compaction, recall and session exit. Both project directories stayed empty.
This is actual native compaction recovery, not a recreated conversation summary.

## Unavailable binding and visible hook failure

After the session closed, the test driver temporarily moved only the disposable
binding file to a retained sibling path. No runtime, managed hook/cache or bank
file was changed. A fresh native session in the second project directory reported
MCP initialization failure. On a read-only unrelated arithmetic prompt, the native
UI displayed the Mandalore memory-unavailable hook warning twice (the two lifecycle
events) and answered 56. There were no tool calls, repair, memory writes or fallback
bank access in that turn. Every bank file remained unchanged.

The driver restored the original binding byte-for-byte. A new session in the
second directory started without the failure. A read-only prompt asked for the
remembered project names/preferences and explicitly treated a quoted “this is the
way” as non-operative. The agent used two recalls and history, answered correctly,
and made no save/journal/sync calls. It stated that remote freshness was not checked.

This demonstrates an unavailable-binding hook warning and MCP-startup failure,
not every possible handler crash or timeout. It does not claim arbitrary hostile
source immunity or silent recovery without restoring the missing configuration.

## Interrupted work

In that recovered session, the agent started a requested script that only slept
45 seconds, with explicit no-write/no-journal/no-retry direction. Escape produced
the native conversation-interrupted state. The observed sleep PID was still alive
19 seconds after start; it later ended. Escape did not prove subprocess termination.

The next prompt ended the test and requested only a read-only project-name recall.
The agent made one MCP recall (879-byte envelope) and returned Nimbus Finch. It
did not resume/retry the command, poll it, repair warnings or write/synchronize
memory. After session exit, all bank files still matched the original pre-compact
baseline, and the previous bank and both project directories were unchanged.

## Measurements and remaining scope

[Native summary](native-summary.json) preserves actual memory-operation sequences,
turn durations, compaction/interruption counts and native aggregate counters, but
no transcript. Every recorded recall operation has a measured text envelope;
the collector handles ordinary and labeled parallel-result wrappers. Aggregate
native counters include inherited context and compaction, not isolated Mandalore
token cost. Two-scope recall and history for an explicit rename-history question
are not categorized as unnecessary calls without evidence.

The full bank hash window covers all read-only native sessions and exits; no
automatic synchronization was performed just to make the final state look clean.
`native-sync-status.json` reports a prior delivered head/time, not proof that an
arbitrary remote remains current forever. Here the explicit local bare fixture's
head was also inspected. All three native sessions are closed; the new synthetic
connection remains installed and its prior generation is retained.

The subsequent [live reference and recovery checkpoint](live-recovery.md) covers
native foundling adaptation, same-process externally refreshed reads and bounded
corruption/uncertain-push/offline/concurrent recovery scenarios. It intentionally
changes this synthetic bank; the snapshots above remain the earlier checkpoint,
not its latest state. Physical-machine/platform coverage and the complete
exact-candidate rendered install/update/repair journey remain. Independent
acceptance and public release require their separate gates.
