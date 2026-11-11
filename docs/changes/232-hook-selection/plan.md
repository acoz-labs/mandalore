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
| 3 | Generic single-word matches: 0 injections on at least 10 two-word prompts sharing one generic body word | not measured | new `generic_two_word` fixture (12 prompts, chosen as [below](#how-generic_two_word-is-chosen)); asserted 0 |
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
    one body word from a fixed list of common English words is incidental
    whenever the prompt has another content term. The list is Ogden's Basic
    English (850 words, 1930), a bare list of everyday words, credited in
    `NOTICE`. It is fixed before any run, and no word is added or removed by
    looking at evaluation outcomes. A word counts when it or its base under
    the existing inflection rule (-s, -es, -ed, -ing) is on the list.
    "Is 17 a prime number?" ("number", in the mail signature's body) and
    "How many days are in 30 weeks?" ("day", "week") then inject nothing. The
    eight plain short questions of that two-term shape should keep their
    answers, because their matched word is distinctive ("Sungold",
    "Postgres", "burst", "eu-central", "Zsh", "metric", "cameras"), or the
    prompt's other word is a stop word ("Are short flags okay?"). Step 1
    checks each of the eight against the list before the rule is written;
    one that would be lost is reported in the PR, not fixed by editing the
    list. The 2.1.0 rules stay: a lone body match on a bare number, or
    with two or more other terms unmatched, is still incidental.
  - *Lone summary word (item 1).* A record that matches just one of the
    prompt's content terms, in its summary or identifiers but not as a name
    the prompt capitalizes, is incidental when two or more of the prompt's
    other content terms are absent from the signet. Absent means in none of
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
  - new tuning sets `filler_short` (18), `generic_two_word` (12) and
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
  `docs/releases/2.5.0.md`, `NOTICE` (the word list's credit), and
  `VERSION` to 2.5.0.
- **Hook replay** (`docs/evidence/2.5/`): the replay script, its outputs
  and `hook-bytes.md` ([below](#bytes-per-prompt-and-the-2048-byte-default)).
- Not changed: signet layout, intents, sync, the state file, OAuth, and
  every file under `plugins/`.

### How `generic_two_word` is chosen

The 12 prompts are written before any rule runs on them, and none is
answered by a record. Each has two content terms after stop words. One of
them is a word on the common-English list that occurs in some record's body
but not its summary. The prompts come in two halves:

- 6 whose other term is absent from the signet ("Is 17 a prime number?"
  shape);
- 6 whose other term is in the signet, but not in the record the generic
  word matches ("How many days are in 30 weeks?" shape, where both words
  are elsewhere in the signet).

The matched words come from at least 4 different records, and no prompt
reuses a word from `no_memory` or `collisions`. A shape the rule can't
reach, such as a generic word that is also in a summary, is a limit stated
in `docs/retrieval.md`, not left out of the fixture.

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
  correction, and any new default, goes in the post-release record PR, which
  is subject to the plugin rule then.

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

1. **Fixtures first, failing.** Add `filler_short`, `generic_two_word` and
   `newer_decisions` to the evaluation set and raise `assertEvaluation` to
   the triage numbers. Record every count on unchanged 2.4.0 code in the PR
   (the before column), and save a per-case report with
   `MANDALORE_EVAL_REPORT` for later comparison. Each new count is logged
   here and run once as an assertion to show it fails for the right reason;
   the step that fixes it makes it an assertion in `bin/ci`. Adding 4
   records moves BM25 statistics, so the 2.4.0 rates are re-measured on the
   new signet and the before column uses them. A drop under a triage floor
   from the fixture alone is fixed in the fixture's wording, never by
   lowering the floor.
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
  generic-word list is fixed in advance and holds only everyday words, so a
  distinctive answer word can't be on it, but a record whose answer hinges
  on an everyday body word ("Which day is bin night?") can lose it. That
  shape is a stated limit in `docs/retrieval.md`.
- **Overfitting the tuning split.** Each new stop word must occur in a
  tuning set other than `filler_short`, the generic-word list is fixed
  before any run, every rule is chosen on the tuning split alone, held out
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
  - generic word: both `generic_two_word` shapes inject nothing; a two-word
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
