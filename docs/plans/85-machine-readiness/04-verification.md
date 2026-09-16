# Verification And Release Design

## Test Strategy

- Characterize existing metadata and error-envelope paths before extraction.
- Test declarations and exact evidence matching independently: different artifact,
  source-only, same-version/different-bytes, missing native version, historical
  scenario, unsupported and unknown target. No permissive wildcard matches.
- Use trap executables and network instrumentation plus complete fixture
  inventories to demonstrate the default no-execution/no-product-write boundary.
- Exercise absent and empty profiles, missing PATH, absent/malformed binding,
  retained mismatch, wrapper uncertainty, oversized files, FIFOs, symlinks,
  permissions, cancellation and concurrent replacement observations.
- Verify shared CLI/menu result parity, bounded report/prompt output, error
  sanitization and unchanged ordinary memory catalog.

Likely ownership: `internal/readiness/*_test.go` for declarations/matching/read
bounds/prompts; `internal/install` for retained projection/ownership; existing
binding/memory tests plus metadata-only regressions for shared field validation;
`internal/api` for schemas, absence of binding and cancellation; compiled CLI
tests in `cmd/mandalore` for execution-boundary and menu parity. Avoid repeating
the full matrix at each layer: test each invariant where it can fail and one
compiled representative per meaningful transport boundary.

Compiled fixtures put marker-writing executables on PATH, select explicit empty
native homes and unavailable local network endpoints, and preserve hashes/modes/
empty directories before/after. Trap unexpected record/event/foundling reads
through the internal read boundary, with a real fixture whose remembered files
are invalid/unreadable yet valid manifest/device metadata still assesses. This
distinguishes metadata inspection from full-store validation. Check process exit
and cancellation, not just the collector's timeout. Source/import-boundary checks
supplement actual executions; they do not alone prove lack of side effects.

Self-image tests replace the on-disk path while preserving supplied process
metadata and ensure no claim of loaded-image verification. Receipt tests must
prove foreign binding/profile paths are not followed, altered digest metadata
does not create evidence, and an omitted root never means absent registration.
Stable fixture identity is injected only behind internal test seams; the public
schema cannot accept fake platforms or an evidence catalog.

At least one controlled mutation must make the no-execution or evidence-match
tests fail, then be restored and retested. Keep the failed result as evidence of
test sensitivity; do not weaken assertions to make an implementation pass.

Run focused pinned-toolchain tests, then `bin/container bin/ci`; if Docker is
unavailable use documented `mise exec -- bin/ci` and record that route. Hosted
required checks apply to the exact reviewed head, not an earlier passing draft.

## Red/Green Sequence

Declarations/schema -> pure classification -> bounded static observations ->
shared API/CLI -> menu/prompt -> explicit native handoff -> exact-head rendered
evidence and reconciliation. Each meaningful slice begins with a failing test.

## Acceptance Evidence

Classification: new-or-materially-changed-experience. Follow
`docs/operations/ui-acceptance.md`. Record actual terminal output from synthetic
normal/missing/unknown/unsupported/stale/error/partial/cancel cases, Codex/Pi
selection, prompt view and native handoff. Cover normal/narrow widths,
color/no-color/plain, arrows/Vim, Back, default-safe selection, EOF and Ctrl+C.
Verify terminal restoration. Record exact source/executable identity, scenario,
dimensions and limitations in openable retained evidence.

The actual native matrix runs in the designated test pane using disposable
fixtures and installed native binaries only for the explicit handoff. Do not
activate a personal connection, copy credentials or launch a model to prove a
static report. An unsupported host/evidence combination can be shown through a
synthetic retained-artifact fixture; do not add production fake-host flags for
screenshots or call that native execution on the unsupported platform.

No browser/mobile product surface or approved pixel baseline exists; do not
fabricate either. Terminal recordings plus semantic/width/ANSI checks are the
current strategy. Screen readers, other locales/fonts and additional native
platforms remain unverified unless actually tested. Contributor self-review
does not become exact-candidate product acceptance.

## Rollout

Implementation envelope only. Normal reviewed merge after complete validation;
keep #85 open for nominated immutable-candidate acceptance and release. Do not
alter v1.0.0 or activate a personal native connection.

## Rollback And Recovery

Assessment has no durable state to roll back. An older toolkit continues its
existing supported commands; the new operation is additive. Preserve old report
and connection contracts. Native handoff performs no repair automatically.
Test old version/manifest compatibility and unchanged deeper CLI operations;
only the menu adds the explicit submenu/disclosure. No data backfill is needed.

## Release Prerequisites

New candidate acceptance, retained rendered evidence and publication authority
remain separate gates. Existing Pi merge is not Pi acceptance.

## Production Readiness Preflight

Not applicable to this implementation-only execution envelope: no production
mutation, credential slot, publication or personal activation is included.
