# Verification And Release Design

Use synthetic banks and failing-first tests. Keep the discovery's executable
model as a design oracle, not a substitute for production integration tests.

1. Format1 characterization: unchanged create/read/write/sync behavior; format2
   unsupported by released readers; no old-client disclosure from upgraded data.
2. Engine schemas/causality: missing/invalid refs, cycles, record mismatch,
   stale expected heads, concurrent corrections/withdrawals/restores, divergent
   restores, repeated input and all merge orders. No clock winner.
3. Upgrade faults: before/after each durable step, manifest publication,
   checkpoint and delivery; preserve old evidence and exact identity. No false
   success, implicit rollback, downgrade or automatic upgrade.
4. Sync: one and two upgraded clones, old clone, remote-only upgrade, divergent
   content/events, denied transition, stale consent and interrupted adoption.
   Verify append-only protections outside the literal supported transition.
5. Retrieval surfaces: recall/scoring, scope counts, native context, explicit
   history, current/historical export, same-origin promotion and independent
   journals. Withheld text must not leak into ordinary guidance or summaries.
6. Retention preview: explicit policy, cutoff boundaries, exact journals, metadata
   only, missing relationships stated, bounded/refused oversized selection and
   complete source invariance. No TTL or apply side effect.
7. Typed API/CLI/MCP: schemas, read-only denial, literal arguments, cancellation,
   partial receipts, compact everyday context and delivery semantics.
8. Compiled CLI plus real Codex/Pi synthetic sessions in the authorized Herdr lab.
   Pin exact source/binary/plugin identities; no personal migration or install.

Full pinned host and hosted CI, privacy audit and exact-head engineering review
precede merge. Product acceptance and immutable-candidate release remain separate.
No new rendered menu is required; native model behavior needs actual observation.
Production readiness preflight is not applicable to the implementation-only
envelope. Production migration requires separate explicit authority and evidence.
