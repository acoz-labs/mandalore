# Handoff

1. Review pinned public synchronization code and native Git feature probes.
2. Finalize/self-review this plan with exact source and required CI evidence.
3. Build failing-first local init/checkpoint/boundary tests, then reconciliation,
   offline, provenance/conflict and cancellation cases; keep the engine independent.
4. Wire actual shared operations and conservative receipts; do not add a stub sync.
5. Promote the contract into docs/synchronization.md plus architecture, interface,
   runbook, development, format/extraction attribution and product status.
6. Reconcile implementation against this pack, remove it after promotion, record
   exact-head self-review and passing native/hosted checks before merge.

Keep #5 open through native policy wiring and multi-machine acceptance; completing
the backend alone is not the full issue. #4 depends on these real shared operations.
Use feature/local-first-sync after the planning gate. Reopen design for destructive
reconciliation requirements, mandatory provider accounts, unbounded child processes
or inability to preserve user work under cancellation/concurrency.
