# Solution Design: local-first synchronization

- **Status:** Draft
- **Issue:** #5
- **Planning PR:** Pending
- **Repository basis:** d1f78b2b4139b394d5c12536bc5e74a6fda4042b
- **Execution envelope:** implementation

## Decision

Add a separate Git synchronization package over the signet engine and its shared
writer lock. Expose real initialization, checkpoint, status and sync operations
through the existing CLI/MCP dispatcher. Preserve local history and distinguish
transport delivery, semantic conflict and current-model-context freshness.

## Decision Spotlight

- Local saves never need network access. Explicit checkpoint creates a local Git
  commit; sync checkpoints before bounded fetch/integrate/push.
- Require the exact standalone signet repository and main branch. Refuse linked
  worktrees, redirected Git environments, in-progress operations and unknown work
  rather than touch another project or overwrite staged/partial edits.
- Validate append-only evidence and remote candidate identity/structure before
  changing the local branch. No reset, force-push, automatic conflict editing or
  deletion of historical evidence.
- Git delivery and semantic agreement are separate receipt fields. Concurrent
  revisions remain visible and require a superseding user decision, not a text merge.
- Use native machine-local Git/SSH credential configuration without copying tokens
  or predecessor assistant/account-role adapters. No interactive auth prompts in
  unattended sync; authentication failure leaves committed local work pending.
- Native session-start hooks stay read-only. Automatic bounded sync is requested
  by the host agent only when the current task permits memory mutation; no-save
  instructions prohibit journaling and sync too. Actual Codex wiring is #6.
- Status describes the last checked remote head/time, never universal freshness.
  Reads reload local memory; they do not rewrite context already read by a model.

## Needs Attention

Native harness event coverage and real two-machine acceptance remain #6/#10;
this design supplies their backend and policy, not evidence that they ran.
Public release, live migration and credential enrollment are outside this envelope.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Owner-authorized exact-head engineering self-review permits implementation.
Record reviewed source, actual Git feature probes and CI; do not claim independent
product acceptance or replace later native lifecycle/candidate verification.
