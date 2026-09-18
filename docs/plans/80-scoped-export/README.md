# Solution Design: scoped memory reports

- **Status:** Draft
- **Issue:** #80
- **Planning PR:** #118
- **Repository basis:** ee8cf60727c9c47b73964ca8b3bd3762ec04cb89
- **Execution envelope:** implementation

## Decision

Provide typed/CLI preview and explicit apply for a versioned, indented JSON
report. Selection is explicit; defaults expose current unambiguous records only.
History, exact journal entries and citation details are separate opt-ins. Export
is not a restorable signet, synchronization or automatic sanitization service.

## Needs Attention

Exact-head engineering review under ADR0003 precedes implementation. Synthetic
fault, alias, source-invariance and every-field redaction tests are mandatory.
Candidate acceptance/release and any export of real personal data are separate.

## Decision Spotlight

- No implicit whole-bank selection and no journal scope inference.
- Omit whole reviewed field categories instead of inventing secret matching.
  The report identifies omitted categories and incomplete provenance.
- Never write an unredacted intermediate report. Preview is metadata-only but
  still sensitive; it is not a public manifest.
- No new menu or daily memory-tool schema overhead. Agents use typed CLI calls.
- Reject stale/ambiguous inputs instead of silently expanding selected data.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Record full repository basis, planning PR and exact-head self-review, then pass
hosted checks. Stacked implementation follows ADR0003; no live-data export or
release authority is granted by this plan.
