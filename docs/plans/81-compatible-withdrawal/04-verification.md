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

## Implementation slice: local activation and recovery

The format-upgrade package now tests explicit stopped-writer acknowledgement,
strict preparation serialization, refusal to use recovery to start a new upgrade,
and interruption/cancellation at preparation, receipt publication/durability and
manifest replacement/durability. Recovery reuses the prepared evidence identity;
binding/source drift refuses continuation. Successful local activation separately
reports that neither a Git checkpoint nor delivery occurred. Original portable
evidence and existing Git metadata are compared byte-for-byte after activation.

The transition inspector validates the original immutable Git candidate in an
owned temporary directory and permits only the prepared receipt and reviewed
manifest replacement in the live tree. Ordinary preview remains read-only and
does not create that temporary candidate. These are synthetic package tests, not
native acceptance, completed synchronization support or authorization to upgrade
a personal signet. Typed interface and native acceptance work remain.

The sync slice adds synthetic checkpoint/delivery tests for upgraded banks,
explicit local opt-in before remote adoption, independent receipts converging,
forged hashes or bases, old-evidence edits, identity changes and downgrades. Each
receipt is checked against its ancestor format1 commit and raw portable inventory;
base bytes are bounded and original evidence must survive. The actual local
activation transaction is also followed by ordinary sync/checkpoint in a package
integration test. Concurrent visibility decisions must be delivered as conflicts
without exposing their content in recall, while explicit history remains intact.
