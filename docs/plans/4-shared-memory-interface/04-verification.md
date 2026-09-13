# Verification

Use TDD around binding isolation and adapter equivalence. Port selected synthetic
binding/MCP tests after reviewing dependency closure; build no private fixtures.

- Create/bind/inspect from unrelated cwd; two independent signets and device IDs.
- Binding inside the signet through normal or symlink paths, unknown/malformed
  versions, oversized/trailing input, stale root identity, duplicate enrollment
  publication and missing local binding.
- Same remember/correction/recall/scopes/history/journal results and error codes
  through CLI and an in-process/native-stdio SDK client, with typed discovery.
- Defaults, paired scopes, limits/bytes and no hidden writes on reads, startup,
  discovery, errors or read-only mutation attempts.
- Unknown/duplicate fields, wrong types, trailing JSON, oversize input, unknown
  operation, broken pipe, EOF and cancelled contexts.
- Receipts distinguish local publication from sync; partial I/O is inspectable,
  not a retry-safe or remotely delivered claim.
- Compile and run the actual CLI in the designated Herdr testing pane using
  disposable synthetic directories; validate stdout/exit status and cwd independence.
- Full race/vet/module/format/privacy checks and platform builds; distinguish
  native protocol tests from a real model/native-plugin conversation.

## Acceptance and rollback

The first slice delivers a usable local CLI/MCP interface, not installed native
plugins or final #4 sync acceptance. #5 supplies actual Git operations through
the same interface before #4 closes. Preserve fixture/evidence identity for the
later #10 candidate. No existing binding/configuration is rewritten.

## Production Readiness Preflight

Not applicable: implementation-only envelope, no deployment, secret injection,
activation or release. Artifacts/native installation/independent acceptance
remain the later distribution and harness delivery slices.
