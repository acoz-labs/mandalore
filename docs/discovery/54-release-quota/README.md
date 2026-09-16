# Discovery: bounded anonymous release retrieval

- **Status:** Draft
- **Discovery issue:** #54
- **Discovery PR:** pending creation
- **Repository basis:** 29c4d48a697f5ccdbddcd87b3f145ba2048df240
- **Recommended decision:** approve one bounded delivery outcome
- **Gate 1:** awaiting exact-head engineering self-review under ADR 0003
- **Confidence:** High for observed duplication; medium for the reduction design
- **Private evidence:** none

## Decision sought

Reduce repeated anonymous release requests within a single bounded operation and
make quota refusals actionable. Preserve immutable release/source/asset identity,
freshness before activation, staged-byte verification, redirect limits and the
explicit local-candidate trust distinction. Do not borrow GitHub CLI credentials,
add unattended retry/sleep loops or weaken existing fail-closed behavior.

## Audience and critical tasks

Users install/update a verified public release through CLI or menus; agents use
the same typed interface. Maintainers separately verify every published asset.
Each needs to know whether a refusal is quota-related, when retry may be useful,
and which local changes occurred. A prior successful local save/install receipt
does not prove current remote freshness.

## Evidence

The original #54 incident is a post-publication quota exhaustion observation,
not proof that every first installation hits the limit. The current anonymous
client never reads native credentials. Its generic error merges all non-200,
non-404 refusals and incorrectly uses a universal no-installation phrase even
when an enclosing operation must report its own partial-state boundaries.

At the repository basis, a synthetic HTTP fixture measured:

| Journey | Requests |
| --- | ---: |
| Fresh plan | 4 |
| Apply after that plan | 18 |
| Fresh plan plus apply | 22 |
| Update from another retained digest: plan plus apply | 22 |
| Replay of the completed exact plan | 0 |
| Full anonymous publication verification | 15 (10 asset requests) |

[Probe source and limitations](probe.md) permit reproduction without contacting
GitHub. Inspection retrieves metadata, manifest, tag and checksums. Installation
replans before and after staging; each of two staged downloads also reinspects
the whole release. Full verification downloads already-verified manifest/checksum
bytes again. Annotated tags and redirects can add requests; these are fixture
request counts, not live rate-limit debits or a latency benchmark.

GitHub documents anonymous primary limits per source IP, response-header quota
information, and recovery using Retry-After/reset information; secondary limits
are a separate possibility. Prefer returned headers over polling a quota endpoint.
Sources: [rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
and [best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api),
checked September 16, 2026. No inference that anonymous conditional requests have
the documented authenticated-request quota exemption.

## Assumptions and unknowns

Immutable verified metadata may be reused only inside a bounded live operation,
not trusted from disk merely because it was once valid. Actual CDN routing,
annotated-tag depth and shared-IP traffic vary. Quota headers may be absent,
malformed, inconsistent or stale; guidance must remain bounded and advisory.
An ordinary 403 is not necessarily a rate limit. Exact field names and whether
to use a small operation-owned download session belong in solution design.

## Competing options

1. Improve messages only: useful but leaves measured repeated retrieval intact.
2. Borrow credentials or maintain a cross-command metadata cache: changes trust,
   privacy or freshness boundaries unnecessarily. Not selected.
3. Reuse verified bytes/metadata within a bounded operation, retain boundary
   refreshes and add typed recovery guidance: selected. Requires regression tests
   proving that fewer requests do not skip source changes or partial-state checks.

## Decision

Select one delivery outcome combining reduced retrieval and actionable quota
guidance. Use fresh operation-owned verified data for asset staging, keep the
public single-asset entrypoint safe for independent callers, and retain final
source/destination checks before activation. Full verifier must still hash every
asset and check the publication ledger plus final release/tag state; already-read
manifest/checksum bytes may satisfy their own byte checks in the same operation.

Rate errors should expose only allowlisted, validated status/retry timing fields,
not raw provider bodies, headers, URLs or credentials. Preserve enclosing install
phase/possible-effects evidence. Never automatically re-save or re-apply to retry.
Do not change published v1.0.0, version format or release authorization gates.

## Success and stop signals

Require measured lower request counts for both fresh and update journeys and
full verification, with zero-request exact replay retained. A provisional target
is at most 13 requests per plan+apply and 13 for full verification in the same
simple-tag/no-redirect fixture; solution design must justify any adjustment.
Stop and revise if reducing calls requires trusting a serialized preview, skipping
an asset digest or final source check, widening redirects or importing credentials.

## Candidate outcome map

### O1 — Efficient anonymous release operations and quota recovery

- Disposition: selected recommendation, not yet materialized.
- Outcome: reduce duplicate operation-local retrieval and distinguish quota
  recovery from generic refusal through typed and rendered interfaces.
- Acceptance: deterministic request counts; exhausted/secondary/ordinary-403 and
  malformed-header cases; changed release/tag/assets during staging; corruption,
  redirect and auth isolation; partial-stage preservation; explicit retry after
  changed conditions; actual synthetic rendered recovery guidance.
- Dependencies: this discovery's exact-head review and solution design.
- Sequence: one scoped delivery issue, plan, failing-first implementation,
  full/native verification, reviewed merge, separate candidate acceptance/release.

## Privacy and evidence handling

Use inert synthetic release bytes and in-memory HTTP fixtures. No live quota
exhaustion experiment, private installation inventory, tokens or workstation
paths belong in public evidence. Never change personal launchers during testing.

## Decision Spotlight and Gate 1

The product tradeoff is less repeated network work without weakening the trust
checks, not silently adding authenticated access. ADR 0003 permits a distinct
exact-head engineering self-review for this roadmap discovery. Required hosted
checks precede merge; implementation waits for the bounded delivery plan. This
does not authorize publication, personal installation or product acceptance.
