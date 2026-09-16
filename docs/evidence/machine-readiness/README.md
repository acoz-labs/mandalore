# Machine-readiness implementation evidence

Issue #85 / draft implementation PR #87. This is incremental engineering
evidence, not complete feature verification, native rendered evidence, candidate
acceptance or release authority. Fixtures contain synthetic data only.

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
public `connection_assess` API tests remain intentionally red until the shared
assessor is implemented; the PR must remain draft.

## Limits

- Reads can affect OS access-time/cache bookkeeping; no product state is written.
- Cancellation is checked between reads; it is not a hard kernel/filesystem
  deadline or a sandbox against a hostile local user.
- Full fixture inventories, no-network transport probes, binding/receipt
  projections, shared CLI/menu behavior and exact-head rendered evidence remain
  subsequent implementation work.
- Pi's historical native evidence is engineering-only. This work does not
  promote it to acceptance or modify the immutable 1.0.0 artifacts.
