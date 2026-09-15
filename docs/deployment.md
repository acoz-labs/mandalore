# Delivery and releases

## Current state

Delivery profile: **artifact**. There is no staging service or hosted production
application. GitHub Releases distribute the CLI and native plugin package.
[v1.0.0](releases/1.0.0.md) is published from the owner-accepted retained bytes.

The product now has a pinned [local candidate builder](development.md#local-distribution-candidates),
checksummed platform/plugin assets, a compatibility manifest and read-only release
inspection and [CLI installation planning](interface.md#read-only-cli-installation-planning).
[Engineering evidence](evidence/distribution/README.md) records the
actual scope tested. Explicit plan/apply now supports owned CLI activation,
identity-checked recovery and retained rollback. The interactive installer/update
journey uses those same operations, with a separately confirmed native handoff
prepared and applied by the verified newly installed runtime. A separate candidate
build/retention workflow and provenance/payload-verified nomination path are now
implemented; [their contracts and current verification limits](development.md#retained-actions-candidates)
are explicit. The [same-byte publisher component](development.md#same-byte-publisher-implementation)
now has simulated-provider staging, byte verification and retry coverage. The
artifact finalizer verifies the published product before updating its ledger.
The retained-candidate promotion workflow is integrated. Actual
[installer/recovery recordings](evidence/distribution/recovery/README.md) and
[fresh native candidate sessions](evidence/distribution/native-candidate.md) provide
source-bound contributor evidence. The release record distinguishes subsequent
hosted retention/nomination, scoped owner acceptance, actual publication and
post-publication verification from those earlier local engineering results.

The repository-specific publisher extends the template's nomination, acceptance
and ledger workflows; the shared template is unchanged. Workflow YAML and local
engineering artifacts alone do not establish a successful hosted release.

## Release contract

1. Merge reviewed and validated release-bearing work. Engineering self-review
   for the current MVP follows [the owner-authorized deviation](decisions/0001-mvp-self-review.md);
   it does not substitute for independent product acceptance below.
2. Build and retain one immutable candidate from the exact nominated source.
3. Record OS/architecture, runtime/plugin versions, digests, checks, known limits,
   supported schema/protocol versions and linked issues.
4. Obtain independent authorized acceptance of the actual artifact. Distinguish
   native macOS/Linux tests from compilation and model-free protocol checks.
5. Publish the same accepted bytes through a Git tag and GitHub Release; never
   rebuild mutable source during promotion.
6. Verify downloadable assets, checksums/manifests, fresh install, update and
   recovery before completing the issue/project release ledger.

Repository creation does not authorize a release. `VERSION` declares the intended
SemVer release, initially 1.0.0, with tag `vVERSION`. Candidate identity additionally
binds the exact commit and manifest digest; a shared version label is insufficient.
Current asset names and compatibility fields are defined by the v1 manifest.
The first same-byte publication used the guarded local promoter after explicit
owner authorization; no Actions credential was provisioned. Never treat old
My Friday releases as compatible Mandalore updates just because they contain Go.

## Configuration and recovery

Current variables: `CI_RUNNER=ubuntu-latest`, `DELIVERY_PROFILE=artifact`,
`STAGING_REQUIRED=false`, `ACCEPTANCE_COMPLETES_DELIVERY=false`,
`ACCEPTANCE_REQUIRED_CHECKS=ci`. Deliberately configure `ACCEPTANCE_ACTORS` for
independent acceptance before release; do not copy private account policies.
No provider service-account credentials are bundled or required.

Real publication also requires immutable releases to be explicitly enabled.
GitHub's [immutable-release settings endpoint](https://docs.github.com/en/rest/repos/repos?apiVersion=2026-03-10#check-if-immutable-releases-are-enabled-for-a-repository)
requires repository Administration:read. The publisher refuses before creating a
draft if that setting is disabled, missing or inaccessible, and rechecks it before
publishing. The planned scoped workflow token alone must not be assumed to provide
this administration read. The workflow now defines an optional
`RELEASE_POLICY_READ_TOKEN` secret binding for a deliberately authorized credential
with repository Administration:read. The command supports the same explicit
environment variable. It is used only for the policy GET, not release/upload APIs
or child authority-check processes. If absent, the publisher tries the explicit
release token and refuses if the setting cannot be read. The publisher itself
does not configure secrets, actors or repository policy. For v1.0.0, the owner
explicitly authorized enabling immutability and local publication. Acceptance
actors were deliberately configured through the recorded MVP acceptance policy.

This additional optional binding is a documented deviation from the earlier
no-new-secret assumption, required by GitHub's policy-read permission contract.
Provisioning remains a deliberate release prerequisite, not permission to copy
a maintainer's personal credential into Actions. The scoped workflow token remains
the release/ledger credential; the separate policy credential needs no write access.

Retain old immutable binaries and connection sources. Preview update/repair
paths, reject unknown ownership, and preserve memory, auth, sessions and native
settings. A binary rollback is not a safe data-schema downgrade by itself.
Retries reuse matching artifacts/tags and reject mismatches. Partial publication
is visibly incomplete. See [runbook](runbook.md).
