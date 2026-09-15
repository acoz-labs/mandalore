# Verification and release

## Failing-first layers

1. Characterize the retained baseline packet/native counters and correct answers.
   Add deterministic synthetic cases in foundling retrieval tests for default
   size/count, exact serialized budgets, paging all twelve same-term hits without
   duplicates/omissions, first-item budget refusal, nonadvancing/invalid offset,
   invalid explicit zero, UTF-8/JSON expansion and stable ranking.
2. Source safety: changed/disconnected/conflicted registration between pages,
   supplied stale pin at page zero, wrong bank, unchanged source/signet files,
   empty/missing results not absence. Preserve existing promotion/provenance and
   source-observation regressions. Repeated same-pinned reads are deterministic;
   source change is freshly detected rather than served from a cache.
3. Shared API/CLI/actual SDK stdio compare defaults, explicit old-size requests,
   new search inputs, budget errors and serialized provenance/continuations.
   Read-only calls succeed and do not mutate. Exact source identity, pin, SHA,
   locator and registration remain sufficient for existing verified promotion.
4. Menu tests exercise default search, Next/Back, unavailable/stale continuation,
   default and explicit 8192-byte read, invalid byte count and cancellation.
   Capture and inspect actual synthetic terminal recordings of the changed flows,
   using existing keyboard/no-color/narrow-width patterns. No screen-reader or
   other-platform acceptance claim without corresponding evidence.

## Native quality and cost matrix

Use fresh synthetic bindings/workspaces in the dedicated test lab; never the
owner's active pane or private references. Keep model/reasoning/runtime identities
and fixture content recorded. Candidate ordinary consultation must preserve the
correct multi-document current/historical/negation answer, reach a late passage
past the first preview, page to a relevant result beyond the first ten, and honor
an explicitly requested full/deep review. Include source-change refusal and a
read-only request with a quoted trigger/instruction in the reference. No implicit
promotion, journal, sync or reference repair.

Compare search -> selected excerpt -> deliberate expansion sequences, actual
text ranges and duplicated bytes, serialized payload/schema sizes, model request
count, final request context and aggregate/cached input separately. Do not count
MCP wrapper removal again as this feature's saving. Record failures and query
variation. Smaller packets without correct answers or discoverable complete
evidence are a failure. Return to design if native expansion loses quality or
systematically creates more repeated reads; do not weaken the questions.
Single runs are behavioral observations, not p95/latency/statistical claims.

## Required validation

Pinned Go 1.26.4 full CI, targeted race tests and strict schema/process tests;
edited native skill validation; exact-head hosted CI and distinct self-review.
Document host fallback if Docker is unavailable. Cross-builds are not native
Linux acceptance. Native raw logs remain private; publish sanitized fixtures,
measurement procedures, actual outcomes and limits.

## Production Readiness Preflight

Implementation-only envelope: no credential changes, deployment, live activation,
bank migration, release or existing artifact replacement. New immutable candidate
nomination and independent exact-artifact acceptance precede publication. Existing
v1.0.0 remains unchanged. No new release secret or infrastructure is required by
the retrieval design. Rollback before adoption is source/runtime selection with
no data migration; never erase registrations or learned provenance to roll back.
