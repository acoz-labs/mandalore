# Solution Design: phase-bound cancellation verification

- **Status:** Final
- **Issue:** #10
- **Planning PR:** #37
- **Repository basis:** 0fe7e0e1eb175943995b0d0d5e54130ff65b074e
- **Execution envelope:** implementation

## Decision

Repair the timing-dependent API cancellation test uncovered during retained-candidate
verification. Observe fetch entry before explicitly cancelling, retain independent
deadline/early-phase evidence checks, and print structured diagnostics. Product
timeouts, runtime behavior and candidate bytes remain unchanged.

## Needs Attention

No implementation blocker. Full #10 acceptance, independent actors and publication
remain outside this narrow test-repair slice. The original failed run stays recorded.

## Decision Spotlight

Readiness is an observed subprocess marker, not elapsed time. A slow-start fixture
must still prove the same fetch cancellation invariants. Deadline coverage tests
the actual configured timeout separately and does not demand an unobserved phase.
Test subprocesses and goroutines must finish on success and failure.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Contributor self-review follows decision 0001 and binds the final planning head
in PR #37. The complete pack has no implementation-blocking unknown; required
checks precede merge. This does not constitute independent product acceptance.
