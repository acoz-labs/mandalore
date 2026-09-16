# Context

Exact retained candidate 6f9bb5e produced one duplicate first-error wrapper
before the model read its linked selector reference. Later results were selected
correctly. Error fidelity, valid recovery and read-only inventories passed.
This is a sequencing failure, not a selector-equivalence failure.

Immutable evidence: [c5507c1 checkpoint](https://github.com/acoz-labs/mandalore/blob/c5507c1e4ea89e6d75b4963aeeccbe0de5752c8e/docs/evidence/candidates/1.1.0/continuity-rendering.md).
Ordinary matched recall reduced memory-result bytes but increased total context
for this small task. All 21 exact-selector synthetic cases passed.

Original #56 criteria and PR #58's privacy, compatibility and scope constraints
remain unchanged. No new UI, identity, learning rule or synchronization behavior.
