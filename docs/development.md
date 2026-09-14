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

### Cancellation regression tests

Observe the intended subprocess phase before cancelling a phase-specific test;
do not infer fetch entry from a short whole-operation timeout. The API regression
uses a PATH-local synthetic Git wrapper, a fetch-entry marker and explicit parent
cancellation with bounded worker cleanup. Normal and deliberately slow startup
must both preserve post-checkpoint fetch evidence. A separate blocked-first-Git
case verifies the real one-second operation deadline and conservative checkpoint-
phase evidence before fetch. Both cases check that the observed process ended and
the signet still validates. Failures print the structured envelope, not a pointer.

Run `mise exec go@1.26.4 -- go test -race -count=10 -run '^TestSync(CancellationExposesPartialWriteEvidence|DeadlineBeforeCheckpointRetainsEvidence)$' ./internal/api`
for repeated focused verification, then the complete `bin/ci`. This test repair
does not change the product's timeout or retry contract. Keep original failures
and controlled reproductions in review evidence; a passing retry alone is not a
root-cause analysis or independent acceptance of a candidate.

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

## Retained Actions candidates

The manually dispatched **Build retained candidate** workflow checks out its exact
default-branch event commit, runs CI and builds one complete payload. The SHA-pinned
upload action retains all eight files as one archive for up to 90 days, subject to
repository policy. Each run attempt has its own name; overwrite is disabled. The
summary records source, run/artifact IDs and archive digest. Building/retaining is
not nomination, independent acceptance or publication.

**Nominate artifact candidate** now accepts source SHA plus run/artifact IDs, not a
free-form accepted artifact string. It checks out trusted default-branch verification
tooling (not arbitrary input source), inspects official metadata, downloads the exact
raw archive with extraction disabled, verifies it independently, retains the JSON
verification receipt, then runs the existing release and issue-nomination gates.
It derives `mandalore:SOURCE_SHA:sha256:MANIFEST_DIGEST` from verified payload bytes.
The artifact archive digest is a separate transport identity, not the manifest digest.

The maintainer helper can be invoked with the pinned Go toolchain:

```sh
bin/candidate-transport inspect --source FULL_SHA --run-id RUN_ID --artifact-id ARTIFACT_ID > transport.json
bin/candidate-transport verify --receipt transport.json --archive downloaded.zip > verified.json
```

For promotion, `verify --expected-identity ACCEPTED_IDENTITY` additionally requires
the independently accepted identity. Verification refreshes metadata before and
after inspecting the archive; a saved local receipt alone is not provenance. The
helper never extracts or executes payloads, installs software, writes memory or
nominates an issue by itself. It is repository-maintenance tooling, not a memory
MCP operation or personal capability.

An explicitly supplied `GH_TOKEN` is used only for fixed same-repository metadata
GETs; redirects and unrelated endpoints are refused, and raw provider errors and
credentials are not printed. The separate pinned download action handles artifact
transport with its scoped workflow token. Ordinary public release inspection remains
anonymous and does not borrow this credential.

Checks bind repository IDs, default branch, workflow identity/path, successful run,
source, attempt, archive ID/name/size/digest and expiry. JSON sizes, ZIP size and
central-directory parsing are bounded. Every archive member must belong to the
fixed regular-file inventory; source/toolchain, checksums, payload hashes and plugin
content identity must agree. ZIP64/multi-disk, nested paths, links, duplicates,
corruption and changed receipts are rejected. This format's eight bounded payloads
do not require ZIP64. Expired/deleted/replaced artifacts require new inspection and,
where identity changes, fresh nomination—not pretending a rebuild is accepted.

The Actions pipeline still requires a real default-branch candidate run and its
own hosted verification after merge. Local HTTP/ZIP fixtures and a live negative
provenance check are not successful hosted candidate retention or nomination.
The guarded promotion/finalizer path is implemented below; real hosted execution
and final #11/#10 acceptance remain incomplete.

Primary contracts: [GitHub artifact metadata](https://docs.github.com/en/rest/actions/artifacts?apiVersion=2026-03-10),
[workflow-run metadata](https://docs.github.com/en/rest/actions/workflow-runs?apiVersion=2026-03-10),
[pinned upload action](https://github.com/actions/upload-artifact/blob/043fb46d1a93c77aae656e7c1c64a875d1fc6a0a/action.yml)
and [pinned download action](https://github.com/actions/download-artifact/blob/3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c/action.yml).

## Same-byte publisher implementation

`internal/distribution.Publisher` is a maintainer-only publication component used
by the guarded promotion command below. The caller must prove
transport provenance, exact independent acceptance and publication authority before
invoking it; its expected-identity argument alone does not prove those prerequisites.
It does not nominate candidates, close issues or execute/build candidate code.

The component verifies the complete selected local payload and uses bounded,
authenticated discovery to find a matching release, including drafts. Source/tag,
deterministic ownership body, prerelease state and existing asset identities must
agree. It creates a draft only when absent, reuses verified matching assets, uploads
only missing assets and checks all remote bytes before publishing. Each upload uses
an owned bounded byte buffer whose hash has already been checked, rather than a
mutable file stream. After publication, anonymous inspection/downloads verify the
tag, immutable release, manifest/checksums and every payload again.

Its receipt records identity/source, release ID/URL, asset IDs/digests, the last
verified phase and any uncertain operation. Provider timeouts and failed writes
retain possible remote effects. Retry re-inspects the same identity rather than
replacing assets or publishing twice. Unexpected assets, `starter` uploads, foreign
release bodies, broken/existing wrong-source tags and changed state are preserved
for inspection. There are no DELETE, asset-overwrite or policy-write endpoints.
Discovery is bounded to ten pages of 100 releases; metadata responses are limited
to 128 KiB and requests share a ten-minute context deadline. A limit refusal does not permit
creating another release without resolving the incomplete discovery.

The release credential is scoped to fixed repository release/tag operations and
the upload host. A separate explicit policy-read credential, when supplied, is
used only for the immutable-release settings GET. Without one, the release
credential must itself have that read permission. Metadata/write redirects are
refused; binary redirects to supported asset CDNs receive no credentials. Raw
provider response/error content is not included in failure messages or receipts.

Simulated-provider tests cover exact bytes and pre/post-publication downloads,
matching published retries, uncertain create/upload/publish responses, partial conflicts,
corruption, cancellation, changed local files/links and remote tags/assets, policy
changes, bounded malformed discovery and credential/redirect isolation. Payloads
are inert fixtures, not native candidate acceptance or a successful hosted release.
The read-only verifier, guarded transport-to-publisher command and artifact ledger
integration are implemented. Real hosted verification remains pending. No actual
credential or repository policy is configured by these tests.

## Guarded retained-candidate promotion

`bin/promote-candidate` is explicit maintainer publication tooling, not a memory
operation or a personal-agent capability:

```sh
bin/promote-candidate --receipt transport.json --archive downloaded.zip \
  --identity ACCEPTED_IDENTITY --staging-parent EXISTING_SCRATCH_DIRECTORY
```

This command can publish a release. It requires an explicitly supplied `GH_TOKEN`,
the official repository context, existing release-check configuration, and the
explicit `RELEASE_ISSUES`/`RELEASE_SUMMARY` for issue authority. It refreshes official
transport metadata, verifies the entire archive against the expected identity,
and stages exact bytes in a new private directory. Members are created exclusively
as non-executable regular files; nothing is overwritten or executed. Failed
extraction retains the partial directory and does not return a verified payload.

The command refreshes transport again, invokes both `bin/release-gate
--require-acceptance` and `bin/finalize-release artifact --preflight`, and refreshes
transport once more before calling the publisher. Source, version and identity
for those guards come from verified payload data, not ambient overrides. No flag
skips a guard. Child process groups have cancellation/deadline handling; their
output is discarded rather than placed in receipts. A refusal names the read-only
guard to run directly for diagnostics. The optional policy-read credential is not
passed to those child processes.

The command returns phased JSON and a nonzero exit on failure. Its local receipt
includes the retained staging directory and any publisher receipt, including an
uncertain operation. A partially completed command must not be interpreted as a
published or ledger-complete release. Retrying creates fresh local staging but
reuses only the same matching remote release/assets.

The `Release artifact` workflow now accepts source SHA, accepted candidate identity,
and retained run/artifact IDs. It performs the initial acceptance check, inspects
provenance, downloads the exact raw ZIP through the pinned action, verifies it,
and invokes the guarded promoter. Only successful publication continues to the
finalizer, which independently verifies the public release again. The product
version is derived from the verified manifest, not a separate input. Serialized
runs, trusted control checkout, pinned helpers/actions and all existing acceptance
checks remain in place.

After a promotion attempt, the workflow retains transport, byte-verification and
publication/ledger evidence for 90 days without overwriting previous receipts.
Machine-local staging paths are removed. Missing/truncated process output produces
an explicitly unconfirmed recovery record, not a fabricated successful publication.
Only a successful finalizer sets `ledger-complete`. These fixtures and source-level
workflow checks do not establish successful real Actions promotion or independent
product acceptance; neither has occurred.

## Published verification and artifact finalization

The maintainer helper shares the publisher's anonymous post-publication verifier:

```sh
bin/release-publication selection --version VERSION --identity ACCEPTED_IDENTITY
bin/release-publication verify --version VERSION --identity ACCEPTED_IDENTITY
```

`selection` validates explicit coordinates locally; it does not prove existence,
provenance or acceptance. `verify` inspects the immutable published release,
source tag, exact manifest/checksums, ownership body and every asset's bytes,
then refreshes metadata. It emits success JSON only after complete verification;
a failed check does not emit a partial receipt as success. Neither command
reads GitHub CLI authentication or changes remote/local product state.

`bin/finalize-release artifact` now requires `RELEASE_VERSION` as well as the
existing accepted SHA/identity and explicit issue set. After the existing
issue-specific workflow-authored nomination/acceptance checks, it calls the live
read-only verifier before any label/comment/body/closure mutation. It never creates
an empty calendar release. Existing release comments must match the canonical
product tag/URL and exact artifact. Partial/closed retries reverify the published
product; a local tag or saved publisher receipt cannot replace that check. Accepted
open-issue `--preflight` remains usable before publication; a preflight that relies
on already-released issue evidence additionally verifies the published product.

The release workflow checks out trusted default-branch control tooling, serializes
release runs, and uses pinned Go for the helpers. It promotes the exact retained
candidate and then verifies/finalizes the published accepted product. These wrappers
compile trusted **control tooling**, not the accepted
product. This qualifies the plan's literal no-compiler wording: promotion must never
rebuild or replace accepted product payloads, but verification tooling may be built
from its separately pinned trusted control revision.

`internal/releaseworkflow` runs the actual finalizer script in disposable Git
repositories with fake GitHub/verifier commands and no provider credentials. It
tests closure ordering, unaccepted/unlinked/foreign-status denials, wrong/incomplete
published identities, omitted issues, partial-ledger recovery, closed retries and
preservation of the production deployment/calendar-release path. HTTP tests exercise
the real verifier separately; fake script receipts alone do not prove remote bytes.

## Bootstrap prerequisites

The packaged POSIX bootstrap currently requires curl 8.4+ plus `sha256sum` or
`shasum`, and invokes the release-install journey after checking the platform
binary against official-release checksums. Curl 8.4+ is required because earlier
versions do not enforce the size limit during unknown-length transfers.
See [curl's size-limit contract](https://curl.se/docs/manpage.html#--max-filesize).
The manual alternative is downloading and verifying the platform binary yourself.
Release discovery, planning, explicit plan/apply, the interactive install journey
and selected native handoff are implemented. Actual hosted promotion and final candidate
acceptance remain incomplete; do not present an engineering candidate as a released
installer.

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
