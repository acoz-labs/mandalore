# Discovery: assess a new machine without certifying untested behavior

- **Status:** Decision proposed for engineering self-review
- **Discovery issue:** #15
- **Discovery PR:** #84
- **Repository basis:** 3818c1b6e4f98dbdf6a5cb5d7a0e8b9f12da2234
- **Recommended decision:** select O1, a non-executing readiness assessment
- **Gate 1:** awaiting exact-head engineering review under ADR 0003
- **Confidence:** High on execution boundary; Medium on implementation shape
- **Private evidence:** none

## Decision sought

Define structured support/dependency/test declarations and a read-only assessment
usable from the menu and shared CLI interface. Separate what Mandalore intends
to support, what has actually been tested, and what this invocation observes.
Optional agentic follow-up must describe verified gaps, not grant repair authority.

## Audience and critical tasks

A user connects an existing signet on another machine or changes native harness.
They need to identify missing setup and unverified combinations without confusing
a green structural inspection, cross-build or successful login with an end-to-end
memory test. The agent should consume the same facts as the human-facing menu.
Private capabilities, package management and credential enrollment stay outside
this memory-only product.

## Current evidence

At the basis, source inspection establishes:

- `cmd/mandalore/version.go` reports runtime OS/architecture, source/version,
  protocols, readable/writable signet formats and Codex package identity. It has
  no common support/dependency/native-evidence declaration. Runtime architecture
  describes the executing build, not necessarily physical CPU identity.
- `internal/distribution/manifest.go` validates four executable targets:
  Darwin/Linux, amd64/arm64. A declared/downloadable target is not proof of native
  harness execution or exact-candidate acceptance.
- `internal/install/doctor.go` checks owned Codex registration, retained bytes,
  binding and native inventory. It expressly leaves login, hook trust, live MCP,
  remote freshness and active context not-tested. Missing profile avoids native
  invocation that might create it.
- `internal/install/pi_doctor.go` similarly distinguishes retained structure,
  native identity/version and registration from login/live tools/freshness/context.
  `pi_apply.go` currently accepts exactly native Pi 0.85.1, not an inferred range.
- `plugins/pi/package/package.json` declares Node >=22.19.0; development CI uses
  Node 24.1.0. Node is not a prerequisite of the standalone Go memory engine.
- `docs/development.md` separates pinned test/build dependencies from runtime
  needs and says cross-builds are not native acceptance. Git synchronization needs
  merge-tree --write-tree and commit-tree; an arbitrary version string alone does
  not establish feature availability or provider authentication.
- Native evidence is in Markdown and separate immutable recordings: Codex and Pi
  have measured macOS arm64 cases, not a universal OS/harness support certificate.
  Existing evidence must be traced to exact identities before any machine-readable
  claim is selected. A newer source commit is not automatically covered.
- The menu delegates Armorer inspection to existing shared operations. Preserve
  this single implementation path; do not add a second diagnosis engine in UI.

No live user profile, signet, credentials or provider account was inspected for
this survey. The following native probe used only disposable selected profiles;
it did not launch a model or test the user's login.

### Native discovery probe

On 2026-09-16 UTC, an actual CLI probe ran in the designated testing terminal on
Darwin arm64, using source `161f95971aad0b91f8f9cfc531e07dbd7481a6bd`, Go 1.26.4,
Node 24.1.0 and executable SHA-256
`2f979dbdcb42988be5a35c428062b73d070d0bb42df00026154951d64ce70469`.
It invoked `connection armorer` with explicit disposable state/profile paths and
selected native executables: Codex 0.154.0 and Pi 0.85.1. Before/after inventories
covered fixture directories and regular-file content hashes, including new empty
directories. Native output was parsed into bounded check names, never published
as raw logs. Each process had a 15-second deadline and 1 MiB output limit.

| Harness / starting profile | Failed check | Fixture changes | State directory created |
| --- | --- | --- | --- |
| Codex / absent | native-profile | None | No |
| Codex / existing empty | connection | native/tmp/ and native/tmp/arg0/ directories | No |
| Pi / absent | native-profile | None | No |
| Pi / existing empty | connection | None | No |

All four invocations returned exit 1, `connection.failed`, a present structured
report and `healthy: false`. Both Codex reports retained the five not-tested
boundaries listed above; Pi retained its four. These expected setup failures are
not product defects. In the empty Pi case, source inspection shows the missing
registration returns before the native version probe; it does not establish that
every Pi native inspection is free of writes.

The initial observer read only successful-envelope `result` and missed failed
reports under `error.connection_report` / `error.pi_connection_report`. That
observer was corrected and all four cases repeated in fresh fixtures. Both runs
observed the same directory changes. Do not interpret the first run's null report
fields as missing product diagnostics. The second run is the report evidence.
The terminal returned to its shell; no authentication was copied or exercised.
This observes selected fixture contents, not every filesystem write, OS cache,
access timestamp, network effect or descendant process on the machine.

**Consequence:** an inventory command is not necessarily a zero-write command.
Existing Armorer's native-execution boundary must remain visible; the new default
assessment must not invoke it silently. This conclusion is based on measured
behavior, not on assuming that a command named "list" cannot write.

### Evidence applicability

The [published 1.0.0 record](../../releases/1.0.0.md) identifies source
`f899cf6a2a255f3b6b35dcd778c672f799c65eb2` and manifest
`c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea`.
Its [immutable native receipts](https://github.com/acoz-labs/mandalore/blob/1c13e2230c47a34d77c2a51f11a428cc08ebf2d3/docs/evidence/final-candidate/README.md)
establish actual CLI/MCP execution on all four targets, but native Codex
conversations only on Darwin arm64. The earlier receipt's pending verdict is not
acceptance; the later release record links the actual owner verdict separately.

Pi's [final implementation evidence](https://github.com/acoz-labs/mandalore/blob/a102ab8b6142c7e5c62ba04fbb33b4907436286e/docs/evidence/pi/final-head-41a45b7/README.md)
identifies source `41a45b730c74070391af0305a1283e136b9ae8de`, manifest
`e6a9919de1ddbad3d3514a7c5519336505057929ad83ea009b851d57ac7addc1`
and package `79d2491186a6da434bd1e7fd8ea212136595247e90162c5bb3beabc6639614f8`.
Its native scenarios used Pi 0.85.1, Node 24.1.0 and Darwin arm64. These remain
engineering results, not product acceptance or Linux native evidence.

Therefore test records need component, scenario, exact tested identities,
platform, applicable native versions, verification level and immutable source
reference. Version labels and four successful cross-builds are insufficient.
Current development bytes must not inherit an earlier artifact's verified state.

## Competing options

| Option | Benefit | Decision |
| --- | --- | --- |
| Add a support table only | Small and readable | Insufficient: no current-machine observations or scoped follow-up |
| Run current Armorer implicitly | Reuses deep checks | Rejected default: executes selected binaries and can write native state; missing PATH selection can abort before a full report |
| Shared non-executing assessment, explicit deeper handoff | Works before setup; truthful side-effect boundary; menu/agent parity | Selected |
| Universal compatibility registry / background probe service | Potentially broad automation | Not selected: unnecessary network, trust, lifecycle and maintenance scope |

## Decision and user flow

Provide one versioned support/dependency/evidence declaration and one shared
assessment operation for the memory runtime plus Codex/Pi integration. The menu
and agent-facing administration CLI consume that operation. Do not inflate the
ordinary memory MCP catalog with installation data or change the existing Armorer
operation's meaning. The Armorer menu may offer the new assessment alongside its
existing explicitly native checks.

1. Select the harness and show the selected paths and executing runtime identity.
   Offer existing local defaults; absent binaries/profiles/bindings are findings,
   not a reason to create them or abort every other observation.
2. Inspect bounded, known local metadata without subprocesses, network, writes,
   directory creation, locks or temporary workspaces. Use filesystem/PATH presence,
   executable format when safely detectable, owned receipts and binding metadata;
   do not read remembered content, credentials, transcripts or arbitrary plugins.
   Reuse safe validation primitives rather than duplicating memory semantics.
3. Show independent dimensions per component: intended support, observed setup,
   applicable evidence and not-tested behavior. Unknown is a useful result.
4. Offer a separately selected existing Armorer check with a clear native-process
   boundary: it can create native logs/caches. This is not automatic repair and
   does not make provider login, live tools or active context tested.
5. Offer an optional follow-up prompt assembled from typed findings. It describes
   gaps and proposed verification, makes current authorization explicit and asks
   the agent to inspect before making changes. No agent launches, installs,
   credential enrollment, sync or native configuration rewrite occurs merely
   because the user viewed or copied the prompt.

Keep arrows/Vim navigation, Back/cancel, plain/no-color output and readable narrow
terminals. Human and JSON results must use the same observations, with no single
green "everything works" verdict. Missing setup should give one practical next
action, not a wall of technical details. Exact public flag/schema names and
rendered product-design acceptance belong to the selected outcome's plan.

### Report semantics and boundaries

- **Support:** supported contract target, unsupported target or unknown. Distinguish
  process/build architecture from physical host/translation information; report
  unknown rather than executing host commands to guess it.
- **Setup:** present/structurally verified, missing setup, inconsistent or unknown,
  with stable typed reasons and explicit scope. A path on PATH is not proof of
  runnable, compatible or authenticated software.
- **Evidence:** exact-match verified scenario, historical/nonmatching evidence,
  or none/unknown. "Stale" means identities no longer match, not that software is
  broken. Matching evidence does not prove that this user's current session works.
- **Dependencies:** distinguish standalone runtime, Git synchronization, native
  harness and Pi's Node requirement from development-only toolchain pins. A
  presence check cannot certify Git features or the Node runtime actually used by
  a wrapper; leave those untested until separately authorized verification.
- **Provenance:** bind declarations to the executing toolkit and report an owned
  retained connection runtime separately. Do not execute that retained binary or
  assume it has current toolkit metadata. Unknown/differing identity is visible.
- **Evidence integrity:** reviewed bundled records, explicit schema/limits and
  immutable evidence pointers; no automatic online registry. Do not embed a
  binary's own final hash into itself. Post-build exact-artifact results stay in
  separate receipts and can inform later reviewed catalogs; an unknown current
  artifact is preferable to a circular or source-only certification claim.
- **Failure:** malformed/oversized metadata, unsafe file types and cancellation
  produce typed bounded findings or an explicit incomplete assessment. No partial
  result is represented as complete or repaired. No raw stderr or arbitrary
  file content is inserted into a generated agent prompt.

## Assumptions and remaining design questions

This is a targeted memory-integration assessment, not a vulnerability scanner,
hardware inventory, operating-system support guarantee or capability adapter.
OS read caches/access times are outside a no-product-write guarantee. Concurrent
filesystem changes mean observations are a snapshot, not a lease on readiness.

Solution design must resolve the exact bounded static receipt/binding projection,
component identity matching (including script/wrapper indirection), declaration
location and test-evidence schema, cancellation/error envelopes and menu labels.
These are implementation decisions within O1, not permission to execute binaries,
parse arbitrary user source or broaden what counts as verified. If static checks
cannot decide a question, report unknown and name the explicit next check.

## Success and stop signals

Require truthful structured dimensions, scoped deterministic observations first,
menu/agent parity and synthetic missing/stale/unsupported/cancelled scenarios.
Stop for new trust/data authority, inferred platform acceptance or private
capability adaptation. Prove default no-execution/no-network/no-product-write
behavior with trap executables and complete synthetic fixture inventories, not
only successful report snapshots. Do not convert "untested" into "unsupported".

## Candidate outcome map

### O1 — Assess memory integration readiness on a new machine

- **Disposition:** selected for materialization after exact-head review/merge.
- **Outcome:** structured support, dependency and evidence declarations plus a
  non-executing current-machine assessment, shared CLI/menu flow and optional
  bounded agent follow-up prompt. All parts are one coherent delivery outcome;
  a documentation-only matrix does not satisfy #15.
- **Acceptance:** distinguish supported/exact-tested, supported/not-tested,
  historical evidence, unknown and unsupported without universal certification;
  identify missing native binary/profile/binding, stale retained identities and
  unmet dependencies. Test Codex and Pi, malformed/oversized metadata, wrapper
  uncertainty, permissions, special files, cancellation and partial results.
  Assert no subprocess, network, signet/native-config write or directory creation
  during assessment. Verify CLI/menu parity and prompt sanitization/no inherited
  repair authority. Record actual rendered normal/error/cancel/narrow/plain and
  arrows/Vim cases against the implementation head in the test pane. Separately
  test the explicit native-check handoff and disclose its side-effect boundary.
- **Dependencies:** existing installation/Armorer APIs (#7) and merged Pi
  integration (#13); #13 candidate acceptance is not claimed by this delivery.
- **Sequence:** shape a self-contained issue with this immutable discovery head,
  complete product and solution design, implement with TDD, reconcile/promote docs,
  run exact-head engineering review/checks and retain candidate acceptance gate.

No second delivery issue for a universal registry, package manager or private
capability adaptation is selected or parked by this discovery; these are excluded
product scope rather than an implied future commitment.

## Privacy and evidence handling

Public records contain synthetic fixture-relative paths, versions, hashes and
check names only. No personal hostname, account, login status, absolute workstation
path, signet ID/content or native transcript is needed. Local assessments may show
selected paths to their operator; generated prompts should use bounded neutral
findings, not copy private raw reports. No telemetry or report publication is
automatic. Existing personal installations and v1.0.0 remain untouched.

## Decision Spotlight

Readiness is not a single pass/fail fact. A supported platform can lack setup;
a configured integration can lack exact native test evidence. The useful default
is to explain these differences without executing the software being assessed.

## Gate 1

ADR 0003 authorizes a separate contributor engineering self-review of the exact
final head. Record findings and current CI before merge; then materialize O1.
This decision grants no personal installation, Pi acceptance, new release,
credential enrollment or automatic repair authority.
