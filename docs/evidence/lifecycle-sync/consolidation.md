# Explicit consolidation delivery

## Intent clarification after candidate verification

The #100 follow-up distinguishes absence of useful new content from an explicit
task restriction. Saying there is nothing new to remember or journal does not
itself mean read-only/no-save/no-sync. On a direct cue, make the permitted bounded
delivery attempt without filler; a missing connection is a limitation to report,
not an implicit successful completion. No-journal alone prohibits journaling,
not an otherwise authorized delivery attempt. Explicit read-only/no-save/no-sync
restrictions remain unchanged and take priority.

This is shared Codex/Pi guidance, not a deterministic hook or permission parser.
The retained 1.1 counterexample and its unchanged pending work are preserved in
[the engineering failure receipt](https://github.com/acoz-labs/mandalore/blob/d4e80dedf68e5fd430c7eec5eee7eba413773bb8/docs/evidence/candidates/1.1.0-b894586/engineering/failures/consolidation.json).
Historical passing observations below are not acceptance of this clarification;
fresh native verification is tracked in #100. No package is activated implicitly.

## Original implementation evidence

Contributor engineering evidence, 2026-09-15, for #61 / PR #63, selected outcome
O3 of #55 / discovery PR #60. Planning PR #62 reviewed
`4d2355ddde26c270e7256edd7759941014781f35`, merged as
`f8500ccb5f29a74aede577eebc7e5cb36a00f725`. This is not independent product
acceptance, a personal installation update or permission to release.

## Identity and reproducible fixture

Codex 0.154.0 on macOS arm64, `gpt-6-astra` with high reasoning. Baseline skill
is the pre-change skill at `4584af179073312f7abaf8695713d31c74863298`;
candidate skill is from `58962e2dc4197fefbb6d6b573b53320f9b6a04e7`.
Both use the same retained development executable, SHA-256
`b0f972e94b122b37fd14ddf4799ba20a8fe3dada88418fdd7ca50303c9d82327`, built
from `e829a6c7e11e04eb4901583fff8735f63855abd5`. Runtime code is unchanged by
this skill-only correction. This is an externally identified, unstamped
development binary, not a nominated artifact. The separately loaded workspace
skill, not the binary's embedded old skill, is the treatment under test.

Each case creates a separate synthetic signet through supported CLI commands,
initializes Git, configures a disposable local bare origin and synchronizes the
initial state. It then remembers fictional project Quartz Robin and checkpoints
without pushing. Assert local HEAD differs from origin main before the session.
Capture a SHA-256 inventory of tracked signet files and both commit identities.
The workspace contains only a link to the selected `.agents/skills/this-is-the-way`.
No real hosting remote, personal bank or credential fixture is involved.

Launch Codex exec with `--ignore-user-config`, `--enable code_mode`,
`--skip-git-repo-check`, JSON events and a final-answer file; use `-a never`,
`-s read-only`, explicit model/reasoning and the synthetic workspace. Unset
`MANDALORE_BIN` and `MANDALORE_BINDING`; disable the installed Mandalore plugin
with a CLI override, and explicitly configure the test executable's stdio MCP
server with the case's binding. The MCP server is writable: the shell sandbox
does not make this external service read-only.

The first candidate encountered native MCP approval refusal before server
invocation. Preserve that result rather than treating it as delivery. Fresh
matched baseline/candidate and subsequent cases use documented per-tool policy:

```toml
[mcp_servers.mandalore.tools.memory_sync]
approval_mode = "approve"
[mcp_servers.mandalore.tools.memory_remember]
approval_mode = "approve"
[mcp_servers.mandalore.tools.memory_journal_append]
approval_mode = "approve"
```

These are invocation-local controls for the synthetic server only, not production
configuration changes. See [Codex MCP configuration](https://learn.chatgpt.com/docs/extend/mcp).
Existing native authentication/resources can be inherited; no claim of full
native-home isolation, lifecycle-hook acceptance or cross-client coverage.

## Native outcomes

Baseline and candidate receive the identical prompt:

> There is no new knowledge or useful journal entry to add from this conversation.
> This is the way.

| Case | Observed memory operations | Tracked content / delivery |
| --- | --- | --- |
| Original baseline | None | Unchanged; prior commit remains pending |
| Initial candidate, approval refused | One attempted sync; not dispatched | Unchanged; unverified delivery correctly reported |
| Fresh authorized baseline | None | Unchanged; prior commit remains pending |
| Fresh authorized candidate | One sync, budget 3 seconds | No new commit or memory; existing HEAD delivered |
| Phrase explained, explicitly not invoked | None | Unchanged; no delivery |
| Direct cue plus explicit no-sync | None | Unchanged; prior pending work untouched |
| Direct cue plus read-only/no modifications | None | Unchanged; prior pending work untouched |
| Direct cue, unavailable origin | One sync, budget 3 seconds | Unchanged; pending/fetch, committed locally, not delivered |
| Remember fictional hiking route Birch Loop plus cue | Two recalls, one remember, one sync | One new fact, no journal; new HEAD delivered |
| Direct cue, MCP connection disabled | None | Unchanged; missing connection reported, no setup/fallback |

The quoted case asks what the phrase means and explicitly says it is not being
invoked. The no-sync case prohibits delivery of previously pending work too.
The read-only case prohibits saving, journaling, checkpointing, repair and changes
to earlier pending work. These negative cases retain a writable MCP connection
and the same per-tool approvals: absence of writes is agent behavior, not server
rejection. No fallback shell writes appear in those completed native event logs.

For unavailable-origin testing, move only the disposable bare origin to a retained
sibling location before launch, leaving the configured origin unavailable. The
retained origin's main ref remains unchanged. The tool returns `ok: true` with
`state: pending`, `phase: fetch`, `checkpointed: true`, `delivered: false`.
The agent correctly distinguishes a completed attempt from successful delivery,
does not retry and does not add filler content.

For successful cases, compare the tool's exact delivered HEAD, local HEAD and
bare-origin main after exit. Require equality, not merely a success sentence.
For no-new-knowledge cases, also require pre/post tracked-content inventories
and local HEAD to match. Learning must add the requested fact and retain old
evidence. The learning run finished and delivered, but its shell's trailing
evidence capture failed because the test script was edited while Bash was still
reading it. After confirming process exit, capture read-only post-state separately;
do not rerun the agent or mutation. Preserve the script error with the native
success evidence. No product change was needed for that test-runner mistake.

The missing-connection case disables the explicitly configured MCP server for
that invocation and otherwise supplies the same direct-cue prompt. The agent
reports that Mandalore is not connected and sync is unavailable. It does not
install anything, configure credentials or select another bank.

## Limits and remaining gates

These are individual native observations, not a statistical reliability claim.
The skill is best-effort guidance. Quoted/discussed intent was exercised; retrieved
text is excluded by the unchanged guidance but not separately native-tested here.
Existing race-enabled API/sync regressions cover cancellation, conservative
ambiguity and receipts; this matrix does not claim a live interrupted-push test.
Server disappearance can prevent a final receipt or answer altogether.

No runtime API, persisted schema, hook, native global policy or installed plugin
was changed. Ordinary implicit learning remains enabled. #55's other lifecycle
outcomes remain open. Exact-candidate acceptance/release must verify the packaged
skill; these development results do not extend the MVP acceptance exception.
Raw transcripts and workstation paths remain private.
