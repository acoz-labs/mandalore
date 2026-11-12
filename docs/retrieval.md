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
  order by revision ID. From 2.5.0, of two decisions (or two preferences) in
  one scope on one subject, the newer goes directly ahead of the older when
  the older ranks above it (see [2.5.0](#250-filler-collisions-generic-words-and-newer-decisions)).
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

### 2.5.0: filler, collisions, generic words and newer decisions

[#232](https://github.com/acoz-labs/mandalore/issues/232) fixed the limits
2.1.0 left: filler around a short question, prompts that inject through a
lone summary word, two-word prompts that share one generic word, prompts that
name something the signet lacks, and two decisions on one subject. Each rule
was chosen on the tuning split alone.

- **Question clauses.** Besides the whole prompt, the hook judges each of its
  first 3 question clauses: text ending in `?`, from the previous sentence
  boundary (a newline, or ".", "!", ":" or ";" followed by a space). Each is
  recalled and judged on its own. A record's relevance is its best over the
  whole prompt and its clauses, the relative cut and the top 3 apply to that,
  and records rank by it. "Before we start, remind me: Which editor do I use?
  One line is fine." is judged on "Which editor do I use?" too, so the
  wrapper's words don't dilute it. A clause can add or raise a record, never
  drop one. "okay", "right", "fine", "please" and "check" join the hook's
  stop words.
- **Generic words.** A record whose only match with a two-term prompt is one
  body word from a fixed list is incidental. The list has 50 common English
  nouns in three closed classes: units of time; numbers and quantity; and
  kind, part and form (`internal/memory/generic_words.go`). A word's plural
  counts. "Is 17 a prime number?" and "How many days are in 30 weeks?" get
  nothing, while "What's the burst allowance?" and "Are we growing Sungold?"
  keep their answers: their words aren't on the list.
- **Lone summary words.** A record that matches one of the prompt's terms, in
  its summary or identifiers, on a word the prompt doesn't capitalize, is
  incidental when another of the prompt's terms is in no record at all: the
  prompt is about something the signet lacks. A prompt's first word counts as
  capitalized here when every other term is a name ("Kopia or Borg?", "Kopia
  vs Borg"). So when Kopia is in several records' summaries, those inject its
  records as "Borg or Kopia?" does. The first word still anchors nothing and
  never counts as a name the signet lacks, so a prompt whose answer rests on
  a name's anchor can still depend on order (see the known limits). "How do I install Redis on
  Ubuntu?" and "What theme should we pick for a wedding?" get nothing; "Which
  editor do I use?" (one term) and "Does Beacon work at a wedding?" (a name)
  still inject.
- **Names the signet lacks.** A prompt that names something no record holds
  (a word it capitalizes mid-sentence, in no record: "in Rust?", "on a GitHub
  account") gets a record only if one of its terms is in that record's
  summary or identifiers. A record that shares only body words with it isn't
  relevant, however well it covers them. "How do I enable two-factor
  authentication on a GitHub account?" and "Convert this variable name to
  kebab-case in Python." get nothing; "Which cloud region do we use for the
  Kubernetes cluster?" still gets the region decision.
- **Newer decisions.** Two current decisions, or two preferences, in one
  scope are on one subject when their summaries share at least two thirds of
  their content terms and their bodies at least 30% (Jaccard, an inflection
  counting as the same term). "Editor" and "Editor theme" aren't. Recall puts
  the newer (later `effective_from`, then `recorded_at`, then ID) directly
  ahead of the older. The hook injects the newer whenever it injects the
  older, even if only the older matches, and never adds the older for the
  newer's sake. Neither supersedes the other.

### 2.5.1: lowercase either-or prompts

2.5.0 judged a bare either-or prompt by its capitals. In "Kopia or Borg?"
the first word is exempt from the lone-summary-word rule and "Borg" is a
name, so Kopia's records inject. In "kopia or borg?" no word is a name or
the first-word exception, so the match on "kopia" was dropped because
"borg" is in no record, and the prompt got nothing
([#264](https://github.com/acoz-labs/mandalore/issues/264)). 2.4.0 injected
Kopia's records in every case.

- **Bare either-or.** A prompt (or question clause) of two or more words,
  each one content term, joined by `or` or `vs` in any case, with a final
  `?`, `.` or `!` allowed: "kopia or borg?", "borg vs kopia", "kopia or Borg?".
  `vs.` isn't a connective, because the word after it starts a sentence, so
  "Kopia vs. Borg" stays as in 2.5.0. "versus", wrappers without a `?`
  clause ("quick one, kopia vs borg") and words that give no content term
  ("go or rust?", "this or that?") aren't bare either-ors either.
- **The rule only adds.** For a bare either-or whose capitals don't already
  give it one lead word, every word that isn't a name becomes a lead word.
  A lone summary match on a lead counts, as on a name; a lead anchors
  nothing and never counts as a name the signet lacks. The hook judges the
  prompt as 2.5.0 does and again with those leads. The second judgement's
  records and conflicts that match the prompt in their summary or
  identifiers, and that the first didn't keep, are additions. An added
  decision or preference brings the newer record on its subject, just
  ahead of it, as the first judgement does, and the two go in together or
  not at all. When 2.5.0 injects anything,
  its output is built as before, line for line, and the additions follow
  as whole lines in the room it leaves, ahead of the scopes. When 2.5.0
  injects nothing, the additions are packed as any records are. So
  "panel or turbine?", whose body matches 2.5.0 injects, keeps all 4 lines,
  and the 3 summary matches the rule adds don't fit in the 2048 bytes
  left. Capitalised forms get one judgement, as before. The second costs one more
  recall, only for prompts the rule reaches: about 2 to 6 ms at the median
  on the 1000-record benchmark corpus and the evaluation signet.
- **What it changes.** Lowercase and mixed-case bare either-ors inject what
  their capitalised form injects, on top of what 2.5.0 injected. That
  includes prompts no record answers: with a "Coffee order" record, "tea or
  coffee?" injects it, as "Tea or Coffee?" already did on 2.5.0 and both
  forms did on 2.4.0.

Proof, on synthetic data ([evidence](../evidence/2.5/either-or.md)): the
evaluation's per-prompt report is identical to 2.5.0's on every tuning set,
since it holds no lowercase bare either-or. The hook replay is byte-identical.
`TestEitherOrProbes` runs 28 lowercase bare either-ors (12 that share a
summary word with no record answering them, 8 that share nothing, 8 pairs of
known words) and asserts, per prompt, that the rule adds only records the
capitalised form injects and drops none that 2.5.0 injects.

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

#232 added six tuning sets and 4 records (254 in all):

- `filler_short`: 18 short questions over existing records, each in the
  issue's wrapper ("Before we start, remind me: {q} One line is fine.").
- `generic_two_word`: 12 two-word prompts no record answers, whose only match
  with any record is a list word in a body. Half have their other word in no
  record, and half have it also a list word, matched by a different record.
  `generic_guard`: 6 prompts that share a list word with some record and a
  distinctive word with their answer ("What year is Postgres?"), which must
  keep it.
- `named_guard`: 10 questions a record answers that also name something no
  record holds ("Which editor do I use when writing Rust?"), added in #232's
  code round 1 before the names rule ran on them. `either_or_limit`: 8 "X or
  Y?" questions whose X is only in the answer's body and whose Y is a name no
  record holds ("Helix or Neovim?"), added in code review round 2. The names
  rule costs them their answer, so the set is a stated-limit tripwire,
  asserted at its count (0 of 8), so a change that fixes or worsens it
  shows.
- `newer_decisions`: two pairs of same-scope decisions on one subject
  ("Tidepool task queue", "Spam threshold"), 6 queries matching both, and 2
  that only the older answers.

The 4 records alone moved no existing count on 2.4.0 code after their
summaries were reworded ("Tidepool job queue" collided with a cover-letter
prompt).

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
floors, which are #232's triage numbers, set before its work started. The
held-out floors are checked when the held-out split is computed.

| Floor | Tuning | Held out |
| --- | --- | --- |
| `memory_context` top-1 / top-3 | 78% / 86% | 75% / 84% |
| `memory_recall` top-1 / top-3 | 75% / 90% | 60% / 80% |
| False injection, at most | 6 of 62 | 2 of 28 |

It also fails if fewer than 35 of the 38 short questions or 16 of the 18
filler-wrapped ones get their answer in the hook's top 3, if more than 3 of
the 24 collision prompts or any generic two-word prompt injects, if a
generic-word guard prompt loses its answer, or if fewer than 8 of the 10
named guard prompts keep theirs. It fails if a newer decision doesn't rank
first in recall and the hook, or an older-only query loses the older from
recall's top 3. It fails if any tuning wrapper's top-3 is under 75%, if the
hook carries recall's first record in under 90% of the prompts where that is
the case's record, or if session prompts average over 800 injected bytes.
`TestNewerFirstCostsNoCase` fails if the newer-first rule lowers any case's
rank in recall or the hook, and `TestSameSubjectPairs` if it pairs any
records but the two fixture pairs.
`MANDALORE_EVAL_REPORT=FILE` writes every outcome, for a per-case comparison
between two builds.

| Measure | Split | 2.0.3 | 2.1.0 | 2.1.0, no kind boost | 2.5.0 |
| --- | --- | --- | --- | --- | --- |
| `memory_context` top-1 / top-3 | tuning (880 prompts) | 18.4% / 44.8% | 78.9% / 87.4% | 71.6% / 83.2% | 82.2% / 89.2% |
| | held out (1232) | 33.3% / 61.9% | 77.3% / 84.6% | not measured | 79.5% / 86.9% (6d8c672) |
| `memory_recall` top-1 / top-3 | tuning (88 queries) | 22.7% / 37.5% | 77.3% / 93.2% | 75.0% / 88.6% | 77.3% / 93.2% |
| | held out (44) | 11.4% / 29.5% | 75.0% / 88.6% | not measured | 75.0% / 88.6% (6d8c672) |
| False injection | tuning (62) | 71.0% (44) | 14.5% (9) | 12.9% (8) | 4.8% (3) |
| | held out (28) | 67.9% (19) | 14.3% (4) | not measured | 10.7% (3) (6d8c672) |
| | sealed (36) | | 13 (2.4.0) | | 6 |
| Short questions, answer in the hook's top 3 | tuning (38) | 25 | 35 | 34 | 36 |
| Filler-wrapped short questions, answer in the hook's top 3 | tuning (18) | | 12 (2.4.0) | | 18 |
| Collision prompts injected | tuning (24) | 23 | 7 | 7 | 2 |
| Generic two-word prompts injected | tuning (12) | | 8 (2.4.0) | | 0 |
| Named guard prompts, answer in the hook's top 3 | tuning (10) | | 9 (2.4.0) | | 8 |
| "X or Y?" limit prompts, answer in the hook's top 3 | tuning (8) | | 8 (2.4.0) | | 0 |
| Newer decision first, recall / hook | tuning (6) | | 0 / 0 (2.4.0) | | 6 / 6 |
| Injected bytes per prompt, 20-turn session | | 1030 | 572 (353 with `session`) | 594 | 560 (355 with `session`) |
| Warm p95, 250 records | | | 12 ms | | 15 ms (254 records) |

The 2.1.0 column is measured on the 250-record signet. The 2.4.0 figures for
#232's sets are on the 254-record signet, whose other numbers match 2.1.0's
except context top-1 (79.1% / 87.6%). The held-out and sealed false
injection miss their floors: see [Held-out exposure](#held-out-exposure-for-250).

The 2.0.3 column runs the same test file against 816ea1b. The 2.5.0 column is
the frozen code, b17ceb1, except the held-out figures, which are from the
code before #232's code round 1 (6d8c672). The sealed row is the reviewer's
batch, run on b17ceb1. The kind boost was
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

#### 2.5.0's curve

On the 2.5.0 code (b17ceb1) and signet, the bars stay as they were (the bold
row). Only stricter bars inject on fewer no-answer prompts, at a cost in hits
and in the worst wrapper:

| Coverage / evidence / anchor / relative | Top-1 / top-3 | Worst wrapper top-3 | False injection | Bytes | Short questions | Collisions | Filler-wrapped | Generic |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| none | 82.3% / 89.3% | 85.2% | 7/62 | 660 | 36/38 | 5/24 | 18/18 | 0/12 |
| 0.25 / – / – / 0.65 | 79.3% / 85.8% | 52.3% | 3/62 | 560 | 36/38 | 2/24 | 18/18 | 0/12 |
| 0.20 / 0.70 / 0.70 / 0.65 | 82.2% / 89.2% | 84.1% | 4/62 | 560 | 36/38 | 2/24 | 18/18 | 0/12 |
| 0.25 / 0.60 / 0.70 / 0.65 | 81.9% / 89.3% | 85.2% | 5/62 | 560 | 36/38 | 2/24 | 18/18 | 0/12 |
| **0.25 / 0.70 / 0.70 / 0.65** | **82.2% / 89.2%** | **84.1%** | **3/62** | **560** | **36/38** | **2/24** | **18/18** | **0/12** |
| 0.30 / 0.80 / 0.70 / 0.65 | 80.5% / 87.6% | 78.4% | 2/62 | 560 | 35/38 | 2/24 | 18/18 | 0/12 |
| 0.25 / 0.70 / 0.70 / none | 82.2% / 89.8% | 84.1% | 3/62 | 1189 | 36/38 | 2/24 | 18/18 | 0/12 |

The lone-summary-word rule was first planned to need two of the prompt's
other terms absent from the signet. That left 4 of 24 collisions, with the
hook at 83.2% / 90.2%. With one, collisions fall to 3 of 24 and the short
questions rise to 36 of 38, for a point of top-1. The names rule then took
false injection from 5 to 3 of 62 and collisions from 3 to 2, and no other
tuning count moved. Of the rules tried for names, it was the only one that
kept 8 of the 10 named guard prompts (the plan lists the others). The two
collision prompts left share two body words with a record ("How do I pick a
phone number for a new business line?") or two summary words ("How do I
flush the DNS cache on Windows?"). The 3 tuning no-answer prompts left ("How
do I read a config.toml file in Rust?", "Format a date as 24-hour time in
JavaScript.", "How do I cache API responses in a Flutter app?") name
something absent but match a record's summary as closely as the named
guard prompts match their answers'. They are
[#262](https://github.com/acoz-labs/mandalore/issues/262)'s.

#### Held-out exposure for 2.5.0

#232's held-out split was computed once, on the frozen code (6d8c672), after
every tuning decision. While the generic-word list was drafted, one rough
lexical check read the 44 held-out variant phrasings alongside the tuning
ones, with no recall or hook run, and found no answer the list would drop.
After it, one word was removed from the draft ("way", a function word) and
none added. The rule reaches only two-term prompts, and just one held-out
variant is that short.

The run met every floor but one: the hook injects on 3 of the 28 held-out
no-answer prompts, over the floor of 2 (2.4.0: 4). The requester ruled it a
deviation
([PR comment](https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6802782106)).
The split isn't used for more tuning, nobody on the implementer side has
looked at which prompts inject, and it wasn't run again after code round 1.
The bar was re-judged at the same rate on the reviewer's sealed no-answer
batch, hashed on #232 before code: at most 3 of its 36 prompts, and no more
than 2.4.0.

- **First gate, on 6d8c672:** 7 of 36, against 13 for 2.4.0, with no prompt
  2.4.0 doesn't also inject on
  ([result](https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6802849026)).
  The work went back to tuning on the tuning split, which added the names
  rule.
- **Second gate, on b17ceb1:** 6 of 36, against 13 for 2.4.0, again with no
  new prompt
  ([result](https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6803097976)).
  The three coverage-path collisions are unchanged from 2.4.0.

The owner decided to ship on the second condition alone, recorded as a
second deviation
([PR comment](https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6803169121)).
The coverage-path collisions that remain are
[#262](https://github.com/acoz-labs/mandalore/issues/262). The sealed batch
stays sealed and isn't used for tuning.

Known limits, all lexical: a question whose only distinctive word is common in
the signet, inside an instruction wrapper, can fall under every bar
(`orchard-compost`'s "How often should the compost be turned?"). A question
whose answer hinges on one word in a record's body gets nothing once two more
of its words go unmatched ("Which port does Unbound listen on?"), and an
unrelated two-word prompt that shares one word with a record can get it,
unless from 2.5.0 the word is on the generic-word list. A prompt that
shares two or more words, or a summary word, with an unrelated record can
still get it (from 2.5.0, a lone summary word beside a word in no record
doesn't). A question that names something no record holds can lose its
answer when its one matched word is in the answer's summary: 2 of the 10
named guard prompts, "Which editor do I use when writing Rust?" (which 2.4.0
misses too) and "Which shell do I use on my MacBook?". They have the shape of
"How do I install Redis on Ubuntu?". An "X or Y?" question whose X is only
in the answer's body and whose Y is a name no record holds ("Helix or
Neovim?", "Postgres or MySQL?") gets nothing: the name is absent and the
answer isn't matched in its summary. It depends on the order: "X or Y?", "X
vs Y", "Should I use X or Y?" and "Do we use X or Y for this?" get nothing,
while "Y or X?", "X?" and "What about X?" keep the answer, because a word that
starts the prompt isn't a name. 2.4.0 answers all 8 of the evaluation's
`either_or_limit` questions; 2.5.0 answers none, and `TestEitherOrOrders`
asserts all 8 reversed. When X is in several records' summaries instead, a
two-name "X or Y?" or "X vs Y" injects X's records as "Y or X?" does (see
[2.5.0](#250-filler-collisions-generic-words-and-newer-decisions)), in
lowercase too from 2.5.1 ([2.5.1](#251-lowercase-either-or-prompts)), but not
with `vs.` or inside a wrapper without a `?` clause. In lowercase, the
body-only shape keeps its answer in every order ("helix or neovim?"), on 2.5.0
and 2.5.1, because no word is a name the signet lacks. The
first word still anchors nothing, so where a name's record rests on its
anchor, order can still matter. With three names, "Beacon or Cortana or
Siri?" gets nothing while "Siri or Cortana or Beacon?" gets the Beacon
record, as on 5b87ef5. With two, "Beacon or Cortana?" gets 1 record (0 on
5b87ef5) while "Cortana or Beacon?" gets 2, as before. Beacon is in one
record's summary. A two-word prompt whose answer hinges on a generic body word
("Which night is bin night?") gets nothing, and a prompt whose shared word is
generic but in a summary ("What year is it?") isn't caught by the
generic-word rule. Two decisions on one subject whose wording differs more
than the bars allow keep their BM25 order, and a head revised for a typo
counts as newer. A name typed in lower case, or first in its sentence
("Beacon: what is it?"), doesn't anchor; on the tuning split the Beacon case
then hits in 75% and 85% of its prompts instead of all of them, the misses
being heavily wrapped ones. The hook drops words
under three letters, so a record found by recall through "go" can miss.
Synonyms still miss. Five recall queries rank their answer lower than 2.0.3
did; see [the release notes](releases/2.1.0.md#known-deviations).

## Reproduce measurements

Use Go 1.26.11 on a disposable development checkout. These opt-in benchmarks create
only synthetic temporary banks/references. They do not use native authentication,
an installed agent binding, remote repositories or private memory.

```sh
mise exec go@1.26.11 -- go test ./internal/testfixture ./internal/memory -run 'TestCorpus|TestNearestRank|TestRetrieval' -count=1
mise exec go@1.26.11 -- go test -v ./internal/memory -run '^$' -bench '^BenchmarkRetrieval$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.11 -- go test -v ./internal/server -run '^$' -bench '^BenchmarkMemoryContext$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.11 -- go test -v ./internal/foundlings -run '^$' -bench '^BenchmarkFoundlingRetrieval$' -benchtime=20x -count=1 -timeout=20m
mise exec go@1.26.11 -- go test -v ./cmd/mandalore -run '^$' -bench '^BenchmarkCompiledRetrieval$' -benchtime=20x -count=1 -timeout=20m
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
