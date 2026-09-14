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
