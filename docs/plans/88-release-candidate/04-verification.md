# Verification and release design

Run required bin/ci (container first; pinned host fallback if unavailable).
Review that only VERSION and docs changed, no historical assets/fixtures or
native config. Use the actual retained workflow to test both package stamping,
four executable manifest digests, eight-file inventory and source binding.
A label-only/unit test is not candidate integrity or native acceptance.

Repeat synthetic exact-artifact tests for all included outcomes. Retain a compact
manifest with source, manifest/transport/binary/plugin hashes, native versions,
command receipts, actual recordings and failures. Provider login stays in place;
no copying auth. Use the dedicated test workspace, never the user's active pane.
Separate deterministic library tests, native/model engineering checks and human
product verdict. No claimed acceptance from automated assertions alone.

## Production Readiness Preflight

Preparation ends before publication. Existing same-byte publisher and immutable
release policy remain unchanged; no secret provisioning, deploy or activation.
Human authorized acceptance of every nominated issue is still required, followed
by explicit publication authority. Current 1.0.0 remains the rollback reference;
do not downgrade a changed schema or rewrite memory. Retention is up to 90 days;
verify expiry/provenance before nomination and again at promotion. Do not record
a successful release receipt until publication and verification actually occur.
