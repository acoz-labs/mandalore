# Decision

## Alternatives

1. Keep only existing regression tests. Low cost, but they do not establish corpus
   growth or native context behavior and would leave #9's outcome unmeasured.
2. Add deterministic quality cases, reproducible scale/payload measurements and
   a small native comparison; fix demonstrated bounded issues. Selected.
3. Add a persistent lexical index or semantic/vector retriever now. Potentially
   useful at scale, but introduces freshness, rebuild, ranking and dependencies
   before identifying the actual bottleneck. Deferred, not rejected universally.
4. Reduce all context ceilings or remove checks for faster turns. Rejected:
   requested ceilings do not equal actual tokens, and hiding omissions/conflicts
   or provenance would compromise the product contract.

## Selected boundaries

The evaluation fixture is synthetic, deterministic and inspectable. Assert exact
expected current IDs and forbidden IDs rather than asking another model to grade
truth. Classify genuine lexical limitations explicitly; do not convert every
natural-language phrase into a mandatory semantic matching feature.

Benchmark ordinary recall, scope routing and prompt-hook work separately from
foundling source verification. Separate fixture generation, opening, operation,
JSON encoding and native/model overhead. Include history depth and conflicts as
well as current-record counts. Do not let an internal scoring microbenchmark
stand in for actual filesystem-backed reads.

Keep exact freshness and isolation behavior through any local optimization.
Per-operation reuse may be considered only with failing-first regression and a
measured benefit; no persistent stale cache, source hash shortcut, weaker schema
validation or silently skipped corrupt record is authorized.

Only small native guidance/schema improvements responding to measured behavior
are preauthorized, for example beginning with the documented compact defaults,
reusing already-discovered explicit scope IDs when context is sufficient, and
avoiding a second read of a complete unchanged cited excerpt. Preserve explicit
history when provenance/conflicts require it and reread after source/local changes.
Keep the tool set and wire/data format stable unless a new design justifies change.

## Follow-up triggers

Record provisional investigation triggers, not universal performance promises:
warm scoped local recall p95 exceeding 250 ms at 1000 current records; fresh
process recall p95 exceeding one second at that scale; or more than two redundant
retrieval calls in a simple scoped continuity task after guidance correction.
Record actual corpus/history/host and sample counts with any comparison.
If triggered, profile first and scope a follow-up; do not automatically install
an index or broaden this slice into a retrieval redesign. Correctness regressions
remain blocking regardless of timing.

A future index must be disposable/rebuildable, bound to exact signet/source
identity, detect external writes/Git changes, preserve graph/scope/privacy
semantics, and operate without a mandatory inference service. Semantic alternatives
must demonstrate improvement on held-out questions without treating historical
text as current guidance. No specific package is selected here.
