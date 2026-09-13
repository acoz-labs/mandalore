# Delivery and releases

## Current state

Delivery profile: **artifact**. There is no staging service or hosted production
application. GitHub Releases will distribute the CLI and native plugins. No
Mandalore runtime or release is published yet.

The template supplies nomination, acceptance and release-ledger workflows, not
this product's build matrix, checksummed assets, installer or update manifest.
Those remain tracked work; workflow YAML alone does not establish distribution.

## Release contract

1. Merge independently reviewed and validated release-bearing work.
2. Build and retain one immutable candidate from the exact nominated source.
3. Record OS/architecture, runtime/plugin versions, digests, checks, known limits,
   supported schema/protocol versions and linked issues.
4. Obtain independent authorized acceptance of the actual artifact. Distinguish
   native macOS/Linux tests from compilation and model-free protocol checks.
5. Publish the same accepted bytes through a Git tag and GitHub Release; never
   rebuild mutable source during promotion.
6. Verify downloadable assets, checksums/manifests, fresh install, update and
   recovery before completing the issue/project release ledger.

Repository creation does not authorize a release. SemVer, asset naming and
promotion commands are finalized in the distribution issue. Never treat old
My Friday releases as compatible Mandalore updates just because they contain Go.

## Configuration and recovery

Current variables: `CI_RUNNER=ubuntu-latest`, `DELIVERY_PROFILE=artifact`,
`STAGING_REQUIRED=false`, `ACCEPTANCE_COMPLETES_DELIVERY=false`,
`ACCEPTANCE_REQUIRED_CHECKS=ci`. Deliberately configure `ACCEPTANCE_ACTORS` for
independent acceptance before release; do not copy private account policies.
No provider service-account credentials are bundled or required.

Retain old immutable binaries and connection sources. Preview update/repair
paths, reject unknown ownership, and preserve memory, auth, sessions and native
settings. A binary rollback is not a safe data-schema downgrade by itself.
Retries reuse matching artifacts/tags and reject mismatches. Partial publication
is visibly incomplete. See [runbook](runbook.md).
