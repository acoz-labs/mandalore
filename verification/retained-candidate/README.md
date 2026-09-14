# Verification-only retained platform probes

This branch extends the successful Linux AMD64 collector from PR #40 to the
remaining Linux ARM64 and macOS AMD64 CLI targets. It is an evidence collector,
not runtime implementation, an intended product merge or a candidate rebuild.

The standard CI job still uses the unchanged `CI_RUNNER=ubuntu-latest` variable.
Two additional explicit public runner variables select only the two test jobs:
`CI_RUNNER_LINUX_ARM64=ubuntu-24.04-arm` and
`CI_RUNNER_MACOS_AMD64=macos-15-intel`. They are ordinary, non-secret test
configuration; no acceptance/release variable or private runner is changed.
GitHub documents both as [standard public runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
This is a documented verification-only extension of the default single-runner
development configuration; main and its runner policy remain unchanged.

Only a same-repository PR named `verification/retained-platforms` runs the extra
jobs, after standard CI passes. Platform cases do not cancel one another. Each
job has a 15-minute deadline and retains separately named synthetic evidence.
Checkout persistence is disabled on the candidate-execution jobs. No release,
acceptance or repository write permission is requested by the workflow.

The trusted transport verifier checks current successful-run provenance and the
complete archive before extraction. The archive hash and all payload checksums
are checked again. Host OS/architecture must match the explicit target; translated
Intel execution on Apple silicon is refused. Only the retained platform binary is
tested; compiling the verifier/source tests does not replace that executable.
Candidate execution runs with an empty credential environment and minimal PATH.

Actual probes cover CLI correction/history, bounded stdio MCP, a refused journal
mutation in read-only mode and matching file hashes, local Git delivery and
fresh-clone recall, owned CLI activation, and explicit synthetic migration with
an unchanged original snapshot and exact historical numeric value. Evidence
contains selected synthetic receipts, not bindings or model transcripts. No
native agent session, global installation or real memory is involved.

Contributor pre-run self-review: fixed source/run/artifact/archive identities;
explicit standard public runner variables without changing the default; expected
native platform checks; cross-platform `shasum`; no credentials passed to the
candidate; local synthetic data/Git endpoints; bounded MCP/job execution; quiet
privacy scan before upload. Wrong-platform invocation must refuse before any
download. Passing results remain contributor verification, not independent
product acceptance or a platform-wide native Codex claim.
