# Retrieval engineering measurements

Status: baseline, same-fixture engine comparison and native guidance/learning
evaluation complete under #9 / PR #34. This is contributor evidence, not independent
immutable-candidate acceptance or an application performance SLA.

## Source and method

Baseline runtime and benchmark implementation:
`a1fba490f33890bd0fe69027012d1f71659809c1`.
Go 1.26.4, Darwin arm64, Apple M1 Pro, GOMAXPROCS 10; operations are serial.
Host Git reports 2.50.1 (Apple Git-155); these local synthetic benchmarks do not
perform remote synchronization.
Measurements ran sequentially in the designated Herdr test pane. No cache purge,
parallel benchmark suite, private bank or native credential was used.

The full memory run used `-benchtime=20x -count=1` without `-v`; all 36 cases
passed in 596.306 seconds including fixture setup. A supplemental verbose run
with `-benchtime=1x` retains fixture metadata; its timings are smoke observations,
not the 20-sample distribution. Hook, foundling and compiled CLI runs used `-v`
and 20 iterations and all passed. Commands and units are in
[the retrieval guide](../../retrieval.md).

The supplemental/profile/adapter runs were taken before any runtime change, with
only pending punctuation regression tests and the documentation's verbose flag
correction in the worktree. Those tests were excluded by `-run '^$'`; benchmark
code, fixture construction, runtime and plugin content still matched the baseline
head. The separate red test demonstrated terminal-period misses before the fix.
Do not label these development binaries immutable release candidates.

Warm means repeated validated filesystem reads, not persistent cached knowledge.
Fresh process still benefits from the OS page cache. Fixture setup/validation and
CLI build are excluded from benchmark accounting. CLI allocation figures are for
the parent runner, not the child. Requested budgets are ceilings, not consumed
tokens. Profile percentages are sampled, cumulative nodes overlap and percentages
use the whole profile denominator even after stack filtering.

## Selected baseline results

All p95 values below use 20 timed operation samples, nearest rank. The complete
matrix and allocation figures are retained in the raw outputs.

| Operation / corpus | p95 | Returned bytes per operation |
| --- | ---: | ---: |
| Warm narrow recall, 100 records / depth 1 | 22.65 ms | 826 |
| Warm narrow recall, 1000 records / depth 1 | 199.7 ms | 826 |
| Warm narrow recall, 10000 records / depth 1 | 2116 ms | 826 |
| Warm narrow recall, 1000 records / depth 2 | 349.6 ms | 826 |
| Warm narrow recall, 1000 records / depth 10 | 1573 ms | 826 |
| Prompt hook, 1000 records / depth 1 | 390.2 ms | 3852 |
| Prompt hook, 10000 records / depth 1 | 4208 ms | 3858 |
| Compiled CLI recall, 10000 records / depth 1 | 2241 ms | 869 |
| Verified reference read, 1000 files | 52.60 ms | 1241 |

The 1000-record depth-1 corpus has 2013 files and 1211038 bytes; the
10000-record corpus has 20013 files and 12108238 bytes. A matching record in the
selected project is one of ten explicit scopes. Source objects and full history
are still validated, even when the returned answer is tiny or empty. The
10000-record narrow operation allocated about 406 million bytes per call in Go
benchmark accounting. Bounded output alone is not bounded internal work.

The focused baseline profile attributes 1.42 seconds of sampled cumulative time
to `deviceExists` within 3.70 seconds under `Service.Recall`. Code inspection
shows the same device is checked for every revision and each referenced source:
2000 checks per operation in the 1000-record, single-device fixture. This supports
investigating per-validation reuse, not a persistent cache or weaker source checks.
Broad/empty/no-match timings are similar; an index is not preselected from them.

## Bounded correction comparison

After source: `47e4caf6aa8bca39f970608ead49fec2004a7dff`, clean throughout
collection. Same host, Go version, fixtures, operation names and 20-sample method
as the baseline; only the documented period fix and per-validation device reuse
changed runtime behavior. All 50 paired cases passed (36 memory, 3 hooks, 8
foundling, 3 compiled CLI). Every pair returned the same average serialized byte
count. That equality is a payload observation, not a substitute for the semantic
and freshness regression tests. Source, graph and conflict checks remain enabled.

| Operation / corpus | Baseline p95 | After p95 | Returned bytes |
| --- | ---: | ---: | ---: |
| Warm narrow recall, 100 records / depth 1 | 22.65 ms | 16.43 ms | 826 |
| Warm narrow recall, 1000 records / depth 1 | 199.7 ms | 143.7 ms | 826 |
| Warm narrow recall, 10000 records / depth 1 | 2116 ms | 1581 ms | 826 |
| Warm narrow recall, 1000 records / depth 2 | 349.6 ms | 251.9 ms | 826 |
| Warm narrow recall, 1000 records / depth 10 | 1573 ms | 1158 ms | 826 |
| Warm narrow recall, 1000 records / depth 2 with conflicts | 357.1 ms | 245.4 ms | 1289 |
| Prompt hook, 1000 records / depth 1 | 390.2 ms | 283.9 ms | 3852 |
| Prompt hook, 10000 records / depth 1 | 4208 ms | 3122 ms | 3858 |
| Compiled CLI recall, 1000 records / depth 1 | 218.2 ms | 160.2 ms | 869 |
| Compiled CLI recall, 10000 records / depth 1 | 2241 ms | 1776 ms | 869 |
| Verified reference read, 1000 files | 52.60 ms | 51.26 ms | 1241 |

The 1000-record depth-1 warm/narrow operation's mean moved from 195560108 to
133744769 ns/op, and allocated bytes from 40686761 to 32301993 B/op. At 10000
records it still allocates 322096568 B/op. The foundling read mean was slightly
slower (48.84 to 49.21 ms) despite its lower p95: that separate verifier was not
optimized, and small differences are not evidence of a new fast path. These are
single sequential host comparisons, not statistically established speed guarantees.

The depth-1 1000-record case is below the provisional 250 ms warm/one-second CLI
investigation triggers. The depth-2 warm case slightly exceeds 250 ms, and depth-10
clearly exceeds it. History growth therefore remains a measured follow-up, not
solved scale. The baseline profile and remaining per-operation full graph/source
scan motivate scoping further validation/I/O investigation before selecting an
index. A lexical-only index cannot by itself remove the current integrity scan;
freshness, corrupt-data refusal and exact scope/conflict semantics cannot be traded
away. No new index, persistent cache or validation shortcut is included here.

All after commands used `-v -benchtime=20x -count=1 -timeout=20m`. Total command
durations, including setup, were 433.831 s memory, 88.999 s hooks, 6.096 s foundlings
and 56.128 s compiled CLI. The tested CLI SHA-256 was
`9cdba5f96e664aeb25c62210211bbf66a6c5a692b77a712d9049d6f74b1509d1`;
an immediately subsequent clean build for the native test had identical bytes.
The baseline compiled CLI digest was not retained; its exact implementation source
and commands are recorded above. Do not treat that baseline as an immutable
distribution candidate. No timing below describes an OS-cold filesystem.

## Retained evidence

| Artifact | SHA-256 |
| --- | --- |
| [Memory baseline](memory-baseline.txt) | `a35e9c71adc744c610eef2ac8e304a17cb16a46c8e3c733e9107bfe20124d740` |
| [Verbose memory metadata](memory-metadata.txt) | `e23e759d68b11a085ea80156db4623a6958071ba7d2135a5fce6ec8680f31208` |
| [Hook baseline](hook-baseline.txt) | `cc8ee5314ce79fa0861a723eebd3e072529a724182f9a066777f287dc07a0d8c` |
| [Foundling baseline](foundling-baseline.txt) | `284105ee2fee843b178974e4cd75bbd0bca426d88004da3b02fa6593b0930d48` |
| [Compiled CLI baseline](cli-baseline.txt) | `2b61208ec3766031c02ad725d4b6aea91e36fd789fced0a8d09575532d38e28d` |
| [Profile benchmark](profile-baseline.txt) | `9a5749657d9041f9d5e8056335a24e4d4027ae35b4509166b2b393d6ab84eee9` |
| [CPU profile summary](cpu-top.txt) | `8156cc8b972c320c1290287a4500b668f9b371a15ccb448f8a7a69ea411081fa` |
| [Allocation profile summary](allocation-top.txt) | `239524e4eec10a16c7e3de579e22cf79a9f958acd894293cff1fee1d959bb193` |
| [Memory after](memory-after.txt) | `ec0c985883d4f4269bd36e6637e68c35a202fd0149b5af9b47e3345c748ff5a1` |
| [Hook after](hook-after.txt) | `7b93cde1663d5dd40a1710b0ebd79e848888de47a80ea6bec866a261bbb6b8fb` |
| [Foundling after](foundling-after.txt) | `218a50c34b2dcba7d983e86e326eab6fb1d7fd911ca9afb6168157351dd769db` |
| [Compiled CLI after](cli-after.txt) | `2cd14d2325e201b9b3fdc1aed100ccf1a242f8d4652c0146dec6f00a7f0b4d20` |

The profile command selects only the 1000-record depth-1 warm/narrow benchmark,
with `-cpuprofile` and `-memprofile`, then uses `go tool pprof -top -cum
-nodecount=30 -focus 'Service.*Recall'` (plus `-sample_index=alloc_space` for
allocations). This excludes setup call stacks from the displayed focus, not from
the collected profile's denominator. The local profiling test binary SHA-256 is
`e8ee80086b6d1425715f157c3a7e95ad8398efbc32c334ce394ac556b8f1839d`.
Raw profiling binaries are not published because debug metadata can embed local
paths; sanitized text summaries suffice to review this engineering observation.

The separate [native call ledger](native.md) records the behavioral comparison
and correction of the earlier excerpt-completeness assumption.

No native model behavior, token-cost attribution, OS cold-cache performance,
other-host timing or real historical-memory adoption is established by these
benchmarks. Native guidance validation and exact-candidate acceptance remain
separate work, as do any larger index/freshness design decisions.
