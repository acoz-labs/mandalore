# Context

## Problem And Desired Outcome

[Issue #13](https://github.com/acoz-labs/mandalore/issues/13) makes the same signet
useful in ordinary Pi sessions, including passive continuity and learning, not
only explicit memory commands. Codex MVP #10 is accepted. A user should connect
Pi, start it from a project, and continue remembered decisions without importing
threads, credentials, an assistant identity or a second memory format.

## Current State

At the repository basis, `plugins/pi/README.md` is a placeholder.
`internal/api` owns typed schemas, read-only enforcement and delivery receipts;
MCP exposes the 18 bound, non-CLI-only operations. `internal/binding` selects an
explicit machine-local identity, while `memory.Service` verifies bank identity.
`internal/codex/hooks.go` supplies read-only, bounded local orientation/recall.
`internal/install`, the connection API and console currently manage Codex only.
Two skills under `plugins/codex` separate memory from on-demand administration.

Distribution format 1 has a fixed inventory: four executables, Codex ZIP,
bootstrap, manifest and checksums. `internal/distribution/install_io.go` strictly
decodes the current version response; adding an unknown Pi field there would
break existing updaters. Public v1.0.0 artifacts remain immutable.

## Actors And Critical Journeys

- Existing signet user: choose Pi in setup, inspect effects, explicitly connect,
  then use ordinary Pi from any cwd with inherited native resources intact.
- Conversational user: recall, confirm a changed decision, retain supersession
  history and provenance, and distinguish a local save from remote delivery.
- Read-only user: recall without saves, journals or synchronization. Enforced
  read-only connections also reject mutations at the Go boundary.
- Maintainer/Armorer: inspect an explicit connection, preview update or repair,
  preserve unknown files, and explain partial native registration or stale context.
- User with separate banks/profiles: no cwd fallback or silent bank switching;
  a second profile can be connected explicitly to a different signet.

## Acceptance And Non-Goals

Deliver the full issue: native integration, lifecycle mapping, passive learning,
explicit cue, read-only limits, Codex/Pi continuity, isolation, provenance,
offline/concurrent sync, bounded context, native installation/update/recovery.
No Pi-specific memory backend, model access proxy, credential management, new
assistant launcher, copied sessions, Claude integration or lifecycle writer.
Context retrieval is not promised to remember every conversation verbatim.

## Constraints, Dependencies, And Risks

Pi executes trusted extension code with the user's permissions. Installation is
an explicit trust decision; hashes establish identity, not publisher trust or a
sandbox against a malicious local account. Native package installation may edit
the selected profile's settings, but unrelated resources remain Pi-owned.
Mandalore must not run npm install or install Pi/Node implicitly.

Support the existing darwin/linux amd64/arm64 runtime targets. Actual native
results must identify platform, Pi and Node versions; cross-builds are not native
acceptance. Use existing native authentication only in authorized synthetic model
tests; command-only probes need none. Public evidence must be synthetic.

## Evidence, Assumptions, And Unknowns

Inspected installed Pi 0.85.1 and upstream tag commit
`d981de1229ef899957bbe968bc8dcda02a21f477`:

- [Extensions](https://github.com/earendil-works/pi/blob/d981de1229ef899957bbe968bc8dcda02a21f477/packages/coding-agent/docs/extensions.md)
  and local type/runner sources provide tools, per-turn system-prompt additions,
  session events, cancellation and tool inventory. Preserve the chained prompt.
- [Packages](https://github.com/earendil-works/pi/blob/d981de1229ef899957bbe968bc8dcda02a21f477/packages/coding-agent/docs/packages.md)
  and [settings](https://github.com/earendil-works/pi/blob/d981de1229ef899957bbe968bc8dcda02a21f477/packages/coding-agent/docs/settings.md)
  support local-path packages containing extensions and skills. Local packages
  are registered, not copied; settings may normalize their paths relatively.
- A disposable native install/command/remove probe on macOS arm64, Pi 0.85.1,
  Node 24.1.0 verified startup/resource/shutdown events and retained native
  read/bash/edit/write tools. Native validation accepted five representative
  shared schemas and rejected unknown fields, including nested combined-save
  input. An initial probe incorrectly used `write` instead of `record`; corrected
  input and failure assertions passed the repeated native run.

That probe is contract evidence, not model behavior, all-schema coverage, release
acceptance or Linux execution. No blocker remains for designing the integration;
native coercion, event ordering, interruption and update ownership require the
explicit implementation tests below, not assumptions of equivalence with Codex.
