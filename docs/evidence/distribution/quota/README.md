# Anonymous release quota: engineering reconciliation

Discovery [#54](https://github.com/acoz-labs/mandalore/issues/54), selected O1
delivery [#74](https://github.com/acoz-labs/mandalore/issues/74), planning
[#75](https://github.com/acoz-labs/mandalore/pull/75), implementation
[#76](https://github.com/acoz-labs/mandalore/pull/76). ADR 0003 permits recorded
engineering self-review, not independent product acceptance or publication.
The PR's current reconciliation/review comment binds the final head and its
separately retained exact-head native recordings. Earlier checkpoints are not
relabeled final-head evidence.

## Requirements and implemented behavior

| Requirement | Implementation and verification |
| --- | --- |
| Bounded request reduction | `release_efficiency_test.go`: fresh/update plan 4 + apply 9 = 13 (baseline 22), exact replay 0; full verifier 13 (baseline 15), eight distinct assets fetched once each. Shared-client concurrent operations each retain their own verification. |
| Original bytes, not input trust | Private inspection retains original manifest/checksum bytes; public plan/view has no private state. Staging rejects missing/mismatched/corrupt private input, retains non-normalized valid manifest bytes and validates selected binary identity. No cache survives an operation. |
| No weaker source/activation checks | Standalone DownloadAsset retains its pinned-ID refresh. Apply preserves pre/post source/destination checks and post-probe byte verification. Tests refuse stale preview, source/tag/asset changes during staging, corrupted binary, probe-modified bytes, unsafe redirects and auth borrowing. Existing retained/pending/ownership tests still run. |
| Full publication verification | Metadata/ledger matching, every asset digest and final release/tag checks remain. Manifest/sums already read in this operation satisfy only their own asset checks. Changed/corrupt/incomplete publication tests still fail closed. |
| Actionable quota, bounded exposure | `release_retry_test.go`: 429, supported 403 primary/unspecified, ordinary403, header-only timing, duplicates/overflow/control/stale/excessive values, no refusal-body read, cancellation and no autonomous retry. Optional advice has bounded numeric/status/UTC fields. |
| Truthful effects and recovery | API/menu tests retain phase, pending record and may-write/inspect flags. A transport refusal does not claim whole-operation effects. Actual fixture apply fails during staging, preserves fresh destination/old launcher and succeeds only on explicit retry after the refusal clears. |
| Separate journey costs | Menu plans/applies without extra inspect. Bootstrap test records two additional asset downloads. Redirects, annotated tags and fetching the script are outside the simple fixture count. No live quota-debit claim. |

## Red/green evidence and provenance

The [approved discovery and baseline probe](https://github.com/acoz-labs/mandalore/tree/ba87c86a6b16e95aa6684a1b853ad39c53e48bd3/docs/discovery/54-release-quota)
are immutable source evidence. The [approved six-file design](https://github.com/acoz-labs/mandalore/tree/4c958b6ae5e712411d6854aefe3ea20b0c43a17b/docs/plans/74-release-quota)
is retained in planning history. Test commit `27652be` failed the 13-request
budgets with observed 22/15 before implementation `40a7a5b`; original-byte
characterization already passed and was preserved. Refusal test commit `9f26faa`
failed on generic/misleading messages before `d1e8979` implemented typed guidance.

The initial implementation and quota slices passed full pinned host CI and
hosted CI. Final-head checks and evidence must be recorded separately on PR76.
Docker was unavailable, so the documented Go 1.26.4/Node 24.1.0 host fallback
was used. A signing-interrupted rebase was aborted; an overlapping host run was
explicitly excluded. No forced branch rewrite or personal installation occurred.

## Native evidence method and limits

`TestReleaseQuotaNativePresentation` is compiled from the exact reviewed source
and run in a real macOS terminal. The production renderer, navigation and
receipt presentation execute; the API responses are expressly synthetic. This
test-only seam is absent from the production binary. It is not proof of real
GitHub quota exhaustion, published-byte installation or candidate acceptance.
The partial result is a composition test, not a claim that production performs
a quota-sensitive network call after activation. HTTP and filesystem effects
are verified separately by distribution/API tests. The explicit-retry native
test uses actual fixture apply with an inert executable verifier, not a real
provider or payload execution.

Final matrix: 80-column color timing; 32-column plain/NO_COLOR unknown timing;
80-column NO_COLOR ordinary refusal; 80-column and 32-column partial receipt;
default-No Enter and Escape/Back; actual synthetic apply/refusal/retry regression
run in the terminal. Retain unmodified BSD script recordings, exact source and
driver hashes, platform/dimensions, action/expected result, semantic assertions
and visual-review findings on a separate evidence branch linked by immutable
commit. This avoids changing the source head merely to attach its own evidence.
No recording containing the earlier fixture's workstation temporary paths is
eligible for public retention; regenerate with corrected synthetic values.

Keyboard, no-color labels, narrow wrapping and plain mode are bounded
accessibility checks. No approved pixel baseline, screen-reader certification,
alternate fonts/locales, native Linux/Intel Mac or browser/mobile results are
claimed. Four-target builds are not native acceptance. The first recording does
not grant product approval.

## Documentation promotion and drift

| Temporary material | Durable destination |
| --- | --- |
| Mechanism, request counts and anonymous trust | [development](../../../development.md), distribution code/tests |
| Typed errors, timing, effects and memory-schema exclusion | [interface](../../../interface.md) |
| Explicit retry and preserved pending state | [runbook](../../../runbook.md), [README](../../../../README.md) |
| Baseline, decisions and review history | Immutable discovery/plan links above and this reconciliation |
| Native methods, findings and limitations | This directory and PR-linked exact-head evidence branch |

The six-file temporary plan and discovery pack are retired after this promotion;
their full reviewed history and probe remain recoverable in Git. Original bytes
are passed through an explicit private planning capture rather than a generic
download-session/cache abstraction. This preserves the planned ownership and
count contract without new public state. No runtime/UX product scope changed.

## Remaining gates

Before ready/merge: final-head native matrix, final full/hosted checks, separate
exact-head self-review and current reconciliation comment. Keep #74 open after
merge for nominated immutable-candidate acceptance/release. Neither this document
nor passing tests authorize a new publication, personal activation or modification
of immutable v1.0.0. Other roadmap outcomes, including lifecycle sync and Pi
acceptance, are not completed by this release-quota improvement.
