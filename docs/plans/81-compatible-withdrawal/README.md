# Solution Design: compatible withdrawal and retention preview

- **Status:** Draft
- **Issue:** #81
- **Planning PR:** Pending
- **Repository basis:** fe45b94ed5e3fc852db3217f8cce93cc4c29d33c
- **Execution envelope:** implementation

## Decision

Implement the owner-approved opt-in format transition in the same signet and Git
repository, then causal record withdrawal/restore and bounded retention preview.
Preserve existing evidence and explicit history. No automatic upgrade or expiry.

## Needs Attention

The transaction contract requires review and fault-test proof before shipping.
Remote format adoption requires a separate explicit local upgrade, not a standing
consent flag. Discovery PR121 is merged and records the owner decision and
synthetic old-reader/causal evidence. Native migration, acceptance and release
are not authorized by this plan.

## Decision Spotlight

- Format gating deliberately makes old readers refuse upgraded banks.
- Visibility events are append-only; correction alone never restores a record.
- A restore does not authorize an unseen concurrent correction.
- Same-bank continuity is selected over creating a second bank.
- Old offline clones and already-returned context cannot be revoked.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Record exact-head engineering self-review under ADR0003, the planning PR number,
resolved transaction contracts and passing checks before implementation. Owner
approval selects the material direction; it is not candidate acceptance or
permission to upgrade a personal bank.
