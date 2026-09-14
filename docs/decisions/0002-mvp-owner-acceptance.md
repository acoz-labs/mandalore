# Candidate-specific MVP owner acceptance

Status: explicitly authorized by the product owner during the acceptance walkthrough.

Implementation PRs were filed through the owner's GitHub account. The owner
authorized personally performed product review through that same account.
This supersedes the independent-author restriction only within the scope below;
it is not an acceptance verdict or permission for agent self-acceptance.

## Exact scope

- Repository: `acoz-labs/mandalore`.
- Source: `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`.
- Artifact: `mandalore:5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f:sha256:c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
- Existing nominated delivery issues #2–#12, including #10's distribution and
  foundling prerequisites. Foundation #1 is already closed. Control issue #42
  must not be added to the old product candidate's implementation set.
- Workflow actor must match repository variable `MVP_ACCEPTANCE_OWNER` and also
  belong to the normal `ACCEPTANCE_ACTORS` allowlist. No private actor name is
  embedded in public code.
- Each exception submission requires `owner_review_confirmed: true`. The
  workflow default is false; merely configuring an owner cannot approve work.

The fail-closed `bin/mvp-owner-acceptance` predicate is shared by both candidate
and linked-implementation author checks. Any mismatched selector retains normal
refusal. Independent reviewers keep the existing path without owner confirmation.

## Review, recording and release

The owner personally performs the review, evaluates relevant retained evidence
and limitations, and gives an explicit approved or changes-required verdict.
An agent may prepare tests, observe results and submit that actual verdict when
directed. It must not infer approval from this policy or from engineering tests.
Record fresh evidence, actual reviewer identity, candidate and reviewed scope.
Workflow comments disclose owner-exception use rather than claiming independent
author approval. Nomination, ancestry, required checks, issue-specific evidence
and implementation-set binding remain unchanged.

This maintainer-control change does not rebuild the reviewed artifact. Acceptance
does not authorize publication, real-memory migration or credential changes.
Existing release authorization and post-publication verification remain separate.

## Revocation

Clear `MVP_ACCEPTANCE_OWNER` to prevent further exception submissions. That does
not erase earlier decisions; preserve their audit trail and explicitly record a
changes-required verdict when appropriate. Other candidates, artifacts and issues
require a separate authorized policy change. This repository-local deviation must
not propagate to the shared template.
