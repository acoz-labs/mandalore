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
| Cwd, native inheritance, separate banks, two machines, origin, external reads | Native inherited-resource/cwd observations; separate synthetic banks; [two physical Macs](physical.md); [native peer continuity and sync recovery](native-peer/README.md); earlier same-live-MCP local refresh | Native peer tests cover two macOS ARM64 hosts and Codex 0.153.4/0.154.0, not all native platforms |
| Compaction, interruption, failed hooks, ambiguous writes, corruption, offline/concurrent sync | Actual native compaction/warnings/interrupted work plus compiled conflict/offline/corrupt/uncertain-push fixtures | Escape did not prove child termination; controlled push-response loss was local, not actual packet loss; no blanket failure coverage |
| Supported OS/architecture execution | Native retained CLI/MCP execution on all four targets; matrix below | Native Codex conversations and rendered terminal journeys are not verified on every platform |
| Context/retrieval cost without private transcripts | Measured MCP schema/response bytes and native operation summaries; [retained-binary 100/1000/10000-record retrieval](retrieval/README.md), plus separate source-level #9 evaluations | Aggregate native counters do not isolate Mandalore token cost; fresh-process single-revision measurements do not cover every retrieval mode |
| Exact-digest install/update/repair journey | [21 fresh rendered recordings](ui/README.md), [distinct retained switching/refusals](ui/switch-and-refusals.md) | Contributor judgment only; no live public release/bootstrap yet |
| Historical references and provenance | Actual CLI/MCP missing/changed-source handling, native selective promotion, source preservation | No arbitrary hostile-source immunity or bulk migration claim |
| Explicit legacy compatibility (#8 dependency) | [Fresh retained-artifact migration](migration/README.md); [actual single-plugin inventory and held-lock refusals](migration/native/README.md); [native legacy-name coexistence refusal and cleanup](migration/coexistence/README.md) | Synthetic memory-only conversion and inert legacy-name fixture; no actual predecessor writer execution, real-memory adoption or remote writer-quiescence proof |

## Platform matrix — do not collapse these columns

| Platform | Retained candidate CLI/MCP execution | Native Codex conversation | Other engineering evidence |
| --- | --- | --- | --- |
| macOS ARM64 | Yes; full local checks and continuity on two physical machines | Yes, 0.153.4 on A and 0.154.0 on B, including native peer recall/learning/sync recovery | Actual installer/repair UI and migration tests; version-specific limits recorded |
| macOS AMD64 | Yes; [native retained CLI/MCP, sync, install and migration](platforms/README.md) | Not verified | Actual Intel macOS hosted runner, translated execution refused |
| Linux AMD64 | Yes; [retained native CLI/MCP, sync, install and migration](linux/README.md) | Not verified | Exact retained binary on the existing Ubuntu x86_64 runner; source-level CI remains separate |
| Linux ARM64 | Yes; [native retained CLI/MCP, sync, install and migration](platforms/README.md) | Not verified | Actual ARM64 Ubuntu hosted runner |

The distribution declares all four targets; all four retained CLI binaries now
have native execution evidence. No translation/emulation or cross-build is
relabeled as native architecture acceptance. The existing local container VM was
not started. The remaining two native routes used explicit standard public runner
variables on a verification-only branch, without changing the default CI runner.

## Authority and next work

Foundation #1 completed separately through docs PR #39 with actual PR/main CI and
main audit. Issues #2–#9/#12 remain in acceptance, not unimplemented merely because
their issues are open. #10 remains in progress.

The bounded native coexistence check is now complete. The limitations above
remain limitations, not requests to expand into arbitrary additional test cases.
The next required step is an independent exact-candidate acceptance decision,
including judgment on the documented sync wording, retrieval scale and platform
coverage. Published
download/bootstrap cannot pass before an actual authorized release exists; it must
be a post-publication verification gate, not a fabricated pre-release result.

Independent acceptance is also outstanding. Repository rules require an acceptor
other than the implementation author. Contributor self-review, additional test
processes and a second machine do not supply that person or their approval. The
owner has been asked who will perform and record that acceptance. No actor or
release configuration has been invented or changed, and no publication is claimed.

## Issues 1–10 closeout map

The current GitHub issue bodies were re-read on 2026-09-14. This map preserves
their full outcome scope; it does not check acceptance boxes on the owner's behalf.

| Issue | Implementation and engineering evidence | Required next disposition |
| --- | --- | --- |
| #1 Foundation | Merged PR #39; exact predecessor ledger, project fields/views/dependencies and successful hosted checks | Closed; no runtime acceptance implied |
| #2 Engine | PR #20; [extraction/dependency/license inventory](../../memory-extraction.md), engine regression/race tests and retained native CLI/MCP matrix | Independent candidate acceptance; do not reintroduce assistant-platform code |
| #3 Format | PRs #20/#22; [format/schema/provenance contract](../../signet-format.md), invalid-input/graph/binding tests, retained correction/history/migration receipts | Independent acceptance; unknown formats remain explicit refusals |
| #4 Interface | PRs #22/#24; [typed CLI/MCP contract](../../interface.md), strict/error/no-save/cwd/cancellation tests and actual retained stdio calls | Independent acceptance of the shared local interface, not remote MCP hosting |
| #5 Synchronization | PR #24; [sync phases/state/limits](../../synchronization.md), real-Git regression suite, retained conflict/offline/uncertain-push and native two-Mac recovery | Independent acceptance; remote delivery is not every peer's freshness |
| #6 Codex | PR #26; actual native learning/cue/no-save/inheritance/cwd/compaction/hook-warning/external-refresh and coexistence checks | Independent native acceptance; evaluate observed broad sync wording, do not infer universal model behavior |
| #7 Setup UX | PR #28; [setup contract](../../setup.md), 21 exact-candidate rendered journeys and second-machine connection | Independent owner journey; live public download/update remains post-release verification |
| #8 Migration | PR #30; supported-schema census, exact preserved source/output history and refusals, native lock/inventory/coexistence evidence | Independent synthetic compatibility acceptance; real memory migration still needs separate explicit authority |
| #9 Retrieval | PR #34; [quality/corpus/cost methods](../../retrieval.md), source-level evaluations/profiles and retained binary measurements | Independent judgment on documented lexical misses and growth cost; no cold-disk or isolated model-token claim |
| #10 MVP proof | Retained artifact matrix and checkpoints linked above cover contributor verification of the requested scenarios | Independent acceptor must repeat/evaluate required journeys and record issue-specific decisions against this candidate |

Current authoritative checks found the retained artifact unexpired, no published
GitHub releases, no product-acceptance workflow runs, no candidate acceptance
statuses, and no `ACCEPTANCE_ACTORS` repository variable. The workflow and
`bin/record-product-acceptance` require an authorized actor distinct from each
linked implementation PR author. Another process, model, account invented for
this run, or machine does not satisfy that independent review requirement.

No acceptance workflow was dispatched just to produce an expected authority
failure. The owner must identify the independent reviewer and authorize the
appropriate configuration; release authorization remains a separate step under
#11. This evidence branch changes documentation only, not those policies.
