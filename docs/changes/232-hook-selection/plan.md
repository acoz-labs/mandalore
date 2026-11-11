# Plan: make the prompt hook's selection robust to filler, collisions and generic words

## Issue

Refs #232 (https://github.com/acoz-labs/mandalore/issues/232). Outcome: the
`memory_context` prompt hook injects less on prompts no record answers,
finds short questions inside filler, ignores a single generic shared word,
and ranks the newer of two same-scope decisions on one subject first, with
no regression in recall or context rankings. Ships as minor release 2.5.0,
server only. No file under `plugins/` changes: not the hook, `.mcp.json`,
skills, README or `plugin.json` (see [Plugin changes, by kind](#plugin-changes-by-kind)).

### Acceptance numbers, and what proves each

The triage comment on #232 set these before work started. They are used as
written. "Tuning" and "held out" are the two splits of the synthetic
evaluation set (`internal/server/testdata/context-eval.json`,
[docs/retrieval.md](../../retrieval.md#evaluation-set)). The held-out split
is computed once per review round, after the code is frozen, with
`MANDALORE_EVAL_HELDOUT=1`, and no tuning decision looks at it.

| # | Acceptance (triage) | Baseline (2.4.0) | Proved by |
| --- | --- | --- | --- |
| 1a | False injection on no-answer prompts: at most 6 of 62 tuning | 9 of 62 | existing `no_memory.tune`; `TestContextSelectionEvaluation` asserts ≤ 6 |
| 1b | … and at most 2 of 28 held out | 4 of 28 | existing `no_memory.held_out`; asserted when held out is computed |
| 1c | Name collisions through the coverage, evidence and summary paths: at most 3 of 24 | 7 of 24 | existing `collisions.tune`; the tripwire moves from 7 to 3 |
| 2a | Filler-wrapped short questions: at least 16 of 18 answered in the hook's top 3 | 7 of 18 in the 2.1.0 review | new `filler_short` fixture (18 prompts); asserted ≥ 16 |
| 2b | Plain short questions stay at 35 of 38 or better | 35 of 38 | existing `short.tune`; the floor moves from 90% to 35 of 38 |
| 3 | Generic single-word matches: 0 injections on at least 10 two-word prompts sharing one generic body word | not measured | new `generic_two_word` fixture (12 prompts whose sole match with any record is a list word, chosen as [below](#how-generic_two_word-is-chosen)); asserted 0; plus a `generic_guard` set (6) that must keep injecting |
| 4a | `memory_context` top-1/top-3 at least 78%/86% tuning | 78.9% / 87.4% | `rates` over the tuning split; floors move from 75/85 to 78/86 |
| 4b | … and at least 75%/84% held out | 77.3% / 84.6% | held-out split; floors move from 65/80 to 75/84 |
| 4c | `memory_recall` top-1/top-3 at least 75%/90% tuning | 77.3% / 93.2% | floors move from 72/88 to 75/90 |
| 4d | Session prompts average at most 800 injected bytes | 572 (scripted session, in process) | the hook replay with session IDs ([below](#bytes-per-prompt-and-the-2048-byte-default)), as the requester ruled; the in-process 800-byte tripwire on the scripted session stays in `bin/ci` |
| 4e | Bytes per prompt measured in real sessions and reported; the 2048-byte default changes only if the measurement supports it | see [Bytes per prompt](#bytes-per-prompt-and-the-2048-byte-default) | hook replay sessions (the requester's ruling) plus the evaluation's byte table |
| 5 | Two same-scope decisions on one subject: the newer ranks first for queries matching both, and no existing case regresses | not measured | new `newer_decisions` fixture; an assertion that no existing case's expected record drops in rank with the rule on against off; a tripwire on the pairs the rule forms |

The 2.1.0 review's 18 filler-wrapped questions aren't in the repository, so
`filler_short` is a new set of 18 written for this change: short questions
over existing records, each in the issue's wrapper ("Before we start, remind
me: {q} One line is fine."). Step 1 records its count on 2.4.0 before any
code changes. The bar is 16 of 18 whatever that baseline is.

The floors that already exist in `assertEvaluation`
(`internal/server/context_calibration_test.go`) only rise, to the triage
numbers. No floor is lowered. The per-wrapper top-3 floor (75%) and the
"hook carries recall's first record in 90%" floor stay.

The reviewer's sealed probe batches (hashes on #232) check the same numbers
at code review. This plan doesn't use or look for them.

### Deviation on triage 1b, and the gate that replaces it

On the frozen code (6d8c672), the hook injects on 3 of the 28 held-out
no-answer prompts. The bar is at most 2, and 2.4.0 injects on 4. Every other
triage number is met. The requester ruled on it on 2026-11-11
(https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6802782106):

- It's recorded as a deviation, and that held-out set isn't used for any
  more tuning. Nobody on the implementer side has looked at which prompts
  inject.
- The bar isn't loosened. It's re-judged at the same rate on the reviewer's
  sealed no-answer batch (36 prompts, hashed on #232 before code). The
  candidate at 6d8c672 must inject on at most 3 of the 36, and on no more of
  them than 2.4.0 does.
- If either condition fails, the work goes back to tuning within its round
  budget, or closes as not shipped. Any further relaxation goes back to the
  requester.
- The requester's private eval before publication is unchanged.

First gate, on 6d8c672: **failed**
(https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6802849026).
The candidate injected on 7 of the 36 (2.4.0: 13; no new prompt): plain 1,
coverage 3, evidence 2, summary 1. Under the ruling the work went back to
tuning, as code round 1 of 4, below.

### Code round 1: names the signet doesn't hold

The lead's direction: tune how the coverage and evidence paths handle name
collisions, deciding on the tuning split only, with every other bar still
met. The 28 held-out no-answer prompts are spent for 1b and weren't run
again, and the sealed prompts stay unseen.

- **What the tuning split showed.** Of the 5 tuning no-answer prompts and 3
  collisions still injecting at 6d8c672, 6 name something no record holds
  ("in Rust?", "on a GitHub account", "in Python", "in JavaScript", "in a
  Flutter app", "in Spanish grammar"). They inject through coverage or
  evidence on words the record shares.
- **A new guard set first.** No tuning prompt asked a real question that also
  named something absent, so any rule here was untested against losing
  answers. `named_guard` holds 10 such questions ("Which editor do I use when
  writing Rust?", "Which shell do I use on my MacBook?"), written before any
  rule ran on them. The frozen code already loses 2 of the 10 (2.4.0 loses
  the first of them too, so 2.5.0 loses one more), both to the
  lone-summary-word rule: their one matched word is in the answer's summary
  and the name is absent. That's the shape of "How do I install Redis on
  Ubuntu?", which the rule exists to drop, so it stays a stated limit.
- **Rules tried, on the tuning split:**

  | When the prompt names something no record holds | False injection | Collisions | Named guard | Other bars |
  | --- | --- | --- | --- | --- |
  | as at 6d8c672 | 5/62 | 3/24 | 8/10 | met |
  | only a name anchor counts | 0/62 | 2/24 | 1/10 | met |
  | evidence doesn't count | 4/62 | 2/24 | 6/10 | met |
  | evidence doesn't count, coverage bar × 1.5 | 0/62 | 2/24 | 3/10 | met |
  | **a record needs one of the prompt's terms in its summary or identifiers** | **3/62** | **2/24** | **8/10** | **met, unchanged** |
  | as above, also when 2 or more terms are in no record | 3/62 | 1/24 | 8/10 | top-1/top-3 80.6%/87.3%, worst wrapper 75.0%, at its floor |
  | as above, also when half the terms are in no record | 3/62 | 2/24 | 8/10 | short questions 33/38, under the bar |

- **Chosen: the bold row.** A prompt that names something no record holds
  is about something else, unless a record matches one of the prompt's terms
  in its summary or identifiers. A record that shares only body words with
  it isn't relevant. "Names" are the words the hook already treats as names
  (capitalized mid-sentence); "no record" is the same document-frequency
  test as the lone-summary-word rule. It's the only row that loses no named
  guard. The three no-answer prompts it leaves ("How do I read a config.toml
  file in Rust?", "Format a date as 24-hour time in JavaScript.", "How do I
  cache API responses in a Flutter app?") match their record's summary as
  closely as the guards match theirs, so no lexical rule tried here
  separates them.
- **Code.** Ranking reports, per query, which names no candidate holds
  (`Packet.AbsentNames`) and which heads match a query term in the summary or
  identifiers (`Packet.Titled`). The hook's `relevance` applies the rule.
  Recall is unchanged.
- **Tuning numbers on the new frozen code:** false injection 3 of 62,
  collisions 2 of 24, plain short questions 36 of 38, filler-wrapped 18 of
  18, generic two-word 0 of 12, generic guard 6 of 6, named guard 8 of 10,
  newer decision first 6 of 6 in recall and the hook, older-only 2 of 2,
  hook top-1/top-3 82.2%/89.2%, recall 77.3%/93.2%, scripted session 560
  bytes.

Second gate, on b17ceb1: **failed**
(https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6803097976).
The candidate injected on 6 of the 36 (2.4.0: 13; no new prompt): plain 1,
coverage 3, evidence 1, summary 1. The three coverage-path injections are
unchanged from 2.4.0.

### Code review round 2

The review found that the names rule also costs "X or Y?" questions their
answer, when X is only in the answer's body and Y is a name no record holds
("Restic or Borg?"). The lead's decision: no rule change. The shape is
named as a known loss in the release notes and the retrieval guide, and a
new tuning set, `either_or_limit` (8 fresh prompts, none the reviewer's),
asserts it at its current count, 0 of 8 (2.4.0: 8 of 8). The newer-first
switch now turns off the hook's half too, so the on/off test compares both;
production behaviour is unchanged, and the tuning bars are the same.

### Second deviation: shipping on the second condition

The owner decided, through the requester, to ship on the second condition
alone: the candidate injects on no more of the sealed batch than 2.4.0, and
on no prompt 2.4.0 doesn't. The 3-of-36 miss is recorded as a second
deviation (https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6803169121).

- The remaining coverage-path name collisions, unchanged from 2.4.0, move
  to #262.
- The sealed batch stays sealed and isn't used for tuning.
- Code review proceeds on b17ceb1. A later change that touches selection or
  packing reruns the tuning bars, not the sealed batch.
- Publication still waits for the requester's private eval on the candidate
  built from the merge commit.

### Code round 4: either-or questions on a name in summaries

A synthetic repro shows an order-dependent either-or limit. In a signet with
three records whose summaries name "Kopia" and none naming "Borg", 5b87ef5
injects nothing for "Kopia or Borg?" or "Kopia vs Borg", and all three
records for "Borg or Kopia?". The fix had to make the first two behave like the third, with every
tuning bar met, in the one remaining code round. The work used the tuning
split and the repro only.

- **Cause.** The hook never counts a prompt's first word as a name. In
  "Kopia or Borg?" only "Borg" is one. The records match only "Kopia", in
  their summaries, so the lone-summary-word rule, which exempts names, finds
  a lone summary match beside a word no record holds and drops every record.
  In "Borg or Kopia?", "Kopia" sits mid-sentence and is a name.
- **Rejected first attempt: every capitalized word exempts.** That fixed the
  repro, but the exemption then also covered a sentence-initial imperative:
  "Format a date as 24-hour time in JavaScript." (a tuning no-answer prompt)
  injected 974 bytes instead of 460.
- **Rejected second attempt: the first word is a name when every other term
  is one.** That fixed the repro with no tuning difference, but it made the
  first word count as an absent name too. "Neovim or Helix?" then needed a
  summary match, and the reversed body-only shape fell from 8 of 8 to 1 of 8.
- **Fix.** The prompt's first word, when capitalized and every other term is
  a name, is passed to recall as `Query.Lead`. A lone summary match on it
  isn't incidental, as on a name. It anchors nothing and never counts as an
  absent name.
- **Results.** The repro injects 3 records for "Kopia or Borg?", "Borg or
  Kopia?" and "Kopia vs Borg" (5b87ef5: 0, 3, 0). The body-only shape is
  unchanged in every phrasing measured, and `either_or_limit` stays at 0 of 8.
  The tuning split's per-prompt report has 0 differences from 5b87ef5, and
  every replay row is identical. Evidence: `docs/evidence/2.5/either-or.md`.
- **Tests.** `TestMemoryContextEitherOrOnASummaryName` asserts the 3 records
  in all three phrasings and fails on 5b87ef5. `TestEitherOrOrders` asserts
  the reversed body-only shape at 8 of 8, which the second attempt would
  have failed. `TestContextLead` covers when the first word counts.
- VERSION stays 2.5.0: the release isn't published yet.

### What changed while building

Each change was decided on the tuning split, before the held-out run.

- **The lone-summary-word rule fires when any other prompt term is absent
  from the signet, not two.** With two, the collision prompts stay at 4 of
  24 ("How do I snapshot a VM in VirtualBox?" has only one other term), over
  the bar of 3. With any, they're at 3 of 24, plain short questions rise
  from 35 to 36 of 38, and the hook's tuning top-1/top-3 goes from 83.2%/90.2%
  to 82.2%/89.2%, above the bars.
- **Plain -s/-es plurals of list words count as list words.** The existing
  inflection rule needs a base of four letters or more, so it would miss
  "days" and "hours". The 50 words are unchanged.
- **Clause boundaries.** ".", "!", ":" and ";" end a clause only when a space
  or the end of the prompt follows, so "Apache-2.0?" and "10:00" stay whole.
- **"just" isn't added** to the stop words: it's already a function word.
- **Newer-first pairs come from every current record in the query's
  scopes**, not just the matching ones. That way the hook can bring the
  newer decision when only the older matches the prompt, as the plan
  requires. Recall still moves only records that both match.
- **The pair records' wording.** "Tidepool job queue" collided with "Write a
  cover letter for a job in Berlin.", and "Mail spam threshold" cost the
  newsletters case its hook hits. They became "Tidepool task queue" and
  "Spam threshold" before any rule ran (step 1's comment on the PR).
- **The replay ran before the held-out run**, so a change to the default
  couldn't follow it. The default stayed 2048.
- **Sessionless MCP.** The replay's `tools/call` carries the `_meta`
  protocol version, client capabilities and client info, and the
  `Mcp-Method` and `Mcp-Name` headers that `2026-07-28` requires.

## Files that change

- **Hook selection** (`internal/server/context.go`):
  - *Question clauses (item 2).* Besides the whole prompt, the hook scores
    each question clause: a run of text that ends in `?`, cut at the
    previous sentence boundary (`.`, `!`, `?`, `:`, `;`, a newline). At most
    3 clauses are scored, the first 3 in the prompt, so a prompt costs at
    most 4 recalls. A prompt that is one question is its own only clause and
    is scored once, as today. "Before we start, remind me: Which editor do I
    use? One line is fine." is then judged on "Which editor do I use?" as
    well, so wrapper words outside the stop-word list ("remind", "start")
    can't dilute it.
  - *How clause results combine.* Each query (the whole prompt and each
    clause) is recalled and judged on its own: its own incidental rule and
    its own coverage, evidence and anchor measures. A record's relevance is
    the largest, over the queries that returned it, of its relevance there
    (the largest of the three measures over its bar, 0 if incidental). The
    hook ranks records by that relevance, ties by the whole prompt's recall
    order, then by revision ID. The relative cut (0.65 of the best) and the
    top 3 apply to that one ranking, and packing is unchanged. So a clause
    can add a record or raise one, never lower one the whole prompt already
    judged relevant.
  - *Stop words (item 2).* A request word joins the hook's stop-word list
    only if it occurs in a tuning set other than `filler_short`, so the set
    built in the issue's wrapper can't justify its own words. Candidates and
    where they occur: "please" (tuning wrappers "{q} One line please, no tool
    calls." and "Context check … Just the answer, please."), "okay" ("Are
    short flags okay?"), "right" ("Metric, right?"), "fine" ("Is
    eu-central fine?"), "just" and "check" (tuning wrappers). A word without
    such an occurrence ("remind") is left to the clauses.
  - *Ranking only, no format change.* `ContextInput`, the output envelope
    and the session de-duplication stay as they are.
- **Relevance rules** (`internal/memory/memory.go`, `bm25` and
  `recallHeads`; `internal/memory` exports stay hook-agnostic):
  - *Generic word (item 3).* A record whose only match with the prompt is
    one body word from the generic-word list is incidental whenever the
    prompt has another content term. A word counts when it, its plain -s or
    -es plural, or its base under the existing inflection rule (-s, -es,
    -ed, -ing) is on the list. On a
    prompt of three or more terms a lone body match is already incidental
    under 2.1.0, so the rule changes only two-term prompts. The 2.1.0 rules
    stay: a lone body match on a bare number, or with two or more other
    terms unmatched, is still incidental.
  - *The generic-word list.* It is written for this change, under the
    repository's own licence, and kept in one file
    (`internal/memory/generic_words.go`), with no third-party word list and
    no `NOTICE` entry. The rule for what goes on it is fixed here, before
    any run:
    - a word is on the list only if it is a common English noun in one of
      three closed classes: units of time; numbers and quantity words; and
      general nouns of kind, part and form;
    - every member of each class below is on the list, and nothing else.
      No technical term, proper name (no weekday or month names), adjective
      or verb is on it, and no word that is already a stop word or a
      function word;
    - the file holds exactly the words below, and adding a word later
      means changing this rule in a plan, not editing the file to fit an
      outcome.

    The list, in full (50 words):
    - *Time:* second, minute, hour, day, week, month, year, decade,
      century, morning, afternoon, evening, night, weekend, tomorrow,
      yesterday, time.
    - *Numbers and quantity:* number, amount, count, total, size, percent,
      half, quarter, dozen, hundred, thousand, million, billion, pair,
      couple, lot.
    - *Kind, part and form:* thing, kind, type, sort, part, piece, rule,
      case, example, fact, idea, point, place, name, word, list, form.

    ("way" is a function word already, and "today" a hook stop word.)

    *Held-out exposure, for `docs/retrieval.md`'s note.* While the list
    was drafted, one rough lexical check read the 44 held-out variant
    phrasings alongside the tuning ones, with no recall or hook run, and
    found no expected record dropped. Afterwards, one word changed: "way"
    was removed, because the plan's own rule excludes function words and
    "way" is one. No word was added. The read can reach at most one variant:
    the rule changes only two-term prompts, and just 1 of the 44 variants
    has two terms or fewer when bare (the reviewer's count at plan
    confirmation). That variant keeps its answer under
    the list, so the read could have moved about 16 of the 1232 held-out
    prompts at most, and only towards keeping an answer. The reviewer's
    sealed batches are unaffected.
  - *How 35 of 38 plain short questions holds (triage 2b).* The rule only
    reaches a two-term prompt whose record matches one body word on the
    list. A rough check of all 38 plain short questions against this list,
    with the stop words above, finds none whose answer it drops. The eight
    of that two-term shape match on "Postgres", "cameras", "burst",
    "flags", "Sungold", "eu-central", "metric" and "Zsh", none of which is
    on the list. In "Are short flags okay?", "Is eu-central fine?" and
    "Metric, right?" the other word becomes a stop word, so they're
    one-term prompts the rule doesn't touch. "What's the burst allowance?"
    keeps its hit, and "Are the cameras isolated?", which misses today, isn't
    barred from coming back. The same check over the 88 bare tuning case
    phrasings finds no expected record dropped. (The rough check also read
    the 44 held-out variant phrasings, lexically only, with no recall or
    hook run; see the exposure note below.) Step 1
    repeats the check in Go against the real tokenizer for all 38 short
    questions and all 88 tuning case phrasings, and posts the result in the
    PR before the rule is written. If it finds one, the rule as written fails 2b. Step 1 then stops and
    reports to the lead, without editing the list.
  - *How the triage 3 fixture gets 0 injections.* The fixture holds only
    prompts whose sole match with any record is a list word: every record
    that matches either of the prompt's two terms matches just one of them,
    in its body, and that term is on the list (see
    [below](#how-generic_two_word-is-chosen)). Each such record is then a
    lone generic body match, which the rule makes incidental, so the prompt
    injects nothing by construction. "Is 17 a prime number?" ("17" has under
    three characters, so its terms are "prime" and "number") is in it: only
    the mail signature's body matches, on "number". The assertion is 0
    injections over the whole fixture.
  - *What the rule does to "How many days are in 30 weeks?"* No record
    holds both words. Today it injects 6 records, 3 matching only "days"
    and 3 only "weeks", each in the body. Both words are on the list, so
    all six become incidental and the prompt should inject nothing. It's a
    `no_memory.tune` prompt, so it counts towards 1a (at most 6 of 62).
    Step 1's check reports the rule's effect on it.
  - *Guard against over-suppression.* A prompt that shares a list word
    with a record and also a distinctive term with its answer ("What year
    is Postgres?" for the Tidepool database decision, "How many hours of
    Helix?" for the editor preference) should inject, because the
    distinctive term is a real match. Such prompts go in a separate guard
    set, `generic_guard`, never in the 0-injection fixture. Each guard
    prompt must keep its expected record in the hook's top 3.
  - *Lone summary word (item 1).* A record that matches just one of the
    prompt's content terms, in its summary or identifiers but not as a name
    the prompt capitalizes, is incidental when another of the prompt's
    content terms is absent from the signet (two or more, as first planned,
    left 4 of 24 collisions; see [What changed while
    building](#what-changed-while-building)). Absent means in none of
    the records the hook's recall ranks: every current record and competing
    head in the visible scopes, the same set BM25 takes document frequencies
    from, not the top results. "What theme should we pick for a wedding?"
    ("theme"; "pick" and "wedding" absent) and "Recommend a good font for a
    resume." go through this path today. The rule 2.1.0 rejected dropped
    every lone summary match and lost "Which editor do I use?"; this one
    leaves a one-term question alone and needs the other terms absent, not
    merely unmatched by the record. A rough count on today's fixture fires
    it on no case question or paraphrase for its expected record.
  - *Newer decision first (item 5).* Two current records are on one subject
    when all of these hold:
    - they are in the same scope and of the same kind, both decisions or
      both preferences;
    - their summaries have the same content terms, after function words
      and inflections, or differ by at most one term of three or more
      (Jaccard at least 2/3 over summary content terms);
    - their bodies share at least 30% of their content terms (Jaccard over
      body content terms).

    On today's fixture no pair passes. The closest are "Editor" with
    "Editor theme" (summary 0.5, body 0.0) and "Tidepool primary database"
    with "Tidepool cache choice" (summary 0.2, body 0.38). When both
    records of a pair are in a recall result and the older ranks above the
    newer, the newer moves directly ahead of it; nothing else moves.
    "Newer" is the later `effective_from` of the current heads, then
    `recorded_at`, then revision ID. In the hook, if the older record is
    relevant and the newer isn't, the newer is added just ahead of it, so
    the hook never injects the old outcome without the new one. The older
    record is never added for the newer's sake. Neither record supersedes
    the other, both stay current, and recall's output shape doesn't change.
- **Byte default** (`internal/server/context.go`, `contextDefaultBytes`):
  changes only under the rule in [Bytes per
  prompt](#bytes-per-prompt-and-the-2048-byte-default).
- **Evaluation** (`internal/server/testdata/context-eval.json`,
  `internal/testfixture/evaluation.go`, `internal/server/context_eval_test.go`,
  `internal/server/context_calibration_test.go`):
  - new tuning sets `filler_short` (18), `generic_two_word` (12),
    `generic_guard` (6) and
    `newer_decisions` (2 pairs of same-scope decision records, 4 records
    added to the signet, with 6 queries that match both records of a pair,
    plus 2 queries that match only the older record, which must still find
    it);
  - the report and log gain the new counts, and `assertEvaluation` asserts
    every triage number in the table above;
  - a no-regression assertion for item 5: the evaluation runs with the
    newer-first rule on and off over the same signet, and fails if any
    existing case's expected record ranks lower with it on, in recall or
    in the hook;
  - a pairing tripwire: the test lists every pair the newer-first rule
    forms on the evaluation signet and fails on any pair outside
    `newer_decisions`;
  - `TestContextThresholdCurve` adds columns for the new sets, so the
    trade-off of each new rule is logged on the tuning split.
- **Unit tests** (`internal/server/context_test.go`,
  `internal/memory/memory_test.go` or a new `ranking_test.go`): each rule on
  a small synthetic signet, failing paths included (see [Proof](#proof)).
- **Docs**: `docs/retrieval.md` (hook selection rules, evaluation set,
  floors, curve and the new numbers, held-out exposure for this change),
  `docs/server.md` (the `memory_context` default, if it changes),
  `docs/releases/2.5.0.md`, and `VERSION` to 2.5.0.
- **Hook replay** (`docs/evidence/2.5/`): the replay script, its outputs
  and `hook-bytes.md` ([below](#bytes-per-prompt-and-the-2048-byte-default)).
- Not changed: signet layout, intents, sync, the state file, OAuth, and
  every file under `plugins/`.

### How `generic_two_word` is chosen

The fixture has 12 prompts, at least the triage's 10. They are written
before any rule runs on them, and none is answered by a record. Each has
two content terms after stop words. Every record that matches either term
matches just one of them, in its body and not its summary or identifiers,
and that term is on the generic-word list. So no prompt in it shares a
non-list word with any record. List words in record bodies on today's
signet include "time", "point", "minute", "week", "month", "list", "day"
and "year". The prompts come in two halves:

- 6 whose other term is absent from the signet ("Is 17 a prime number?"
  shape);
- 6 whose other term is also a list word, matched by a different record
  ("How many minutes are in a month?" shape).

The matched words come from at least 4 different records, and no prompt
reuses a word from `no_memory` or `collisions`. Step 1 posts, for each
prompt, every record that matches it and on which word, so the reviewer
can check the construction before any rule runs. A prompt that fails it is
replaced by one that meets it, before the rule is written, never after.

`generic_guard` has 6 prompts, each answered by a record. Each prompt
shares a list word with its record or another record, and a distinctive
non-list term with its answer, as in "What year is Postgres?". Each must
keep its expected record in the hook's top 3; a guard prompt that loses it
fails the evaluation.

### Plugin changes, by kind

This PR touches no file under `plugins/`, so it merges without waiting on
the owner's confirmation. The repository rule (#229, and the requester's
ruling on #250) sorts plugin changes into two kinds: (a) wording-only
changes that apply only when the server lists a tool, which may merge on
the reviewer's evidence from the live server, and (b) changes to hooks or
`.mcp.json`, or unconditional tool calls, which wait for the requester's
confirmation after the server upgrade.

- **Hook (`hooks/hooks.json`): no change.** Item 4's session handling is
  already shipped: since 2.1.1 (9b5c80f, the file's last change) the hook
  sends `"session": "${session_id}"`, and the native checks for #228
  ([evidence](../../evidence/2.1/claude-session.md)) showed Claude Code
  substitutes the session ID and the server de-duplicates per session. The
  hook doesn't send `budget_bytes`, so a new default reaches users through
  the server. Every change in this plan is in the server's handling of the
  same call. If any step finds the hook itself must change, that is a kind
  (b) change: the plan is updated in this PR, and that part merges only
  after the requester confirms their server runs 2.5.0.
- **Native re-run.** Re-running the #228 native check
  (`claude-session-native.sh`) needs a Claude Code credential in the
  environment, and none is available. It is not re-run, and the PR says so.
  The 2.1 native evidence stands for "the hook passes the session", because
  `hooks.json` hasn't changed since; the hook replay shows the 2.5.0
  candidate de-duplicating on the hook's input as defined.
- **`.mcp.json`, skills, `plugin.json`: no change.** The plugin stays 2.3.0
  on purpose.
- **Plugin README: not in this PR.** It states the byte default and is
  already stale ("1.5 KB by default" where the default is 2048). That
  correction, and any new default, goes in the post-release record PR. That
  PR touches a plugin file, so while #229 is open it merges only after the
  requester confirms the live server runs the version whose default the
  README states. That holds even if only "1.5 KB" changes to "2048", and
  the #250 exception doesn't cover it, since a byte figure isn't wording
  gated on a listed tool. It needs no `plugin.json` bump, because Claude
  Code doesn't load the README.

### Bytes per prompt and the 2048-byte default

On 2.4.0 code, over the 880 tuning prompts of the evaluation set (measured
for this plan with a throwaway test, `budget_bytes` set per run, removed
afterwards):

| Budget | Prompts injected | Average bytes per prompt (all / injecting) | p50 / p95 / max | Top-1 / top-3 | Top-3 answers clipped | Scripted session average |
| --- | --- | --- | --- | --- | --- | --- |
| 1024 | 844 | 679 / 708 | 664 / 1019 / 1023 | 78.9% / 87.4% | 0 of 769 | 400 |
| 1536 | 844 | 800 / 835 | 664 / 1522 / 1535 | 78.9% / 87.4% | 0 of 769 | 469 |
| 2048 (default) | 844 | 918 / 957 | 664 / 2031 / 2046 | 78.9% / 87.4% | 0 of 769 | 572 |

On the synthetic set the budget doesn't change a hit: the answers are
short, and the extra bytes are clipped long notes ranked under them. That
alone doesn't support a lower default, because real records are longer than
the fixture's answers, which is the case the 2.1.0 change to 2048 was for.

The measurement in real sessions follows the requester's ruling of
2026-11-11 (https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6801761482):
a replay of the plugin's hook counts as measuring "in real sessions", and no
Claude Code credential is needed. The ruling's "real hook script" means the
hook exactly as `hooks.json` defines it, since the hook is not a script. The
prompt set and session grouping below are fixed before any number is taken.

- **What the replay sends.** The replay (`docs/evidence/2.5/hook-replay.py`,
  synthetic data only) reads the `UserPromptSubmit` entry of
  `plugins/claude/hooks/hooks.json` as shipped: its tool and its input
  template, `{"event": "prompt", "prompt": "${prompt}", "session":
  "${session_id}"}`, and its 3 second timeout. It fills `${prompt}` and
  `${session_id}` as Claude Code does, a JSON string substitution. It sends
  the `tools/call` for `memory_context` over the real MCP HTTP endpoint of a
  loopback candidate server (`http://127.0.0.1:PORT/mcp`, the local form
  `.mcp.json`'s `server_url` takes), serving a copy of the synthetic
  evaluation signet. It uses the protocol Claude Code uses with this server
  (`2026-07-28`, sessionless, as recorded for #249). It never calls
  `contextOf` in process. The measured bytes are the length of the reply's
  `hookSpecificOutput.additionalContext`, which is what Claude Code adds to
  the turn, the same measure the evaluation uses.
- **Sessions, in this order, each with its own fixed synthetic session ID:**
  - S0: the evaluation's 20 scripted session prompts, in fixture order;
  - S1 to S8: one per scope, in the order scopes first appear in the
    fixture's records. Each holds, for every case whose answer is in that
    scope, in fixture order, the bare question and then the bare
    paraphrase; then the plain short questions answered in that scope; then
    the tuning no-memory prompts whose position in `no_memory.tune` is
    k − 1 modulo 8 for session Sk.

  Together they hold the 20 session prompts and each bare tuning prompt
  once: 88 case phrasings, 38 short questions and 62 no-memory prompts, 208
  in all. Wrapped prompts are left out because a real session doesn't
  repeat one question under 9 wrappers. That is also why the in-process 918
  bytes, over the 880 wrapped tuning prompts with no session, is not this
  measure.
- **Two runs per budget.** Each session runs once with its session ID and
  once with the `session` input empty. The second run is the same prompts
  with no de-duplication, so the difference is what per-session
  de-duplication removed.
- **Reported per run, in `docs/evidence/2.5/hook-bytes.md`:** injected
  bytes per prompt (average, maximum, and the share of prompts over 800
  bytes), overall and per session; the bytes de-duplication removed; and,
  per case prompt, whether the answer's line was injected whole, clipped or
  not at all. The 2048 runs always happen. The 1536 and 1024 runs use server
  builds that differ only in `contextDefaultBytes`, because the hook sends
  no `budget_bytes`.

**The 800-byte cap** (triage 4d) is judged on the 2048 run with session IDs,
the way the hook runs: the average over all 208 prompts must be at most 800.
`hook-bytes.md` and the PR report the no-session average next to that
judged number, so the share de-duplication removes is visible. For scale,
the reviewer's in-process run of this layout on 2.4.0 code (round 2, not
the replay) gave 339 bytes with session IDs and 627 without.
If it is over 800:

1. Selection changes come first: the relative cut and the packing of
   records after the top 3, chosen on the tuning split's curve, within every
   other triage number.
2. If no such change brings it under, the default comes down, to the
   largest of 1536 and 1024 at which the average is at most 800 and the
   default rule below holds.
3. If neither works, criterion 4d is reported as failed to the lead, with
   the numbers, and nothing ships until it is decided.

**The default rule**, set before measuring: the default stays 2048 unless
the replay's numbers support lowering it. They support it when, at the lower
budget, in both the replay and the evaluation, no answer that was whole at
2048 is clipped and no answer is lost. The default then comes down to the
smallest of 1536 and 1024 that meets that. The plan's numbers and the
replay's are reported either way, in `docs/retrieval.md` and the PR.

Limits, stated in the evidence: the signet is synthetic, so its record
lengths stand in for real ones, and no model answers the prompts. The
requester's private eval on a copy of their own memory covers real data on
the candidate before publication, and its injected-bytes figure must not
rise above their baseline. Its results stay out of the repository.

### Release

2.5.0, the next minor after 2.4.0 (the live server's version). The `VERSION`
bump and `docs/releases/2.5.0.md` are in this PR; `plugin.json` stays 2.3.0.
The PR says `Refs #232`. After merge, a candidate is built from the merge
commit (`bin/build-artifacts`) and verified (`bin/promote-local-candidate
--verify-only`, then the evaluation and the hook replay run on the candidate's
binary). The work then stops before `bin/sdlc-release
nominate`: the requester runs their private recall eval on that candidate,
on a copy of a signet, never the live one. Nomination and publication follow
only after they report back.

## Order of work

1. **Fixtures first, failing.** Add `filler_short`, `generic_two_word`,
   `generic_guard` and
   `newer_decisions` to the evaluation set and raise `assertEvaluation` to
   the triage numbers. Record every count on unchanged 2.4.0 code in the PR
   (the before column), and save a per-case report with
   `MANDALORE_EVAL_REPORT` for later comparison. Each new count is logged
   here and run once as an assertion to show it fails for the right reason;
   the step that fixes it makes it an assertion in `bin/ci`. Adding 4
   records moves BM25 statistics, so the 2.4.0 rates are re-measured on the
   new signet and the before column uses them. A drop under a triage floor
   from the fixture alone is fixed in the fixture's wording, never by
   lowering the floor. Before the generic-word rule is written, run the
   list check on all 38 plain short questions and the 88 tuning case
   phrasings, and post it ([above](#files-that-change)).
2. **Question clauses and stop words** (item 2). Unit tests, then the
   evaluation; log the curve.
3. **Generic word rule** (item 3), then **lone summary word rule** (item
   1). Unit tests, evaluation, curve for each.
4. **Newer decision first** (item 5). Unit tests on recall and the hook,
   the on/off no-regression assertion, the pairing tripwire, evaluation.
5. **Freeze and run held out** once: `MANDALORE_EVAL_HELDOUT=1`. If a
   held-out floor fails, that is reported, not tuned against; any change
   after it is a new round and its held-out run is counted as exposure.
6. **Bytes per prompt** with the hook replay on a build of the frozen
   head, as above, then the cap and default decisions. After merge the
   replay runs again on the candidate, which also shows its per-session
   de-duplication.
7. **Docs, release notes, `VERSION`**; `bin/ci`, `bin/sdlc check`,
   `govulncheck`. Simplify, then mark the PR ready.

Each step leaves `bin/ci` passing and the branch reviewable.

## Risks

- **Losing real answers to the new incidental rules.** A paraphrase whose
  unmatched words are absent from the signet could look like a collision.
  Contained by the triage floors on top-1/top-3 (tuning and held out), the
  plain short-question floor, the per-wrapper floor, the per-case
  comparison against 2.4.0 (each case that loses its answer is listed in
  the PR), and the requester's private eval before publication. The
  generic-word list is fixed in advance by its class rule and holds only
  generic nouns, so a distinctive answer word can't be on it. A two-term
  question whose answer hinges on a generic body word ("Which night is bin
  night?") can lose it. That shape is a stated limit in
  `docs/retrieval.md`.
- **Overfitting the tuning split.** Each new stop word must occur in a
  tuning set other than `filler_short`, the generic-word list follows a
  class rule fixed before any run, every rule is chosen on the tuning split alone, held out
  runs once per round after freezing, and the reviewer's sealed batches use
  wrappers and prompts this plan hasn't seen. Clauses are chosen over a
  longer stop-word list for that reason.
- **More recalls per prompt.** Up to 4 recalls on a multi-sentence prompt.
  The hook's 3 s timeout fails open. `BenchmarkMemoryContext` (1000 records)
  is run before and after; the warm path's bound in `bin/ci` stays as the
  spec sets it.
- **Newer-first misfiring.** A refinement ("Editor theme" beside "Editor")
  or two unrelated decisions with similar wording in one scope could pair
  up. The pairing needs near-identical summaries and overlapping bodies,
  the tripwire fails on any pair it forms in the evaluation signet outside
  `newer_decisions`, and a unit test holds the "Editor" and "Editor theme"
  shape apart. A wrong pair would reorder the two records and, in the hook,
  could add the newer one: one unasked-for record and its bytes, never the
  older one. A pair whose wording differs more than the bars allow keeps
  BM25 order, which is a stated limit. A head revised for a typo counts as
  newer; the release notes say so.
- **Byte default too low for long records.** Guarded by the rule above and
  the requester's eval; the default stays 2048 unless both support it.
- **Plugin installs follow main (#229).** Nothing here changes installed
  plugin behaviour, so merging isn't a rollout. Servers on 2.4.0 keep
  working with the unchanged plugin.

## Proof

- **Unit tests**, each with a failing path:
  - clauses: a wrapped question finds its answer; a no-answer question
    clause inside a wrapper that shares a summary word with a record injects
    nothing ("Quick one before we start: what theme should we pick for a
    wedding?"); a clause can raise a record but not drop one the whole
    prompt found relevant; a fourth clause is ignored; a prompt with no `?`
    behaves as today;
  - generic word: both `generic_two_word` shapes inject nothing; a
    `generic_guard`-shaped prompt keeps its answer; a two-word
    question on a distinctive body word ("Are we growing Sungold?") still
    injects; a bare-number match stays incidental;
  - lone summary word: a three-term collision injects nothing; a one-term
    question on the same summary word ("Which editor do I use?") still
    injects; a capitalized name still anchors;
  - newer first: the newer of a pair ranks first in recall and the hook; a
    pair in different scopes, or of different kinds, keeps BM25 order;
    "Editor" and "Editor theme" don't pair; an older-only query still finds
    the older record; equal `effective_from` falls back to `recorded_at`
    then ID; the hook adds the newer when only the older is relevant, and
    doesn't add the older when only the newer is.
- **Evaluation** (`TestContextSelectionEvaluation` in `bin/ci`): every
  triage number in the table, on the tuning split; held out with
  `MANDALORE_EVAL_HELDOUT=1` once per round. Before and after numbers and
  the curve go in the PR and `docs/retrieval.md`.
- **Item 5 no-regression**: asserted in `bin/ci`, with the newer-first rule
  on against off; any existing case whose expected record ranks lower fails.
- **Per-case comparison** for the other rules: `MANDALORE_EVAL_REPORT` from
  2.4.0 code and from the head, diffed per case; every case that loses rank
  is listed in the PR, and the triage floors apply.
- **Hook replay** on the frozen head's build and again on the candidate:
  the hook from `hooks.json` as shipped, over the real MCP endpoint, one
  session ID per session, de-duplication within a session and none across
  sessions, the 800-byte cap, and the bytes-per-prompt report, with the
  script and outputs under `docs/evidence/2.5/`.
- **Not done, and said so in the PR**: the #228 native Claude Code re-run,
  which needs a credential (see [Plugin changes](#plugin-changes-by-kind)).
- **Repository checks**: `bin/ci` (host form with `CLAUDE_CONFIG_DIR`
  unset, as there's no TTY for the container), `bin/sdlc check`,
  `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`, with
  `GOFLAGS=-count=1`. `bin/check-old-reader` isn't needed: the signet
  layout, intents and sync paths don't change.
- **What the verifier exercises on the real artifact**: the candidate
  binary from the merge commit, serving a copy of the synthetic evaluation
  signet: the reviewer's sealed batches through `memory_context` and
  `memory_recall`, the hook replay, and `--version` reporting
  2.5.0.

## Rollback

Reinstall 2.4.0 on the server (atomic swap plus `launchctl kickstart -k`).
Nothing is persisted by this change: no signet format, schema, intent or
state-file field changes, and the session memory is in-process, so a
rollback loses only the current sessions' de-duplication, as any restart
does. The plugin is unchanged, so it needs no rollback. If the byte default
came down, 2.4.0 restores 2048. No data or release step makes undoing
harder; a published 2.5.0 stays in the release history, and the release
notes name 2.4.0 as the rollback target.
