# Lifecycle event probe — discovery evidence

Observed 2026-09-15 with Codex 0.154.0 on macOS arm64, model
`gpt-6-astra`/high. This is contributor discovery evidence for #55, not a
Mandalore synchronization implementation or product acceptance.

## Method and limits

A disposable directory contained a Node.js 24.1.0 command hook which read stdin
and appended only event name, opaque session/turn IDs, event source/trigger/reason,
permission mode, tool name and boolean presence of prompt/transcript fields.
It returned `{}`. It did not retain prompt text, paths, tool arguments or tool
outputs; it did not invoke Git or Mandalore. Each handler had a two-second budget.
All nine handlers used the same reviewed command, registered through session
configuration for SessionStart, UserPromptSubmit, PreToolUse, PostToolUse,
PreCompact, PostCompact, Stop, Interrupt and SessionEnd.

First, noninteractive `exec --ignore-user-config` completed the requested constant
answer but produced no probe events. That launch did not prove hook support or
failure: no trusted configuration was established for its new handlers.
Next, the interactive native UI requested directory and hook trust. The operator
reviewed the exact command and trusted the nine authored probe definitions using
the user's existing synthetic-test authorization. No blanket trust-bypass flag
was used. Native trust receipts persisted for these session-flag definitions.

The interactive UI also listed existing user/plugin hooks despite an attempted
Mandalore plugin-disable override. Therefore this is **not an isolated plugin
acceptance test**. Personal memory tools were never requested; native existing
context hooks were not treated as evidence of the probe's implementation. CLI
read-only flags were supplied, but observed hook `permission_mode` was
`bypassPermissions`; this discrepancy remains unresolved. No sandbox enforcement
claim rests on these observations. All test prompts prohibited memory mutations
and sync. Actual model actions were constant answers and two disposable sleeps.

## Observed sequence

Aliases below preserve equality relationships without publishing native IDs.
All events have the same session ID, including after resume.

| Action/event | Turn ID | Observed detail |
| --- | --- | --- |
| SessionStart | absent | source startup |
| UserPromptSubmit → Stop | T1 | Constant answer; no tools |
| PreCompact → PostCompact | TC | Manual `/compact`; TC differs from T1 |
| SessionStart | absent | source compact; before next prompt |
| UserPromptSubmit → Stop | T2 | Explicit no-save/no-sync instruction; no tools |
| UserPromptSubmit | T3 | Request one `sleep 20` timing probe |
| PreToolUse | T3 | Bash, before sleep |
| New instruction submitted while sleep runs | — | Explicit no-sync/no-save, no new commands |
| PostToolUse | T3 | Sleep finished after approximately 20 seconds |
| UserPromptSubmit | T3 | Queued instruction delivered **after** PostToolUse |
| Stop | T3 | Agent acknowledged the restriction; no retry |
| SessionEnd | absent | `/quit`, reason other, normal process exit |
| SessionStart | absent | Explicit resume of the same synthetic session |
| UserPromptSubmit → PreToolUse | T4 | Second `sleep 20` interruption probe |
| Interrupt | T4 | Escape during the command |
| SessionEnd | absent | `/quit` after interruption |

For the timing probe, PostToolUse was logged at 19:19:02.039 UTC and the queued
UserPromptSubmit at 19:19:02.107 UTC. Herdr had confirmed the new prompt submission
while the agent was still working. Thus this was not a prompt entered after the
tool ended. Both prompt events used the same native turn ID. No Stop or
PostToolUse for the interrupted second command appeared before the final exit.

Permission mode was unchanged across ordinary and explicitly prohibited tasks.
Manual compaction and SessionEnd did not supply that field in this run.
SessionEnd had neither a prompt nor a turn ID. These facts are consistent with
the documented fields, but runtime observations—not documentation alone—establish
the sequence above. See the [official hook contract](https://learn.chatgpt.com/docs/hooks).

## Consequences for the proposed design

1. **Native turn ID is not a prompt generation.** Steering can reuse it, so an
   allow lease keyed only by session/turn may survive a new restriction.
2. **PostToolUse is not a safe permission-refresh boundary.** It can precede the
   newly queued prompt's hook. Do not attach unconditional sync or consume an
   earlier turn-level permission there.
3. **Manual compaction is not the prior task's turn.** Do not infer authority
   from the last seen turn, or interpret missing authority as permission.
4. **Exit/resume require fresh state rules.** Exit lacks turn identity, and resume
   preserves session identity. A session-keyed allow file would need additional
   freshness and invalidation proof.
5. **Permission mode is not semantic consent.** It cannot distinguish no-sync
   tasks even when native tools are broadly allowed.

The findings rule out the naive session/turn cache and post-tool sync designs.
They do not yet prove a safe replacement or imply that any live sync occurred.
Next investigate a fresh prompt-generation handshake with fail-closed state and
inline write-and-deliver semantics. Explicit consolidation can remain an agent
decision and finish with bounded sync without requiring a new memory write.

Automatic compaction, failed/disabled invalidation hooks, concurrent sessions,
subagents, hard process crash and the real Git timeout/delivery matrix remain
unverified here. Existing synchronization tests do not prove this lifecycle
authorization contract. No release or installed Mandalore runtime was changed.

## Follow-up: can the transcript establish a fresh prompt generation?

A second, fresh synthetic session on the same native version used a separately
reviewed command hook. It read at most the trailing 2 MiB of the native transcript
and retained only event metadata, byte count, latest user-message timestamp and
SHA-256 digests. The native `response_item` / `message` / `user` shape selected
the latest user entry; `turn_context` supplied a comparison turn ID. No transcript
body, prompt text, tool arguments or workstation path was retained by the probe.
This parser is an investigative instrument, not proposed product code or a stable
native interface. Partial trailing lines were excluded and read/parse failures
would be recorded as unavailable. All observed reads were below the bound.

The operator reviewed and trusted the exact nine session-local handlers, with
two-second budgets. Existing native hooks were also present: the same isolation
limitation applies. No Mandalore tool was called. Tasks were a constant answer,
one `sleep 20`, a queued prohibition, manual compaction and normal exit.

| Event | Native turn | Latest user entry visible in transcript |
| --- | --- | --- |
| First UserPromptSubmit | A | Earlier setup context, not current prompt |
| First Stop | A | First prompt |
| Sleep request UserPromptSubmit | B | First prompt, not sleep request |
| PreToolUse | B | Sleep request |
| PostToolUse | B | Sleep request, not queued prohibition |
| Queued prohibition UserPromptSubmit | B | Still the sleep request |
| Stop after acknowledging prohibition | B | New prohibition |
| PreCompact / PostCompact | C | New prohibition; transcript turn remains B |
| SessionEnd | absent | New prohibition; transcript turn remains B |

The queued instruction was submitted while the sleep was running. PostToolUse
read the previous prompt at 19:50:59.554 UTC. The new prompt's own hook ran at
19:50:59.624 UTC and still saw that previous transcript entry. The new user entry
was timestamped 19:50:59.650 UTC; Stop saw it at 19:51:05.279 UTC. Every observed
UserPromptSubmit had a mismatch between its prompt digest and the latest user
entry in the transcript. The native model acknowledged the prohibition; no extra
tool, save or synchronization was performed. The process exited normally.

This reproduces the old-prompt window with transcript evidence. Reading the
transcript does not repair the post-tool freshness problem. Stop seeing the
current prompt in this case is encouraging but not proof of a fail-closed
authorization protocol across disabled/failed hooks, subagents, crash/resume,
truncated transcripts or native upgrades. No claim that all lifecycle approaches
are impossible follows from this result. Prompt-generation identity remains
unresolved; no permission-cache implementation is selected.

## Inline-delivery comparison basis

The successful new-learning case in the [consolidation matrix](consolidation.md)
used separate native code-mode requests for save and sync, not one batched code
cell. The saved receipt reached model context at 19:40:30.127 UTC; the model
requested sync at 19:40:32.858 UTC, a 2.731-second interval. This is one observed
decision/round-trip gap, not a latency benchmark or an observed lost delivery.
Both operations ultimately succeeded. A combined save-and-deliver operation
could remove that second model decision after dispatch, but cannot promise
delivery through cancellation, offline state or a save the model never requested.
Native operation selection, schema/context cost and suppression still require
before/after verification before claiming the proposed improvement.
