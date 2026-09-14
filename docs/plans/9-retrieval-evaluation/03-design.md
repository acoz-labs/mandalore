# Technical design

## Evaluation surface

Add test-only fixture helpers and table-driven cases next to the memory tests,
with small public synthetic scenarios for signet-wide preferences, project-specific
preferences, renamed entities, current decisions, superseded history, concurrent
heads, future-effective revisions, ambiguity and no matches. Include distinct
projects sharing keywords. Known aliases present in current text must retrieve
current IDs; an alias found only in a superseded revision must not resurrect it.
Scope discovery supplies stable IDs; it is not evidence of which ambiguous project
the user meant. Do not guess from cwd or turn scoped preferences into global rules.

Exercise the public service and compiled/shared interface for quality and budgets.
Keep low-level relevance cases as diagnostics, not the sole quality gate. Record
expected result IDs, conflict IDs, excluded IDs, matching/omission counts and JSON
bytes. An empty/truncated response must retain its limitation notice. Full claims
must not be clipped to improve apparent result count.

## Scale measurements

Add opt-in Go benchmarks or a small test-only measurement entrypoint using
100, 1000 and 10000 current records, with independently reported history-depth
and multi-scope/conflict variants. Expensive fixture creation is outside timed
sections and clearly labeled. Test-only direct fixture generation, if needed to
avoid quadratic public-writer setup, must use valid canonical objects and run
complete validation before measurements; it is not a new production write route.

Measure warm repeated operations on an existing service, opening a fresh service
plus recall, and a bounded compiled CLI sample including process startup. Report
sample count and median/p95 for timed samples, benchmark ns/op, B/op and allocs/op
where available, corpus bytes/files/current and total revision counts, Go/Git/OS/
architecture, exact head and binary identity. Do not equate fresh process with
cold disk, purge caches or require privileged performance tooling.

Measure narrow, broad, empty and no-match queries; scope inventory and a representative
prompt hook; reference searches separately at bounded file/byte counts. Record
actual serialization sizes and caps. Go benchmark means alone are not p95, and
end-to-end CLI samples do not isolate filesystem time. Do not place noisy wall-time
thresholds in required CI; correctness, fixed bounds and benchmark compilation are
deterministic gates. Retain raw synthetic measurement output and reproduction steps.

## Freshness and authority

Use the same open service to read before/after an external valid revision append,
supersession, conflicting head and bank identity replacement. Observe subsequent
reads or explicit refusal without a service restart. Existing real-Git sync tests
remain part of the full suite. Any optimization must retain invalid/corrupt-data
failure, distinct signet isolation and source pin verification.

Native comparison uses fresh empty project directories and disposable banks, the
installed supported Codex version and the designated pane. Preserve inherited
native resources/authentication; use the authorized native test permissions, not
an alternate private home or copied credential. Scope prompts as read-only for
retrieval measurement and verify unchanged source/bank files afterward. A separate
authorized save/supersession prompt verifies that compact guidance still learns.

Record ordered Mandalore tool calls, query/scopes/limits, requested ceilings versus
returned bytes, repeated complete excerpts, wall time and final factual result.
If visible aggregate token counters are retained, label their native/session-wide
scope and cache contributions; never present them as Mandalore-only cost.
Do not copy raw private/native transcripts into the public repo. Synthetic receipts
and a sanitized factual ledger are sufficient; artifacts must be privacy-reviewed.

## Failure and bounded improvement

First record baseline results, including misses and long runs. Profile or reproduce
before changing code. Fix each actual behavior defect with a failing regression;
compare the identical fixture and measurements afterward. Timing-only improvements
must retain semantic equivalence. Repeated native calls are not automatically bugs:
classify provenance checks, actual changes and recovery separately from needless
duplicate retrieval. If no bounded fix is justified, ship the useful evidence and
document the limitation with a scoped follow-up rather than add complexity.
