# Handoff

This is a bounded implementation/evaluation slice, not a semantic search platform.
The smallest complete outcome is reproducible quality/scale/context evidence,
documented lexical and operating limits, plus regression-tested bounded fixes if
the evidence justifies them. A table of speculative performance targets alone is
not completion.

## Order

1. Add deterministic quality/fixture/measurement-sanity tests and capture baseline
   at the exact repository basis. Existing fixture semantics must be validated.
2. Add opt-in scale and end-to-end payload/latency measurements; retain native
   observations from #12 as prior evidence, not a final candidate baseline.
3. Reproduce/profile any material issue; apply the smallest in-scope correction
   with a failing regression and same-fixture comparison. Preserve graph, scope,
   full-claim, freshness, privacy and cancellation boundaries.
4. If guidance changes, test fresh native read-only continuity and authorized
   learning in disposable banks through the designated pane. Do not tune by
   silently dropping provenance or suppressing warranted recovery.
5. Reconcile findings, limitations, thresholds and next actions into durable docs;
   complete full/local/hosted checks and exact-head engineering review.

Likely ownership: `internal/memory/*_test.go`, `internal/codex/*_test.go`, shared
API/MCP process tests and narrow existing production functions only when justified.
No permanent test framework abstraction is needed unless fixtures actually share
the same semantics. Scale experiments stay opt-in; CI owns deterministic correctness.

## Documentation and review

Promote measurements/reproduction into a focused `docs/retrieval.md` and retained
synthetic evidence; link relevant contracts from `docs/interface.md`, native plugin
guidance, architecture/development and roadmap. Keep units, exact sources, hardware
class and limitations together. Do not embed private hostname, paths, account policy
or raw native transcripts. Update #9 and candidate #10 follow-up from actual results.

Use `design/9-retrieval-evaluation` for this planning-only PR, then
`feature/retrieval-evaluation` for implementation after the authorized exact-head
planning self-review. Planning content is temporary; implementation promotes its
decisions and deletes all six files before leaving draft. No independent review or
acceptance is fabricated.

## Reopen design only when necessary

A new persistent cache/index, new remote/inference dependency, relaxed integrity
validation, wire/storage break, broad hook redesign, different semantic-authority
rule, private-data experiment or live/release action is outside this plan. Record
the measured need and return to scoped design/authority rather than implement it
as an optimization. Ordinary fixture details and bounded measured refinements
remain delegated.
