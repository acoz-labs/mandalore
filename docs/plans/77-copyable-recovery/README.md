# Solution Design: copyable recovery commands

- **Status:** Draft
- **Issue:** #77
- **Planning PR:** #116
- **Repository basis:** 8b74cfad3324f4cdef0036dc1a2c46b96209677d
- **Execution envelope:** implementation

## Decision

Keep readable wrapped receipts, but render the recovery command separately as
one literal logical line. No formatter-added wrapping, indentation, styling or
quote changes. Terminal soft wrapping is distinct from inserted newline bytes.

## Needs Attention

Real normal/narrow terminal evidence and shell interpretation tests are required.
Engineering self-review is authorized by ADR0003; candidate acceptance and
publication remain separate. No clipboard access or automatic retry.

## Decision Spotlight

The command line may exceed the prose width. This deliberate exception preserves
copyable bytes; wrapped prose explains selecting the full logical line. Unsafe
control-bearing input is suppressed, not rewritten into a different command.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Record exact-head self-review and pass hosted CI before reviewed merge. Stacked
implementation may begin after recorded planning review under ADR0003.
