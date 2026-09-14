# Implementation handoff

Implement in bounded slices: public format census and read-only preflight;
validation reuse and exact transformation; retained snapshot/publication;
typed CLI receipts and synthetic end-to-end checks; native coexistence reporting
and migration/runbook documentation. Keep ordinary memory reads and MCP context
unchanged. No predecessor framework dependency, capability import or old runtime
execution is needed to decode the memory-only format.

Use design/8-memory-migration for planning and feature/memory-migration for the
implementation. Record exact-head engineering self-review under decision 0001
before implementation; required hosted checks still apply. Review meaningful
validation changes and refusal diagnostics against the issue criteria.

Promote namespace/support boundaries, opt-in steps, original snapshot limits,
fresh-session handoff and rollback into docs/migration.md, docs/interface.md and
docs/runbook.md. Record validation ownership in architecture/development docs,
native/synthetic evidence in a focused migration receipt, and upstream attribution
in NOTICE for any copied code/fixtures. Reconcile the exact implementation head,
then delete all six plan files and the empty directory before readiness.

Keep #8 open after engineering merge for immutable-candidate acceptance. #12
foundlings is the route for arbitrary historical resources and assistant repos;
do not silently broaden this converter to satisfy that distinct product need.
Reopen design if supporting a new source format, executing source code, carrying
Git/native credential state, changing semantic interpretation or mutating a live
source becomes necessary. Ordinary implementation choices remain delegated.
