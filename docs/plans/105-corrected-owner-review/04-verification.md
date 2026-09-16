# Verification

Extend the existing predicate matrix to the fourth pair across issue IDs0–110.
Add mixed old/new sources/artifacts, malformed selectors, missing owner/reviewer,
false/absent confirmation, excluded control issue105 and ordinary independent
review. Preserve all older scope tests.

Extend synthetic recorder scenarios across all eleven new issue IDs, covering
both author checks, no confirmation, unlisted owner, normal independent review,
wrong nomination and failed release gate. Refusal must make no external writes.
Run new tests red before adding the predicate; then race-enabled tests and full
pinned local/hosted CI. No-rendered-impact control change; no new native UI test.

## Production Readiness Preflight

Not applicable: implementation-only maintainer controls. No acceptance dispatch,
actor configuration, product rebuild, release, deployment or live activation.
