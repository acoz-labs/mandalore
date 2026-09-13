# Repository Instructions

This repository follows a role-based, tool-agnostic SDLC.

## Product and privacy boundary

Mandalore is a memory-only successor to My Friday. Follow `docs/product.md` and
`docs/migration.md`. Codex is first; Pi and Claude Code are later tracked work.
The repository and identifiers are lowercase `mandalore`; Mandalore is the
display name, a signet is a private bank and `this-is-the-way` is the memory skill.

Port reviewed memory-only components and tests from the pinned public source.
Do not copy predecessor Git history, private-agent capabilities, personal memory,
machine inventory, workstation paths, account policies or raw transcripts.
Use synthetic examples. Public attribution and upstream licenses remain intact.
No implicit live migration, plugin removal, bank rewrite or public release.

The initial template-derived bootstrap is authorized repository setup, not
runtime acceptance. Substantive porting follows the design/review workflow below.
Do not manufacture independent approval or acceptance.

## Current MVP review authorization

For the current issues #1–#10 delivery effort, the product owner has authorized
contributor self-review and delegated ordinary design decisions. Record decisions,
review findings, tests and the reviewed commit honestly; do not label self-review
as independent approval or impersonate another reviewer. Surface consequential
choices, deviations and unresolved risks at handoff so the owner can redirect.

A recorded self-review of the planning head is sufficient to begin implementation;
work may be stacked on that head while the planning PR awaits CI/merge. Do not
stop for separate product sign-off on ordinary in-scope decisions. This supersedes
the separate reviewer/approval and plan-merge prerequisites below for this effort.
Passing required checks, actual runtime verification, privacy boundaries and
explicit live migration/public release authority are not waived. Missing hosted
checks are not success. Independent product acceptance required by release
automation remains separate from engineering self-review.

Keep the durable rationale in `docs/decisions/0001-mvp-self-review.md`.

## Roles

- Product owners set intent, priority, and product judgment.
- Maintainers shape work, review PRs, verify releases, and preserve durable
  repository knowledge.
- Product design reviewers shape user flows, interaction behavior, visual direction,
  accessibility and localization requirements, and implementation-ready design
  acceptance criteria for substantive user-facing work.
- Contributors implement code, tests, docs, branches, and PR responses.

## Workflow

1. Capture an ambiguous product opportunity as a discovery issue. Use the
   owning repository whenever one is known.
2. Develop the decision in one scoped, contributor-authored pull request under
   `docs/discovery/<issue>-<slug>/`. Keep one complete `README.md`; add optional
   files only when the evidence or outcome map needs depth. A maintainer records
   product authority with an approval on the exact final head, then merges it.
3. Materialize only selected or deliberately deferred outcomes as self-contained
   delivery issues linked to the approved discovery head and outcome key.
4. Shape a bounded delivery issue before implementation.
5. For substantive user-facing work, complete the product-design gate before
   solution design.
6. Complete one contributor-authored solution-design planning PR under
   `docs/plans/<issue>-<slug>/`. Resolve maintainer findings internally, obtain
   the final product-authority approval, merge the planning-only PR, and only
   then move the issue to `Ready`.
7. Use `discovery/<issue>-<short-topic>` for discovery and
   `design/<issue>-<short-topic>` for the planning-only PR, then branch the
   approved implementation from `main` with `feature/<short-topic>`.
8. Use TDD for meaningful behavior changes.
9. Keep changes small, scoped, and documented.
10. Commit, push, open a draft PR linked from the issue lifecycle, and keep it
   draft while reconciliation is prepared.
11. Reconcile that draft's current head with the approved plan, explain drift,
   record the documentation-promotion matrix, update durable repository docs
   from the shipped behavior, and delete the temporary issue plan. Replace the
   current reconciliation and bind it to the exact head commit.
12. Verify the current reconciliation and plan removal. For meaningful rendered
    changes, complete `docs/operations/ui-acceptance.md` against the exact PR
    head and attach its evidence manifest. Then mark the PR ready with
    validation, design, docs, and deploy evidence.
13. Merge only after review and required checks.
14. Keep the linked issue open after merge when the change requires staging,
    product acceptance, or a production release.
    Candidate association uses only an explicit top-level `Refs`, `Closes`,
    `Fixes`, or `Resolves` PR line; narrative mentions and closed issues do not
    authorize nomination.
15. Accept only the exact nominated commit and immutable artifact. Service
    nominations come from staging; artifact repositories use the staging-free
    artifact-nomination workflow. The contributor who implemented the change
    must not be its sole product acceptor. Durable acceptance must be bound to
    the issue and its current implementation pull-request set; issue labels or
    comments alone are not release authority. Every lifecycle-linked
    implementation merge must be contained in the accepted candidate.
    UI acceptance repeats the hands-on scenario matrix and records fresh,
    openable exact-candidate evidence; contributor-local screenshots are not
    staging acceptance.

Generated managed-standard adoption issues may omit an issue-local Solution
Design plan only when they link the immutable reviewed template commit, carry
the managed-standard marker, change no product outcome, record repo-specific
deviations, pass CI, receive independent maintainer review, and pass the
post-merge stewardship audit. They complete after the reviewed merge passes
that audit, are excluded from candidate nomination, and do not authorize a
production promotion.

## Engineering Rules

- Prefer existing project patterns over new abstractions.
- Preserve framework-native behavior; do not force a shared visual style across
  unrelated products.
- Use YAGNI: build only what the current brief needs.
- Use SOLID as review pressure, not ceremony.
- Keep project knowledge in repo docs, not chat.
- Keep related behavior, data, contracts, authorization, invariants, failure,
  and operation coherent by capability. Do not copy one permanent document per
  solution-design stage.
- Keep Solution Design review in its planning pull request. Issues contain the
  compact product contract and lifecycle links, not copies of planning files.
- Do not commit secrets.
- Treat local and development-preview URLs as ephemeral review aids, not
  acceptance or release evidence.
- Build a release candidate once and promote the same immutable artifact through
  staging and production when this repository has deployed environments.
- Do not invent staging for an artifact repository; nominate its exact verified
  commit and artifact before acceptance and release.
- Use container-first validation unless this repo documents a different path.
- When host-local language execution is supported, commit exact runtime versions
  in `mise.toml`; keep ecosystem version files synchronized when other tooling
  still consumes them.

## Validation

Run:

```sh
bin/container bin/ci
```

If container support is not ready yet, run:

```sh
bin/ci
```

Document any required deviation in `docs/development.md`.
