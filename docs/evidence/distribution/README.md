# Distribution engineering evidence

This is contributor verification for draft implementation PR #36, not a published
release, immutable-candidate nomination or independent acceptance.

## First real reproducibility run

Source: `5f81257346dbd0deace438f711dcf6d4d2f2e8e9`, clean before and after.
Environment: macOS ARM64, Go 1.26.4, designated native verification pane. Docker
was unavailable; the documented pinned-host fallback was used. Both invocations
used `bin/build-artifacts --output NEW-DIRECTORY` from the same committed source.

Each build made a separate Git export, stamped the plugin only in that export,
used a fresh private Go build/module cache, and built all four Darwin/Linux
amd64/arm64 targets. Direct comparisons of all eight output files passed:
four executables, Codex ZIP, bootstrap, manifest and checksum file. The combined
local script completed in approximately 55 seconds; this is not a performance
guarantee or an OS-cold cache measurement.

- Manifest SHA-256: `f71e898ae5c0885c211d4647a74702086a68eb8f18c3a9462fb7b4095f4d3a21`.
- Checksum file SHA-256: `186282341414caab2c5f14139d9a6022cd115a38ddeecc0c990e77fc3d623bbf`.
- Exact public asset inventory: [build-manifest.json](build-manifest.json).
- Native ARM64 CLI `version` agreed with the manifest's source commit, release
  version, actual embedded plugin content/version, toolchain, protocol, schema
  read/write versions and target.

The `1.0.0` version is the intended stable target, not evidence of publication.
These early candidate bytes are retained local engineering fixtures, not the
final artifact for #10. The remaining CLI install/update and publication work
must be completed before nomination. A later source change requires a new build
and its own exact identity; do not relabel this evidence as testing that change.

## Failing-first and review findings

- Runtime provenance tests failed on missing fields before implementation.
- Bootstrap's valid-handoff test failed before the script existed; synthetic
  tests then passed for verified handoff and checksum, duplicate, missing-file,
  download, foreign-final-host, old-curl, target and version denials.
- Export/build tests failed before the interfaces existed. They cover traversal,
  Git state, symlinks/hardlinks/special files, duplicates, source cleanliness,
  existing output, process cancellation and actual excessive subprocess output.
- The new build process test exposed an embedded `bytes.Buffer.ReadFrom` bypass
  of `Write`. A real native-command regression then reproduced the same older
  runner defect: a 2 MiB subprocess output succeeded despite its 1 MiB limit.
  Both runners now use a named buffer field; real-process red/green tests cover
  the corrected path, not only direct calls to `Write`.
- Go 1.26.4 deliberately omits linker flags from build metadata with `-trimpath`.
  Static inspection verifies actual Go module/target/toolchain settings, not
  nonexistent linker-flag evidence. Source/version provenance is established by
  the controlled export/build and manifest, with native runtime reporting tested
  separately. Checksums alone do not authenticate a publisher.

## Limits and remaining evidence

No native Intel Mac or Linux execution, cross-host rebuild comparison, screen
reader test, fresh release-install menu, native plugin activation, live download
or GitHub Release publication is established by this run. No real signet, native
connection, authentication or session was changed. Structural plugin validation,
hosted CI and later UI/candidate evidence are recorded in the PR at their own
exact heads; none substitutes for independent acceptance.

After this first run, bootstrap review tightened redirect handling to validate
each destination before contacting it, rather than checking only the final host.
The synthetic tests include a bounded redirect loop. That source change does not
retroactively change the retained manifest above; final nomination needs newly
built and independently accepted bytes from the completed implementation.

## CLI installation preview

A later compiled development CLI was exercised against the unchanged real candidate
above on macOS ARM64, through the designated verification pane. `release plan`
and typed `release_plan` emitted byte-identical successful JSON with an intentionally
invalid memory binding. The plan selected the exact manifest and native binary
digests, a new user prefix and an absent launcher; no prefix was created. A separate
synthetic pre-existing regular launcher was refused. Before/after hashes matched
for all eight candidate files and the foreign launcher.

Failing-first tests cover missing plan interfaces, then CLI operation discovery;
passing tests cover exact identities/effects, canonical selection, strict parsing,
owned/retained fixture recognition, changed receipt bytes, redirected directories,
foreign/dangling launchers, FIFO receipts, corruption, writable directories and
pending/unknown ownership. Mocked published planning pins a specific release and
fetches only manifest/checksum assets, never executable payloads. Memory MCP remains
16 tools and rejects direct calls to both release administrative operations.
Self-review reproduced a parser gap with a plan whose destination observations had
been removed. Red/green tests now require the complete ordered directory observations
and consistent receipt, launcher and retained-runtime claims. Parsing still does
not replace fresh source/filesystem verification at apply time.

Retained states in these preview tests are constructed fixtures. This verifies
preview and denial behavior, not successful installation, stale-plan apply,
concurrent activation, recovery, rollback execution or connection updates.
The later activation evidence below is separate from this preview run.
The PR checkpoint binds this later verification to its own implementation head;
it does not change the earlier reproducible candidate's source or manifest identity.

## Native CLI installation, update and rollback

Implementation/candidate source: `2a59cecd92259b22cc219161db0ebec676f61bf7`.
The pinned builder exported this clean commit and created a second local candidate:

- Manifest SHA-256: `3c2c405c0d68bec6c9c9a23a7677c5cee01f8bdaa95174cff79f7e39e474cbaa`.
- Native Darwin ARM64 binary SHA-256: `4975862877d654f855a733a8c4c35a1753e3dd46b50f94cf94f1436923c67f95`.
- Embedded plugin SHA-256: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.

Actual commands in the designated macOS ARM64 pane used the new candidate CLI,
the earlier reproducibility candidate and a fresh temporary prefix. They installed
the earlier candidate, verified its `version`, updated to this candidate through
typed `release_apply`, and verified the new source/version. The installed new CLI
then applied its own reviewed retained rollback plan. The older runtime's `version`
output matched its pre-update output byte-for-byte. Restoring the newer retained
runtime likewise matched its post-update version output. Both content-keyed runtime
files remained present; no pending record remained after success.

Both candidates declare the same intended `1.0.0` version, so this verifies selection
by exact manifest/source identity rather than a mutable version directory. It is
not a published SemVer upgrade or a production downgrade promise. Before/after
SHA-256 inventories matched for all files in both candidate directories and the
separate synthetic signet. No native plugin connection was switched.

A separate compiled-CLI native run verified fresh install, read-only refusal before
creating a prefix, exact completed-plan replay with no destination change, and
byte-identical typed/CLI replay results. Candidate/synthetic-bank inventories and
an unrelated synthetic native-configuration sentinel remained unchanged.

Model-free regressions cover initial/update interruptions after pending record,
launcher activation and receipt publication; exact-plan recovery; rejection of
changed plan/runtime/receipt/launcher/pending state; stale plans; competing writers;
retained rollback; real permission-denied link/receipt writes on a non-root host;
verified/corrupt mocked public downloads; and production refusal of inert payloads.
Runtime metadata tests bind version/source/toolchain/platform/protocol/schema/plugin
fields. Update-recovery tests caught the prior-receipt serialization mismatch;
exact JSON text now survives the strict decoder. A failing CLI regression reproduced
rounding filesystem IDs above float64's exact range; typed plan unwrapping preserves
the full integers. Neither fix relies only on a passing happy-path mock.

Full local pinned-toolchain CI passed at the implementation commit; container-first
validation was attempted and Docker remained unavailable. Hosted checks are linked
in the PR at their exact heads. These runs do not establish native Linux/Intel Mac
release execution, power-loss recovery, interactive installer UX, native plugin
handoff, final candidate acceptance or publication. The full #11/#10 delivery and
release gates remain open.
