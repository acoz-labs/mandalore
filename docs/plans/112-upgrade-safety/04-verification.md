# Verification and release design

Use the numbered red/green and native/terminal matrix in
[Technical design](03-design.md#verification-and-acceptance). Add delegated-runtime
coverage proving an old apply implementation cannot bypass the parent guard.
Run `mise exec -- bin/ci`; this repository permits pinned host fallback when
container execution is unavailable. Hosted CI remains required separately.

## Acceptance Evidence

Bind tests, native identities and synthetic recordings to the implementation
head. Do not copy personal credentials, signets or raw private transcripts into
evidence. Native macOS testing and Linux cross-builds are different claims.
The selected default-defer flow is an in-pattern terminal change requiring real
normal/narrow and plain/TUI verification, not just string assertions.

## Production Readiness Preflight

Not applicable to the implementation-only execution envelope. No release or live
installation occurs under this plan. Subsequent candidate nomination, product
acceptance and publication require their existing separate gates.

## Rollback And Recovery

Denied replacement leaves native registration/cache untouched. Acknowledged
partial failure is inspected using existing phase receipts and retained bundles;
never retry or roll back automatically. The guard does not override ownership,
stale plan or edited-file refusals.
