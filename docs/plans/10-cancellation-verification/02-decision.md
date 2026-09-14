# Decision

Increasing the one-second timeout would reduce failure frequency but leave the
phase assumption unproven. Relaxing the assertion to accept any phase would stop
testing post-checkpoint evidence. Both are rejected.

Use observed fetch readiness plus explicit parent cancellation, with normal and
delayed-start cases. Keep a separate short-deadline case that stalls the initial
Git boundary and expects checkpoint-phase cancellation with conservative ambiguity.
Retain strict input-range tests. No new public test hooks or dependency injection
surface is needed: the existing PATH-local wrapper is sufficient.

Structured JSON failure diagnostics must expose the envelope/phase rather than an
opaque pointer. No production timeout, sync guard, retry policy or assertion about
actual delivery is weakened. The change has no rendered impact.
