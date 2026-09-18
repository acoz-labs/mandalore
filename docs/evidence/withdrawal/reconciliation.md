# Withdrawal implementation reconciliation

Scope: issue81, discovery PR121, approved solution plan PR122, implementation
PR123. Engineering self-review is authorized under ADR0003. This document is a
coverage map, not product acceptance. The PR review binds it to an exact commit.

## Plan-to-implementation coverage

| Requirement | Implementation and verification |
| --- | --- |
| Stable IDs, exact heads, distinct withdrawal/restoration | `internal/memory/visibility_service.go`; service tests cover stale heads, corrections, legacy refusal, publication failure and late cancellation receipts. |
| Causal concurrency, no clock winner | `visibility.go` and `visibility_records.go`; causal-state, invalid-graph, deep-graph and delivery-permutation tests. Divergent restores remain withheld; content conflicts remain unresolved. |
| Shared read boundary | Memory service filters before recall scoring; scope projection exposes visibility counts; native context consumes the same service. Integration tests and actual fresh Pi recall establish withheld behavior. |
| Explicit history and restoration discovery | Content/visibility histories retain evidence. Native testing exposed missing fresh-session ID discovery; bounded `memory_withheld` supplies routing metadata, not bodies. Both native harnesses then restored without signet-file scans. |
| Export and independent evidence | `internal/exportreport/visibility_test.go` covers dual history/withdrawn opt-in, conflicts, redaction and snapshot pins. Journal independence and exact-origin foundling re-promotion guards have storage tests. No inferred suppression of journals or external documents. |
| Opt-in format upgrade, preserved evidence | `internal/formatupgrade` preview/apply/recover pins binding, root, HEAD, manifest and raw portable inventory. Tests compare original evidence/Git metadata and inject interruption at durable phases. Recovery cannot start a new upgrade or ignore drift. |
| Older readers and runtime transition | Design-stage released-reader probe plus compiled CLI scenario establish refusal and unchanged offline-copy behavior. Disposable installation driver establishes safe old-updater refusal and verified-new-executable installation without bank migration. |
| Narrow synchronization transition | `internal/sync/upgrade_*` validates ancestor proof and original blobs, requires per-clone opt-in, rejects tampering/downgrade, converges independent receipts and preserves concurrent visibility conflicts. Actual activation-to-sync integration is tested. |
| Explicit retention preview only | `internal/retention` tests policy/selection pins, all structural heads, cutoff equality, missing timestamps, source sharing, separate journals, metadata-only output, cancellation, source invariance and oversized-result refusal. No apply or TTL exists. |
| Typed interfaces and read-only boundaries | API visibility/upgrade/retention tests cover strict inputs, receipts and denial before decoding. MCP schemas, CLI dispatch and Pi catalog tests cover exposure. Upgrade and retention remain CLI-only. |

## Documentation promotion

| Temporary material | Durable home |
| --- | --- |
| Context, alternatives, owner choice | ADR0004 and `docs/privacy.md` |
| Causal schemas and read semantics | `docs/signet-format.md`, `docs/architecture.md` |
| Upgrade transaction and cross-clone consent | `docs/synchronization.md`, `docs/runbook.md` |
| Typed commands, limits, receipts | `docs/interface.md`; on-demand harness visibility/administration references |
| Design oracle and released-reader characterization | `docs/evidence/withdrawal/design-model/` |
| Compiled CLI, native models, runtime installation | `docs/evidence/withdrawal/README.md` and reproducible drivers |
| Verification/handoff | This reconciliation and exact-head PR engineering review |

The temporary discovery and plan are removed after promotion; their original
reviewed forms remain in Git and PR121/122. No private transcripts, workstation
paths, credentials or personal memory are included in public evidence.

## Deliberate refinements

The implementation makes the upgrade inventory's raw framing explicit, including
portable metadata rather than reusing the narrower export digest. Independent
per-clone upgrades replace an ambient remote-upgrade permission flag. Both refine
the approved preservation/consent contract without broadening authority.

`memory_withheld` is the native-test-driven discovery addition. Ordinary recall
does not fall back to it after an empty result. It requires deliberate historical
inspection/restoration intent; its scoped result is bounded metadata only.

## Remaining gates and limits

Full pinned host and hosted CI and exact-head engineering review precede merge.
The merged tree must also be checked for temporary-plan removal. Keep issue81
open pending exact immutable-candidate product acceptance and publication.

Native evidence covers the recorded macOS arm64/Codex/Pi combinations, not native
Linux or every model/wording. The engineering installation test is not a public
bootstrap download of a new release. Runtime updates do not opt a bank into
format2. No personal installation/migration, deletion, expiry, history rewrite,
offline-copy revocation or previously-read-context revocation is authorized or
claimed. These boundaries survive successful CI and merge.
