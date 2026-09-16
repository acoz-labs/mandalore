# Machine-readiness implementation evidence

Issue #85 / draft implementation PR #87. This is incremental engineering
evidence, not complete feature verification, native rendered evidence, candidate
acceptance or release authority. Fixtures contain synthetic data only.

The [source-bound menu checkpoint](menu/README.md) retains actual terminal
recordings, the narrow-layout finding and its corrected retest. It is partial
rendered engineering evidence, not the final-head or candidate matrix.
The [configured/adverse-state checkpoint](menu/configured-d2ce027/README.md)
adds both native handoffs, stale identity, malformed selection and a known
incompatible retained artifact. [Reconciliation](reconciliation.md) tracks the
requirement map, durable documentation and remaining final-head/release gates.

## Evidence-matcher sensitivity

Baseline: `52839e9cf31dd18bf6cc1d5a186fe8976a72a24c`.

Temporarily changed the missing recorded-identity branch in
`internal/readiness/evidence.go` from appending
`record-<identity>-missing` to returning without a reason. This deliberately
makes an incomplete historical record act as a wildcard.

Ran:

```sh
mise exec -- go test -count=1 ./internal/readiness -run '^TestEvidenceMatcherRequiresEveryScenarioIdentity$'
```

Observed exit 1, with the specific assertion:

```text
incomplete recorded evidence became a wildcard {synthetic verified []}
```

Restored the original implementation without weakening the test. The focused
readiness package then passed `go test -race -count=1`; the restored source was
identical to the baseline. This establishes sensitivity to that specific false
positive, not every possible matcher defect.

## Bounded file fingerprints

Added tests before implementing `hashFile`; the first run failed compilation
because the function and result type were absent. Implementation streams the
regular file through SHA-256, using the same bounded no-follow reader as metadata.
It returns an on-disk fingerprint, not loaded-image or interpreter verification.

Tests cover exact bytes and size, executable-mode observation, a marker-writing
script that must not run, no changed fixture state, explicit launcher symlink
resolution, rejection of redirected managed paths, directories/FIFOs/invalid
paths/oversized files, cancellation, and concurrent growth/replacement/mode
changes. No selected program, Git command or provider is executed by hashing.

A second red/green test exposed a real intermediate defect: changing a launcher
symlink during the read returned the old target's digest. The initial
`TestHashFileRejectsRetargetedLauncher` failed with the assertion
`retargeted launcher returned a stale identity`. Retaining and rechecking the
original selected path fixes that case. The test remains in the suite.

After this fix, `go test -race -count=3 ./internal/readiness`,
`go vet ./internal/readiness` and `git diff --check` passed using the pinned host
toolchain. These are component checks, not full CI or native acceptance. The
At that checkpoint, the public `connection_assess` API tests remained
intentionally red until the shared assessor was implemented.

## Binding metadata projection

Added a pure shared validator for signet/device/authorship metadata and a bounded
readiness projection. Existing store opening reuses the same extracted field
rules. New tests first failed because these functions were absent, then passed
with the implementation. The projection reads only the selected binding (16 KiB),
its manifest and enrolled-device metadata (4 MiB each). A reader allowlist asserts
those exact three reads and budgets. Remembered-content locations are deliberately
unusable in the fixture; no store/content validation is invoked.

The tests cover malformed/duplicate JSON, unsupported format, identity mismatch,
invalid roots and device-path traversal, legacy markers, binding-inside-bank,
redirected metadata, cancellation before and during inspection, missing targets,
bounded error classification and private-name/error canaries. Successful results
mean only that these selected identity fields agree, not that the memory graph or
native connection works.

An additional failing-first test found that a dangling managed-file symlink was
reported as absent setup. The reader now rejects the final redirected component
before resolving it. `TestInspectBindingDanglingRedirectIsNotMissingSetup`
retains that regression check.

After the fix, the readiness, memory and binding packages all passed
`go test -race -count=1` and `go vet`. The readiness package also cross-built for
Darwin/Linux on amd64/arm64. These package builds do not establish native behavior
on those platforms. API, retained-connection and rendered validation were still
pending at that checkpoint.

## Retained selection and shared assessment

Extracted existing Codex/Pi ownership decoders into pure byte validators and
added a narrow projection. Synthetic nonexistent-path tests establish that
projection needs no profile, package, runtime or bank to exist. Foreign
harness/root/state/profile/native/binding selections and malformed/oversized
receipts return no followable runtime path. Existing installer race tests pass.

The readiness observer now reads the selected receipt and fingerprints only its
owned runtime after matching the operator's native/binding selection. Tests
cover changed runtime bytes, altered runtime paths, stale binding/native bytes,
foreign selection and an unselected root. Successful static verification names
only runtime/selection consistency; missing package files do not become proof of
package or active native registration health.

Implemented shared `connection_assess`, human `connection assess`, explicit
harness selection, bounded non-executing defaults, independent report dimensions,
trusted internal process stamps and fixed-text optional follow-up guidance. The
previously red API tests now pass. CLI/shared-call reports match for both harnesses.
Invalid flags, untrusted JSON identity fields, duplicate JSON and cancellation
are rejected with the expected protocol/exit semantics. The memory MCP surface
remains separate: this new operation is CLI-only and needs no bound service.

A compiled CLI test places marker-writing Codex, Pi, Git and Node executables on
PATH. Both harness assessments return successful JSON without triggering them or
creating the selected state/profile/binding. Invalid proxy endpoints are present;
this does not by itself prove every possible network path is blocked. Stronger
transport/network instrumentation and complete fixture inventories remain work.

The first full `mise exec -- bin/ci` run passed: Pi adapter tests, all Go race
tests, vet, public-content checks and four target builds. Docker's daemon was
unavailable, so the documented pinned host-toolchain fallback was used.

Subsequent self-review added failing-first regressions for three intermediate
defects: silently skipping a relative PATH entry, following a retained runtime
when the selected native file lost executable permission, and recommending native
verification before reviewing a missing profile. Fixes preserve unknown PATH
selection, stop before the retained read and suggest connection review. A separate
classification test distinguishes a known matching artifact target, a known
incompatible target and unknown artifact architecture. Current-head validation
is recorded in the implementation PR; earlier passing CI is not a substitute.

## Assessment menu and boundary follow-through

Added the Armorer assessment/inspection submenu, shared default selection with
single-field editing, compact independent component results, explicit details and
prompt views, and a default-No native handoff. Both the direct inspection entry
and report handoff disclose the selected program/profile and possible native
log/cache effects. A failed native check leaves the earlier assessment snapshot
unchanged and does not retry or repair it.

Plain-menu tests cover missing setup without writes, explicit-only prompt display,
single-field edits and clearing the retained root, correcting invalid selection
without leaving the journey, Back, both harnesses' native disclosure, default-No,
incomplete affirmative input, explicit dispatch and native failure without retry.
A failing-first cancellation test found that an already-cancelled report rendered
a completed summary before reading the next action. The report now checks context
before rendering; the regression requires no output after pre-cancellation.

A development preview in the dedicated terminal test pane exercised the actual
80-column, 32-row inner PTY (the host viewport was larger): Vim navigation to the
Armorer, missing Codex setup, optional prompt display, arrow navigation to the
native disclosure, its default-No response, Back and Exit. The selected missing
state/profile/binding remained absent. This was an uncommitted development build,
not final-head or candidate acceptance evidence. It exposed a harness-specific
wording defect: the Codex report listed an untested Node requirement. A new test
failed for Codex, then passed after moving that requirement into the Pi branch.
Exact-head recordings and the full rendered matrix remain pending.

The compiled boundary test now inventories every fixture path before and after
each harness assessment, including bytes, modes, modification times and empty
directories. Marker-writing native/Git/Node programs remain unexecuted. A local
instrumented proxy/provider endpoint counts TCP connections, including failed
requests; neither assessment contacts it. This observes configured endpoints,
not arbitrary direct sockets or system-wide network activity. The child receives
explicit selected paths and does not override the user's home directory.

Additional Pi retained tests allowlist the exact receipt/runtime reads and their
budgets, work without a staged package tree, and reject changed profile, binding,
native bytes and malformed receipt before following a runtime. These supplement
the shared ownership decoder tests; they do not claim installed-package health.

Both packaged Armorer skills now distinguish assessment from executing native
inspection and route detailed interpretation to their administration references.
The two skill validators and Codex plugin validator pass in an isolated PyYAML
6.0.2 environment. No marketplace, live plugin, authentication or personal
connection is changed by this source update. Focused race tests pass; full and
hosted validation for the committed slice is recorded in PR #87.

## Committed-build narrow-layout finding

The first committed menu build, source
`b757751886fc3e19a0bb4edeab6b8ead98d6fb29`, passed local full CI and
[hosted CI](https://github.com/acoz-labs/mandalore/actions/runs/35053701426).
Its 32-column no-color/plain preview exposed status words split across lines
in the compact component rows (`pr` / `esent`). A failing-first regression
reproduced that output at 31 usable columns. The report now stacks Support,
Setup and Evidence fields below 60 usable columns, retaining compact rows at
normal widths. The regression verifies intact status words in both layouts.
Rendered retesting and final-head evidence binding remain required.

The plain-mode EOF fixture's initial exact `stty -g` comparison also flagged a
terminal difference. Investigation found only Darwin's `PENDIN` bit changed
(`0x20000000`, retype-pending-input state); canonical input, echo, signals and
all control characters were unchanged. The fixture comparison now excludes
only that transient bit and discloses the exclusion. This is a test-harness
correction, not a product terminal-mode change.

## Limits

The assessment pipeline also has a replacement regression: a private
`os.Executable` path seam reports a fixed fixture path, the file at that path is
renamed and replaced, and a second assessment must measure the new disk bytes
while keeping the supplied process stamps and embedded package unchanged. Unknown
replacement bytes cannot gain verified evidence from source/version labels.
This complements the exact catalog matcher tests. It models path replacement,
not native loader behavior or attestation of a loaded process image. No public
input can override that seam, process platform or bundled catalog.

- Reads can affect OS access-time/cache bookkeeping; no product state is written.
- Cancellation is checked between reads; it is not a hard kernel/filesystem
  deadline or a sandbox against a hostile local user.
- Broader adversarial coverage and exact-head native rendered evidence remain
  subsequent implementation work. The development preview and configured-network
  instrumentation above have explicit limits. The overall feature/PR remains draft.
- Pi's historical native evidence is engineering-only. This work does not
  promote it to acceptance or modify the immutable 1.0.0 artifacts.
