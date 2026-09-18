# Local candidate acceptance and delivery

The current workflow is [portable verification](operations/local-verification.md).
Use a fixed merged commit and retained artifact. The maintainer performs the
scenarios below, including the issue's specific criteria, and records observed
results with openable evidence in the structured acceptance report. Run the
configured validation on that same commit and publish the receipt independently
of the implementation author. No owner sign-off or Actions runner is required.

Use `bin/sdlc-release nominate`, `bin/sdlc-evidence run --phase acceptance
--candidate ... --report ...`, and `bin/sdlc-evidence publish`. Then run the release
gate before any promotion. Execute the repository-specific procedure below,
record the release scenarios, and use `bin/sdlc-release record` and `finalize`.
The acting agent must perform the work; a report is an attestation, not a substitute
for execution. Record failure honestly and resolve it before release.

Retain a private attempt record **before** deployment/publication. Bind it to the
issue, accepted digest, prior deployed artifact, backup/recovery plan and target.
Serialize changes to the target. After interruption, inspect actual external
state before retrying; do not repeat an uncertain migration or publication.
Publish sanitized logs and reports, keeping credentials/config/data outside Git.
The release report must name the previous artifact (or explicit initial release)
and recovery procedure, as well as each configured release criterion.

Persistent staging is optional. Temporary isolated acceptance instances are
allowed and should be removed after evidence capture. Never use production data,
queues or external side effects for destructive acceptance. Promotion uses the
same accepted bytes, never a fresh build from moving main.

## Candidate and acceptance

Use `mise exec -- bin/build-artifacts --output NEW_DIRECTORY` from the selected
clean source to produce retained platform/plugin assets and compatibility
manifest. Use the exact `manifest.json` SHA-256 as the portable artifact identity
and supply that file with `--artifact-file`. The manifest binds every retained
payload's size and digest; keep all eight files together outside source control.
Preserve the product's `mandalore:SOURCE_SHA:sha256:MANIFEST_DIGEST` identity too.
Run `mise exec -- bin/promote-local-candidate --verify-only --directory DIR
--identity PRODUCT_IDENTITY` to verify every retained file without publication.

- `issue-criteria`: exercise the assigned behavior and documented compatibility
  boundaries without exposing private memory or credentials.
- `retained-distribution`: verify the manifest, every asset hash, source SHA and
  product candidate identity; never rebuild during publication.
- `native-runtime-and-plugin`: run the actual candidate on supported native
  macOS/Linux targets and exercise CLI/protocol/plugin behavior. Cross-compilation
  is not native acceptance. Preserve the scope of unsupported/unavailable targets.
- `installation-and-recovery`: exercise fresh install, update, owned activation,
  identity checks and retained rollback with isolated synthetic state, including
  rejection of unknown ownership and incompatible schema changes.

## Publication and recovery

After portable acceptance, use `mise exec -- bin/promote-local-candidate
--directory DIR --identity PRODUCT_IDENTITY --issue ISSUE --expected-actor
MAINTAINER`, supplying the independent maintainer's process-scoped credential.
This command re-verifies every local payload, invokes the portable gate against
the accepted manifest digest, then calls the existing same-byte publisher. It
does not require an Actions transport receipt or rebuild product bytes. Preserve
the returned publication receipt, including any uncertain pending operation.
The legacy Actions-backed promoter remains a separate optional path. Preserve
publisher safeguards:
immutable release policy, exact version/tag identity, payload verification,
scoped credentials and retry inspection. Historical owner exceptions do not
authorize new self-acceptance. Do not rotate unrelated service credentials.

- `published-same-bytes`: verify downloaded release assets/checksums/manifests
  match the accepted candidate. Use `bin/release-publication verify` with the
  exact product version and candidate identity as documented by the publisher.
- `public-installation`: verify fresh public install, update and recovery from
  the published bytes; record the product-page closeout or its explicit separate
  scope/status. A release is not proof that all machines auto-updated.
- `immutable-release-and-recovery`: verify the immutable tag/release posture,
  preserve prior artifacts and configuration sources, and record safe recovery
  for data/schema changes. Never retarget an existing release.

If the publisher needs a narrowly scoped policy-read credential, resolve that
access explicitly; do not silently skip the provider policy check.
