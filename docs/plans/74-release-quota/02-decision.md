# Solution Decision

## Decision Drivers

Preserve immutable bytes, source freshness, fail-closed activation and anonymous
privacy. Reduce measured duplicate reads with minimal state. Keep recovery
under explicit caller control and output truthful about partial effects.

## Competing Approaches

1. Messages only: minimal but leaves the selected duplication outcome unsolved.
2. Disk/client-wide cache or authenticated fallback: reduces requests but adds
   expiry, credential and cross-command trust concerns outside this outcome.
3. Private operation-owned inspection result plus typed refusal: selected.

## Adversarial Comparison

The public serialized `ReleaseView` is caller-controlled and cannot authorize a
download by itself. A client cache keyed by version would allow an old preview
to substitute for the required live refresh. Remarshalling a parsed manifest
changes its byte identity. Skipping final refresh would miss source changes
during executable verification. Selected reuse retains the actual verified
bytes only within the live call and keeps boundary refreshes. It saves nine
requests for plan+apply and two for full verification in the bounded fixture.

## Selected Approach

High confidence in the bounded mechanism, conditional on failing-first tests
proving exact bytes, fresh ownership and race behavior. No new dependency or
generic cache abstraction. Export only the error/advice types needed by API;
keep verified data private to distribution.

## Decisions Ledger

| Decision | Rationale |
| --- | --- |
| Preserve original manifest bytes privately | Digest identifies received bytes, not normalized JSON |
| Keep independent DownloadAsset refresh | Existing callers need a safe standalone operation |
| Keep both apply refresh boundaries | Fewer requests must not trade away source/destination checking |
| Verify every asset once per full operation | Previously verified manifest/sums satisfy their own checks only |
| Header-only quota classification | Avoid provider-body disclosure and unreliable text matching |
| Omit invalid timing rather than clamp | Clamping could encourage a retry earlier than provider guidance |
| No automated recovery | A wait duration never authorizes another mutating apply |
