# Solution Design: cancellable plain terminal input

- **Status:** Final
- **Issue:** #99
- **Planning PR:** #101
- **Repository basis:** 7d9ec5c3cd2083e41bf56c47aee3e4295fe0aef9
- **Execution envelope:** implementation

## Decision

Let a cancelled plain-menu wait return without waiting for its underlying blocking
reader. Preserve the current bounded line parser and every approval check.

## Needs Attention

The retained 1.1 candidate failed native cancellation. It must not be accepted
using a replacement executable's results. A fix needs new artifact nomination;
the existing exact-candidate owner exception does not cover a replacement.

## Decision Spotlight

Cancellation is denial, never a default answer. A worker may remain inside an
uncancellable terminal read until stdin closes or the CLI exits; no retry or
subsequent menu read starts after cancellation. Do not change terminal modes or
make all process input globally nonblocking.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)

## Final Gate

Exact-head contributor self-review under ADR0003, current issue criteria, required
CI and the implementation-only envelope. Not independent product acceptance.
