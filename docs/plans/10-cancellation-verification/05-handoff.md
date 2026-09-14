# Handoff

Implement one bounded test-only PR from the reviewed planning head:
`internal/api/sync_test.go`, with a small contributor note in `docs/development.md`.
Do not change runtime code, plugin instructions, schemas, workflow permissions or
release policy. If evidence identifies a runtime defect, return to design for that
different change rather than hiding it in a test adjustment.

Traceability: observed fetch + strict partial status covers post-checkpoint API
cancellation; initial stall + operation deadline covers pre-checkpoint timeout;
existing input tests cover supported timeout values. Repetition/full-suite and
hosted checks cover stability and regressions, not independent native acceptance.

Before readiness, reconcile with this pack, document actual tests and any drift,
promote the recurring testing lesson to development guidance, then remove the six
temporary files and their empty directory. Keep #10 open for its complete original
criteria. Publish no private fixture paths, raw native transcripts or credentials.
