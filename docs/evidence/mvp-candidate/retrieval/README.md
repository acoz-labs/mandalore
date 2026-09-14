# Exact retained-candidate retrieval measurements

Contributor engineering measurement on macOS ARM64, not independent acceptance,
a universal latency promise or isolated model-token accounting.

Candidate source `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`, executable
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
The binary is the retained artifact, not a locally rebuilt substitute.

## Method and driver

The verification-only [driver head `f2bffad829c7e184fbe49cc6908e044cf6174a6c`](https://github.com/acoz-labs/mandalore/tree/f2bffad829c7e184fbe49cc6908e044cf6174a6c)
changes only the existing compiled-CLI benchmark and adds its test rationale.
It reuses the validated synthetic corpus and nearest-rank measurement helper,
requires an explicit candidate path, and checks exact executable bytes plus
source/OS/architecture before any measurements. No product merge is intended.

Container validation was attempted; Docker was unavailable. The documented
Go 1.26.4 host fallback passed the full source suite. The driver with no selected
binary [refused as expected](missing-driver.txt), rather than building or locating
another CLI. The positive run was launched and observed in the designated pane:

```sh
MANDALORE_EVAL_BIN=/absolute/path/to/retained/mandalore_1.0.0_darwin_arm64 \
  mise exec go@1.26.4 -- go test -v ./cmd/mandalore -run '^$' \
  -bench '^BenchmarkCompiledRetrieval$' -benchtime=20x -count=1 -timeout=20m
```

Each corpus has one revision and source per record distributed across ten scopes.
The query is an exact signal in one explicit project scope; the driver validates
the exact expected revision on every response. Go's calibration run is additional
to the 20 measured samples per corpus. Fixture creation, version/hash checks and
before/after file fingerprints are outside timings. The child receives a minimal
PATH-only environment, an explicit synthetic binding and an unrelated empty cwd.

Timings include fresh process startup, memory opening/read, serialization, pipe
transfer and result validation. They do not mean cold disk: creation, validation
and file fingerprinting have already read the fixture. Result bytes include the
CLI envelope/newline; parent allocations do not describe the child. No real bank,
native model, credential, cache clearing or administrator operation is used.

## Results

| Records | Fixture files | Fixture bytes | Median | 95th percentile | Response bytes |
| --- | --- | --- | --- | --- | --- |
| 100 | 213 | 121,318 | 35.92 ms | 39.50 ms | 869 |
| 1,000 | 2,013 | 1,211,038 | 155.9 ms | 179.3 ms | 869 |
| 10,000 | 20,013 | 12,108,238 | 1,512 ms | 1,623 ms | 869 |

All expected revisions, read-only fixture fingerprints and unchanged-cwd checks
passed. [Measurement output](measurements.txt) retains reported values and test
outcomes. For publication only CRLF was normalized to LF and the CPU inventory
line removed; no measurement values or failures were removed or rewritten.

The 1,000-record result is below the documented provisional one-second
fresh-process investigation trigger. At 10,000 records a scoped recall takes
about 1.5 seconds on this host: bounded response size does not imply bounded
filesystem work. This deserves consideration before adopting a large historic
corpus, but is not evidence of a regression, an automatic indexing decision or
a newly invented acceptance threshold. Warm persistent-MCP distributions, deep
revision/conflict corpora, other platforms and model-level overhead are not
measured by this run. Earlier source-level evaluations cover different cases and
remain explicitly separate.
