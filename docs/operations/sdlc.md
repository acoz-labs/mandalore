# SDLC And Release Flow

This repository follows a role-based, tool-agnostic SDLC.

## Standard Flow

1. Capture an ambiguous opportunity as a discovery issue and move it to
   `Discovery` in the repository-linked Project.
2. Review one proportional `docs/discovery/<issue>-<slug>/` pull request. Gate 1
   is an authorized approval on the exact final head followed by merge; issue
   labels, comments, and Project state are navigation, not authority.
3. Materialize selected or deliberately deferred outcomes as self-contained
   delivery issues with immutable PR/head/outcome provenance. Promote durable
   decisions and remove the temporary discovery pack in a reviewed cleanup.
4. Shape acceptance criteria, risks, tests, and docs expectations for a bounded
   delivery issue.
5. Complete the product-design gate for substantive user-facing work.
6. Create one temporary issue plan under `docs/plans/<issue>-<slug>/` and one
   planning-only pull request. Complete context, solution decision, technical
   design, verification/release design, and implementation handoff there.
5. Resolve contributor/maintainer review internally. Obtain one final product-
   authority approval, record the execution envelope, merge the approved plan,
   and only then move the issue to `Ready`.
6. The contributor branches from `main` using `feature/<short-topic>`.
7. The contributor writes tests first for meaningful behavior changes.
8. The contributor implements, runs checks, reviews the diff, commits, pushes,
   opens a draft implementation pull request, and links it from the issue.
9. Against that draft's exact current head, the contributor reconciles the
   implementation with the approved plan. Every drift has a located reason;
   unexplained drift is resolved or returned to solution design. The
   contributor updates durable repository docs, deletes the temporary issue
   plan, replaces the current PR reconciliation, pushes, and verifies the
   reconciliation and plan removal before the PR leaves draft. An independent
   maintainer reviews the reconciled current head and complete diff.
10. Product design reviewers assess the implemented experience when the design
   gate applied. For every meaningful rendered change, the contributor
   completes the UI Acceptance Evidence contract against the exact PR head and
   attaches an openable evidence manifest. Maintainers review the complete
   change. The contributor addresses feedback.
11. Merge after approval and passing checks.
12. Keep a release-bearing issue open and move it from `Review` to
    `Acceptance`; do not equate merge with completion.
13. When this repository has staging, automatically deploy the successful
    immutable `main` candidate and run staging smoke checks.
14. An acceptor other than the implementing contributor exercises the original
    criteria against the exact candidate and records evidence.
15. Move an approved issue to `Ready for Production`. Return a rejected issue to
    `In Progress`; capture non-blocking discoveries as separately prioritized
    follow-up issues.
16. Promote the accepted artifact through the repository's production workflow.
17. After production smoke checks pass, create the production Git tag and
    GitHub Release, record the release on the issue, close it, and move it to
    `Done`.

## Delivery States

The repository-linked GitHub Project uses these delivery states:

```text
Inbox -> Discovery -> Done
Inbox -> Shaping -> Solution Design -> Ready -> In Progress -> Review -> Acceptance
Acceptance -> Ready for Production -> Done
Acceptance -> In Progress
```

- `Review` means a pull request is receiving engineering and design review.
- `Discovery` means an ambiguous product decision is being evidenced and
  reviewed. It does not grant approval authority.
- `Solution Design` means one repository-native planning pull request is being
  authored, challenged, revised, and prepared for the final product-authority
  approval.
- `Acceptance` means the reviewed change is merged and its exact candidate is
  receiving independent hands-on product verification.
- `Ready for Production` means the candidate is accepted and may be promoted
  under this repository's release policy.
- `Done` means the documented delivery profile is complete. For a production
  service, that includes a verified production deployment and GitHub Release.
  For a discovery issue, it means the decision, outcome links, promotion, and
  reviewed discovery-pack removal are complete; child outcomes need not be
  shipped.

`Waiting` remains the state for a real external dependency or owner decision.
Do not use it merely because acceptance found an implementation defect.

## Solution Design And Reconciliation

Use one planning-only pull request and one temporary issue directory copied from
`docs/plans/_template/`. The issue body remains the compact product contract and
lifecycle ledger. Inline comments and review summaries preserve the
contributor/maintainer exchange without copying design prose into the issue.

The standard pack contains context, solution decision, technical design,
verification/release design, and implementation handoff. A narrow change may
combine them; a broad design may split a substantial view. Scale content and
diagrams to risk, but keep the same evidence, authorization, test, rollout,
rollback, non-goal, and handoff questions.

Intermediate document completion is an internal quality checkpoint, not a
product-authority gate. After maintainer findings are resolved, one final
approval accepts the complete plan and its execution envelope:

- `implementation` authorizes code, tests, documentation, and the reviewed
  implementation pull request;
- `through-staging` additionally authorizes merge, staging, and independent
  product acceptance under repository policy;
- `through-production` additionally authorizes promotion and verification under
  the repository's production release policy.

Material product-contract change, contradictory verified evidence, a new trust
or data boundary, an unplanned irreversible risk, or work outside that envelope
returns to Solution Design. Ordinary implementation detail does not.

The generated managed-repository-standard adoption issue is the narrow
low-ceremony exception to an issue-local Solution Design plan. It must link the
immutable reviewed template commit and machine-readable adoption marker, change
no product outcome, document repository-specific deviations, pass CI, receive
independent maintainer review, and pass the post-merge stewardship audit. This
exception ends at merge and never authorizes production.

The merged planning pull request is the `Ready` gate. Reconciliation is the
pre-review check: compare the diff with the approved plan, explain drift with a
commit, test, review, or recorded decision, promote durable knowledge, and
delete the temporary plan. Git history and the planning pull request preserve
the journey; current repository docs describe what actually ships.

## Documentation Promotion

The implementation handoff nominates likely documentation destinations.
Reconciliation makes the final choice from the actual code and tests, then
records the design concern, shipped result, repository destination, action, and
useful provenance.

Choose the destination by the durable question it answers:

| Type | Use it for |
|---|---|
| `docs/architecture.md` | System shape and major boundaries |
| `docs/architecture/<capability>.md` | One capability's behavior, data, interfaces, authorization, invariants, failure, and operation |
| Contract or security reference | A substantial stable surface with a distinct audience |
| `docs/decisions/*.md` | Consequential choices likely to be questioned later |
| `docs/development.md` | Contributor setup, tests, and tooling |
| `docs/deployment.md` | Candidate, environment, promotion, verification, and rollback contract |
| `docs/runbook.md` or `docs/runbooks/` | Recurring diagnosis and recovery |
| User or admin guide | Shipped usage and administration behavior |

Flows and ERDs are embedded views in the smallest relevant system or capability
document, not permanent files corresponding one-for-one with design stages. Use
`docs/architecture/0000-capability-template.md` for a durable end-to-end
capability. Promoted diagrams show the shipped state, cite code or schema
locations, and explain semantics or invariants the diagram cannot express.
Link authoritative generated schemas instead of duplicating them.

Update docs in the implementation branch before review. Reconciliation records
`Update`, `Create`, `Supersede`, or `Not needed` for each durable concern. An
independent maintainer verifies both the destination and content. `Not needed`
requires a reason why no future contributor, operator, user, or administrator
needs the knowledge after the issue closes.

## Delivery Profile

Every repository declares exactly one profile in its repo-specific section:

- `service`: a running application or site with staging and production targets.
- `artifact`: a package, document set, media output, dataset, or other versioned
  deliverable whose candidate can be accepted without a long-running service.
- `non-deployable`: exploratory or internal source with no release target yet.

The state names remain consistent, but environment gates apply only when the
environment exists. A missing staging or production target is a documented
capability gap, not a fictional deployment.

## Development Preview

During implementation, a contributor may expose the current branch through an
ephemeral private preview so a reviewer or product owner can answer a bounded
question. Record the branch, commit, URL, scenario, owner, and expiry in the
issue or pull request.

A development preview is mutable and may use development data. It is not
staging, product acceptance, production authorization, or release evidence.
Keep the application bound to loopback and use the repository or operator's
approved private reverse-proxy mechanism; do not create public ingress merely
for review.

## Staging And Product Acceptance

For `service` repositories, successful CI on `main` builds one immutable
candidate and automatically deploys that same candidate to isolated staging.
Staging must not share production secrets, writable storage, or irreplaceable
data. The deployment records its commit, artifact identifier or digest, URL,
and smoke-check result in GitHub.

Product acceptance is independent of implementation self-checks. The acceptor
must pass the immutable artifact identifier into the acceptance workflow,
verify the original criteria against that exact commit and artifact, and record:

- commit SHA and artifact digest or identifier
- staging deployment and workflow links
- criteria and scenarios exercised
- durable links to exact-candidate evidence; meaningful rendered changes require
  the fresh screenshot, recording, trace, or visual-diff manifest defined in
  `docs/operations/ui-acceptance.md`
- acceptor and timestamp
- approved or changes-required decision
- migrations, rollback concerns, and linked follow-up issues

An agreed acceptance failure blocks that candidate and returns the issue to
`In Progress`. A newly discovered improvement outside the accepted scope does
not block release; create a linked issue and prioritize it normally.

## UI Acceptance Evidence

Use `docs/operations/ui-acceptance.md` for every meaningful browser or native
interface change. It separates functional coverage, visual-regression
detection, and first-version product judgment; defines the scenario matrix and
evidence manifest; blocks UI pull-request readiness without openable exact-head
evidence; and requires fresh independent exact-candidate evidence during
product acceptance. Repositories choose a stack-appropriate visual backend and
record it there rather than inheriting a universal framework or hosted service.

For `artifact` repositories, `Nominate artifact candidate` verifies the exact
successful commit without inventing staging and attaches the same evidence to
the immutable built artifact. For `non-deployable` repositories, merge-time verification may finish
the issue only when the issue and PR explicitly record that no release exists.

Approved or rejected acceptance writes the stable `product-acceptance` commit
status and an artifact-digest-specific status. The issue comment records the
plain artifact identifier and a machine-readable SHA/artifact marker. Release
gates require both successful status contexts, and release finalization requires
the matching issue marker, so changing an artifact input on the same commit
cannot reuse unrelated acceptance.
Every implementation pull request linked from the issue lifecycle must be
merged into the accepted candidate's ancestry; a merely merged PR that is not
present in that SHA cannot borrow acceptance authority.
Each application staging or artifact-validation result also nominates the
issue's active candidate with a machine-readable SHA, artifact digest, source,
source id, and intent.
Evidence-only reruns of the already accepted exact candidate do not supersede
that nomination. Acceptance fails closed unless its SHA and artifact match the
latest nominated application candidate, so an older historically staged build
cannot be approved after a corrective candidate appears.
Because a release candidate is cumulative, service staging and artifact
nomination re-nominate every open issue already in acceptance or a release-ready
state along with open issues linked by an explicit top-level `Refs`, `Closes`,
`Fixes`, or `Resolves` line in the new merge's pull request. Narrative issue
mentions and closed issues are not nomination authority. Finalization fails while
any member of that candidate still awaits acceptance; commit-wide status
evidence cannot substitute for each issue's ready label and exact acceptance
marker.

## Automation Contract

The managed workflow files fail closed and use these repository variables:

- `CI_RUNNER`: approved runner label; required by every job.
- `ACCEPTANCE_ACTORS`: comma-separated GitHub logins allowed to record product
  acceptance; required.
- `ACCEPTANCE_REQUIRED_CHECKS`: comma-separated check-run names; defaults to
  `ci`.
- `STAGING_ENABLED`: `true` only when `bin/deploy-staging` and
  `bin/verify-staging` implement a real isolated target.
- `STAGING_REQUIRED`: `true` for service releases; set `false` only for a
  documented artifact or non-deployable profile.
- `STAGING_ENVIRONMENT`: defaults to `staging`.
- `STAGING_URL`: stable URL for the staged candidate.
- `PRODUCTION_ENABLED`: `true` only when `bin/deploy-production` and
  `bin/verify-production` are implemented and rollback is documented.
- `PRODUCTION_URL`: canonical production URL.
- `PRODUCTION_REQUIRED_SECRETS`: comma-separated injected environment names,
  or the explicit value `none`; values are never recorded.
- `PRODUCTION_ACTIVATION_MODE`: `deploy-command` or `separate-command`.

`Deploy staging` follows successful trusted `main` CI, deploys the exact CI SHA,
records a GitHub Deployment, and labels linked issues `delivery:acceptance`.
The deployment request sets `required_contexts` to an empty list because the
workflow already proves candidate CI through its trusted `workflow_run`
handoff. This prevents an unrelated concurrent main-branch audit from racing or
deadlocking staging on a single shared runner.
Staging deployments serialize and never cancel an in-flight deployment; this
keeps deployment status and rollback evidence complete even when merges are
close together.
`Product acceptance` verifies the candidate, artifact-matched staging
deployment when staging exists, authorized acceptor, successful checks, and
implementer independence before writing both shared candidate statuses and
workflow-authored evidence bound to the issue, artifact, authorized acceptor,
and current set of linked implementation pull requests. Finalization recomputes
that set, rechecks the latest workflow-authored application nomination, and
requires the exact workflow-authored status and acceptor. Human-authored marker
text is never authority, so a mutable issue label or forged comment cannot
borrow another issue's acceptance.
`Release production` requires the same
artifact identifier, runs the executable production-readiness preflight,
promotes the supplied immutable artifact, records its exact deployment receipt,
verifies and activates production, creates the tag and GitHub Release, clears
pre-release labels, updates the lifecycle Release link, and closes the included
issues.

Promotion, exact-candidate receipt, verification/activation, and ledger finalization are
separately resumable. A receipt records application SHA, artifact, deployment
id, control/workflow SHA, linked issues, and rollback target. Once promotion
completes, an evidence failure leaves a `verification pending` receipt;
`finalize-only` re-verifies the live exact artifact without redeploying.
`force-promote-and-finalize` is the explicit recovery when a fresh promotion is
actually required. A successful receipt remains valid historical evidence even
if a newer environment deployment marks it inactive.
If execution stops after deploy but before that marker, the retry runs
`bin/verify-production-candidate` first and records the promoted state when the
exact artifact is already present. The probe returns `0` only when that candidate
is active, `3` only when the provider conclusively shows it is inactive, and any
other status for an inconclusive query or operational error. Only the explicit
inactive result permits the default path to deploy again; an inconclusive result
stops, while `force-promote-and-finalize` remains the deliberate override. A
historically successful receipt is verified but never reactivated during
evidence-only finalization.

Service releases call `bin/finalize-release production`; artifact repositories
use the managed `Release artifact` workflow and `bin/finalize-release artifact`. The
operator still supplies the explicit release issue set. Before publishing a
ledger, the finalizer compares that set with every open issue carrying the
mode's release-ready label and fails closed if one was omitted or lacks matching
SHA/artifact acceptance evidence. GitHub query failures also abort instead of
appearing as an empty set. This keeps carried-forward accepted work complete
without inferring scope from every historical pull request.
Managed-standard adoption issues and lifecycle entries that declare
`Release: not applicable` are excluded from candidate nomination. Their
completion boundary is the reviewed merge plus stewardship audit, not product
acceptance or release.
GitHub-generated release notes supply commit-to-pull-request associations; the
finalizer never guesses PR numbers by parsing `#123` fragments from titles or
commit subjects.

Release finalization is retry-safe. A rerun reuses an existing matching
SHA-specific tag and GitHub Release only when the Release ledger also names the
exact artifact, fails if the matching tag points to another commit or artifact,
and does not duplicate the exact release comment. For each included issue, it
preserves the comment as recovery evidence, removes all transient delivery
labels, applies `delivery:released`, and closes the issue. An interrupted run
can therefore finish the remaining issue transitions without rewriting the
tag, Release, acceptance status, or earlier comments.
When `ACCEPTANCE_COMPLETES_DELIVERY=true`, an exact approval retry remains
idempotent after closure, while a later authenticated `changes-required`
decision reopens the accepted issue and writes blocking status evidence.
Label cleanup only requests deletion for labels observed on the issue and
propagates deletion failures; a retry re-reads current state and safely
completes any remaining transitions.

GitHub Project `Status` remains the planning view. The private control-plane
release closer aligns `Status=Done`, `Release=<tag>`, and `Next Action` with the
GitHub-native release ledger after the product workflow closes the issue.

## Product Design Lane

Complete product design review before implementation for net-new user flows,
substantive redesigns, design-system foundations, or changes where hierarchy,
interaction, accessibility, or product identity materially changes. Small copy
fixes and in-system adjustments may bypass the gate when the issue records why.

The design gate produces the smallest useful implementation contract:

- target users, critical task, context, and measurable outcome
- project-specific design thesis and relevant existing patterns
- flow or information architecture, including loading, empty, error, success,
  permission, and recovery states
- semantic token or component-behavior decisions when foundations change
- accessibility, responsive, localization, motion, and performance acceptance
  criteria
- evidence required for review, including desktop and mobile screenshots for
  visual changes
- the UI acceptance classification, scenario matrix, retained evidence
  manifest, and visual-regression strategy defined by
  `docs/operations/ui-acceptance.md`

Product design review owns design intent and critique, not feature priority or
production code. Contributors map the contract to framework-native components
and document justified deviations. Heuristic review identifies risks; tests,
representative users, and production evidence determine whether the design
works.

## Dependency Stewardship Lane

Dependency updates follow the standard flow, with extra automation and gates:

1. Dependabot opens version-update pull requests after the configured cooldown.
2. Security updates bypass the normal cooldown and are triaged promptly.
3. Dependency PRs run CI and dependency review when available.
4. Patch updates may merge through the dependency merge steward only when the
   repo's dependency policy allows it and non-steward checks pass.
   The pull-request event checks once and exits; it never occupies the shared
   runner while waiting. Check-completion events and the daily sweep re-evaluate
   candidates.
5. Major updates, new dependencies, runtime-sensitive packages, Docker/base
   images, and workflow-permission changes require human review.
6. Production deploys still require this repo's staging and production policy.

See `docs/operations/dependencies.md`.

## Actions Runner Lane

Every GitHub Actions job reads the required repository variable `CI_RUNNER`.
Its value identifies the approved runner label for this repository. There is no
GitHub-hosted fallback: an unset or quarantined variable must fail closed rather
than silently consume hosted-runner capacity. Preserve existing job IDs and
check names when adopting this contract so release gates and review rules keep
working.

Self-hosted runners may execute repository code with host-level consequences.
Use this lane only for reviewed private repositories, keep public-repository
access disabled, and do not route outside forks or otherwise untrusted code to
it. Pull-request CI may install changed dependencies, so the contributor and
automation trust boundary must be explicit before a repository is admitted.

Dependency review evaluates changed manifests through its reviewed action
without checking out pull-request code. Dependency stewardship uses reviewed
actions plus a trusted base checkout. Audit jobs run after changes reach
`main`. Release and deployment jobs retain their existing permissions,
environment gates, manual attestations, and secret-handling rules; changing the
runner does not weaken those controls. Pre-provision every CLI or system package
these workflows need.

To quarantine Actions without spending hosted minutes, set `CI_RUNNER` to a
label no runner owns or stop the approved runner. Do not substitute
`ubuntu-latest` as an automatic recovery path.

## Cascading Standard Changes

When the managed repository baseline changes, update this template, publish the
standards, and cascade the change to existing managed repos in `acoz-labs`.

Use:

```sh
bin/publish-repo-standards
```

That command publishes only after the repository Project template has a
verified field/view receipt. Cascade is a separate explicit reviewed wave with
a finite target list; it never enumerates the organization. Cascade work is
tracked as repo issues/PRs and never silently rewrites live repositories.

## Soft Enforcement

Private repositories on GitHub Free may not have server-side branch protection.
When unavailable, use compensating controls:

- The local pre-push hook blocks direct pushes to `main`/`master`.
- `Main branch audit` flags direct pushes after the fact. It gives GitHub up to
  55 seconds to index a squash merge's commit-to-PR association before raising
  an incident, avoiding false alerts immediately after a reviewed merge.
- The dependency merge steward self-verifies safe Dependabot patch updates
  instead of relying on GitHub native auto-merge.
- Production deploy workflows, when added, should verify current `main`,
  required checks, and merged PR association before promotion.

## Production

Merge never deploys production. Production promotion requires the exact
candidate to have successful required checks, successful staging or artifact
verification, independent product acceptance, a rollback path, and no
unresolved P0/P1 release findings.

Normal product acceptance authorizes routine production promotion; repositories
must explicitly identify changes that reserve a separate owner approval because
of security, data, migration, spending, legal, or materially different product
risk. Break-glass releases record the reason and receive after-the-fact review.

Production must deploy the accepted artifact rather than rebuilding mutable
source. After deploy and smoke verification succeed, create a production tag
such as `prod-YYYY.MM.DD-<short-sha>` and publish a GitHub Release containing:

- deployed commit and artifact digest or identifier
- included issues and pull requests
- staging and product-acceptance evidence
- migrations and configuration changes
- production smoke result
- rollback target and operator notes

The Git tag and Release are the production ledger. Publish them after
production verification so they describe software that actually reached
production. If release-ledger publication fails after a successful deploy, the
workflow remains failed until the ledger is repaired.

## Repo-Specific Environment Contract

Replace this template section during repository onboarding:

```text
Delivery profile: non-deployable
Development preview: repository-local only
Staging: not configured
Production: not configured
Release command: not applicable
Smoke command: not applicable
Rollback: not applicable
```
