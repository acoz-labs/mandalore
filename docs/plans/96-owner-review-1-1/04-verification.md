# Verification
Failing-first tests: enumerate allowed and denied issues for the third pair;
mixed historical/new/previous-1.1 pairs, arbitrary source/digest, wrong repository,
missing/extra args, malformed issue, owner identity and confirmation refusals.
Run recorder fixtures for all nine issues through both author checks; verify
allowlisting, normal independent review, nomination and gate failures plus no
external mutations on refusal. Preserve existing tests and authority disclosure.
Run full pinned CI, privacy/whitespace checks, exact-head review, hosted CI and
post-merge audit. Use local synthetic process fixtures, never live acceptance.

## Production Readiness Preflight
Not applicable: maintainer-control change only, Release: not applicable.
No artifact rebuild, nomination, acceptance dispatch, production activation,
credential change or release publication. Retained product bytes stay untouched.
Rollback is a reviewed revert of the added eligibility case; clearing the owner
variable already disables new exception submissions without erasing history.
