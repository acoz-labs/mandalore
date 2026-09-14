# Design

Test setup initializes a synthetic signet and fake HTTPS origin using real local
Git before installing a test-local wrapper on PATH. A marker outside the bank
identifies the first Git invocation; an independent marker identifies ls-remote.
The wrapper blocks that command without making a network request.

For phase cancellation, run the API call with a generous bounded operation budget
in one goroutine. Wait for the fetch marker or early completion/readiness deadline,
then cancel its parent context. Join the call with a bounded wait before teardown.
Assert operation.cancelled, checkpointed=true, phase=fetch, not delivered, ambiguity
and inspect-before-retry. Validate the bank afterward. Run once normally and once
with an initial delay exceeding the old one-second assumption.

For deadline coverage, stall the first Git call beyond a one-second operation
budget. Assert the error envelope retains phase=checkpoint, checkpointed=false,
not delivered, ambiguity and inspect-before-retry, with no fetch marker. Preserve
the memory graph; local diagnostic receipts are not evidence of remote delivery.

Every failure path cancels and joins the worker. Shell quoting handles temporary
paths. No sleeps in the Go test substitute for observed readiness; a controlled
wrapper delay is the explicit slow-start stimulus. Process cleanup assertions and
diagnostics stay confined to test fixtures.
