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
  binding. A new metadata-only projection must not use this full open path.
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
