# Implementation handoff

## Smallest complete outcome

A buildable memory-only library with documented signet data semantics, complete
synthetic regression coverage and no assistant execution dependency. This does
not complete all of #3's cross-component migration/binding acceptance.

## Ordered slices

1. Pin Go/tooling; add failing clean-package/manifest/legacy-refusal tests.
2. Extract atomic files, directory publication, locks and device/source handling.
3. Extract graph/scope/history semantics and signet schema; run corruption and
   supersession regressions before modifying service consumers. Include data-only
   foundling registration/citation validation, not external source retrieval.
4. Extract journals, service validation, lexical recall, budgets and pagination.
5. Audit dependency closure, attribution and every copied fixture; run race/vet,
   platform builds and native filesystem tests where available.
6. Promote actual format, API, failure and development knowledge into durable
   docs, record remaining limits and remove this temporary plan in implementation.

## Review contract

After self-review is recorded on the final planning head under the MVP review
authorization, branch as `feature/memory-engine`. If the plan is not yet merged,
stack on the reviewed planning branch and retarget after merge. Link #2/#3 and
the planning PR. Keep the implementation PR draft until reconciled to its
exact head, evidence is attached and plan-removal/doc promotion are complete.
Use `Refs #2` and `Refs #3`; do not close release-bearing outcomes at code merge.

## Documentation promotion

Update `docs/architecture.md` from target to actual package boundaries; add a
focused `docs/signet-format.md` for the data contract; update development commands,
migration refusal behavior, security boundaries and runbook recovery. Preserve
source attribution in a neutral notice, without copying predecessor narratives.
The format documentation includes foundling citation/registration fields and
clearly distinguishes this data contract from the later #12 management workflow.

## Non-goals and reopening conditions

No CLI/menu, MCP server, plugin install, provider capability, secret manager,
reference import, Git remote creation, live migration or release. No Pi/Claude
implementation. Reopen design for an unavoidable assistant dependency, altered
memory semantics, incompatible storage assumption or a new destructive operation.
