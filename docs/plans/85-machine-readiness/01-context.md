# Context

## Problem And Desired Outcome

A user connecting an existing signet needs practical next steps without mistaking
installed bytes or a successful login for tested memory integration. #85 owns the
complete declarations, assessment, menu/CLI parity and optional prompt outcome.

## Current State

At repository basis b28224827e0749ac4b27cd2fee807526ccf9ecc6:

- `internal/install/doctor.go` and `pi_doctor.go` own existing deeper inspection.
  They retain not-tested boundaries and do not repair or sync. Codex inventory
  invokes native commands; Pi may invoke its version command after registration.
- `cmd/mandalore/connection.go` resolves native defaults but rejects a missing
  PATH executable before returning other structural observations.
- `cmd/mandalore/menu.go` dispatches shared operations and owns no separate
  diagnosis engine. Existing inspect and repair entries are distinct.
- `internal/install/stage.go` / `pi_stage.go` parse bounded owned receipts.
  Ownership paths and artifact hashes are available without executing binaries.
- `internal/binding/binding.go` opens a memory service after parsing its 16 KiB
  binding. Tracing `OpenService` and `ValidateAuthorship` confirms it reads the
  manifest and selected enrolled-device metadata, not records/journals. Preserve
  those validation rules in the bounded metadata path; do not call `Validate`,
  which checks the full signet. The initial draft overstated this read footprint.
- `internal/api/api.go` supplies strict schemas, read-only/CLI-only annotations,
  32 KiB input and 64 KiB output limits. Existing failure reports can reside under
  the error envelope, not only successful `result`.
- `cmd/mandalore/version.go` and embedded plugin metadata report running build
  information; preserve their existing format and the eight-artifact inventory.

## Actors And Critical Journeys

Humans use the menu; agents use the same administration CLI. Both need fresh
machine, already-configured, changed-runtime, unsupported-target and unknown
cases. The default journey must work without native binaries, profiles or a
signet. Prompt viewing and cancelling leave state unchanged. Deeper native
inspection is explicit and preserves its existing untested boundaries.

## Acceptance And Non-Goals

All issue #85 acceptance groups remain in scope. Do not introduce a package
manager, universal compatibility database, private capability scanner, telemetry,
agent launcher, new background process or ordinary memory-tool schemas.

## Constraints, Dependencies, And Risks

Use only Codex/Pi memory integrations and known machine-local metadata. No
credentials, native transcripts or record/journal content. Retained connections
can differ from the executing toolkit. Wrappers and translated processes make
PATH/architecture inference uncertain; report that uncertainty explicitly.
Pi implementation exists, but Pi product acceptance remains pending separately.

## Evidence, Assumptions, And Unknowns

The reviewed discovery contains native empty/missing-profile probes and exact
release/Pi evidence identities. New assessment behavior itself is unimplemented.
Outstanding design work: static receipt selection, narrow metadata readers,
identity matching and complete CLI/menu failure semantics. Neither historical
evidence nor an operator-supplied string may silently become verified status.

## Product Design

Classification: new-or-materially-changed-experience within existing terminal
patterns. Keep `console.Block`, the terminal's ANSI palette and textual statuses;
no new theme, dashboard, graphical surface or progress animation is needed.

Replace the current top-level inspect label with **The Armorer · Assess or
inspect** at the same menu position. Its submenu offers:

1. Assess this machine · No execution or changes (default).
2. Inspect native connection · Runs the selected native program.
3. Back.

Assessment selects Codex/Pi using the existing harness chooser, then shows
defaults without requiring every path to be retyped. Choices: **Assess selected
paths** (default), **Edit selection**, **Back**. Edit selection allows explicit
native binary/profile, installation state, binding and optional retained root.
An empty retained root means no retained selection, not no active connection.
Do not guess a selection by modification time or scan all historical generations.

The report's default view is short, ordered by the user's task:

```text
The Armorer · Machine readiness
  Completed local assessment. No programs run or files changed.

Next step
  Connect the selected signet on this machine.

Memory runtime
  Support: Supported target
  Setup:   Present
  Evidence: No exact evidence for these bytes

Selected harness
  Support: Supported target; native version not checked
  Setup:   Native executable missing
  Evidence: Historical scenario only

Still untested
  Login, native registration, live tools, active context, remote freshness
```

This is an illustrative missing-setup state, not actual test output. Real component
states determine the next-step wording. Show at most one recommended next step,
prioritizing malformed/inconsistent selection, then missing dependencies/binding,
then explicit native verification; unsupported combinations explain the limit
without proposing forced installation. Unknown evidence alone is not a repair
request. Show independent component summaries, not a universal health badge.

Report actions: **Back** (default), **Details**, **Show agent follow-up prompt**,
**Run native checks**. Details shows the same typed report, selected paths and
scoped immutable evidence links. No automatic clipboard or report-file write.
Prompt viewing shows copyable text and returns; it does not start an agent.

Both the direct native-inspection submenu and the report handoff show the selected
program/profile and explain potential native logs/cache writes before a default-
No choice to run it. This adds disclosure to the menu, not a new prompt in the
automation CLI. Reuse the current doctor result/error rendering, including its
not-tested checks. Neither path offers an implicit repair or retries failures.

Missing paths are normal assessment findings. Malformed input stays at selection
with a bounded explanation; corrupt observed metadata stays in the report as a
finding. Cancellation/EOF returns safely using existing menu conventions; no
success report or prompt is emitted after cancellation. A failed deeper native
check does not overwrite the preceding static report or imply a repair occurred.

Accessibility: status words remain when colors are off; all actions have arrows,
j/k, Enter and Back/Escape routes. Reuse plain input/EOF handling and existing
sanitized wrapping at 32 and 80 columns. Do not require interpreting color, icons
or long hashes to choose the next step. Native rendered verification will judge
these exact tasks and record untested screen-reader/locale/platform surfaces.

Product-design approval is recorded on an exact planning head under ADR 0003
before final solution review. Rendered implementation evidence remains separate.
