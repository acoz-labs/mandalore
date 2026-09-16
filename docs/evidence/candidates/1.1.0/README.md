# 1.1.0 retained candidate — engineering checkpoints

This evidence branch is separate from the immutable candidate source. It does
not approve or publish the candidate, change personal installations, or finish
the broader roadmap. Original criteria and the full remaining matrix are in
the [candidate guide](../../../releases/1.1.0-candidate.md).

## Retention and nomination

- Source: `6f9bb5e4d3b79a1c5cf4947b428dcd5b31da0a41` (reviewed PR #90 merge).
- [Build/retention run](https://github.com/acoz-labs/mandalore/actions/runs/35057409524):
  passed source validation, built once, retained all eight files, attempt 1.
- Artifact ID `10431187273`, archive size 70,380,502 bytes; archive SHA-256
  `e40443fb1d5b5e22123eb1c93d733d86dc5babd27cfa0f4f2bc09af73711d688`.
- Candidate identity:
  `mandalore:6f9bb5e4d3b79a1c5cf4947b428dcd5b31da0a41:sha256:431046675befde9e68c0ff30fe3b31f636af18fb41701ce9a2ed4d00487fb120`.
- Retention expiry observed as 2026-12-15T04:54:15Z, subject to repository policy.
- Local maintainer verifier refreshed official provenance and checked every
  archived payload before extraction or execution. [Transport receipt](transport.json)
  and [verified payload/identity](verified.json) retain the exact observations.
- [Guarded nomination run](https://github.com/acoz-labs/mandalore/actions/runs/35057728812)
  independently passed. Workflow-authored exact-identity nomination markers were
  verified on #13, #56, #57, #61, #65, #74, #85 and #88. This is nomination,
  not a human acceptance verdict; all eight issues remain open.

Archive hashes and manifest hashes are different identities. Saved receipts do
not establish current availability; promotion must reverify provenance/expiry.
No Git tag or public Release was created. Published v1.0.0 is unchanged.

## Initial native checkpoint

The actual macOS arm64 executable has SHA-256
`5f19988a729637eea300949bce603796da831c9c1365c9f8dea043f4c0391d89`.
Its version response confirms source above, version 1.1.0, format/protocol 1 and
Codex plugin identity `2a8a47c24a9d7ba8580cac4ad37b6d7796ac94cacfb70c74d81417ee43ffcaa9`.
Embedded Pi package inspection reports version 1.1.0 and SHA-256
`013dcc492565ba180abd59eb50e3a6fbea2cd8ba045ccf067e17d51a050e7fda`.
Actual native versions: Codex 0.154.0 and Pi 0.85.1. Test drivers use pinned
Node 24.1.0; this receipt does not separately attest the native wrapper's interpreter.

Two synthetic signets/local bare origins were created with the exact artifact.
Each has a Cedar Orbit release-day decision: Thursday in A, Saturday in B.
Separate native profiles for both harnesses/banks installed successfully, without
provider authentication or credential copying. Subsequent model tests inherited
existing native login in place, not copied into those profiles. They used the
candidate runtime/package explicitly, unrelated working directories and
gpt-6-astra/high reasoning. These are not a claim of isolated native credentials.

| Actual model scenario | Result |
| --- | --- |
| Fresh Pi ordinary recall, bank A | Answered Thursday through memory_recall, without mentioning Mandalore in the question. |
| Fresh Pi ordinary recall, bank B | Answered Saturday through memory_recall; did not confuse the identical project scope in the other bank. |
| Ordinary confirmed change in A, no remember/cue instruction | Pi recalled and used memory_remember_and_sync once to replace Thursday with Friday. No separate sync call. |
| Fresh Codex recall from a different project directory | Used scopes/recall/history and answered Friday, previously Thursday. |

All three read-only scenarios preserved the complete synthetic bank/remote
fixture inventory (paths, file bytes, modes, mtimes and directories). The learning
case preserved B's bank/remote. Independent reads confirmed exactly two revisions
of A's same record, Pi authorship, retained supersession and stable device ID.
Actual local/bare-remote Git heads equal the successful combined-delivery receipt:
`0a56ae46ef0f8b3c29aec92b4104861a86c9a4c8`, zero semantic conflicts.
That local bare-remote test does not establish hosted Git authentication.

[Structured semantic results](initial-native.json) retain answers, operation
names, measured Pi result bytes, runtime/package identities and head verification.
Pi exposed one full text envelope per tool result and only operation/ok detail
metadata. These results are not a Codex model-visible duplication measurement;
its CLI's nested MCP events alone cannot prove what code mode printed. Raw
sessions, private machine paths, authentication and fixture drivers are not
published here. Native aggregate token counts are not Mandalore-only context cost.

## Remaining gates

The [second checkpoint](continuity-rendering.md) adds opposite-direction
correction, explicit no-sync, quoted-cue and no-filler delivery checks. It also
retains a first-error rendering failure: #56 is not ready for candidate acceptance
until that finding is addressed or explicitly resolved through product review.
Remaining candidate work includes additional prohibited cases,
failure/concurrency/context/lifecycle checks, progressive
foundling quality and actual code-mode presentation, rendered quota/readiness
matrices, isolated old→new update/recovery, and the human acceptance handoff.
Earlier implementation evidence is a baseline, not a substitute for exact-byte
candidate verification. Screen-reader, other-locale/font and other-platform
native acceptance remain unverified. No automatic lifecycle sync is implied.

Engineering self-review may review this evidence; it cannot provide the human
verdict or reuse the MVP-specific owner exception. Publication and live
installation still require their separate authority and unchanged gates.

## Receipt integrity

The JSON receipts are original verifier output or a bounded semantic projection
of observed native runs. Their hashes are recorded in the evidence commit's
review comment; Git preserves the exact published bytes. Re-run the provenance
verifier against the retained archive before relying on transport freshness.
