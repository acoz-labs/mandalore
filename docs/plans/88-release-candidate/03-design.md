# Technical design

The only runtime input change is VERSION. Existing clean-export stamping creates
1.1.0 native/runtime/package identities. Protocol and signet format remain 1;
historical readiness evidence remains historical unless every identity matches.
New version labels must not be mistaken for new evidence.

Permanent guide: docs/releases/1.1.0-candidate.md. Include outcome-to-PR/test
mapping, compatibility, no-release warning, actual acceptance entrypoints,
retention/identity receipts, fixture isolation, untested platforms and rollback.
docs/roadmap.md and docs/deployment.md link the guide.

Use Build retained candidate on reviewed main, then Nominate artifact candidate
with its exact SHA/run/artifact IDs. The nomination must include all seven
open implementation outcomes plus #88; use explicit top-level Refs lines in the
preparation implementation PR, not prose mentions. Verify the resulting issue
receipts. Do not silently nominate unresolved parent discoveries as completed.
