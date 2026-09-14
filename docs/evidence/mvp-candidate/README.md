# Retained MVP candidate: reference/interface checkpoint

Contributor engineering verification for #10, not independent product acceptance
or a public release. This evidence-only branch does not change or rebuild the
tested candidate. Previous candidate evidence remains separately retained at
[`a2182f6`](https://github.com/acoz-labs/mandalore/tree/a2182f6c291d51b71bf7532105232b51df89ffee/docs/evidence/mvp-candidate);
its native results and original regression failure are not relabeled here.

## Candidate identity

- Source: `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`.
- Manifest SHA-256: `c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
- macOS ARM64 executable: `eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
- Embedded plugin identity: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- [Successful retained build](https://github.com/acoz-labs/mandalore/actions/runs/34840171935), artifact `10346171185`, raw archive SHA-256 `3916b8df9270007ff5800c9bb68a53e516301b1f36065293b3844e9a6406616c`.
- [Successful nomination](https://github.com/acoz-labs/mandalore/actions/runs/34840401278), retained verification receipt artifact `10345433974`.

The raw archive was downloaded once, hashed, verified with refreshed successful-run
provenance and complete payload verification, then extracted into a fresh private
directory. All payload checksums and the candidate's own native version/read-only
inspection passed. The candidate was not rebuilt locally. Retention expires under
the repository's 90-day policy; an Actions artifact is not a published release.
No native connection, credential, real bank or release policy changed in this slice.

## Regression repair and source checks

PR #38 merged the test-only phase-bound cancellation repair designed in PR #37.
Compared with the preceding candidate, the source diff is only
`internal/api/sync_test.go` and `docs/development.md`; CLI/plugin runtime behavior
is unchanged, but the new source stamp gives this artifact its own identity.

Thirty focused race-enabled cases passed. Restoring the old one-second limit in
a disposable overlay made the slow-start case fail before fetch with an honest
checkpoint-phase cancellation receipt. Final-head full pinned Go 1.26.4 host CI
passed after an actual unavailable-Docker attempt; [PR CI](https://github.com/acoz-labs/mandalore/actions/runs/34840015529),
[merged-source CI](https://github.com/acoz-labs/mandalore/actions/runs/34840157038),
main audit and candidate-build validation passed. The earlier opaque failure's
exact phase is still unproven; the controlled reproduction demonstrates the bad
timing assumption. [Reconciliation and contributor self-review](https://github.com/acoz-labs/mandalore/pull/38#issuecomment-5663424929)
retain the distinction and all temporary plan files were removed before readiness.

## Actual native CLI reference checks

The retained macOS ARM64 executable ran in the designated test pane from an empty,
unrelated project directory. It created a fresh synthetic signet/binding with
explicit device label `Test Mac A` and actor `Example User`, not an inferred host
inventory. An existing synthetic procedure allowed autonomous routine local work.

The selected historical [source](process.md) contains an obsolete approval rule,
a useful verification lesson, a quoted cue and an inert command fragment.

1. Preview left all bank file hashes unchanged. Explicit registration/connect
   returned a complete receipt and inspection reported available.
2. Search/read returned attributed, explicitly unreviewed historical evidence,
   not current guidance. The complete excerpt's file hash matched the source.
   Reading and current-memory recall left all bank files unchanged.
3. The test driver deliberately replaced the synthetic source bytes. Inspection
   reported changed, search returned `foundling.changed`, and the changed source
   bytes were preserved. After restoring them, the driver temporarily moved the
   source directory: inspection reported unavailable and search returned
   `foundling.unavailable`, without recreating the missing path. After restoration,
   every bank file still matched its pre-read hash. These were explicit fixture
   mutations, not toolkit source writes or automatic repairs.
4. Read-only promotion returned `operation.read_only` with exit 2 and no bank
   changes. The first test driver had incorrectly expected exit 1 and stopped;
   the documented expectation was corrected, state was inspected, and the
   read-only operation was repeated successfully before any promotion.
5. Explicit writable promotion adapted the useful lesson into the same procedure
   record with the exact predecessor, preserving current autonomy. The new
   revision retained current device/actor/CLI authorship and generated a verified
   citation with source identity, registration, pin, relative locator and file
   digest. Unknown original author/date remain absent. No journal or Git setup
   occurred implicitly. Final recall selected only the new effective revision;
   history preserved both. Source bytes and the empty project directory remained
   unchanged; structure inspection was healthy.

This tests caller-authored selective promotion through the public interface, not
whether a model will independently choose an appropriate adaptation or resist
arbitrary hostile reference instructions. Native-agent consultation remains pending
against this new artifact.

## Actual MCP stdio checks and measurements

The same retained executable ran a read-only MCP subprocess from the unrelated
directory. The bounded client initialized protocol `2025-03-26`, listed the 16
memory/reference tools, and verified administration/install operations were not
advertised. Search's structured result exactly equaled the CLI result. Promotion
returned `isError` with `operation.read_only`; the journal remained empty. The
process exited successfully on input closure. All bank files, source bytes and
cwd contents remained unchanged through shutdown.

[Measured results](mcp-summary.json) contain 5 actual requests. Input schemas total
5,966 compact JSON bytes, output schemas 32,678 bytes and descriptions 2,488 UTF-8
bytes. The full tools/list response line was 44,087 bytes. These are serialized
byte counts, not model tokens or isolated native-session context cost. Individual
timings are one local sample, not a benchmark or cross-platform claim.

## Retained evidence and limits

`candidate.json` is the actual verification receipt. `history.json`, `recall.json`,
`journal.json` and `incorporation-source.json` are synthetic toolkit/store outputs.
`reference-search.json` preserves the source/authority distinction and
`mcp-summary.json` the compiled-process results. `SHA256SUMS` covers every evidence
file other than itself. No credentials, local bindings, workstation paths or raw
native transcripts are published.

Keep #10 open. Fresh native learning/cue/journal/compaction/interrupt/hook/foundling
scenarios, recovery/offline/concurrent/ambiguous-write cases, physical-machine
coverage and the full rendered install/update/repair journey still need their
exact-candidate evidence. Independent acceptance and publication prerequisites
remain separate. Cross-builds and a local bare remote are not other-platform or
second-physical-machine acceptance; no live adoption or release is claimed.
