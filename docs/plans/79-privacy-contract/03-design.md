# Technical design

## Components, data and interfaces

No new state or interface. Promote the existing storage/exposure matrix,
correction-versus-erasure explanation and future-operation boundaries into
`docs/privacy.md`. Link it from README, SECURITY and runbook.
Use existing memory/API/sync fixtures; add focused tests in their owning packages.
No new shared testing abstraction unless repeated assertions warrant it.

## Authorization and exposure

Tests create private-permission disposable synthetic banks. Only test-owned
files are removed, to exercise denial; no production or user binding is opened.
Public docs use synthetic examples and immutable public review links.
No log contains real credentials, native transcripts or workstation paths.
Ordinary learning and current no-save/no-sync direction remain unchanged.
Read-only API enforcement is not a sandbox against independent native tools.

## Failure and recovery

Assert denied mutations leave file names/content unchanged, including Git/local
state. Assert deleted committed evidence cannot advance checkpoint HEAD, while
the earlier Git blob remains available. Assert source graph remains valid in the
independent append-only case so the test cannot pass for the wrong reason.
Unexpected failures stop claims; no automatic repair, retry or cleanup of live data.

## Traceability

Storage/exposure and labels: guide plus service/context tests.
Supersession/journal/Git lifetime: memory/API and sync tests.
No-save: enforced API mutation rejection and full fixture inventory.
Incident response: runbook/SECURITY links and bounded primary-source guidance.
Deferred designs and exclusions: privacy guide links to #80/#81.
Existing foundling disconnection and error-redaction tests remain part of full CI.
