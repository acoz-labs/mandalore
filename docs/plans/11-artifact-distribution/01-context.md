# Context

## Problem And Desired Outcome

[Issue #11](https://github.com/acoz-labs/mandalore/issues/11) supplies the actual
distributable product needed for #10 acceptance. A person on another supported
machine must be able to inspect and install the same tested CLI/plugin bytes,
then explicitly connect their chosen memory without building source code.

## Current State

At the repository basis in README:

- `bin/ci`, `mise.toml`, `go.mod` and `Dockerfile.dev` pin Go 1.26.4 and cross-build
  Darwin/Linux amd64/arm64. Cross-build success is not native runtime acceptance.
- `cmd/mandalore/main.go` reports a development version. There is no distribution
  manifest, release asset builder or published update discovery.
- `internal/install` prepares content-addressed connection generations. Preparation
  uses the preparing runtime's embedded `plugins/codex` content, even if a different
  binary was selected; an old runtime must not prepare a new release's connection.
- `cmd/mandalore/menu.go` and `internal/console` provide styled terminal/plain
  journeys with explicit effects, default-No, keyboard and cancellation handling.
- `.github/workflows/nominate-artifact.yml` accepts an arbitrary artifact string.
  `.github/workflows/release-artifact.yml` gates and finalizes a ledger but does
  not package, upload or verify product asset bytes.
- `bin/finalize-release` creates an artifact-dated release without product assets
  and closes eligible issues. Product publication must precede that ledger step.
- Nomination discovers associated PR issue references and outstanding delivery
  labels. Acceptance also requires issue-body implementation PR links. A project
  board's Acceptance column alone does not supply either dependency.
- `docs/deployment.md` correctly says no Mandalore release is published.

## Actors And Critical Journeys

Users: inspect official release, install to a user-owned destination, update the
CLI, optionally update one selected connection, recover an interrupted install,
or select a retained compatible version. Agents use equivalent explicit JSON
operations, not terminal scraping or additional memory MCP privileges.

Maintainers: build a clean exact-source candidate, retain immutable transport
identity, nominate after lifecycle reconciliation, collect independent acceptance,
publish the accepted assets and reconcile the verified release ledger on retry.

Denials include unknown ownership, changed source/digest, unsupported target,
incompatible manifest/schema, expired candidate, missing acceptance, partial or
foreign release assets, unapproved activation and network failure.

## Acceptance And Non-Goals

Cover all six issue acceptance groups: pinned validation, platform packages and
provenance, version/install compatibility, same-byte promotion and recovery,
independent actors, and verified release-driven lifecycle completion.

Do not ship Pi/Claude adapters, predecessor compatibility, automatic Git clone,
remote inference, provider credentials, OS package management, arbitrary scripts,
memory migrations, background self-updates or an assistant launcher.

## Constraints, Dependencies, And Risks

Public assets contain only tracked reviewed code and synthetic evidence. Builds
must not embed workstation paths, credentials, hostnames or local signets. Existing
native auth/settings/sessions and unselected connections remain untouched.

The installer trusts explicitly selected official release bytes as executable
code. A local candidate is a separate, visibly explicit engineering trust path.
GitHub compromise and malicious same-user processes are not solved by checksums.
Filesystem identity checks must still reject links, changed ownership receipts
and concurrent destination changes rather than blindly overwrite.

## Evidence, Assumptions, And Unknowns

Verified: no repository releases exist; `ACCEPTANCE_ACTORS` is not configured;
the current Docker daemon is unavailable. None is grounds for fabricated success.
Physical OS/architecture and independent candidate behavior remain to be measured.

GitHub supports immutable artifact IDs/digests and release asset digests. Published
release immutability is a repository setting, not something a checksum file turns
on. Inspect the actual setting/status before making that claim; changing repository
release policy requires deliberate release setup.

Primary contracts:

- https://docs.github.com/en/rest/actions/artifacts
- https://docs.github.com/en/rest/releases/releases
- https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases
- https://go.dev/src/cmd/go/internal/work/build.go
