# Retained-candidate retrieval probe

Verification-only branch, not a runtime change or intended product merge.
Contributor self-review: reuse the existing validated synthetic corpus and
20-sample measurement helper; replace only the compiled-CLI benchmark's build
with a required explicit, exact-hash retained candidate. The Go test driver is
compiled, but the measured CLI is never rebuilt. Child environment excludes
provider credentials. Read-only corpus hashes and unrelated cwd are checked
outside timings. No private memory, native agent or account is used.

Measure 100/1000/10000 single-revision records, ten scopes, 20 fresh process
reads each. Fixture creation and version/hash validation are outside timings;
child startup, memory read, encoding, pipe transfer and result validation are
included. Fresh process is not cold disk. Parent allocations are not child
allocations; serialized bytes are not tokens. This does not isolate agent
context cost or replace independent acceptance.

The first mandatory run with no candidate path must refuse to benchmark. Then
run with the exact retained macOS ARM64 executable. Record raw output and any
threshold crossings honestly, without changing the candidate or benchmark limits.
