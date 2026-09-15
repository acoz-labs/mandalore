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
