# Implementation Handoff

## Change Tier And Smallest Complete Outcome

Substantive native integration with in-pattern setup UI changes. Completion is an
ordinary Pi session using the same signet safely and passively, plus an agent-ready
installation/update/recovery path. A working shell command or schema probe alone
is not a substitute for the full issue.

## Dependency Order And Reviewable Slices

1. `internal/binding`, shared context assembly, `internal/api`, CLI flags: guarded
   same-byte binding validation and bounded read-only packets; Codex regression
   tests pass without a new wire contract.
2. `plugins/pi` and embedding: dependency-free native adapter, transport, skills,
   catalog filtering and metadata; compiled and native no-model tests pass.
3. Focused Pi installation code plus API/CLI: owned plans, retained generations,
   native registration and conservative repair; fresh and interrupted synthetic
   native flows pass and preserve unrelated profile state.
4. Console/Armorer and distribution: harness choices, exact metadata, embedded
   package stamping, unchanged legacy version/inventory; native UI and old
   installer compatibility evidence retained.
5. Immutable synthetic candidate and cross-harness model matrix; reconciliation,
   durable documentation, plan removal, exact-head self-review and hosted checks.

Keep a single scoped implementation PR unless a discovered independent delivery
boundary warrants a linked split. Each slice must leave the existing Codex path
usable. Do not implement a generic adapter framework in anticipation of Claude.

## Acceptance Traceability

Slices 1–2 establish engine/schema/binding/lifecycle; slice 3 installation/recovery;
slice 4 conversational administration/UI/distribution; slice 5 proves actual
cross-harness learning, isolation, provenance, delivery and context. The detailed
matrix in [verification](04-verification.md) remains the evidence checklist.

## Documentation Promotion

- `plugins/pi/README.md`: native contract/version, connection and usage.
- `docs/architecture.md`, `docs/interface.md`: shared context, guarded calls,
  package metadata and Pi connection operations with honest limitations.
- `docs/setup.md`, `docs/runbook.md`: choosing a harness, update/recovery and
  no-auth-copy ownership boundaries; distinguish installed from loaded.
- `docs/development.md`, distribution docs: pinned JS tests, native scenarios,
  embedded Pi identity and preserved v1 inventory/updater contracts.
- Pi and Codex skills: only affected guidance, validated and consistent for
  memory semantics; record harness differences rather than hiding them.
- `docs/evidence/pi/`: synthetic engineering matrix and openable UI evidence.

## Pull Request And Review Contract

Planning PR changes only this six-file directory and references #13. A distinct
exact-head self-review follows ADR 0003; no independent approval is claimed.
Implementation branch `feature/pi-plugin` may stack on the reviewed plan.
Before its draft is ready, reconcile every acceptance group, record deviations,
promote shipped contracts, remove this temporary plan and retain exact-head UI
evidence. Required hosted CI and honest engineering review precede merge.

Container-first `bin/container bin/ci`; if Docker service is unavailable use the
documented pinned host route and report it. No static test or green cross-build
establishes native authentication, product acceptance or release readiness.

## Explicit Non-Goals And YAGNI Boundary

No credentials, account roles, user capabilities, assistant orchestration,
standalone RAG, native auth/session copying, private-bank migration, new release
format, automatic Pi/Node installation, lifecycle writes or Claude plugin.
Do not resolve the #55 lifecycle authorization problem with a regex or by
relabelling combined save-and-sync as a deterministic lifecycle checkpoint.

## Exceptions That Reopen Design

Return to design if native Pi cannot preserve tool selection/profile ownership,
the explicit binding cannot be maintained without copying user configuration,
existing updaters cannot verify embedded packaging, or event/context behavior
requires a new data/authority boundary. Ordinary test-driven refinements and
specific API names may be reconciled; materially weakening an issue criterion,
publishing or activating a real bank requires new authority.
