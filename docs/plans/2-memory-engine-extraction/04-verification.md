# Verification and release design

## Test strategy

Port the relevant public synthetic tests, then add explicit tests for the clean
format and rejected legacy inputs. Test at the lowest useful layer:

- New signet structure without assistant/capability directories or metadata.
- Device/source validity, rejected unknown/malformed schemas and oversized input.
- Remember/recall/journal/history, explicit scopes and deterministic ordering.
- Supersession chains, invalid graphs, concurrent heads, future-effective data.
- Byte budgets, pagination, missing-query ambiguity and no false completeness.
- No writes after invalid input; no side effects from reads; disk failure,
  symlink/path redirection, lock contention and atomic directory publication.
- A dependency boundary check forbidding native/assistant/provider imports.
- Legacy bank/agent formats rejected unchanged, including mismatched local locks.
- Foundling registration/citation schema validation, rejected private paths and
  credential-bearing locators, invalid pins/traversal, unknown original authors,
  separate original/incorporation provenance and preserved disconnected citations.
- Registration metadata does not appear as ordinary recalled knowledge. Data
  validation performs no source access, network requests or executable dispatch.

## Red/green sequence

Start with package/format boundary tests that fail because the new engine is
absent. Extract file/manifest primitives, then source/device/graph behavior, then
journal/recall/service validation. Re-run characterization after each slice.
Record intentional differences; do not weaken tests to obtain a green port.

Pin the existing source's Go 1.26.4 toolchain before language execution. Retain
only required direct/transitive dependencies and generate module metadata through
Go tooling. Add `go test -race ./...`, `go vet ./...` and static target builds to
`bin/ci` in the implementation, distinguishing native tests from cross-builds.

## Acceptance and rollout

The first outcome is a library and published format docs, not an installable MVP.
Independent review validates the extracted code and evidence. #10 handles the
actual immutable native candidate after CLI, integration and distribution exist.
No private environment or production state is touched by this plan.

## Rollback

Code rollback uses normal reviewed Git changes. No existing signet is rewritten.
Synthetic test data is disposable. Future schema conversion and reverse migration
require explicit #8 evidence; do not imply backward compatibility from shared IDs.

## Production Readiness Preflight

Not applicable to this implementation-only library envelope. No deployment,
secret injection, activation or release command runs here. Product distribution,
independent acceptance and exact-artifact release prerequisites remain #11/#10.
