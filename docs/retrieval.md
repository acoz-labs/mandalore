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

## Hook selection

[#227](https://github.com/acoz-labs/mandalore/issues/227) found the
`memory_context` hook injecting a weak match while `memory_recall` found the
right record. Raw term frequency let long records full of common domain words
outrank a short, specific answer. The packer also skipped a top hit whose whole
claim didn't fit the budget, and any score above 0 injected something, so most
prompts got a record. 2.1.0 changes all three:

- **Ranking (recall and the hook).** BM25 (k1 1.2, b 0.75) over the existing
  field weights: summary hits count four times a body hit and identifier hits
  eight times, with inflections at a quarter. Document frequency and average
  length come from the records the query can return, so ranking within a
  scope ignores other scopes. Function words ("what", "is", "the") still make
  a record match but only break ties, so a query's membership and the
  response shape are unchanged. A decision or preference scores 1.2 times as
  much as other kinds on a non-empty query. Ties, and an empty query's list,
  order by revision ID.
- **Threshold (the hook only).** A recalled record that matches just one of
  the prompt's content terms, and only in its body, is incidental and never
  relevant while two or more of the prompt's other content terms go
  unmatched, or one does and the match is a number: sharing "two-factor" with
  "How do I add two-factor login to Django?" or "600" with "What is 15 percent
  of 600?" doesn't make a record about it. When the matched word is most of a
  short question ("Is it self-hosted?", "Are we growing Sungold?") the record
  is judged on the bars like any other. Otherwise a record is relevant if any
  of three measures clears its bar:
  - *Coverage* ≥ 0.25: its BM25 score as a share of the most a record could
    score on the prompt. Every prompt term counts, weighted by IDF, so a
    prompt about a topic the signet lacks (its terms in no record) covers
    little.
  - *Evidence* ≥ 0.7: its score in units of one fully matched term that no
    other record contains. It doesn't fall when a prompt adds words, so
    instruction preambles ("Without looking anything up, …") don't hide an
    answer that matches several of the question's terms.
  - *Anchor* ≥ 0.7: the rarest of the prompt's names (words capitalized
    mid-sentence) in its summary or identifiers, as that term's IDF over the
    most possible. "Beacon" in "What is Beacon?" anchors its record even
    inside filler; an all-lowercase prompt has no names.

  A prompt with no relevant record gets nothing. A record less than 0.65 as
  relevant as the best (relevance being the largest of the three measures
  over its bar) is left out.
- **Packing (the hook only).** The default budget is 2048 bytes. The top 3
  records always appear, each clipped to an even share with a `…` mark and
  never losing its provenance. Further records appear only whole. Within a
  session (an optional `session` input) a revision is injected once. See
  [the server](server.md#tools).

Explicit `memory_recall` has no threshold and no de-duplication.

### Evaluation set

`internal/server/testdata/context-eval.json` is a synthetic signet of 250
records in eight scopes: 44 short answers (decisions, preferences, facts and
procedures), 36 decisions and preferences on neighbouring topics in the same
scopes, so a record's kind doesn't give the answer away, 26 long status and
research notes that use the answers' words, and 144 generated session notes
full of the same domain and meta words. Hosts use `memory.example.com`.

Each case has three phrasings. The hook sees each alone and inside 15
instruction wrappers ("Answer from context you already have; don't call
tools.", "In one line: …"), 2112 prompts in all. 90 prompts that no record
answers cover acknowledgements, general knowledge, coding questions and
uncovered topics, 42 of them written to collide with the signet's words:
hyphenated words, numbers, versions, file names, code calls and words from
record summaries ("How do I read a config.toml file in Rust?", "Is 17 a prime
number?", "Recommend a good font for a resume."). 38 short questions each hinge
on one distinctive word that is mostly in the answer's body, not its summary
("Are we growing Sungold?", "Metric, right?"), and 24 more prompts no record
answers each share a word with some record ("How do I install Redis on
Ubuntu?"); both were added in round 4 of #227 review. A scripted 20-turn
session measures injected bytes.

**Splits.** The tuning split is each case's question and paraphrase, 9
wrappers, 62 no-memory prompts (30 of them colliding), the 38 short
questions and the 24 collision prompts. Every tuning
decision (the hook's stop words, the incidental rule, the names rule, the
three bars, the relative cut and the kind boost) was made on it alone; each
hook stop word occurs in it. The held-out split is a third phrasing per case,
6 wrappers and 28 no-memory prompts (12 colliding), each written before any
run used it. A prompt is held out if it uses the third phrasing or a held-out
wrapper. The held-out split is computed only with `MANDALORE_EVAL_HELDOUT=1`;
`bin/ci` and every other run score the tuning split alone.

**Held-out exposure.** For 2.1.0 the held-out split was computed in every
evaluation run of the second review round, since the test then always scored
both splits and the tuning-only curve only hid it. Its numbers were shown in
three runs of the hook's development builds (76.3%/83.8%, 77.6%/85.5% and
77.6%/85.5% top-1/top-3), two runs without the kind boost (69.1%/76.1% and
70.7%/77.8%) and one both-split curve. Other runs computed and saved them
without showing them (one with other bars recorded 78.2%/86.6%). No held-out
number informed a decision. From the third round, held-out was computed once
per round, after the code was frozen; the numbers below are round 4's. Round
3's rule gave 77.2%/84.5% and 2 of 28 held-out false injections. Round 4's
adds "What does the SQL HAVING clause do?" (the migrations decision, through
"SQL") and "Explain the 80/20 rule." (the commit style through "explaining"
and a mail note through "rule"): two-word prompts sharing one word with a
record, the trade-off described under the curve below. The held-out split is
easier for 2.0.3 than the tuning split (61.9%
against 44.8% top-3), so similar held-out and tuning numbers partly reflect
easier prompts. The first version of the set had a held-out split that tuning
had looked at; it is part of the tuning split now.

`TestContextSelectionEvaluation` runs it in `bin/ci` and fails under these
floors. The tuning floors sit a little under its rates. The held-out floors
were set before the held-out split was first run and are not moved to fit it;
they are checked when it is computed.

| Floor | Tuning | Held out |
| --- | --- | --- |
| `memory_context` top-1 / top-3 | 75% / 85% | 65% / 80% |
| `memory_recall` top-1 / top-3 | 72% / 88% | 60% / 80% |
| False injection, at most | 15% | 25% |

It also fails if fewer than 90% of the short questions get their answer in
the hook's top 3, if more than 7 of the 24 collision prompts inject (the
count when they were added), if any tuning wrapper's top-3 is under 75%, if the hook carries
recall's first record in under 90% of the prompts where that is the case's
record, or if session prompts average over 800 injected bytes.
`MANDALORE_EVAL_REPORT=FILE` writes every outcome, for a per-case comparison
between two builds.

| Measure | Split | 2.0.3 | 2.1.0 | 2.1.0, no kind boost |
| --- | --- | --- | --- | --- |
| `memory_context` top-1 / top-3 | tuning (880 prompts) | 18.4% / 44.8% | 78.9% / 87.4% | 71.6% / 83.2% |
| | held out (1232) | 33.3% / 61.9% | 77.3% / 84.6% | not measured |
| `memory_recall` top-1 / top-3 | tuning (88 queries) | 22.7% / 37.5% | 77.3% / 93.2% | 75.0% / 88.6% |
| | held out (44) | 11.4% / 29.5% | 75.0% / 88.6% | not measured |
| False injection | tuning (62) | 71.0% (44) | 14.5% (9) | 12.9% (8) |
| | held out (28) | 67.9% (19) | 14.3% (4) | not measured |
| Short questions, answer in the hook's top 3 | tuning (38) | 25 | 35 | 34 |
| Collision prompts injected | tuning (24) | 23 | 7 | 7 |
| Injected bytes per prompt, 20-turn session | | 1030 | 572 (353 with `session`) | 594 |
| Warm p95, 250 records | | | 12 ms | |

The 2.0.3 column runs the same test file against 816ea1b. The kind boost was
kept because without it, on the tuning split, `memory_context` falls under
both its top-1 and top-3 floors and recall top-3 under 90%.

`MANDALORE_EVAL_CURVE=tune` logs the trade-off on the tuning split (`all`,
with `MANDALORE_EVAL_HELDOUT=1`, adds the held-out split):

| Coverage / evidence / anchor / relative | Top-1 / top-3 | Worst wrapper top-3 | False injection | Bytes | Short questions top-3 | Collisions injected |
| --- | --- | --- | --- | --- | --- | --- |
| none (incidental rule only) | 81.4% / 90.2% | 87.5% | 21/62 | 666 | 35/38 | 14/24 |
| 0.25 / – / – / 0.65 | 74.5% / 81.8% | 52.3% | 8/62 | 572 | 35/38 | 6/24 |
| 0.25 / 0.70 / – / 0.65 | 78.3% / 86.8% | 79.5% | 9/62 | 572 | 35/38 | 7/24 |
| 0.20 / 0.70 / 0.70 / 0.65 | 80.2% / 88.9% | 81.8% | 11/62 | 572 | 35/38 | 9/24 |
| 0.25 / 0.60 / 0.70 / 0.65 | 79.8% / 88.3% | 84.1% | 12/62 | 572 | 35/38 | 8/24 |
| **0.25 / 0.70 / 0.70 / 0.65** | **78.9% / 87.4%** | **81.8%** | **9/62** | **572** | **35/38** | **7/24** |
| 0.25 / 0.80 / 0.70 / 0.65 | 77.7% / 86.0% | 77.3% | 9/62 | 572 | 35/38 | 7/24 |
| 0.30 / 0.80 / 0.70 / 0.65 | 76.0% / 84.2% | 77.3% | 8/62 | 572 | 34/38 | 6/24 |
| 0.25 / 0.70 / 0.70 / none | 78.5% / 87.8% | 83.0% | 9/62 | 1179 | 35/38 | 7/24 |
| 0.25 / 0.70 / 0.70 / 0.80 | 78.9% / 86.2% | 81.8% | 9/62 | 428 | 35/38 | 7/24 |

The chosen bars are the ones from before the colliding prompts were added;
with the incidental and names rules they still give the best tuning top-3 at
the lowest false injection short of losing hits; the round-4 incidental rule
didn't move them. Two of the nine no-memory tuning prompts it still injects
for share one word with a record and have one other word: "Is 17 a prime
number?" (the mail signature's "no phone number") and "How many days are in
30 weeks?" (retention and lifetime notes). The incidental rule can't tell them
from a short question such as "Metric, right?"; it lets them through, which
the 38 short questions bought (27 of 38 in the top 3 under the round-3 rule,
35 now). The other seven match the signet on two or more words, or on a
summary word:
"How do I enable two-factor authentication on a GitHub account?" (mail
provider settings), "Convert this variable name to kebab-case in Python."
(flag naming), "How do I read a config.toml file in Rust?" (the CLI's config
path), "Format a date as 24-hour time in JavaScript." (the date-format
preference), "How do I cache API responses in a Flutter app?" (notes on the
API cache), "What theme should we pick for a wedding?" (editor theme) and
"Recommend a good font for a resume." (terminal font). A rule that also
dropped single summary-word matches removed the last two but lost the
answers to "Which editor do I use?", "Which units do I prefer?" and three
more, the same shape, so it wasn't taken. The relative cut barely moves hits
but halves the bytes.

The 24 collision prompts inject on 7, the same 7 as under the round-3 rule
(2.0.3: 23). None goes through the incidental rule: "How do I install Redis
on Ubuntu?", "How long does a Canada goose live?", "What width should a
kitchen countertop be?" and "How do I snapshot a VM in VirtualBox?" match a
summary word ("install", "live", "kitchen", "snapshot"); "How do I flush the
DNS cache on Windows?" matches summary words ("cache", "windows") and two body
words; "How do I pick a phone number for a new business line?" and "Explain
the imperative mood in Spanish grammar." match two or more body words. They
are the summary-word and two-word limits below, not addressed in 2.1.0.

Known limits, all lexical: a question whose only distinctive word is common in
the signet, inside an instruction wrapper, can fall under every bar
(`orchard-compost`'s "How often should the compost be turned?"). A question
whose answer hinges on one word in a record's body gets nothing once two more
of its words go unmatched ("Which port does Unbound listen on?"), and an
unrelated two-word prompt that shares one word with a record can get it. A prompt that
shares two or more words, or a summary word, with an unrelated record can
still get it. A name typed in lower case, or first in its sentence
("Beacon: what is it?"), doesn't anchor; on the tuning split the Beacon case
then hits in 75% and 85% of its prompts instead of all of them, the misses
being heavily wrapped ones. The hook drops words
under three letters, so a record found by recall through "go" can miss.
Synonyms still miss. Five recall queries rank their answer lower than 2.0.3
did; see [the release notes](releases/2.1.0.md#known-deviations).

## Reproduce measurements

Use Go 1.26.10 on a disposable development checkout. These opt-in benchmarks create
only synthetic temporary banks/references. They do not use native authentication,
an installed agent binding, remote repositories or private memory.

```sh
mise exec go@1.26.10 -- go test ./internal/testfixture ./internal/memory -run 'TestCorpus|TestNearestRank|TestRetrieval' -count=1
mise exec go@1.26.10 -- go test -v ./internal/memory -run '^$' -bench '^BenchmarkRetrieval$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.10 -- go test -v ./internal/server -run '^$' -bench '^BenchmarkMemoryContext$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.10 -- go test -v ./internal/foundlings -run '^$' -bench '^BenchmarkFoundlingRetrieval$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.10 -- go test -v ./cmd/mandalore -run '^$' -bench '^BenchmarkCompiledRetrieval$' -benchtime=20x -count=1 -timeout=20m
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
opening plus recall, and scope pages. Prompt-hook measurements run the server's
`memory_context` over a 1000-record signet with its read cache: ranked recall,
the scope list and the emitted text. Foundling
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

The 1.x [MCP presentation comparison](evidence/mcp-rendering/native.md) addresses
duplicate compatibility envelopes, not retrieval algorithms. In four synthetic
Codex runs, candidate guidance removed eight duplicate representations per run
and retained correct answers, historical attribution and truncation notices.
Final-request input context was about 5% smaller. Aggregate input usage varied:
one candidate run used an extra orchestration round and increased aggregate
input, while its replicate did not. Added reference/code cost is included in the
reported context and byte counts. These observations are not universal cost,
latency or compaction guarantees. Unknown results safely remain verbose.

A later retained-candidate small-task check reduced three duplicate wrappers
to zero but increased final-request input by 699 tokens: selector/discovery
overhead exceeded saved payload bytes. A first-error run also bypassed the
selector before loading its reference. The [sequencing follow-up](evidence/mcp-rendering/first-result.md)
retains that counterexample and the targeted correction; these observations must
not be restated as a general token-cost reduction.

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
