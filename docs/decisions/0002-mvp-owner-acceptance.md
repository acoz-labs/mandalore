# Candidate-specific MVP owner acceptance

Status: explicitly authorized by the product owner during the acceptance walkthrough.

Implementation PRs were filed through the owner's GitHub account. The owner
authorized personally performed product review through that same account.
This supersedes the independent-author restriction only within the scope below;
it is not an acceptance verdict or permission for agent self-acceptance.

## Exact scope

- Repository: `acoz-labs/mandalore`.
- Original source: `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`.
- Original artifact: `mandalore:5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f:sha256:c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
- Original pair: nominated delivery issues #2–#12 only, including #10's
  distribution and foundling prerequisites.
- Replacement source: `f899cf6a2a255f3b6b35dcd778c672f799c65eb2`.
- Replacement artifact: `mandalore:f899cf6a2a255f3b6b35dcd778c672f799c65eb2:sha256:c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea`.
- Replacement pair: nominated delivery issues #2–#12 and #45 (The Armorer).
  Source/artifact pairs cannot be mixed. Foundation #1 is already closed.
  Maintainer-control issues #42 and #48 are outside both product candidate sets.
- Workflow actor must match repository variable `MVP_ACCEPTANCE_OWNER` and also
  belong to the normal `ACCEPTANCE_ACTORS` allowlist. No private actor name is
  embedded in public code.
- Each exception submission requires `owner_review_confirmed: true`. The
  workflow default is false; merely configuring an owner cannot approve work.

The fail-closed `bin/mvp-owner-acceptance` predicate is shared by both candidate
and linked-implementation author checks. Any mismatched selector retains normal
refusal. Independent reviewers keep the existing path without owner confirmation.

## Superseding scope extension — 2026-09-14

After personally driving the replacement candidate's Armorer health, repair
preview/application and fresh-session recall walkthrough, the owner explicitly
authorized extending the exception to that exact candidate and #45. Issue #48
records the extension; it does not reinterpret previous "done" messages as an
acceptance verdict. The original pair remains eligible only for its original
issues. The added pair does not authorize future builds, ancestry-based matching,
agent self-acceptance or publication. No acceptance was recorded by this change.

The [sanitized walkthrough observations](https://github.com/acoz-labs/mandalore/blob/06f2dede753ec4065941017033cdbcafd1ba68c9/docs/evidence/armorer/retained-candidate.md)
distinguish the actual checks from retained historical evidence and their limits.
They are not a substitute for issue-specific acceptance evidence or owner verdict.

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
