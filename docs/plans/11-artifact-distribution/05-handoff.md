# Implementation Handoff

## Change Tier And Smallest Complete Outcome

Substantial cross-cutting artifact/security and user-facing delivery change. Ship
one coherent candidate format and guided owned-CLI install/update path with explicit
selected-connection activation, plus same-byte gated publication and honest recovery.
A checksum file without actual distribution or an empty release ledger is incomplete.

## Dependency Order And Reviewable Slices

1. `internal/distribution`, `VERSION`, build metadata and `bin/build-artifacts`:
   strict manifest and deterministic package identity tests red then green.
2. Official/local candidate resolution and bounded verification: malformed source,
   metadata, network and transport fail closed; retain verified coordinates.
3. Local plan/apply and receipts: foreign/stale/partial cases protect existing
   programs; repeatable clean install/update/rollback with unchanged signets.
4. `cmd/mandalore` CLI-only contracts, reviewed bootstrap and new-runtime connection
   handoff: exact matching embedded plugin; no exposure in memory MCP.
5. Existing console journey: automated keyboard/plain/effects tests and actual
   normal/narrow recordings reviewed against issue product-design criteria.
6. `.github/workflows`, product release helpers and artifact finalizer adaptation:
   same retained bytes, real preflight denials, remote verification before ledger,
   idempotent matching retry and explicit public-release authority.
7. Full validation, reproducibility evidence, native smoke, documentation promotion
   and exact-head reconciliation. Keep acceptance/release prerequisites visible.

## Acceptance Traceability

Issue criteria 1–2 map to slices 1–2 and hosted/local build evidence; criterion 3
to slices 1/3/4/5 and manifest/CLI/UI contracts; criterion 4 to slices 3/4/6/7 and
download/install/update/rollback evidence; criterion 5 to preserved acceptance gates
and deliberate later configuration; criterion 6 to publisher/finalizer fixtures and
later verified GitHub release receipts. Detailed cases are in `04-verification.md`.

## Documentation Promotion

- `docs/deployment.md`: candidate/build/provenance/version/acceptance/publication and
  repo-specific template adaptation, actual commands and release prerequisites.
- `docs/runbook.md` and README: first install, CLI vs selected connection updates,
  explicit trust/effects, compatibility, partial state and retained rollback.
- `docs/development.md`: reproducibility commands, pins/container fallback, tests,
  bounded process cleanup, supported build targets vs tested native targets.
- `docs/interfaces.md` and architecture: actual typed release contracts and local
  ownership/verification boundaries; no unnecessary new permanent plan mirror.
- `docs/evidence/distribution`: reviewed synthetic build and terminal evidence,
  exact heads/digests/results and explicit untested surfaces.
- Plugin documentation only if actual distribution guidance changes; use relevant
  skill/plugin authoring instructions before editing those assets.

## Pull Request And Review Contract

Planning-only branch `design/11-artifact-distribution` changes only this six-file
directory; linked with `Refs #11`. Final exact-head engineering self-review under
the delegated MVP policy is recorded honestly. Implement on `feature/artifact-distribution`
after the planning review; stack only as permitted by repository instructions.

Keep implementation PR draft until current-head reconciliation, tests/hosted CI,
privacy and rendered evidence are complete. Promote actual behavior to durable docs
and delete all six temporary files before readiness. A passing engineering review
does not close release-bearing issues or satisfy independent candidate acceptance.

## Explicit Non-Goals And YAGNI Boundary

No generic updater framework, package management, signing-key enrollment, background
service, credential copying, arbitrary hook bus, assistant identity, new adapters,
data schema rewrite or automatic predecessor migration. No live publisher rehearsal
that creates a public release just to test a workflow. No additional account setup
or acceptance-policy weakening to remove the independent-actor prerequisite.

## Exceptions That Reopen Design

Reopen for a different activation/data/trust contract, mandatory new runtime or
provider dependency, inability to retain/promote the exact candidate, an incompatible
schema change, unplanned destructive cleanup or publication beyond this envelope.
Routine test findings, safe implementation corrections and documented supported-host
limits do not require serial product permission.
