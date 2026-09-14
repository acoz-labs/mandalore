# Project setup receipt

## Published project

[Mandalore, project 35](https://github.com/orgs/acoz-labs/projects/35) is public
and linked only to `acoz-labs/mandalore`. It was copied from the organization's
Repo Project Template without copying draft issues or predecessor work items.

The copy was kept private while inspecting and sanitizing metadata. Owner
options in the public copy are Product owner, Maintainer and Contributor;
the source template was not modified. Native GitHub assignees may identify
actual contributors; the role field does not pretend someone accepted a task.

## Verified structure

All 22 field names/types match the template. Status, Area, Priority, Size,
Execution Lane and Confidence choices match; Owner is the documented privacy
deviation. Release remains unset until verified publication. Estimates and
confidence are not invented merely to fill empty cells.

| View | Layout | Filter | Columns/grouping |
| --- | --- | --- | --- |
| View 1 | Table | None | Template visible fields, no grouping |
| All Work | Table | None | Template visible fields, no grouping |
| Discovery | Table | `label:"work:discovery"` | Template visible fields, no grouping |
| Delivery | Board | `-label:"work:discovery"` | Status columns |

The template-visible fields are Title, Assignees, Status, Linked pull requests
and Sub-issues progress. GraphQL field/view/filter/grouping inspection verified
the copy; this is not a claim of a separate browser usability test.

All 16 successor product issues are attached with Status, Area, Priority, Owner role,
Execution Lane, Target and Next Action. Three milestones distinguish Memory MVP,
Cross-harness continuity and Later improvements. Exact live state remains in
GitHub, not this receipt.

One additional bootstrap diagnostic, #19, tracks missing hosted Actions runs.
It is attached with its own status and next action; the board now has 17 issues.

## Initial status decisions (historical bootstrap)

- #1: Review; archive complete, bootstrap receipt and hosted CI evidence pending.
- #2/#3: Solution Design; PR #17 needs independent review and final approval.
- #4–#11: Shaping, not implementation-ready merely because an issue exists.
- #13: Waiting for Codex acceptance; #14: Waiting for Pi acceptance.
- #12/#15/#16: Parked improvements. Baseline privacy applies immediately.

The old project is closed, retaining its items/statuses. The old repository is
archived with its branches, open issues/PRs and releases intact. Archival is
reversible; it does not delete data or complete unfinished delivery work.

## Checks and limitations

Local foundation validation and documentation-link checks passed using the
documented host fallback; Docker was unavailable. Source and planning branches
were published after account authorization for workflow and project access.
Hosted CI evidence must be checked separately: registered/enabled workflows do
not establish successful runs. No runtime port, independent product acceptance,
native installation or release occurred during bootstrap.

## Foundation reconciliation — 2026-09-14

The live project was re-inspected through GitHub: public, open, linked only to
`acoz-labs/mandalore`, with 22 fields, the four named views/layouts/filters above
and 17 issue items (the 16 product outcomes plus #19). No draft predecessor items
were copied. This refresh did not change views, field definitions or the template.

All 17 items have a real status and next action. Current phase groups are:

| Issues | Status and next step |
| --- | --- |
| #1 | Review: reconcile the complete predecessor ledger and this receipt through a docs-only PR and hosted CI |
| #2–#9, #12 | Acceptance: implementation merged; retain outcome-specific engineering evidence and complete independent exact-candidate acceptance |
| #10 | In Progress: remaining native/platform audit and independent candidate acceptance; contributor test results are not acceptance |
| #11 | Acceptance: retained candidate and nomination exist; deliberately configure independent acceptance/release authority before publication |
| #13, #14 | Waiting: Pi after Codex acceptance, Claude after Pi acceptance |
| #15, #16 | Parked: scoped readiness and privacy lifecycle improvements, not hidden MVP implementations |
| #19 | Shaping: hosted CI is operational; historical trigger cause remains unproven |

Stale next actions were refreshed, including #11's superseded candidate identity.
The current tested candidate is source `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`,
manifest `c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
[Retained contributor evidence](https://github.com/acoz-labs/mandalore/tree/4e2a6280c3a04d014d4eda66a1a4890f5e5a03e8/docs/evidence/mvp-candidate)
includes native memory/recovery checks, two-physical-Mac CLI continuity and 21
installer recordings. This ledger-only update does not rebuild or relabel it.

### Dependency meaning

An implementation prerequisite and an acceptance dependency are different phases.
The issue bodies preserve the engineering sequence: #2/#3 establish engine/format,
#4 the shared interface, #5 synchronization, #6 native integration, #7 setup,
#8 explicit migration and #9 measured retrieval. Foundling schema work coordinates
with #3, implementation is #12. #10 consumes their integrated artifact from #11.
Those implementation prerequisites have merged; their still-open acceptance
issues are not evidence that their code is unavailable.

The initial project had no native blocker links. This audit added and re-read the
actual still-waiting GitHub relationships: #13 blocked by #10, #14 blocked by #13,
and #15 blocked by #13. #15's existing #7 readiness prerequisite remains documented;
#7's implementation is merged and #10 will accept the integrated setup surface.
No synthetic cycle was added between #10 and #11: candidate construction precedes
#10, while public promotion follows independent acceptance. Nor are completed
engineering prerequisites made into new unresolved native blockers merely because
their acceptance issues stay open.

### Archive and authority

The [complete predecessor ledger](../migration.md#open-predecessor-work-ledger)
accounts for the observed 21 open issues and five open PRs, pins each PR head and
maps the memory-only source boundary. The archive still exposes the pinned feature
branch, transition notice and three historical releases. No predecessor item,
branch, tag or release was changed by reconciliation.

Foundation completion is not permission to accept or publish a runtime. The
initial missing-Actions incident in #19 does not negate later verified hosted CI,
but successful later runs do not establish its historical cause. Independent
acceptance actors, live migration and release settings remain outside this audit.
