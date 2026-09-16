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

## Release Prerequisites

New candidate acceptance, retained rendered evidence and publication authority
remain separate gates. Existing Pi merge is not Pi acceptance.

## Production Readiness Preflight

Not applicable to this implementation-only execution envelope: no production
mutation, credential slot, publication or personal activation is included.
