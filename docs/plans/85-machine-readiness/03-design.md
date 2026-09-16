# Technical Design

## Component And Behavior Flow

Proposed flow: select harness/paths -> validate bounded input -> inspect known
local metadata -> classify support/setup/evidence separately -> show report ->
Back, optional prompt, or explicitly selected existing native inspection.
No report generation step applies a mutation or launches an agent.

Keep one assessment implementation behind a new administration operation and a
human CLI wrapper. The menu must invoke that operation rather than reimplementing
checks. Preserve existing Armorer command behavior and memory MCP catalog.

## State And Data Model

Proposed versioned report: selected harness; executing toolkit identity;
independently selected retained connection identity when supplied; declared
component requirements; typed observations; scoped evidence matches; untested
boundaries; complete/incomplete status; bounded follow-up text when requested.
No durable machine report, receipt cache, signet revision or consent file.

Declarations cover standalone memory runtime, Git synchronization, Codex and Pi
integration, plus Pi's Node requirement. Development toolchain pins are not
general runtime requirements. Evidence needs immutable source pointers, component,
scenario, artifact/package hashes, platform, applicable native versions and
verification level. Unavailable identity fields cannot be wildcard matches.

## Interfaces And Contracts

Final names and field schema remain design work. Requirements:

- Shared operation is CLI-only, requires no bound memory service, read-only and
  non-network. Human CLI/menu defaults must not invoke native programs.
- Accept an explicit harness and selected absolute metadata paths. A missing
  dependency is a finding; malformed input is a typed error. Do not guess an
  active connection from the newest directory or a remembered path.
- Never read a whole native profile. Explicit owned connection roots can expose
  known receipts and hashes after ownership validation. Bindings require a
  metadata-only reader; no memory-service open or record graph scan.
- Static presence/identity and tested behavior have distinct statuses. Keep
  unsupported, unknown and historically tested readable without color.
- Cancellation never reports completion. Partial observations, if retained,
  must be marked incomplete and must not generate a success/repair claim.
- Viewing a prompt returns text only. The next agent task must recheck the state
  and respect its actual authorization. No shell-ready arbitrary path snippets.

## Authorization And Data Exposure

The operator authorizes inspection of selected setup metadata, not signet content,
credentials, native sessions or external providers. Use bounded regular-file
reads and safe path handling; refuse FIFOs/devices and ambiguous redirection.
Local output may display selected paths, but no private file contents or raw
errors belong in public evidence or generated prompts.

Existing native inspection is a separate explicit choice with selected executable
and profile shown and its possible cache/log effects disclosed. Repair, sync,
install and login remain distinct operations with their existing authority.

## Failure, Recovery, And Observability

Missing components identify the next setup step. Corrupt metadata says inspect or
select another intact receipt, not overwrite it. Unknown wrappers or absent exact
evidence suggest explicit verification, not automatic upgrades. Re-run a read-only
assessment after a separate authorized change; there is no stored assessment to
repair. Ordinary memory use is not gated by this report.

## Design Traceability

Before final review, map every issue acceptance group to concrete schema fields,
reader boundaries, CLI flags, menu states and tests. This draft deliberately does
not claim those contracts are implementation-ready yet.
