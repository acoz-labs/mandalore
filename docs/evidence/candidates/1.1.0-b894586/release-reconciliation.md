# 1.1.0 release reconciliation

Audit dated September 16, 2026. **Not yet ready for publication.** This is an
evidence/checklist update, not changed candidate bytes, a new acceptance policy,
an issue-level approval or authority to update an existing installation.

## Follow-up: acceptance authority resolved

**Later engineering result:** fresh retained-byte testing found plain-menu
cancellation and direct-cue intent failures, now tracked as
[#99](https://github.com/acoz-labs/mandalore/issues/99) and
[#100](https://github.com/acoz-labs/mandalore/issues/100). See the
[complete follow-up and counterexamples](engineering/README.md).
This candidate is not ready for product sign-off or publication. The table below
is the earlier audit snapshot; the follow-up records what was actually refreshed,
what failed and which remaining rendered matrices were stopped. No verdict or
candidate authority transfers to a future correction.

After this audit the owner explicitly authorized the narrowly scoped personally
performed same-account review extension. [Control issue #96](https://github.com/acoz-labs/mandalore/issues/96)
and reviewed [PR #98](https://github.com/acoz-labs/mandalore/pull/98), merged as
`7d9ec5c3cd2083e41bf56c47aee3e4295fe0aef9`, implement that decision for this exact
source/artifact pair and its nine nominated issues only. Local full CI,
[exact-head PR CI](https://github.com/acoz-labs/mandalore/actions/runs/35130922685)
and the [main audit](https://github.com/acoz-labs/mandalore/actions/runs/35131096955)
passed. Failing-first scope/recorder tests preserve the old scopes and normal
allowlisting, confirmation, author checks and release gates. No actor
configuration was changed.

This supersedes item 3's unresolved authority question below, not the original
audit evidence or any remaining tests/verdicts. The owner must still personally
review and supply actual issue-level judgments; no acceptance was dispatched.
Main now includes control-policy commits, while the retained product source and
manifest remain the exact values below. Rechecked manifest and macOS arm64
runtime hashes are unchanged. This control issue/PR is not added to the retained
product implementation sets. No rebuild, nomination, publication or live update.

## Verified current state

- Product source: `b89458617b49d2eb0a8686dcd75c26aa7ea6f6ba`; the later control
  commits described above are on main without changing these retained bytes.
- Manifest SHA-256: `7995f820a7182a95b828851afce12bacea7108f82afac9369d307a43d961f45a`.
- [Build](https://github.com/acoz-labs/mandalore/actions/runs/35105578901),
  [CI](https://github.com/acoz-labs/mandalore/actions/runs/35105564763),
  [main audit](https://github.com/acoz-labs/mandalore/actions/runs/35105564820)
  and [nomination](https://github.com/acoz-labs/mandalore/actions/runs/35105986297)
  succeeded. The nomination remains current on all nine issues below.
- Artifact `10450157997` remains available, not expired; recorded expiry is
  December 15, 2026 at 14:00:18 UTC. GitHub's archive digest matches the retained
  [transport receipt](transport.json). Recheck availability before promotion.
- All ten implementation merges in the table are contained in this candidate.
  No open PRs were present. Latest published release remains v1.0.0 at
  `f899cf6a2a255f3b6b35dcd778c672f799c65eb2`.
- No candidate product-acceptance statuses are recorded. The issue comments'
  nomination markers are readiness for acceptance, not approval.

## Evidence coverage, without transferring verdicts

The [previous-candidate handoff](https://github.com/acoz-labs/mandalore/blob/5d18cbcd5bc91d957ddad9abbb80a32cb7695d46/docs/evidence/candidates/1.1.0-bb16a57/acceptance-handoff.md)
retains the broader engineering matrices and their limitations. Those checks
used `bb16a57`, not this replacement candidate. They remain regression baselines;
unchanged components do not convert their recordings into fresh artifact tests.
The [candidate guide](../../../releases/1.1.0-candidate.md) still defines the full
required matrix. No criterion is silently waived here.

| Issue | Implementation PRs | Current candidate evidence and remaining work |
| --- | --- | --- |
| #56, faithful MCP presentation | #59, #90, #92 | Fresh owner session printed one envelope for each of three memory results and recalled the right fact. Remaining: error/conflict/truncation/fallback and wire-versus-visible comparisons under the full contract. |
| #61, explicit consolidation | #63, #90 | Source CI passes. Direct/quoted/prohibited/offline/missing-connection model scenarios remain on the prior candidate; repeat the applicable current-artifact matrix. |
| #65, save-and-deliver | #67, #90 | Owner recall performed no new saves. Combined save, delivery failure, cancellation/contention and ambiguous-response matrix remains to be refreshed. Do not count a successful read as save-and-deliver acceptance. |
| #57, progressive foundlings | #69, #90 | Prior quality/expansion/provenance checks are retained. Refresh the current-artifact matrix and assess the efficiency result explicitly; overall native context savings have not been demonstrated. |
| #13, Pi integration | #72, #90 | Fresh Codex correctly read an existing Pi-authored correction and preference. Pi authorship occurred on the older candidate. Current Pi setup/lifecycle, reverse continuity, isolation and recovery still need their applicable exact-artifact evidence. |
| #74, quota recovery | #76, #90 | Prior request-count and rendered scenarios remain baselines. Refresh current checks and distinguish exact-source injection drivers from retained-executable runs. Narrow copyability #77 remains separate. |
| #85, readiness | #87, #90 | Fresh human Armorer structural inspection passed. This does not replace the static non-execution/state matrix or full rendered/cancellation matrix for both harnesses. |
| #88, candidate integration | #90, #92, #95 | Identity, CI, eight-asset verification and nomination pass. Broader current-artifact update/recovery/rollback evidence and final human judgments remain outstanding. |
| #93, first-use Codex | #95 | Real retained-binary first-use and diagnostic checks pass; owner completed setup, inspection and automatic recall and accepted that experience. Formal issue acceptance remains subject to authority and workflow gates. |

The #93 issue body lacked the lifecycle implementation link consumed by
`bin/record-product-acceptance`. Reconciliation adds #95 there and adds the
contained integration fix #95 to #88's implementation set. These bookkeeping
changes do not approve either issue; subsequent acceptance must bind the updated
implementation sets.

## Human observation already complete

The [owner-operated observation](https://github.com/acoz-labs/mandalore/issues/93#issuecomment-5701464900)
and [affirmative scenario judgment](https://github.com/acoz-labs/mandalore/issues/93#issuecomment-5701496652)
are bound to this candidate. No repeat of this accepted scenario is requested:

- Fresh Codex profile installation reached verified; structural inspection passed.
- An ordinary question, without naming Mandalore, recalled the current Saturday
  maintenance window and concise-update preference from the intended test bank.
- Earlier Thursday guidance remained superseded; no timezone was invented.
- Synchronization truthfully reported local-only delivery; no new memory/journal
  records were created and the tracked test bank remained unchanged.

This is product judgment for the tested setup-and-recall scenario, not approval
of all nine issues. Raw native session logs remain private; public evidence is
sanitized. No private signet data or credentials are included.

## Remaining sequence

1. **Finish exact-artifact engineering coverage.** Reuse the existing synthetic
   drivers and scenarios against retained bytes; do not rebuild the candidate.
   Start with deterministic CLI, transport, install, readiness, quota and recovery
   checks. Schedule only genuinely missing native/model scenarios; do not consume
   provider allowance simply to repeat the accepted onboarding/recall journey.
   Record each outcome under its actual artifact and executable identity.
2. **Resolve the efficiency judgment.** #57 demonstrates smaller initial payloads,
   deliberate expansion and quality checks, not guaranteed total-context savings.
   Prior repeated previews and an empty EOF read remain disclosed. Evaluate the
   original bounded-budget/repeated-fetch criteria on the refreshed evidence;
   obtain an explicit product judgment or return the affected behavior to work.
3. **Resolve acceptance authority.** The current predicate in
   `bin/mvp-owner-acceptance` permits only two old MVP source/artifact pairs.
   It rejects this candidate. Either use an authorized independent acceptor or
   obtain explicit authority for a narrowly scoped personally performed owner
   review exception, then implement and test that policy separately. Do not
   generalize the MVP exception, impersonate an independent reviewer or treat
   engineering self-review as product approval.
4. **Record actual per-issue verdicts** against the current nomination and
   lifecycle PR sets through Product acceptance, with openable evidence and
   honest limits. A changes-required result may require a new candidate.
5. **Ask for publication authority**, promote the same accepted bytes, then verify
   downloadable assets and delivery ledger. Activating a personal installation
   remains a separate choice. No publication or live activation occurred here.

Lifecycle checkpoints #55, export #80, withdrawal #81, copyable narrow recovery
#77 and Claude support #14 remain outside this candidate's completion claims.
Cross-builds are not native certification of Linux/other architectures. No
screen-reader, alternate locale or font certification is implied.
