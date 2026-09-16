# Solution Design: distinguish no new content from no-save direction

- **Status:** Final
- **Issue:** #100
- **Planning PR:** #102
- **Repository basis:** 7d9ec5c3cd2083e41bf56c47aee3e4295fe0aef9
- **Execution envelope:** implementation

## Decision

Clarify shared Codex/Pi guidance: no useful new content is not an explicit
read-only/no-save/no-sync restriction. Direct consolidation still attempts
delivery once when allowed, without filler; explicit prohibitions still win.

## Needs Attention

One fresh native run skipped the required attempt. Preserve that counterexample;
passing runs elsewhere do not erase it. Model-selected behavior remains fallible.
A changed package requires a new artifact; existing candidate acceptance does
not transfer.

## Decision Spotlight

This fixes interpretation, not authorization. Do not add a phrase-matching hook,
force calls, weaken explicit restrictions, or invent records to cause a sync.
An absent connection is a reportable limitation, not a request to install one.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Exact-head contributor self-review under ADR0003 and required CI, within the
implementation-only envelope. Not independent acceptance or publication.
