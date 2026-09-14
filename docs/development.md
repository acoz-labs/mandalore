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

Synchronization tests additionally require native Git with merge-tree --write-tree,
commit-tree and standalone repository support. CI logs the actual Git version;
unsupported features fail the real two-clone tests rather than being silently
replaced with an in-place merge. The fixtures use local bare remotes and controlled
failure wrappers, not external provider authentication. See [synchronization](synchronization.md).

The Codex plugin and skill also require their native validators when edited.
The development check used the plugin-creator and skill-creator validators with
an isolated `uv run --with PyYAML==6.0.2` environment; PyYAML is a validation
dependency, not a runtime requirement. Go tests exercise the real shell bridge
and compiled hook command. These structural checks do not establish model
behavior: a native forward test caught an ambiguous no-save orientation sentence
that the unit tests could not. Repeat relevant fresh-session scenarios after
instruction changes and record the exact tested binary/plugin identities.
