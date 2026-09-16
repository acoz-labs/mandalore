# Design

In `menu.line`, keep cancellation/output checks, 4096-byte reader boundary,
newline requirement, control-character rejection and :back semantics. Perform
only the read in a worker, return its bytes/error through a buffered channel,
and select between that result and context cancellation. A second context check
prevents a ready newline from becoming approval after a simultaneous interrupt.

No persistent state, new API, goroutine pool, signal handler or dependency.
Never launch another line read after cancellation. Existing callers propagate
the cancellation to `runMenu` or `runReleaseInstall`, which return130.

Tests supply a reader whose read is observably blocked, not sleeps that assume
the read began. Cancel, require the call to return before releasing the reader,
then release/join the worker. Exercise a partially buffered approval and a
confirmation to prove no default or partial input can be accepted.
