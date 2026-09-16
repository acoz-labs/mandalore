# Discovery: assess a new machine without certifying untested behavior

- **Status:** Draft investigation
- **Discovery issue:** #15
- **Repository basis:** 3818c1b6e4f98dbdf6a5cb5d7a0e8b9f12da2234
- **Recommended decision:** follow-up evidence before selecting delivery
- **Gate 1:** awaiting exact-head engineering review under ADR 0003
- **Confidence:** Medium on current gaps; design not selected
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
this survey. These are source observations, not a native readiness probe.

## Unknowns to resolve

1. Which identities make recorded evidence applicable: full artifact, package,
   protocol, native version and platform? How to report historical/stale evidence
   without pretending either universal compatibility or universal failure?
2. What can be observed without executing an arbitrary newly selected binary,
   creating a missing native profile, synchronizing or contacting a provider?
   Reuse existing bounded trusted probes only where their authorization is clear.
3. How should missing setup, unsupported contract, stale evidence and unknown
   coexist? Independent dimensions may be more truthful than one healthy boolean.
4. What is the minimum helpful menu flow and typed schema, including missing
   native/runtime paths, malformed declarations, cancellation and partial results?
5. Can an optional generated prompt name only observed gaps and scoped next steps,
   avoid private content/raw errors, and explicitly require fresh verification
   without promising that the agent can repair unsupported combinations?

## Competing options

Compare extending existing Armorer reports, a shared assessment composing those
reports, and a separate universal compatibility registry. Prefer reuse and no
new background/index/update service, but do not select before investigating
fresh-machine cases. A static support table alone cannot satisfy host assessment.
Silent installation, credential discovery or native configuration rewriting is
not an option in this scope.

## Success and stop signals

Require truthful structured dimensions, scoped deterministic observations first,
menu/agent parity and synthetic missing/stale/unsupported/cancelled scenarios.
Stop for new trust/data authority, inferred platform acceptance or private
capability adaptation. No outcome is selected or materialized by this draft.

## Next investigation and Gate 1

Inspect current runtime/package metadata and retained evidence identities; probe
only disposable profiles and fixtures using the designated test workspace for
native/manual cases. Complete comparison, user flow and outcome acceptance before
exact-head contributor self-review and merging this discovery. No personal
installation, runtime release or Pi product acceptance is implied.
