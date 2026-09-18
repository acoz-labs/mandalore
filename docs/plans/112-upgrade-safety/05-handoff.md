# Implementation handoff

First add failing tests for denied replacement and truly non-mutating replay.
Implement the shared guard, then wire typed/CLI/menu/delegated-runtime entrypoints
and Armorer handoff guidance. Finish native component, complete session and
rendered checks before readiness. Each critical journey maps to the matrix in
[Technical design](03-design.md#verification-and-acceptance).

Promote final behavior into setup, runbook, interface and plugin administration
docs. Remove this temporary plan during implementation reconciliation, recording
the exact reviewed head, drift and documentation-promotion matrix in its PR.
Required checks and exact-head self-review precede merge. Keep #112 open for its
acceptance/release obligations. Stop after #112 and repitch the roadmap.

No lifecycle sync, cache resurrection, active-session monitoring daemon, native
credential setup, memory migration or unrelated issue implementation. Reopen
design if supported native behavior contradicts the guard's correctness, if a
new trust boundary is required, or if compatibility demands a product rescope.
