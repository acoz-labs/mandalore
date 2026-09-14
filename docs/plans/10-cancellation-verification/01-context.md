# Context

During #10 verification, full host CI failed at
`internal/api/sync_test.go:TestSyncCancellationExposesPartialWriteEvidence`.
The test uses a one-second whole-sync deadline while requiring checkpoint completion
and fetch-phase interruption. A controlled slow first Git invocation instead yielded
an honest checkpoint-phase cancellation receipt. Ten isolated repetitions passing
does not invalidate the original failure. Evidence is retained in
[the issue checkpoint](https://github.com/acoz-labs/mandalore/issues/10#issuecomment-5663308376).

The relevant precedent is `internal/sync/cancel_phases_test.go`: a wrapper marks
actual phase entry, then the test cancels. The API must preserve that status in its
public error envelope. This slice changes tests and contributor guidance only;
it is not a runtime fix or completion of the other #10 scenarios.

Use synthetic standalone repositories and fake Git wrappers; never real network,
credentials, memory or native profiles. The original error pointer does not prove
its exact phase, so documentation must distinguish observation from inference.
