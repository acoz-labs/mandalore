# Managed Repo Standard

SDLC template version: `2026.08.13.1`

This repository was created from or adopted into the managed software-repository
standard.

## Expected Baseline

- `AGENTS.md`
- issue templates
- pull request template
- `bin/ci`
- `bin/container`
- `bin/validate-solution-plans`
- `Dockerfile.dev`
- every GitHub Actions job uses the required configurable `CI_RUNNER` label
- no workflow silently falls back to a GitHub-hosted runner
- main-branch audit workflow with bounded retry for GitHub's eventually
  consistent squash-merge association
- dependency update policy and Dependabot configuration
- dependency merge steward workflow and `bin/dependency-merge-steward`
- daily dependency-steward recovery sweep; event-driven checks remain primary
- dependency review workflow, or a documented repo-specific deviation
- docs for development, architecture, capability architecture, temporary
  solution plans, deployment, runbook, decisions, and SDLC
- product-design gate for substantive user-facing changes, including
  implementation evidence and accessibility review expectations
- UI Acceptance Evidence contract separating functional coverage,
  visual-regression detection, and product judgment, with durable exact-head
  PR artifacts and fresh exact-candidate acceptance evidence
- one contributor-authored, independently reviewed solution-design planning PR
  before `Ready`, plus implementation reconciliation and durable documentation
  promotion before review
- one compact Decision Spotlight exposing consequential product, UX, data,
  automation, permission, privacy, and trust defaults at the final plan gate
- machine-checked implementation reconciliation bound to the current draft PR
  head, exact planning PR, and removed plan directory
- temporary issue plan packs under `docs/plans/`, with an explicit execution
  envelope and deletion after the shipped contract is promoted
- purpose-based documentation promotion that keeps final flows, data models,
  contracts, authority, failure, and operational knowledge coherent rather than
  copying one permanent document per design stage
- explicit `Acceptance` and `Ready for Production` delivery states
- linked issues remain open until their documented delivery profile is complete
- issue-, commit-, artifact-, implementation-set-, and acceptor-specific
  product-acceptance evidence independent of the implementer, including
  workflow-authored durable statuses that issue comments or labels cannot forge
- on-demand private development previews for bounded in-progress decisions
- automatic staging of immutable `main` candidates when a staging target exists
- active-candidate nomination from service staging or staging-free artifact
  validation that prevents stale acceptance while preserving evidence-only
  reruns of an already accepted exact candidate; release gates authenticate the
  workflow author and recheck the latest application nomination
- production promotion of the accepted artifact, followed by smoke verification,
  an immutable Git tag, and a GitHub Release ledger entry
- executable production readiness covering named secret slots, deploy,
  activation, exact-artifact verification, rollback, and project-specific checks
- an exact promoted-candidate probe that resolves the deploy/receipt crash
  window into active, conclusively inactive, or inconclusive; only a
  conclusively inactive result permits an ordinary retry deployment
- an exact production receipt containing application SHA, artifact, deployment
  id, control revision, linked issues, and rollback target
- shared, retry-safe release finalization that rejects omitted release-ready
  issues and stale candidate evidence, propagates GitHub API failures, reuses
  matching Git tags and Releases, and clears transient delivery labels before
  applying `delivery:released`
- evidence/finalization retries resume a promoted receipt without redeploying;
  forced re-promotion is a separate explicit operation
- explicit `service`, `artifact`, or `non-deployable` delivery profile so a repo
  never pretends to have an environment that does not exist
- exact mise runtime pins when the repository supports host-local language
  execution
- private-repository and trusted-contributor boundaries for self-hosted jobs
- unchanged permission, secret, environment, and manual-attestation gates for
  privileged workflows

## Repo-Specific Deviations

Record intentional deviations here with rationale.

- Bootstrap basis: template `feeb93836b967d6c100e58b4384fc067e73a0ac9`.
  Generic files were privacy-reviewed before seeding fresh history. The first
  commit initializes the repository; subsequent changes use reviewed PRs.
- Public CI explicitly uses `CI_RUNNER=ubuntu-latest`. No self-hosted runner or
  automatic fallback is used for public code.
- Artifact profile, no staging service. Product-specific distribution is tracked,
  not operational merely because template workflows exist.
- Host `bin/ci` is supported when Docker is unavailable; exact runtime pins are
  required before adding language builds.
- No private control plane is required for the issue/project/release ledger.

`ACCEPTANCE_REQUIRED_CHECKS` may name one check exactly, including punctuation
such as a comma. For multiple checks, use the legacy comma-separated form only
when none of the individual names contains a comma.

Set `DELIVERY_PROFILE` to `service`, `artifact`, or `non-deployable`. A service
with no production target may set `ACCEPTANCE_COMPLETES_DELIVERY=true`; accepted
artifacts otherwise become ready for release, while production services become
ready for production. In completion-on-acceptance mode, an exact approval retry
is idempotent after closure and an authenticated later rejection reopens the
issue with blocking evidence.
