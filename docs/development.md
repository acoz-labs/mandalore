# Development

Run `mise exec go@1.26.4 -- bin/ci` from the repository root. It checks the managed
solution-plan contract, shell syntax, documentation, public fixture hygiene,
workflow runner configuration, formatting, module integrity, race-enabled tests,
vet, and engine/CLI builds for macOS/Linux on amd64/arm64. Native execution is only
established on the host running tests; a cross-build is not runtime acceptance.

`bin/container bin/ci` is the container entrypoint. If Docker is unavailable,
use the supported host fallback `bin/ci` and report the route actually verified.
Go is pinned to 1.26.4 in mise, the module toolchain, CI and the development
container. The module language baseline is Go 1.26.0. CI rejects a different
active toolchain; do not install or use an unrecorded latest runtime.

Use synthetic banks and disposable project workspaces. Native integration tests
may inherit existing native resources and authentication with explicit local
runtime/binding selection; do not replace CODEX_HOME or copy authentication,
private transcripts or real memory into acceptance fixtures.

## Public CI

Every job uses the explicit `CI_RUNNER=ubuntu-latest` repository variable,
following the public predecessor's pattern. No public fork code goes to private
runners. This is an intentional configuration, not an automatic fallback.
Release/acceptance workflows retain their separate permissions and checks.

Local validation is not independent product acceptance. Record exact source and
artifact identities, actual native results and pending checks in the issue/PR.

Build the development executable with `go build -o ./mandalore ./cmd/mandalore`
under the pinned toolchain. See [interface](interface.md) for synthetic examples.
The compiled-process regression builds its own temporary executable and exercises
actual stdio without a model, authentication, native settings or global install.

## Local distribution candidates

From a clean committed repository, run:

```sh
mise exec go@1.26.4 -- bin/build-artifacts --output ../mandalore-candidate
```

The output must not exist, its parent must exist, and it must not overlap the
source repository. This builds local engineering artifacts only: no release,
tag, native connection or signet is created. `VERSION` declares the intended
release version; unstamped development binaries continue to report `0.0.0-dev`.

The builder exports the exact commit, refuses unsafe archive entries, stamps
the native plugin in that private export and compiles all four supported targets.
Go 1.26.4, disabled CGO, trimpath, isolated build/module caches, readonly module
resolution and the public Go module proxy/checksum database are explicit. Ambient
Go workspace/configuration and provider tokens are not passed to those children.
No home-directory or shell configuration is changed. Each tool process has a
bounded output and five-minute deadline; the build has a twenty-minute deadline.
Cancellation cleans its process group and the invocation's private export/cache.

The candidate contains four raw executables, a deterministic Codex ZIP, reviewed
bootstrap, manifest and checksums. Every payload digest and the separate embedded
plugin identity are verified before a no-replace directory publication. Retrying
an existing destination is deliberately refused; select a new output directory.
Candidate integrity is not native behavior, independent provenance or acceptance.

Repeat with another absent output directory and compare all eight files to measure
reproducibility. [Recorded engineering evidence](evidence/distribution/README.md)
includes two actual isolated builds and native metadata verification. This does
not establish cross-host reproducibility or native Linux support by itself.

The packaged POSIX bootstrap currently requires curl 8.4+ plus `sha256sum` or
`shasum`, and invokes the release-install journey after checking the platform
binary against official-release checksums. Curl 8.4+ is required because earlier
versions do not enforce the size limit during unknown-length transfers.
See [curl's size-limit contract](https://curl.se/docs/manpage.html#--max-filesize).
The manual alternative is downloading and verifying the platform binary yourself.
Release discovery and read-only installation planning are implemented. Activation
and promotion are still being implemented;
do not present this early candidate as a complete released installer.

Synthetic bootstrap tests substitute download/host commands and never contact a
provider. Run `go test ./internal/distribution ./internal/install ./cmd/mandalore`
under the pinned toolchain for package/build and process-bound regressions.
The real subprocess output tests are important: directly testing a writer's `Write`
method alone would miss an `io.Copy`/promoted `ReadFrom` limit bypass.

## Other development checks

Menu presentation uses pinned Bubble Tea 2.0.9 and small terminal/ANSI/Unicode
helpers declared in go.mod/go.sum. It remains optional at invocation time: pipes,
`--plain`, `MANDALORE_PLAIN=1` and dumb terminals use line-oriented prompts.
The shared API remains the automation interface, not terminal key simulation.
Tests cover incomplete EOF, default-No, output failure, partial durable setup,
navigation, normal text entry, literal paths and narrow output. Native menu tests
also compare terminal settings before/after cancellation. See
[menu engineering evidence](menu-engineering-evidence.md) for actual coverage and
limits; no-color/text checks do not establish screen-reader acceptance.

Synchronization tests additionally require native Git with merge-tree --write-tree,
commit-tree and standalone repository support. CI logs the actual Git version;
unsupported features fail the real two-clone tests rather than being silently
replaced with an in-place merge. The fixtures use local bare remotes and controlled
failure wrappers, not external provider authentication. See [synchronization](synchronization.md).

Migration tests use synthetic legacy fixtures under
`internal/migration/testdata/bank-v1`, not a user's bank or the old executable.
The converter's preflight shares the memory engine's pure snapshot validation;
changes to this seam require the existing graph, integrity and synchronization
regressions. Tests cover raw extension precision, authorship/history, conflicting
heads, original snapshot bytes/empty directories, lock observations, stale inputs,
typed/CLI parity and retained staging/publication failures. Cross-builds still
do not establish native Linux filesystem/locking acceptance.

The Codex plugin and skill also require their native validators when edited.
The development check used the plugin-creator and skill-creator validators with
an isolated `uv run --with PyYAML==6.0.2` environment; PyYAML is a validation
dependency, not a runtime requirement. Go tests exercise the real shell bridge
and compiled hook command. These structural checks do not establish model
behavior: a native forward test caught an ambiguous no-save orientation sentence
that the unit tests could not. Repeat relevant fresh-session scenarios after
instruction changes and record the exact tested binary/plugin identities.

Foundling fixtures exercise bounded local files and standalone Git sources without
executing source configuration. Their creation commands disable automatic Git
maintenance so background fixture writes cannot race source-preservation snapshots;
a trace regression guards that boundary and failures report changed paths without
dumping contents. Keep the full source-tree assertion, including Git metadata.
The native foundling pilot caught a nonempty-query discoverability/error problem
that deterministic storage tests did not expose; the resulting strict API test,
skill correction and actual fresh-session retest are documented in
[native foundling evidence](evidence/foundlings-native/README.md).

Opt-in [retrieval evaluations](retrieval.md) measure filesystem-backed memory,
scope routing, prompt hooks, foundlings and a compiled CLI using validated synthetic
fixtures. Corpus construction is outside timed sections. Report at least 20
samples for p95 and distinguish returned bytes, allocations, native counters and
fresh processes from OS-cold storage. Expensive scale measurements are not noisy
wall-time gates in CI; deterministic quality, freshness, corruption and budget
regressions remain required. The shared fixture generator is test-only, not a
second supported memory writer.
