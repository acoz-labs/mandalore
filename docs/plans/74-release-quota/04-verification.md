# Verification And Release Design

## Test Strategy

Extend existing fixtures, not live endpoints:

- `internal/distribution/release_test.go`: safe standalone downloads, request
  counts, exact-byte manifest identity, asset corruption/metadata changes,
  redirects/auth isolation, cancellation and concurrent independent operations.
- Install tests: fresh/update plan+apply <=13, exact replay zero; source/tag/asset
  changes while staging, altered destination, changed probe bytes, missing or
  mismatched private inspection, noncanonical-but-valid original manifest
  formatting, partial staging failure and explicit successful retry after the
  fixture's refusal clears. Keep retained/pending/local paths unchanged.
- Publication verifier tests: <=13 requests, exactly eight distinct verified
  assets, ledger/source/final changes still fail, corrupt metadata assets fail
  before reuse, no receipt falsely reaches publication-verified.
- Rate parsing: 429 unknown, 403 ordinary/remaining zero/Retry-After/reset only;
  absent, duplicate, mixed valid/invalid, control, overflowing, negative, huge,
  stale and boundary timestamps; reject raw body/header leakage. One request on
  refusal, no autonomous retry, safe cancellation precedence.
- API/CLI tests: `release.rate_limited`, optional bounded advice, ordinary
  `release.failed`, unchanged unavailable/cancel codes, preservation of staged/
  pending/write state and plain output. Synthetic partial-result injection tests
  error composition; do not invent a network call after activation.
- Menu tests: valid/unknown timing, ordinary refusal, truthful receipt, narrow
  wrapping, no-color labels, plain mode and explicit default-No confirmation.
  Bootstrap test counts its two downloads separately; keep its generic
  pre-runtime refusal, shell/curl restrictions and scratch cleanup unchanged.

## Red/Green Sequence

First add failing request-count/original-byte tests, then private inspection
reuse. Next add failing quota parsing/output tests, implement typed advice and
rendering. Run existing failure/race/recovery tests at each step. Last exercise
explicit retry and native matrix, then full required CI and self-review.

## Acceptance Evidence

In-pattern-visual-change per `docs/operations/ui-acceptance.md`. Capture actual
synthetic native terminal recordings against the exact implementation head:
normal 80-column color, 32-column plain and no-color. Cover valid timing, unknown
timing, ordinary refusal and partial receipt; explicit default-No/Back and retry
after changed fixture state. Retain commands, fixture identity, exit/effects
assertions, environment/dimensions and openable recordings with a manifest.
Use the designated separate test workspace, not the user's active pane.
Test seams must be synthetic-only; no production endpoint override for evidence.

Automated semantic assertions protect error hierarchy and effect flags; no
approved pixel baseline exists and the first recording cannot approve itself.
Keyboard/no-color/plain checks are bounded accessibility evidence, not screen
reader certification. No browser/mobile interface. Native Linux, alternate
fonts/locales and live GitHub/CDN behavior remain explicitly unverified.
Engineering self-review under ADR 0003 is not independent candidate acceptance.

## Rollout

Implementation-only. CI uses pinned tools and container-first validation; use
the documented host fallback if Docker is unavailable. Later nominate an exact
immutable artifact including the merge and obtain separate product acceptance,
fresh native evidence and release authority. Do not modify published v1.0.0.

## Rollback And Recovery

No data migration/configuration change. Prior runtime remains usable; existing
retained-version selection is the explicit rollback path. Preserve pending
receipts and old owned runtime bytes; never infer permission to replace foreign
paths. A later release cannot overwrite the immutable old release.

## Release Prerequisites

No implementation secret or external login required. New candidate acceptance,
artifact identity/version and publication authority are future release gates,
not missing engineering design inputs.

## Production Readiness Preflight

Not applicable to this implementation execution envelope: it cannot activate or
publish production. Existing artifact workflows and anonymous post-publication
verifier remain the later release path; no change to their secret injection or
acceptance rules. Do not dispatch publication during verification of this plan.
