# Verification and release design

## Test strategy

Table-driven public/private/sensitive/restricted recall tests; restricted native
context test; correction preserving explicit history and journal content; retained
Git blob after correction and attempted deletion. Deny remember, journal, combined
save/sync, Git init/checkpoint and sync in read-only mode, with complete before/
after file-content inventory and successful allowed reads. Use existing fixtures.
An independent test removes a committed journal entry, verifies Store.Validate
still succeeds, and expects ErrHistory without changing HEAD.

## Red/green sequence

These are characterization tests of existing behavior, not invented production
fixes. Run new tests against unchanged runtime. Demonstrate assertion sensitivity
with temporary controlled mutations in a disposable worktree (e.g. remove the
read-only gate or bypass append-only protection), then restore exact source and
require tests to pass. Keep mutation edits out of commits.

## Acceptance evidence

No-rendered-impact: only docs and tests change; no UI or native prompt/skill
changes. Actual engine/API/Git fixture execution is required, but fresh provider
login/session recordings would not prove these storage claims. No private bank.
Run focused tests uncached and full repository CI with pinned runtimes; use the
documented host fallback only if container support is unavailable.
Review Markdown structure/links and public-content hygiene at final head.

## Rollout, rollback and release prerequisites

Reviewed docs/tests merge publishes repository knowledge, not a runtime artifact.
Reverting that commit removes docs/tests only; no migration/data rollback exists.
Complete #79 after reviewed merge and post-merge checks. Runtime issues still
require their own candidate acceptance/release. No immutable artifact changes.

## Production Readiness Preflight

Not applicable: implementation-only docs/tests cannot activate or touch
production. No secrets, deploy, native configuration or release command required.
