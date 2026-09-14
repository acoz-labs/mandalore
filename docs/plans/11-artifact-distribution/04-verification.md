# Verification And Release Design

## Test Strategy

Use failing-first Go tests for manifest/build identity and release install behavior
under `internal/distribution`, CLI contracts under `cmd/mandalore`, and scripted
fake-GitHub fixtures for actual workflow helper branches. No live release creation
is needed to verify denial, staging, publication retry or ledger ordering.

Build two isolated clean exports with Go 1.26.4 and compare every payload byte,
including executables/plugin archive/manifest/checksums. Verify all four executable
formats/targets and plugin content identity. Confirm deterministic payload excludes
host paths, credentials, timestamps and workflow IDs. Bound subprocess cancellation
and clean up only owned temporary processes/paths. Preserve reproducibility failure
evidence rather than removing disagreeing fields to manufacture a passing claim.

Manifest fixtures cover unknown/duplicate fields, oversized input, version/schema
incompatibility, missing/duplicate/misnamed assets, unsafe archive members, wrong
target, source/version/plugin disagreement, corruption and trailing content.
HTTP fixtures cover official provenance, altered/expired Actions metadata, redirects,
status failures, truncation, excessive bytes, timeout/cancellation and credential
non-forwarding. Download-action digest warnings alone are not verification.

Installer fixtures cover absent/owned/foreign paths, regular-file and symlink
collisions, tampered receipt/retained binary, stale plans, concurrent apply, failed
rename/receipt write, interrupted download, idempotent reuse and compatible rollback.
Hash sentinel memory/auth/native settings/other connection trees before and after.
Connection handoff tests distinguish old and new embedded plugin content so a new
binary paired with the old plugin cannot accidentally pass. No shell execution.

Bootstrap tests run the actual reviewed script with fixture curl/checksum commands:
missing dependencies, safe target/version mapping, duplicate checksum entries,
bad digest, cancellation, refusal without executing payload, success into the same
CLI preview. Test real supported host tools in the designated native pane.

Publication fixtures exercise missing acceptance/actors, unlinked implementation,
changed candidate or tag, partial draft, matching/mismatching existing assets,
successful published verification, interruption before ledger completion and retry.
Run the actual finalizer with fake API responses: no issue may close before the
complete exact release is published and verified. Preserve production-mode behavior.

## Red/Green Sequence

1. Strict manifest and version consistency failures, then deterministic builder.
2. Source/transport/digest verification failures, then candidate retention tooling.
3. Local install ownership/stale-plan/partial/recovery failures, then plan/apply.
4. CLI/bootstrap/operation discovery and correct-new-runtime handoff failures.
5. Menu interaction/effects tests, then actual terminal recordings and review.
6. Candidate nomination/promotion/finalizer denial and retry tests, then workflows.
7. Full CI, clean reproducibility run, native install/update/rollback smoke and docs.

## Acceptance Evidence

Classification: `new-or-materially-changed-experience`, with the prior issue-linked
product-design contract. Record actual terminal output at normal and narrow
(24-column) widths in color, NO_COLOR and plain modes. Cover checking/no-release,
offline/corruption/unsupported target, default-No, EOF/Escape, owned/foreign paths,
success with connection unchanged, selected connection handoff and partial recovery.
Use existing arrows/Vim controls; do not intercept text-entry j/k/h/l.

Retain synthetic recordings and a manifest bound to the exact implementation head,
build digest, environment/width/input/actions/results. Run console/CLI tests,
inspect ANSI-safe rendering and errors, and record accessibility limits: keyboard,
labels/plain modes tested; screen readers, alternate fonts/locales and untested OS
rendering not inferred. No mobile/browser surface exists for this CLI. No approved
visual baseline yet; a recording is evidence, not independent product approval.

Native engineering tests use the designated Herdr pane and synthetic banks only.
Test downloaded/local candidate CLI version, new plugin identity, fresh Codex
session learning/recall, unrelated cwd, no-save, update and retained rollback.
Keep routine native auth/inheritance; do not copy credentials or alter real banks.
Model-free tests and cross-builds are labeled separately from native behavior.

#10 repeats relevant hands-on behavior on the retained exact candidate with a
deliberately authorized independent acceptor and records actual platform coverage.
Source CI and contributor native tests are not immutable-candidate acceptance.

## Rollout

Implementation envelope permits branch/PR/CI, synthetic installs and candidate
build/retention. Reconcile lifecycle PR links/labels before nomination. Keep issues
open after merge; no public tag/release, acceptor configuration or live migration
is authorized. A later explicit release dispatch promotes accepted retained bytes.

## Rollback And Recovery

Retain original launcher receipt/runtime and native connection generations. Retry
only identity-matching partial states; foreign paths and ambiguous receipts require
inspection. Selecting an older CLI does not rewrite data or make an unsupported
schema safe. Never automatically delete a published release or overwrite assets.

## Release Prerequisites

Configure independent acceptance actors deliberately; ensure all lifecycle-linked
implementation PRs are merged and included, nominate the verified retained candidate,
complete #10 evidence, verify repository immutable-release setup, then obtain explicit
publication authority. Candidate retention must outlast review; expiry requires
retaining and re-nominating a newly verified transport/identity as appropriate,
never pretending an unverified rebuild is the previously accepted artifact.

## Production Readiness Preflight

Not applicable as an authorization to deploy: execution is implementation only.
The current workflow uses GitHub's scoped workflow token, with no new provider
secret slots. Independent `ACCEPTANCE_ACTORS` configuration is absent and releases
do not exist. Product publication helpers and workflows are the work being built;
their execution/verification/rollback fixtures and exact release receipt must pass
before a later publication preflight can succeed. Activation remains explicit and
memory compatibility does not authorize schema migration.
