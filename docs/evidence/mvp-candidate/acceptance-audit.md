# Candidate acceptance audit — 2026-09-14

This is a requirement audit, not an acceptance vote. Candidate source is
`5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
The actual retained Actions artifact `10346171185` was re-inspected: not expired,
retention ending 2026-12-13. The later foundation-only merge `08129ed` changes two
documentation files, not this retained candidate or its runtime acceptance set.

## Current evidence against #10

| Requirement | Actual evidence | Remaining boundary |
| --- | --- | --- |
| Ordinary learning, cue, journal, fresh recall, supersession, no-save | [Native checkpoint](native.md), [live reference checkpoint](live-recovery.md) | Contributor-operated; not independent acceptance |
| Cwd, native inheritance, separate banks, two machines, origin, external reads | Native inherited-resource/cwd observations; separate synthetic banks; [two physical Macs](physical.md); same live MCP process external refresh | Second machine used compiled CLI, not a fresh native agent session |
| Compaction, interruption, failed hooks, ambiguous writes, corruption, offline/concurrent sync | Actual native compaction/warnings/interrupted work plus compiled conflict/offline/corrupt/uncertain-push fixtures | Escape did not prove child termination; controlled push-response loss was local, not actual packet loss; no blanket failure coverage |
| Supported OS/architecture execution | Matrix below | Cross-builds and source-suite tests do not prove execution of all retained platform binaries |
| Context/retrieval cost without private transcripts | Measured MCP schema/response bytes and native operation summaries; [retained-binary 100/1000/10000-record retrieval](retrieval/README.md), plus separate source-level #9 evaluations | Aggregate native counters do not isolate Mandalore token cost; fresh-process single-revision measurements do not cover every retrieval mode |
| Exact-digest install/update/repair journey | [21 fresh rendered recordings](ui/README.md), [distinct retained switching/refusals](ui/switch-and-refusals.md) | Contributor judgment only; no live public release/bootstrap yet |
| Historical references and provenance | Actual CLI/MCP missing/changed-source handling, native selective promotion, source preservation | No arbitrary hostile-source immunity or bulk migration claim |
| Explicit legacy compatibility (#8 dependency) | [Fresh retained-artifact migration](migration/README.md) | Synthetic memory-only conversion; no real-memory adoption or duplicate-native-writer inventory in this checkpoint |

## Platform matrix — do not collapse these columns

| Platform | Retained candidate CLI/MCP execution | Native Codex conversation | Other engineering evidence |
| --- | --- | --- | --- |
| macOS ARM64 | Yes; full local checks and CLI continuity on two physical machines | Yes, Codex 0.153.4 on the first machine | Actual installer/repair UI and migration tests |
| macOS AMD64 | Not yet verified | Not verified | Cross-built payload only |
| Linux AMD64 | Yes; [retained native CLI/MCP, sync, install and migration](linux/README.md) | Not verified | Exact retained binary on the existing Ubuntu x86_64 runner; source-level CI remains separate |
| Linux ARM64 | Not yet verified | Not verified | Cross-built payload only |

The distribution declares all four targets. This audit does not silently redefine
support as macOS ARM64 alone. No translation/emulation or cross-build is relabeled
as native architecture acceptance. The existing local container VM was observed
stopped; it was not started because its unrelated workloads are outside this test.
An isolated execution route is needed for the missing platform checks.

## Authority and next work

Foundation #1 completed separately through docs PR #39 with actual PR/main CI and
main audit. Issues #2–#9/#12 remain in acceptance, not unimplemented merely because
their issues are open. #10 remains in progress.

Next engineering work is retained-binary execution on the missing platforms and
the remaining native/compatibility verification boundaries above. Published
download/bootstrap cannot pass before an actual authorized release exists; it must
be a post-publication verification gate, not a fabricated pre-release result.

Independent acceptance is also outstanding. Repository rules require an acceptor
other than the implementation author. Contributor self-review, additional test
processes and a second machine do not supply that person or their approval. The
owner has been asked who will perform and record that acceptance. No actor or
release configuration has been invented or changed, and no publication is claimed.
