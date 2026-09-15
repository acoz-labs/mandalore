# Verification And Release Design

## Test Strategy

Use failing-first Go tests for shared runtime/installation, Node builtin tests
for the dependency-free adapter, compiled-process tests for transport, and real
Pi for native behavior. Initial exact Node test pin: 24.1.0, matching the inspected
Pi 0.85.1 baseline. Add it to mise/CI/container before requiring Node tests; runtime
packaging itself must not depend on Node, npm or model access. A newer baseline
requires recorded contract verification, not an unpinned latest install.

| Acceptance group | Required evidence |
| --- | --- |
| Shared engine/schema | All bound catalog operations exposed once; CLI-only admin absent; native validation and authoritative strict Go validation; compiled real calls. |
| Passive continuity | Ordinary fresh Pi session recalls without naming Mandalore; confirmed learning and short journal; Codex recalls it and supersedes a decision; Pi reads current value and retained history. |
| Binding/provenance | Two synthetic banks/profiles; cwd changes; binding replaced between calls, malformed guard, root substitution; original bank unchanged on rejection; authored harness `pi` versus `codex`, stable device identity. |
| Lifecycle/boundaries | Startup/resume/reload/fork, fresh per-turn reads, no growing attachment history, native tools preserved, duplicate extension/colliding tool refused; direct versus quoted cue; conversational no-save and enforced read-only. |
| Delivery/concurrency | Local bare remote and two clones; offline/pending, successful sync, concurrent compatible changes, semantic conflicts; combined-save partial evidence retained; cancellation before/after save, no automatic retries. |
| Transport/context | No shell injection; bounded stdin/stdout/stderr, invalid JSON/protocol/exit, timeout/abort child cleanup; one complete envelope, no duplicate body; measured process latency and real model-visible context. |
| Installation/recovery | Real native install/remove/update from immutable synthetic candidates; empty and populated profile, relative paths/resource filters, changed inputs, interrupted phases, foreign registration refusal, repair preserves unrelated state. |
| Distribution compatibility | Existing strict version reader accepts new executable; eight-file format-1 inventory unchanged; reproducible embedded Pi package, old package/runtime mismatch denied; no extra network package install. |

Use `internal/binding`, `internal/api`, compiled CLI tests, a focused Pi adapter
package and Node test files under `plugins/pi`. Test cancellation only after
observing the targeted process phase. Do not treat a stale file/lock as proof of
a live process. Test errors both before dispatch and after mutation may occur.

## Red/Green Sequence

1. Guarded binding and shared bounded context tests; preserve existing Codex hook
   characterization and same-byte version response.
2. Native adapter transport tests with inert children, then compiled Go runtime
   and the actual Pi loader/schema validation without a model.
3. Embedded package/stamping and typed install-plan/receipt tests, then actual
   native registration/update/recovery in an isolated synthetic profile.
4. Menu and Armorer parity, cancellation/default-No, docs and skill validation.
5. Immutable local candidate; native Codex/Pi semantic matrix, context measurements,
   rendered recordings, final reconciliation, full CI and exact-head review.

## Acceptance Evidence

Engineering evidence identifies source SHA, executable/package hashes, candidate
manifest identity, Pi/Node versions, platform, synthetic fixture and actual checks.
Preserve openable synthetic recordings and compact semantic results, not raw
private sessions. Separate static/compiled/native-no-model/native-model evidence.

UI matrix: harness selection/back; preview with no effects; default-No and EOF;
success; stale/foreign refusal; interrupted registration/repair; color, no-color,
plain output and narrow terminal. Retain recordings and inspect labels, keyboard
arrows/vim navigation, focus and terminal restoration. This extends existing
console patterns rather than establishing an unreviewed visual redesign.
No browser/mobile screen exists. Screen-reader or Linux interaction results are
unverified unless actually exercised; no fabricated accessibility verdict.

Pi tool output uses native rendering, not a custom dashboard. Inspect actual
warning, success and partial-error displays for readability and duplicate bodies.
Record input-token/returned-byte changes separately from response length and
do not claim memory cost from elapsed time alone.

## Rollout

Implementation envelope permits reviewed engineering merges and synthetic native
installation, not release or personal activation. Keep #13 open for a nominated
immutable candidate and independent product acceptance. Codex behavior and public
v1.0.0 stay intact. Claude #14 waits for Pi acceptance, not merely a green PR.
The current eight-file build/nomination/acceptance/promotion route remains; Pi
package bytes travel inside the accepted platform executable.

## Rollback And Recovery

Retain prior known owned runtime/package generations. Explicit native removal of
the new registration and reinstallation of the inspected previous package is
connection rollback, not a bank rollback. Never discard memory written meanwhile.
An interrupted native operation requires inventory and receipt inspection before
retry. Unknown edited assets remain untouched and are reported for user action.

## Release Prerequisites

All implementation/hosted checks, exact-candidate native matrix, current issue/PR
association and product acceptance precede publication. Version compatibility
tests must prove old installer behavior; hashes alone do not establish native
integration. Unsupported host/Pi versions must remain explicit limitations.

## Production Readiness Preflight

Not applicable to production execution: this plan's envelope is implementation.
No release credential or live signet activation is needed or authorized. The
existing publication workflow and its exact-candidate receipt/acceptance gates
remain unchanged. A future approved release must reverify those gates and use
the same immutable bytes; engineering captures cannot be relabeled acceptance.
