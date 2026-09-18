# Implementation Handoff

High-risk data-format work must ship coherently: compatibility gate, reviewed
upgrade, causal visibility, retrieval/export integration, sync and recovery.
Do not expose a withdrawal command before every ordinary retrieval surface
respects it. Format1 remains supported without forced migration.

Order: production schema/pure evaluator and red tests; transaction/upgrade proof;
store writes and shared retrieval; synchronization/adoption; export/foundlings/
journals/counts; typed interfaces and retention preview; native integration;
durable documentation and exact-head review. Each slice must retain safety tests.

Promote final format/causal rules to docs/signet-format.md and architecture.md;
upgrade/sync rules to synchronization.md and runbook.md; user distinctions to
privacy.md; commands/schemas to interface.md; native behavior and limitations to
bounded evidence. Preserve the owner decision in a durable ADR. Reconcile from
the actual implementation, remove this temporary pack and retire the discovery
only after its evidence/decision has an appropriate durable home.

Use an implementation branch from reviewed main, a draft PR with top-level
Refs #81 and separate engineering self-review. Keep #81 open while candidate
acceptance/release remains. Do not relabel design proofs as a delivered feature.

Return to design for a need to rewrite old evidence, loosen unrelated append-only
checks, auto-upgrade, conceal independent journal/source data, implement destructive
retention or change the approved same-bank continuity contract. No general policy
engine, encryption subsystem, background scheduler or content-classification model.
