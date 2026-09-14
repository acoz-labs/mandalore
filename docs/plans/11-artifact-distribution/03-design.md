# Technical Design

## Component And Behavior Flow

Build exact clean source → verify deterministic package → retain immutable
candidate → nominate verified identity → independently accept → stage matching
draft assets → verify → publish → verify downloadable release → finalize ledger.
Promotion never invokes the compiler. A missing or changed candidate returns to
nomination, not an in-place artifact replacement.

User flow: inspect release/local candidate → preview destination and compatibility
→ default-No confirmation → verify/install retained CLI → optionally preview one
selected connection with the new runtime → separate confirmation → native apply
→ fresh-session notice. Any cancellation stops subsequent phases. CLI installation
success remains visible if later connection activation fails.

## State And Data Model

`VERSION` is the product SemVer source. Initial target is `1.0.0`; development
builds remain visibly development builds unless built by the release builder.
Build metadata reports commit, version, target, protocol/schema support and embedded
plugin identity. Candidate generation stamps plugin metadata in an isolated clean
source export, never the user's checkout. CLI and standalone plugin use the same
stamped input.

Manifest v1 includes product, exact version/tag/source commit, pinned toolchain,
protocol version, signet readable/writable schema versions and a fixed asset list.
Each asset declares kind, target if relevant, safe basename, byte count and SHA-256.
Embedded plugin content identity is recorded independently of archive compression.
Manifest bytes have a SHA-256 candidate identity; no self-referential digest field.
Checksums cover payload assets and the manifest. Versioned bootstrap is also a
checksummed asset. No current time, hostname, absolute path or workflow run ID
enters the deterministic payload.

Asset names: `mandalore_VERSION_GOOS_GOARCH` for the four executables,
`mandalore_VERSION_codex.zip`, `install.sh`, `manifest.json`, `SHA256SUMS`.
Plugin archive uses sorted entries, fixed timestamps/modes and no symlinks.
Raw binaries remove archive extraction from the initial installer trust path.

The transport receipt separately records repository, source SHA, workflow path,
run ID, artifact ID, artifact ZIP digest, manifest digest and expiry. Nomination
binds `mandalore:SOURCE_SHA:sha256:MANIFEST_DIGEST`. Verify a successful trusted
candidate workflow on the default branch and matching source/run/artifact before
using the payload; artifact name alone is not authority.

Local installation uses a chosen user-owned prefix, with a retained content-keyed
runtime and an owned `bin/mandalore` symlink plus receipt. Plan records exact source
identity, target binary and embedded plugin digests, platform, destination state
and intended effects. Apply revalidates all observed identities. Absent paths may
be created; a launcher must be absent or proven owned with matching receipt and
target digest. Never follow arbitrary destination links, overwrite a regular file,
or interpret an arbitrary directory's existence as ownership. Serialize cooperating
installers and reject stale plans. Atomic staging and rename protect activation;
if receipt/activation recovery is ambiguous, report partial state and refuse blind
retry. Retain the previous valid receipt/target until the new state is verified.

## Interfaces And Contracts

- `bin/build-artifacts`: exact clean source, pinned Go, isolated export, deterministic
  metadata stamping, bounded compile processes, new output directory; emits verified
  payload and candidate summary. Never writes native config or a signet.
- `mandalore release inspect [--version VERSION | --candidate DIR]`: bounded public
  official discovery or explicit local candidate verification. No installation.
- `mandalore release plan [selection] --prefix DIR`: outputs an explicit JSON plan,
  including compatibility and overwrite denial. It may fetch into a temporary
  download area but does not activate a runtime or connection.
- `mandalore release apply < plan.json`: revalidates source, destination and plan,
  installs the CLI only, and reports phase/previous target/unchanged connections.
- `mandalore release install [selection] [--prefix DIR] [--plain]`: interactive
  inspect/preview/default-No/apply journey, shared with the menu. Optional selected
  connection handoff is a second preview/confirmation, not part of CLI apply.
- Existing operation discovery advertises corresponding CLI-only typed contracts;
  these operations are not exposed through the bound memory MCP.
- Menu adds a release install/update journey, not silent startup auto-update. It
  reuses the same planning/apply service and readable effects/errors. Selecting a
  connection update invokes the verified new runtime's existing connection plan
  and apply commands with explicit binding/profile. Bounded JSON stdin/stdout,
  fixed command names and argument arrays; no shell interpolation or user scripts.
- `install.sh VERSION`: reviewed POSIX bootstrap, explicit sanitized version,
  supported `uname` target mapping, `curl` and native SHA-256 tool prerequisites.
  Download to a fresh temporary directory from fixed official HTTPS release URLs
  using time/size bounds. Require exactly one well-formed checksum for the expected
  executable basename. Verify before execution, then invoke that executable's
  `release install --version VERSION` journey. No tar/unzip, jq, Python, sudo, shell edits or
  provider credentials. Cleanup is bounded to the directory it created. A checksum
  mismatch, missing tool or interrupted fetch exits without activation. The verified
  executable rechecks manifest/release metadata before its installation plan.

Strictly parse bounded manifests/plans: reject unknown fields, duplicate keys,
unsupported formats, unsafe names, duplicate/missing assets, malformed digests,
invalid SemVer, incompatible schema/protocol and target disagreement. Network
timeouts, maximum response/asset sizes, redirect host/scheme rules and cancellation
are explicit; no bearer token is forwarded across download redirects. Public user
downloads require no GitHub token. Publisher-only API credentials remain workflow
inputs and are never included in logs/assets.

Workflow changes: build/retain candidate with SHA-pinned official upload/download
actions and explicit retention; nomination consumes verified transport coordinates,
not an unchecked free-form artifact string. Release consumes the same coordinates,
checks acceptance and ledger preflight, creates/reuses a draft `vVERSION`, attaches
only missing matching assets, verifies remote bytes, publishes and verifies again.
An existing nonmatching tag, asset or release body is a conflict, not an overwrite.
Require repository release-immutability policy at real publication time; explain
missing setup without silently changing it. Record candidate and final asset IDs,
digests, commit, release URL and phase for recovery.

Adapt `bin/finalize-release` artifact mode to validate the already published SemVer
product release and its complete verified manifest before closing issues. Preserve
production mode, independent actor/issue-set/source checks, and exact existing
nomination/acceptance markers. Do not emit a second empty calendar release.
Document and test this repository-specific managed-template deviation.

## Authorization And Data Exposure

User selection authorizes only the displayed CLI destination or separately chosen
connection; it does not authorize memory writes, migration, changing other signets,
native accounts or shell settings. Read-only selection never executes downloaded
code; only an explicitly approved bootstrap/install path or connection handoff may
execute verified selected bytes. Execution trust is stated in the preview.

Maintainer workflow authority authorizes candidate retention, not release. Release
requires exact independent acceptance and explicit dispatch under the existing
policy. Build jobs have read-only contents; publisher gets necessary contents and
ledger permissions only. No private account conventions are embedded.

## Failure, Recovery, And Observability

Expose structured phases: inspected, downloaded, verified, runtime-retained,
launcher-activated, connection-unchanged, connection-partial or connection-updated.
Publisher phases distinguish draft-created, assets-staged, assets-verified,
published, publication-verified and ledger-complete. A failure does not erase earlier
successful effects or report synchronization/activation that did not happen.

Retry validates the same identity and existing assets; changed identity requires
a new plan or nomination. Unknown partial local ownership requires inspection,
not takeover. Rollback selects a retained verified compatible artifact through
the same preview/apply contract; no signet files are modified. Connection rollback
uses existing retained-generation repair with explicit compatibility cautions.

## Design Traceability

Pinned builder/CI covers reproducibility; manifest/transport covers matrix and
provenance; release installer/bootstrap/menu covers versioned install/update;
phased publisher plus adapted finalizer covers same-byte promotion and ledger;
unchanged independent actor checks cover acceptance authority; retained receipts
and tests cover partial retry/rollback. See verification for evidence boundaries.
