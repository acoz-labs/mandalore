# Proposed session-level automatic synchronization

Issue #14 delivery plan. The product owner assigned implementation through independent review, native acceptance and verified release. Behavior below remains proposed until implemented and verified. Claude native integration exists; no1.3 release has been accepted. Current installed instructions and permissions remain in force until a reviewed replacement is installed.

## Product contract

An explicitly enabled Mandalore session authorizes automatic synchronization of its selected signet. A disabled session loads no Mandalore skills, MCP tools, hooks or memory context. Disabling skills alone is insufficient. Use a fresh session to verify disabled behavior; disabling cannot remove previously injected context.

Separate content selection from delivery. The agent decides which confirmed knowledge to save and honors requests not to remember or journal particular material. Software synchronizes already-saved records. No transcript harvesting or claim that unsaved model output is durable. A read-only code-review task and a memory-disabled session become separate product concepts; this requires explicit replacement of today's instructions that conflate them. Never silently reinterpret an existing read-only connection as writable/synchronizing.

Remove conversational no-sync as a supported transport mode for the new enabled-session contract. Provide a real native/profile/session disable control. If an in-session request calls for stopping Mandalore, explain and use a supported actual disconnect or restart boundary; do not claim background effects stopped merely because a model acknowledged text. Ordinary remembered content cannot grant or alter synchronization authorization.

## Implementation sequence

1. Implement the revised issue #14 session transport contract. Discovery #55 remains historical; its required transport outcome is included in #14. Retire draft #136 as a wording-only solution; reuse reviewed native integration and correction-ID improvements from #134 and #135. Keep rejected candidate evidence.
2. Add an explicit, versioned session transport policy pinned to the selected binding, signet and owned runtime. Automatic synchronization applies only under that policy. Preserve existing read-only connections and explicit administrative local-only APIs; changing installed permissions requires a reviewed connection update. No signet-format migration is expected.
3. Add one shared synchronization coordinator over the existing Git engine. Serialize/coalesce overlapping attempts per signet; bound runtime/network waits; preserve cancellation, offline receipts, local durability and semantic conflicts. Do not auto-resolve conflicting heads or add a daemon.
4. Wire ordered native lifecycle entry points: startup/resume and before each top-level user turn refresh before assembling memory context. Coalesce adjacent startup/prompt callbacks for the same turn without suppressing checks for later turns. Confirm actual host ordering/cancellation on supported versions. Adapter failure produces an explicit local-only/stale status, not a false freshness claim.
5. Make every successful semantic write in an enabled session enter software-controlled delivery: memories, journals, supersession, withdrawal/restore and applicable promotion/resolution operations. Persist once, then attempt bounded sync; return separate save/delivery outcomes. Accurately declare network/write effects on the active-session tool surface. Preserve local-only administrative command semantics. Do not require the model to select a companion sync tool, and do not duplicate attempts for existing combined operations.
6. Retry pending delivery at the next permitted foreground opportunity, including subsequent turns and session starts. Exit callbacks are supplementary only. No timer/always-on service and no claim that delivery occurs while every harness is closed. Bound online per-turn overhead and avoid repeated warnings for unchanged pending state.
7. Integrate the same contract in Claude, Codex and Pi using their native mechanisms. Claude is the first vertical proof, not a release that leaves other shipping harnesses on conflicting semantics. Align packaged skills, connection plans/receipts, enable/disable controls, Armorer and documentation.

## Acceptance

Use exact retained binaries/packages, actual supported native harnesses and separate synthetic clones connected through a synthetic remote.

- Source agent saves a realistic convention; delivery happens despite no model-selected sync call. A stale destination receives it before its first ordinary answer. Corrections travel in reverse with original record identity and provenance preserved.
- Repeat in an already-open destination at a new top-level turn; exercise resume/compaction and concurrent same-bank sessions without sync storms or interleaved corruption.
- Exercise all mutation paths, offline save and later recovery, interruption after local commit, forced termination before exit callback, and explicit semantic conflicts. Never duplicate a save to retry delivery.
- Prove disabled sessions load no Mandalore context/hooks/tools and cause no Mandalore bank/network activity. Native agent memory is independent. Prove binding retarget protection and separate-bank isolation.
- Verify do-not-remember/no-journal content selection separately from transport of previously saved data; do not confuse model compliance with runtime-enforced disabling. Test existing read-only connections stay read-only and are not silently upgraded.
- Measure startup/per-turn delay, context size and bounded timeout behavior. Verify truthful stale/offline status and absence of transcript copying or native-note resurrection into shared records.

## Release

Independent code review and full configured checks; then build one new immutable candidate, independently accept actual native journeys, publish those same bytes, verify public install/update/recovery and record release provenance. Preserve prior1.2 artifacts and connection rollback. Do not activate or migrate personal signets as part of product acceptance.

## Honest guarantee

Software performs a bounded synchronization attempt at defined entry/write boundaries when Mandalore is enabled. Successful delivery/refresh is evidenced by receipts; network failures remain pending and visible. This does not guarantee network availability, automatic semantic agreement, retroactive removal of existing model context, or that a model saves every useful fact.
