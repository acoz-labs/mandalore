# Retrieval quality and efficiency

Issue #9's engineering evaluations and bounded corrections are implemented.
This guide records their method and limits, not independent candidate acceptance
or a universal corpus-scale performance promise.
Mandalore keeps scoped current evidence separate from history, conflicts and
foundling references. Exact-byte bounds and accurate evidence take precedence
over a smaller but misleading answer.

## Correctness baseline

`TestRetrievalQualityMatrix` exercises the public memory service against synthetic
global/project preferences, renamed projects, retained aliases, historical-only
words, same-keyword projects, future-effective revisions, concurrent heads,
no-match and paginated scope discovery. It asserts exact IDs, omission/conflict
counts, bounded complete responses and unchanged files. Known lexical misses
are labeled; they are not semantic search successes. In the initial baseline,
an absent synonym and a terminal period on a word both missed. The period case
was discovered by the fixture sanity test, not hidden by the benchmark. A focused
failing-first correction now ignores terminal periods while preserving internal
dots/hyphens and exact-match priority. Domains, versions and identifiers are not
split into matching fragments. Absent synonyms remain a lexical limitation.

`TestRetrievalSameServiceObservesExternalChanges` holds one reader open while
another writer adds, corrects and branches a record, then introduces corruption
and replaces the signet identity. The reader must see new local evidence or
refuse invalid/replaced data without a restart. Existing integrity, real-Git,
budget and scope tests remain required too.

The initial [baseline and profile](evidence/retrieval/README.md) showed repeated
device-provenance reads dominating a substantial part of graph validation.
Each graph validation now reuses successful checks of the same device between
revisions and their source metadata. The memo is local to that single validation,
never stored on a service, in a file or across calls. A subsequent read/write
validates current provenance again; errors are not cached. Source objects, source
origins, schema and graph integrity still undergo their existing checks.
This does not make multi-file reads atomic against uncoordinated filesystem edits
or introduce an index. Freshness/corruption tests and a same-fixture performance
comparison are required for this correction.

## Reproduce measurements

Use Go 1.26.4 on a disposable development checkout. These opt-in benchmarks create
only synthetic temporary banks/references. They do not use native authentication,
an installed agent binding, remote repositories or private memory.

```sh
mise exec go@1.26.4 -- go test ./internal/testfixture ./internal/memory -run 'TestCorpus|TestNearestRank|TestRetrieval' -count=1
mise exec go@1.26.4 -- go test -v ./internal/memory -run '^$' -bench '^BenchmarkRetrieval$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.4 -- go test -v ./internal/codex -run '^$' -bench '^BenchmarkPromptHook$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.4 -- go test -v ./internal/foundlings -run '^$' -bench '^BenchmarkFoundlingRetrieval$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.4 -- go test -v ./cmd/mandalore -run '^$' -bench '^BenchmarkCompiledRetrieval$' -benchtime=20x -count=1 -timeout=20m
```

Memory cases use 100/1000/10000 records, plus independent 1000-record depth-2,
depth-10 and conflict variants. Ten explicit scopes share terms. Each revision
has a source object and realistic compact prose. The shared test-only generator
writes canonical objects to a fresh test directory and validates the complete
bank before timing. It is not a supported production writer. Fixture creation
and validation are excluded from measured operation time, not from total command
duration. Small generator tests independently check counts and current/conflicting
heads before its output is trusted.
Use `-v` to retain parent-benchmark fixture counts in the output; without it Go
prints the leaf timings but can omit those setup logs.

Memory measurements cover narrow, broad, empty and no-match reads, fresh service
opening plus recall, and scope pages. Prompt-hook measurements include the actual
read-only adapter, default recall, routing inventory and emitted JSON. Foundling
measurements cover separate 100/1000-file references with source verification,
not just string matching. Compiled CLI measurements include process startup,
binding/recall/encoding, pipe transfer and client decoding/validation from an
unrelated working directory. Build and fixture setup are excluded.

## Interpret the units honestly

- `ns/op`, `B/op`, `allocs/op`: Go benchmark operation mean and benchmark-process
  allocation accounting. Compiled CLI allocation figures describe the parent,
  not the child process. Operations include result serialization and a small
  per-sample clock/bookkeeping cost.
- `p50-ms`, `p95-ms`: nearest-rank samples measured per operation, reported only
  with at least 20 iterations. They are not derived from the Go benchmark mean.
- `result-B/op`: actual average serialized returned bytes, not a requested
  maximum or token estimate. CLI includes its envelope/newline; memory measures
  compact result JSON; hooks include the emitted envelope/newline.
- Fixture logs: current record identities, total revisions, conflicts, files
  and bytes. History and referenced source files count toward real I/O.

Warm means repeated filesystem-backed reads of a validated fixture, not a
persistent memory cache. Fresh service/process does not mean cold disk: the OS
page cache is not flushed. No administrator privileges or cache-clearing commands
are used. Record exact source head, environment, commands, sample counts and raw
outputs; host timings do not establish a universal SLA or native Linux acceptance.
Run suites sequentially for comparable host evidence, not concurrently.

## Native context and follow-up

### Progressive historical references

Foundling search starts with three 512-byte previews within an 8192-byte
serialized-result budget, including provenance. Default reads return up to 1024
content bytes. These are initial defaults, not a whole-task ceiling: explicit
search budgets reach 32768 bytes and reads reach 8192 content bytes per call.
Follow result `next_offset` for more documents using the same query and exact
registration; an excerpt's `next_offset` instead continues UTF-8 bytes within
that document. Every call revalidates the selected source. A result-budget
failure is actionable, not an empty/no-match result. Full contracts and human
menu behavior are in [foundlings](foundlings.md).

The [progressive retrieval evaluation](evidence/foundlings-progressive/README.md)
separates returned source ranges, response bytes and native context. A matched
ordinary task retained all decisive evidence while removing 2048 repeated source
bytes. The first candidate's extra guidance nevertheless increased final context;
shorter guidance produced a modest reduction in a subsequent run. Neither smaller
packets nor a single native observation establishes a general token/latency win.
Explicit deep review, later conclusions and pinned traversal remain available.
No ranking change, index, persistent retrieval cache, automatic summary or limit
on a user's deliberate exhaustive scan is introduced.

The [MCP presentation comparison](evidence/mcp-rendering/native.md) addresses
duplicate compatibility envelopes, not retrieval algorithms. In four synthetic
Codex runs, candidate guidance removed eight duplicate representations per run
and retained correct answers, historical attribution and truncation notices.
Final-request input context was about 5% smaller. Aggregate input usage varied:
one candidate run used an extra orchestration round and increased aggregate
input, while its replicate did not. Added reference/code cost is included in the
reported context and byte counts. These observations are not universal cost,
latency or compaction guarantees. Unknown results safely remain verbose.

Schema sizes, tool response bytes, hook text, requested context ceilings and
native aggregate token counters are distinct. Prior foundling tests observed
a 16000-byte recall request for two records. An earlier report of a redundant
complete-excerpt read was corrected by inspecting actual response flags: the
consultation excerpt was incomplete, so the subsequent read was warranted.
Neither a requested ceiling nor that warranted read proves a billed-token
regression. See
[native foundling evidence](evidence/foundlings-native/README.md).

The [native comparison](evidence/retrieval/native.md) records actual ordered
calls, returned sizes, warranted provenance/recovery reads, correct final results
and unchanged files. The refined skill reuses an already-discovered scope ID
for a fresh read and starts with documented defaults. It preserved warranted
history/reference reads, ordinary confirmed learning, explicit supersession and
fresh-session recall after a rename. Single native before/after observations do
not isolate token cost or establish a latency distribution. Candidate acceptance
remains #10/#11.

Provisional investigation triggers are warm scoped p95 over 250 ms or fresh
process p95 over one second at 1000 current records, and more than two redundant
retrieval calls in a simple scoped task after guidance correction. These trigger
profiling and a scoped follow-up, not automatic index installation or a timing
assertion in CI. Any future index must be rebuildable, detect external changes,
preserve exact identity/scope/graph semantics and need no mandatory inference
service. No specific index or semantic package is selected.
