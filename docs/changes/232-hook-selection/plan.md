# Plan: make the prompt hook's selection robust to filler, collisions and generic words

## Issue

Refs #232 (https://github.com/acoz-labs/mandalore/issues/232). Outcome: the
`memory_context` prompt hook injects less on prompts no record answers,
finds short questions inside filler, ignores a single generic shared word,
and ranks the newer of two same-scope decisions on one subject first, with
no regression in recall or context rankings. Ships as minor release 2.5.0,
server only. The plugin's hook, `.mcp.json`, skills and `plugin.json` don't
change (see [Plugin changes, by kind](#plugin-changes-by-kind)).

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
| 3 | Generic single-word matches: 0 injections on at least 10 two-word prompts sharing one generic body word | not measured | new `generic_two_word` fixture (12 prompts); asserted 0 |
| 4a | `memory_context` top-1/top-3 at least 78%/86% tuning | 78.9% / 87.4% | `rates` over the tuning split; floors move from 75/85 to 78/86 |
| 4b | … and at least 75%/84% held out | 77.3% / 84.6% | held-out split; floors move from 65/80 to 75/84 |
| 4c | `memory_recall` top-1/top-3 at least 75%/90% tuning | 77.3% / 93.2% | floors move from 72/88 to 75/90 |
| 4d | Session prompts average at most 800 injected bytes | 572 | scripted 20-turn session; the existing 800-byte tripwire |
| 4e | Bytes per prompt measured in real sessions and reported; the 2048-byte default changes only if the measurement supports it | see [Bytes per prompt](#bytes-per-prompt-and-the-2048-byte-default) | native session runs plus the evaluation's byte table |
| 5 | Two same-scope decisions on one subject: the newer ranks first for queries matching both, and no existing case regresses | not measured | new `newer_decisions` fixture; per-case comparison report against 2.4.0 |

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
    previous sentence boundary (`.`, `!`, `?`, `:`, `;`, a newline). A record
    is relevant if it clears the bars on the whole prompt or on a clause.
    "Before we start, remind me: Which editor do I use? One line is fine."
    is then judged on "Which editor do I use?" as well, so wrapper words
    that aren't in the stop-word list ("remind", "start", "fine") can't dilute
    it. At most 3 clauses are scored, so a prompt costs at most 4 recalls. A
    one-sentence prompt is its own only clause and is scored once, as today.
  - *Stop words (item 2).* A few more request words join the hook's
    stop-word list ("remind", "fine", "right", "okay", "please" and the like),
    each one occurring in the tuning split, as the current list's rule
    requires. Clauses do most of the work; the list catches wrappers without
    a question mark ("remind me where Unbound listens").
  - *Ranking only, no format change.* `ContextInput`, the output envelope
    and the session de-duplication stay as they are.
- **Relevance rules** (`internal/memory/memory.go`, `bm25` and
  `recallHeads`; `internal/memory` exports stay hook-agnostic):
  - *Generic word (item 3).* A two-term prompt's lone body-only match is
    incidental when its other term is in no candidate record: the prompt is
    about something the signet lacks. "Is 17 a prime number?" ("prime" is in
    no record; "number" is in the mail signature's body) then injects
    nothing, while "Are we growing Sungold?" still counts, because "growing"
    is in the signet. Today's exemption for a two-term prompt applies only
    when both terms are known to the signet.
  - *Lone summary word (item 1).* A record that matches just one of the
    prompt's content terms, in its summary but not as a name the prompt
    capitalizes, is incidental when two or more of the prompt's other
    content terms are in no candidate record. "What theme should we pick for
    a wedding?" and "How do I install Redis on Ubuntu?" go through this path
    today. The rule 2.1.0 rejected dropped every lone summary match and lost
    "Which editor do I use?"; this one keeps a one-term question and needs
    the unmatched terms to be absent from the signet, not merely unmatched
    by the record.
  - *Newer decision first (item 5).* Two current records are on one subject
    when they are in the same scope, are of the same kind (both decisions or
    both preferences), and at least half of their summaries' content terms
    are shared (Jaccard over summary terms without function words, a bar
    tuned on the fixture). When both are in a recall result and the older
    one ranks above the newer, the newer moves directly ahead of it; nothing
    else moves. "Newer" is the later `effective_from` of the current heads,
    then `recorded_at`, then revision ID. The hook treats such a pair as
    one: if either is relevant, both are, in that order, so the hook never
    injects the old outcome without the new one. Neither supersedes the
    other and both stay current; recall's output shape doesn't change.
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
  - `TestContextThresholdCurve` adds columns for the new sets, so the
    trade-off of each new rule is logged on the tuning split.
- **Unit tests** (`internal/server/context_test.go`,
  `internal/memory/memory_test.go` or a new `ranking_test.go`): each rule on
  a small synthetic signet, failing paths included (see [Proof](#proof)).
- **Docs**: `docs/retrieval.md` (hook selection rules, evaluation set,
  floors, curve and the new numbers, held-out exposure for this change),
  `docs/server.md` (the `memory_context` default, if it changes),
  `plugins/claude/README.md` (the byte default it states; see below),
  `docs/releases/2.5.0.md`, and `VERSION` to 2.5.0.
- Not changed: signet layout, intents, sync, the state file, OAuth, and the
  plugin's `hooks/hooks.json`, `.mcp.json`, skills and `plugin.json`.

### Plugin changes, by kind

The repository rule (#229, and the requester's ruling on #250) sorts plugin
changes into two kinds: (a) wording-only changes that apply only when the
server lists a tool, which may merge on the reviewer's evidence from the live
server, and (b) changes to hooks or `.mcp.json`, or unconditional tool calls,
which wait for the requester's confirmation after the server upgrade.

- **Hook (`hooks/hooks.json`): no change.** Item 4's session handling is
  already shipped: since 2.1.1 the hook sends
  `"session": "${session_id}"`, and the native checks for #228
  ([evidence](../../evidence/2.1/claude-session.md)) showed Claude Code
  substitutes the session ID and the server de-duplicates per session. This
  change re-runs that check against the 2.5.0 candidate (step 6) rather than
  editing the hook. The hook doesn't send `budget_bytes`, so a new default
  reaches users through the server, not the plugin. Every other change in
  this plan is in the server's handling of the same call. If any step finds
  the hook itself must change, that is a kind (b) change: the plan is
  updated in this PR, and that part merges only after the requester
  confirms their server runs 2.5.0.
- **`.mcp.json`, skills, `plugin.json`: no change.** The plugin stays 2.3.0
  on purpose.
- **Plugin README (`plugins/claude/README.md`): wording only, and only if
  the default changes.** It says "at most 2048 bytes" and, already stale,
  "1.5 KB by default". It is documentation Claude Code doesn't load, it
  calls no tool and changes no behaviour, so it is neither kind; with no
  `plugin.json` bump, installs don't pick it up. The stale "1.5 KB" is
  corrected either way. The reviewer can ask to keep it out of this PR.

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

The measurement in real sessions: native Claude Code sessions with the
plugin, against a local 2.5.0 candidate serving a copy of the synthetic
evaluation signet, through the logging proxy from #228
(`docs/evidence/2.1/claude-session-proxy.py`). One scripted multi-turn
session (the evaluation's 20 session prompts, then a mix of case questions,
short questions and no-answer prompts) runs once per budget, the proxy
setting `budget_bytes` on the hook's call so the plugin isn't edited. Each
run records per prompt the injected bytes, whether the answer's line was
clipped, and whether the model's answer was right. Results go in
`docs/evidence/2.5/hook-bytes.md` with the scripts, as for #228.

The rule, set before measuring: the default comes down to the smallest of
1536 and 1024 at which, in both the native runs and the evaluation, no top-3
answer that was whole at 2048 is clipped and no answer is lost; otherwise it
stays 2048. The plan's numbers and the native numbers are reported either
way, in `docs/retrieval.md` and the PR.

Two limits, stated in the evidence: the signet is synthetic, so its record
lengths stand in for real ones; and the requester's private recall eval on
the candidate (below) is where real record lengths are checked. Its results
stay out of the repository.

**Access needed.** The native runs need a Claude Code credential in the
environment (`CLAUDE_CODE_OAUTH_TOKEN` or `ANTHROPIC_API_KEY`), because the
script isolates `HOME` and `CLAUDE_CONFIG_DIR` and never uses the live
login. Batvault has no entry for one. The variable is documented in the
evidence script; the PR asks for a Batvault entry. Without it, step 6's
native runs can't happen and the default stays 2048.

### Release

2.5.0, the next minor after 2.4.0 (the live server's version). The `VERSION`
bump and `docs/releases/2.5.0.md` are in this PR; `plugin.json` stays 2.3.0.
The PR says `Refs #232`. After merge, a candidate is built from the merge
commit (`bin/build-artifacts`) and verified (`bin/promote-local-candidate
--verify-only`, then the evaluation and the native session check run on the
candidate's binary). The work then stops before `bin/sdlc-release
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
   the step that fixes it makes it an assertion in `bin/ci`. Adding 4 records moves BM25 statistics, so the
   2.4.0 rates are re-measured on the new signet and the before column uses
   them; a drop under a triage floor from the fixture alone is fixed in the
   fixture's wording, never by lowering the floor.
2. **Question clauses and stop words** (item 2). Unit tests, then the
   evaluation; log the curve.
3. **Generic word rule** (item 3), then **lone summary word rule** (item
   1). Unit tests, evaluation, curve for each.
4. **Newer decision first** (item 5). Unit tests on recall and the hook,
   evaluation, per-case comparison against step 1's report.
5. **Freeze and run held out** once: `MANDALORE_EVAL_HELDOUT=1`. If a
   held-out floor fails, that is reported, not tuned against; any change
   after it is a new round and its held-out run is counted as exposure.
6. **Bytes per prompt** natively, as above, and the default decision.
   Re-run the #228 session check (`claude-session-native.sh`) against the
   candidate build.
7. **Docs, release notes, `VERSION`**; `bin/ci`, `bin/sdlc check`,
   `govulncheck`. Simplify, then mark the PR ready.

Each step leaves `bin/ci` passing and the branch reviewable.

## Risks

- **Losing real answers to the new incidental rules.** A paraphrase whose
  unmatched words are absent from the signet could look like a collision.
  Contained by the triage floors on top-1/top-3 (tuning and held out), the
  plain short-question floor, the per-wrapper floor, the per-case
  comparison against 2.4.0 (each case that loses its answer is listed in
  the PR), and the requester's private eval before publication.
- **Overfitting the tuning split.** Each new stop word must occur in the
  tuning split, every rule is chosen on it alone, held out runs once per
  round after freezing, and the reviewer's sealed batches use wrappers and
  prompts this plan hasn't seen. Clauses are chosen over a longer stop-word
  list for that reason.
- **More recalls per prompt.** Up to 4 recalls on a multi-sentence prompt.
  The hook's 3 s timeout fails open. `BenchmarkMemoryContext` (1000 records)
  is run before and after; the warm path's bound in `bin/ci` stays as the
  spec sets it.
- **Newer-first misfiring.** Two unrelated decisions with similar summaries
  in one scope would swap order. The swap moves one record by one place
  relative to its pair and never adds a record the query doesn't match. A
  head revised for a typo counts as newer; the release notes say so.
- **Byte default too low for long records.** Guarded by the rule above and
  the requester's eval; the default stays 2048 unless both support it.
- **Plugin installs follow main (#229).** Nothing here changes installed
  plugin behaviour, so merging isn't a rollout. Servers on 2.4.0 keep
  working with the unchanged plugin.

## Proof

- **Unit tests**, each with a failing path:
  - clauses: a wrapped question finds its answer; a no-answer clause inside
    a wrapper injects nothing; a fourth clause is ignored; a prompt with no
    `?` behaves as today;
  - generic word: "Is 17 a prime number?"-shaped prompt injects nothing;
    a two-word question whose other word is in the signet still injects;
    a bare-number match stays incidental;
  - lone summary word: a three-term collision injects nothing; a one-term
    question on the same summary word ("Which editor do I use?") still
    injects; a capitalized name still anchors;
  - newer first: the newer of a pair ranks first in recall and the hook; a
    pair in different scopes, or of different kinds, keeps BM25 order; an
    older-only query still finds the older record; equal `effective_from`
    falls back to `recorded_at` then ID; the hook injects both when either
    is relevant.
- **Evaluation** (`TestContextSelectionEvaluation` in `bin/ci`): every
  triage number in the table, on the tuning split; held out with
  `MANDALORE_EVAL_HELDOUT=1` once per round. Before and after numbers and
  the curve go in the PR and `docs/retrieval.md`.
- **Per-case comparison**: `MANDALORE_EVAL_REPORT` from 2.4.0 code and from
  the head, diffed per case; every case that loses rank is listed.
- **Native**: the #228 session check on the candidate (session ID passed,
  de-duplication across `--resume`, older-server fail-open unchanged) and
  the bytes-per-prompt runs, with scripts and outputs under
  `docs/evidence/2.5/`.
- **Repository checks**: `bin/ci` (host form with `CLAUDE_CONFIG_DIR`
  unset, as there's no TTY for the container), `bin/sdlc check`,
  `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`, with
  `GOFLAGS=-count=1`. `bin/check-old-reader` isn't needed: the signet
  layout, intents and sync paths don't change.
- **What the verifier exercises on the real artifact**: the candidate
  binary from the merge commit, serving a copy of the synthetic evaluation
  signet: the reviewer's sealed batches through `memory_context` and
  `memory_recall`, the native session check, and `--version` reporting
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
