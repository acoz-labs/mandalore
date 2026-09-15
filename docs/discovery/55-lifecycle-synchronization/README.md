# Discovery: reliable delivery without overriding the current task

- **Status:** Final for O3; O1/O2/O4 require follow-up discovery
- **Discovery issue:** #55
- **Repository basis:** 0e4329e28e1aaa70f605d4c67ce502171cf0de4b
- **Recommended decision:** approve O3 delivery; retain O1/O2/O4 as unresolved follow-up
- **Gate 1:** exact-head contributor review required under ADR 0003
- **Confidence:** Medium on constraints, Low on safe lifecycle authorization
- **Private evidence:** none

## Decision sought

Choose the smallest mechanism that reliably attempts delivery of already-saved
memory while respecting current read-only/no-save/no-sync direction. Keep semantic
learning agent-driven. This discovery does not authorize network work from existing
hooks, introduce a daemon, change the bank or activate a personal installation.

## Audience and critical tasks

People moving between machines need useful learning to survive a lost session and
reach the next machine without remembering a separate command. They also need a
read-only task to stay read-only, including previously pending delivery. Explicit
consolidation should attempt delivery even when no new memory is worth creating.

## Evidence

At the repository basis:

- `internal/codex/hooks.go` handles only SessionStart/UserPromptSubmit, reads the
  local bank and returns context. Other events are no-ops. It does not persist
  authorization, read transcripts, synchronize or invoke subprocesses.
- `plugins/codex/plugins/mandalore/hooks/hooks.json` registers those two hooks.
- `internal/api/api.go` enforces process-level `ReadOnly`; ordinary tool calls do
  not carry a verified native session/turn authorization identity.
- `internal/api/sync.go` accepts a timeout, not a current-task authorization lease.
  The synchronizer already serializes bank writers, validates merges, refuses
  interactive credentials and retains conservative cancellation/delivery receipts.
- `docs/synchronization.md` and the memory skill describe explicit agent-requested
  sync. That is the supported fallback, not a deterministic lifecycle safety net.

The [official Codex hooks reference](https://learn.chatgpt.com/docs/hooks), checked
2026-09-15, documents PreCompact, UserPromptSubmit, Interrupt and SessionEnd.
SessionEnd is advisory, command-only and limited to three seconds; switching
threads does not immediately end a session. Hook events provide session identity,
and selected events provide turn identity. Permission mode is not a user's
task-specific memory preference. Transcript format is unstable, and tool-hook
coverage is not a complete enforcement boundary. Native event ordering, steering,
cancellation and unsupported-version behavior still need synthetic verification.

## Assumptions

The [native lifecycle probe](../../evidence/lifecycle-sync/native.md) now verifies
startup, prompts, manual compaction, queued steering, normal exit, resume and
interruption in Codex 0.154.0. It changes the design: queued steering reused a
turn ID, and its prompt hook ran after the preceding PostToolUse. SessionEnd had
no turn ID and resume preserved session identity. A session/turn allow cache
and a post-tool callback cannot by themselves enforce current task intent.
The probe also exposed native configuration/isolation discrepancies; it is not
plugin or sandbox acceptance and does not establish a supported minimum version.

Ordinary confirmed learning and its authorized delivery remain enabled; no magic
phrase should be necessary. Current prohibitions take precedence over an older
authorization. A lifecycle callback is an opportunity, not consent. Synced files
do not retroactively update evidence already loaded into the model's context.

Machine-local coordination metadata may be needed, but it must be distinguished
from user memory and from a blanket claim of zero filesystem writes. Its consent,
storage, lifetime and cleanup are unresolved; do not quietly redefine read-only.

## Unknowns to resolve first

1. Can native events and the MCP connection be bound to an unambiguous current
   session/turn, including steering while a tool is active and subagents sharing
   a parent session? Do not use cwd, a caller-supplied thread string alone or a
   previously observed permission mode as authorization. Native evidence now
   rules out turn ID alone: distinguish each prompt generation, including steering.
2. Can every new prompt invalidate earlier permission before a delayed checkpoint
   starts? What happens when invalidation itself fails, hooks are disabled or a
   resume occurs? A stale allow file must not become authorization by default.
   PostToolUse preceding a queued prompt is a reproduced counterexample to the
   naive design, not a hypothetical edge case.
3. Where can coordination state live without leaking prompts into the signet or
   violating a no-write scope? Can the lifecycle event prove freshness without
   depending on arbitrary transcript parsing?
4. How much useful work fits within an exit budget after lock acquisition and
   process cleanup? The sync API currently accepts whole-second budgets from
   1–30; using the full three seconds leaves no exit-hook cleanup margin.
5. Does the installed native version actually deliver the required events for
   interactive, exec, resumed, compacted and interrupted sessions? No minimum
   version is selected from documentation alone.

## Competing options

| Option | Benefit | Main limitation |
| --- | --- | --- |
| Existing agent-requested sync | Simple; agent interprets current direction | Model can omit delivery |
| Unconditional lifecycle sync | Deterministic invocation | Violates current prohibitions; rejected |
| Per-turn authorized lifecycle checkpoints | Retries pending work at useful boundaries | Requires proven freshness/invalidation and failure behavior |
| Explicit write-and-deliver operation | Removes a separate post-save tool call | Does not recover unsaved knowledge or later pending delivery; must allow save-without-sync |
| Native prompt/transcript keyword classifier | Superficially automatic | Quotes, languages and scope changes make permission inference unreliable; rejected |
| Background daemon | Independent retry opportunities | Outlives task intent and adds an unnecessary service/authorization problem; not selected |

## Selected decision and retained investigation

Select **O3** for independent delivery: a direct user request to consolidate
finishes with one bounded synchronization attempt when the task permits it,
whether or not new facts/journal entries were warranted. Do not create filler
memory to trigger synchronization. Existing `memory_sync` already supports this;
its timeout, cancellation and no-empty-checkpoint behavior are tested. This is a
small skill/usage-contract correction, not a new protocol or lifecycle executor.
Native behavioral acceptance must cover direct, quoted and prohibited uses and
real pending delivery without new memory content.

O3 does **not** satisfy the automatic lifecycle safety-net outcome. O1/O2/O4 remain
explicit follow-up discovery in the roadmap; they are not canceled, declared
complete or replaced by an easier successful test. Keep #55 open as their working
discovery tracker after O3 is materialized. The current permission contract stays
in effect while the product question about earlier saved data is discussed.

Investigate bounded write-triggered delivery and per-turn checkpoint authorization
side by side. Do not start by registering more events that directly invoke sync.
Unknown, revoked, stale or unverifiable state must skip network and bank mutation.
Startup/resume remain read-only until current authorization can be established.
Failure of a checkpoint should preserve pending state, not block ordinary recall
or spin up another model turn to force a retry.

The semantic choice still belongs to the agent: interpreting current direction
cannot honestly be advertised as deterministic merely because an executor checks
a structured flag. The engineering objective is deterministic validation and
bounded execution *after* a valid current authorization, with that boundary
visible in the product contract.

## Success and stop signals

Probe only a disposable native profile/fixture in the designated test workspace.
Initially record bounded event names, opaque session/turn identities and ordering;
do not save prompt bodies, transcript contents or provider credentials. Use a local
remote for later delivery checks and retain before/after inventories.

Require ordinary authorized delivery, no-new-memory explicit consolidation,
prohibited tasks with prior pending work, a new prohibition during active work,
stale events, disabled/failed hooks, resume/crash, concurrent sessions, offline
failure, timeout and later recovery. Reuse existing lock/conflict tests rather
than inventing a second Git transport. Trigger text in quotes, tools and foundlings
must not count as direct consolidation intent.

If suppression cannot be enforced for a proposed lifecycle path, defer that path
with a supported agent-driven fallback. Do not weaken current prohibitions or
claim that successful invocation proves delivery. No observed event is not proof
of unsupported behavior without checking its trigger and native configuration.

## Candidate outcome map

Only O3 is selected for delivery by this decision. Other outcomes remain
unresolved and may not activate network-capable hooks based on this approval.

- **O1 — Current-task authorization and native event evidence:** investigate
  first; establish identity, invalidation, suppression, timeout and unsupported
  behavior. Dependency for any automatic lifecycle transport.
- **O2 — Authorized delivery checkpoints:** conditional on O1; select only
  events that satisfy it. Keep local durability, attempted delivery, verified
  delivery and semantic agreement distinct; no implicit conflict resolution.
- **O3 — Explicit consolidation delivery (selected):** sharpen the direct-user flow to
  finish with one bounded attempt even without new facts, while avoiding filler
  writes. Implement and verify independently; no dependency on O1. Acceptance:
  real pending local commit delivered without new memory records/journals;
  direct consolidation with useful new facts preserves normal learning;
  quoted/retrieved phrases do not trigger delivery; explicit no-sync and existing
  read-only/no-save contract suppress it; failures retain honest pending receipts
  without repeated writes or retry loops. No graphical interface change.
- **O4 — Write-triggered transport:** compare combined operation versus retained
  separate tools; select only if it measurably improves reliability without
  forcing network work on authorized local-only saves.

## Privacy and evidence handling

Public evidence uses synthetic names and local remotes. Native source logs stay
private until sanitized; do not publish paths, transcripts or account policy.
Authorization state is operational metadata, never a portable remembered fact.

## Decision Spotlight

The gap is not the list of available hooks. It is whether a later callback can
prove that the user's current task still permits synchronization. Preserve the
working memory experience while establishing that missing contract.

## Gate 1

ADR 0003 authorizes exact-head contributor engineering self-review. Record the
reviewed head and passing CI, merge this scoped O3 decision, then materialize O3
with immutable provenance and its own proportional solution plan. O1/O2/O4 stay
open under #55 and require further evidence/decision before implementation. Promote
O3's durable contract during its implementation; retain the unresolved discovery
pack until the remaining outcomes are decided. No lifecycle authorization or
new-candidate release acceptance is implied by this partial outcome selection.
