# Verification And Release Design

## Test Strategy

Fail first on exact command-line bytes at 24/32/80 columns, plain and colored.
Test spaces, consecutive spaces, apostrophes, Unicode and shell metacharacters.
Extract the displayed line and evaluate it only with an inert shell function
named mandalore and a synthetic pending file: assert exact argv and stdin, never
run installation or recovery. Reject control, newline and invalid UTF-8 command
input without a misleading transformed command. No pending plan means no command.
Keep existing prose width, sanitation, quote and receipt regressions.

## Acceptance Evidence

In-pattern visual change. Capture actual normal and 32-column terminal output in
plain/no-color and TUI modes, including partial receipt, command and Back/default
No flows. Compare literal captured bytes separately from visual readability.
Use source-bound synthetic menu drivers; no live quota exhaustion or real pending
installation. Browser/mobile are not applicable; screen-reader/other-platform
acceptance is not implied. Bind retained recordings to exact source and driver.

## Rollout And Recovery

Run focused console/CLI tests, full pinned host CI and hosted CI. Remove the
temporary plan after promotion/reconciliation. No data migration or rollback
needed; no new state is written. Immutable candidate acceptance/release remains
a subsequent gate, not engineering self-review.

## Production Readiness Preflight

Not applicable to this implementation-only plan; it cannot activate production.
