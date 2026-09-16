# Context

## Problem And Desired Outcome

#74 delivers approved #54 O1: fewer redundant anonymous release reads and
actionable quota guidance, without weakening verified installation.

## Current State

Basis: `6e008087a25daf5a0db6686f1d2669ae5930677f`.
`internal/distribution/release.go` verifies metadata, manifest, source tag and
checksums in four simple-tag requests. `DownloadAsset` repeats that inspection
then downloads one asset. `install_apply.go` replans before and after staging;
`install_io.go` separately downloads manifest and binary. `ParsedManifest`
retains parsed content and original digest, **not original bytes**.
`publication_verify.go` hashes manifest/checksums twice and verifies the
publication ledger plus final metadata/tag. `internal/api/api.go` preserves
install receipts; `cmd/mandalore/menu.go` renders them beside errors.

Discovery's reproduced synthetic fixture measured:

| Entry path | Current | Designed budget |
| --- | ---: | ---: |
| Typed inspect or plan, simple tag | 4 | 4 |
| Fresh/update plan + apply | 22 | 13 |
| Exact completed-plan replay | 0 | 0 |
| Full publication verifier | 15 | 13 |
| Menu install, accepted preview | Same plan + apply | 13 |
| Shell bootstrap plus accepted menu | 2 + 22 | 2 + 13 |

Menu `installRelease` calls `release_plan`, then `release_apply`; there is no
extra `release_inspect` call at this basis. Independently requested inspection
adds four calls. Bootstrap `packaging/install.sh` fetches sums and platform
binary, verifies them and launches the menu; it does not reuse that scratch
binary as authenticated installer state. Counts exclude fetching the script
itself, redirects and annotated tag expansion. No live quota-debit or latency
claim follows from these fixtures.

## Actors And Critical Journeys

Users and agents preview/apply an anonymous published CLI; maintainers verify
publication. Critical failures: rate refusal during preview/staging/final check,
changed source or destination, corrupted bytes, interrupted activation and
explicit retry. Existing pending activation and completed replay remain separate
owned-local-state paths, not remote freshness claims.

## Acceptance And Non-Goals

Meet #74's request counts, immutable identity/race checks, quota classification,
bounded output, receipt accuracy and native evidence. Do not change signets,
plugins, provider authentication, published v1.0.0, bootstrap protocol, release
authority, cache persistence or automatic scheduling.

## Constraints, Dependencies, And Risks

Provider headers are untrusted. Anonymous quota can be shared by unrelated
users. Timing is not a guarantee. Verified-byte reuse must be inaccessible from
JSON/CLI input. Compatibility requires unchanged plan format, receipt ownership,
read-only operation flags, output budgets, public safe-download behavior and
all existing host/redirect/auth checks. No credential or new secret is required.

## Evidence, Assumptions, And Unknowns

Discovery probe and primary GitHub guidance are linked in #54's approved pack.
Reduction is designed, not measured until implementation tests pass. CDN routing,
live quotas and cross-platform native behavior remain unproven. No need to test
by exhausting a real quota. Mac terminal accessibility checks do not prove
screen-reader, other locale/font or Linux native behavior.
