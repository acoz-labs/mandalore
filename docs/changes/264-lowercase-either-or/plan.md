# Plan: keep the known name's records on lowercase either-or prompts

## Issue

Refs #264 (https://github.com/acoz-labs/mandalore/issues/264). Outcome: a
bare either-or prompt written in lowercase ("kopia or borg?", "borg or
kopia?", "kopia vs borg", "borg vs kopia") injects the known name's records
again, as 2.4.0 did, while every capitalised form, every other prompt and
`memory_recall` behave exactly as in 2.5.0. Ships as patch release 2.5.1,
server only. No file under `plugins/` changes.

### Cause, confirmed on main

The hook's lone-summary-word rule (2.5.0, #232) drops a record whose only
match with the prompt is one summary word, when another prompt term is in no
record, unless that word is a name or the prompt's lead word. A word is a
name only when it's capitalised mid-sentence. The lead word (PR #263) is the
first word, only when it's capitalised and every other term is a name. In
an all-lowercase prompt neither holds, so the match on "kopia" is dropped
because "borg" is in no record.

A throwaway test on 6bed590 (removed, not committed), with the issue's three
Kopia-summary records and 30 filler notes:

| Prompt | Records injected |
| --- | --- |
| "Kopia or Borg?", "Borg or Kopia?", "Kopia vs Borg", "Borg vs Kopia" | 3 each |
| "kopia or borg?", "borg or kopia?", "kopia vs borg", "borg vs kopia" | 0 each |
| "kopia or Borg?", "Kopia or borg?" (mixed case) | 0 each |
| "kopia?" | 3 |

On the same build, lowercase forms of the 8 `either_or_limit` pairs, where
X is only in the answer's body and Y is in no record, already keep the
answer at rank 1 in every order ("helix or neovim?", "neovim or helix?",
"helix vs neovim"). That's the case a fix must not break.

### Rejected: judge the prompt as if it were capitalised

Title-casing a bare either-or prompt before naming its words would make
"kopia or borg?" behave like "Kopia or Borg?", which injects 3. But it would
also make "borg" a name no record holds. The hook then needs a summary
match, and the lowercase body-only shape above would fall from 8 of 8 to
0 of 8, as "Helix or Neovim?" does today. That's a per-prompt regression
the contract forbids.

## Files that change

- **Hook selection** (`internal/server/context.go`): `lead` becomes `leads`,
  returning a list. It keeps today's rule unchanged and adds one: when the
  prompt is a bare either-or, every term in it that isn't a name is a lead.
  - *Bare either-or*: the prompt's words, with trailing `?`, `.` or `!`
    dropped, alternate between a word and a connective, `or` or `vs` (also
    `vs.`), in any case, starting and ending with a word, with at least two
    words. Each word must give exactly one content term. "kopia or borg?",
    "borg vs kopia" and "kopia or borg or restic?" qualify; "should i use
    kopia or borg?", "kopia versus borg" and "this or that?" (stop words) don't.
  - *Why capitalised forms can't change*: in an all-capitalised bare
    either-or, every word after the first is a name, so the only non-name
    term is the first word, which today's rule already makes the lead. The
    new rule returns the same single lead, so the query is identical.
  - *What a lead does* (unchanged from 2.5.0): a lone summary match on it
    isn't incidental, it anchors nothing, and it never counts as a name the
    signet lacks. So "borg" in "kopia or borg?" doesn't trigger the hook's
    rule for absent names, and the lowercase body-only shape is judged as
    today.
  - Mixed case ("kopia or Borg?") is covered by the same rule: "kopia" is a
    lead and "Borg" stays a name, as in "Kopia or Borg?".
- **Recall query** (`internal/memory/memory.go`, `internal/memory/service.go`):
  `Query.Lead string` becomes `Query.Leads []string`, and
  `RecallAllNamed` takes the list; `bm25` checks membership in it instead of
  equality. Only the hook sets it, so `memory_recall` builds the same query
  as before.
- **Tests**:
  - `TestMemoryContextEitherOrOnASummaryName` (`internal/server/context_test.go`)
    adds the four lowercase prompts, "Borg vs Kopia", and the two mixed-case
    forms, each asserting the 3 Kopia records. It's the tripwire, and fails on
    6bed590 for the lowercase and mixed rows.
  - `TestEitherOrOrders` (`internal/server/context_calibration_test.go`)
    adds the lowercase forms of all 8 `either_or_limit` pairs, "x or y?",
    "y or x?", "x vs y" and "y vs x", each asserting the answer in the top 3.
    This fails if the fix makes the lowercase second word an absent name, as
    the rejected title-casing would.
  - `TestContextLead` becomes `TestContextLeads`: the existing rows keep
    their answers (the existing `"restic or Borg?"` row changes from no
    lead to `restic`, the one intended change), plus lowercase, three-word,
    `vs.`, "versus", stop-word and sentence rows that get no lead.
  - A unit test in `internal/memory` that a lowercase lone summary match is
    kept when it's in `Leads` and dropped when it isn't, with an absent
    other term.
- **Docs**: `docs/retrieval.md` (the either-or limit paragraph and the
  2.5.1 notes), `docs/releases/2.5.1.md`, `docs/evidence/2.5/either-or.md`
  (a 2.5.1 section with the matrix and the per-prompt comparison), and
  `VERSION` to 2.5.1. README and install pointers move in the
  post-publication record PR, as for 2.5.0.
- Not changed: the evaluation fixture (`context-eval.json`), every tuning
  floor in `assertEvaluation`, the signet layout, intents, sync, the state
  file, and every file under `plugins/` (`plugin.json` stays 2.3.0).

## Order of work

1. **Tripwire first, failing.** Add the test rows above and run them on
   unchanged code: the lowercase and mixed-case rows of
   `TestMemoryContextEitherOrOnASummaryName` fail, the lowercase rows of
   `TestEitherOrOrders` pass. Save the baseline evaluation report with
   `MANDALORE_EVAL_REPORT` on 6bed590.
2. **The fix**: `leads`, `Query.Leads`, the `bm25` membership check, and
   the unit tests. The tripwire passes.
3. **No-regression evidence** (see [Proof](#proof)): the per-prompt
   comparison, the hook replay comparison, and the either-or matrix.
4. **Docs, release notes, `VERSION`**; `bin/ci`, `bin/sdlc check`,
   `govulncheck`. Simplify, then mark the PR ready.

Each step leaves the branch building and reviewable; `bin/ci` passes from
step 2.

## Risks

- **A lowercase bare either-or that names nothing in the signet injects.**
  A lead word's lone summary match now counts, so "tea or coffee?" could
  bring a record whose summary has "coffee" when 2.5.0 dropped it. That's
  what 2.4.0 did, and what "Coffee or Tea?" does today. Contained by the
  per-prompt comparison over every no-memory, collision and guard prompt,
  which must show 0 differences, and by the requester's private eval.
- **Breaking the lowercase body-only shape.** Contained by keeping leads out
  of absent names, and asserted for all 8 pairs in four phrasings.
- **Changing capitalised forms.** Contained by the construction above and
  asserted by the unchanged `TestContextLeads` rows and the zero-difference
  comparison.
- **Three-word lowercase prompts.** Every word becomes a lead and none
  anchors, so "beacon or cortana or siri?" may inject what neither
  capitalised order does. The matrix reports it. Order effects with three
  names remain a stated limit.
- **Main moves under this PR** (#259 is in flight). The baseline report is
  regenerated on the merge base used for review, and main is merged in, never
  rebased or force-pushed.

## Proof

- **The contract's four shapes**: `TestMemoryContextEitherOrOnASummaryName`
  in `bin/ci`, 3 records each, failing on 6bed590.
- **No per-prompt regression on any existing set**, three ways:
  1. *The evaluation, prompt by prompt.* `MANDALORE_EVAL_REPORT` from
     `TestContextSelectionEvaluation` at the merge base and at the head, on
     the same fixture, compared with `docs/evidence/2.5/eval-diff.py`. The
     report holds every tuning set (88 cases in 10 wrappers, recall and the
     hook, short, filler, no-memory, collisions, generic, guard, named
     guard, `either_or_limit`, newer decisions, session bytes). The bar is
     `differences: 0`. Because the comparison is per prompt, every 2.5.0
     tuning bar is met with the same numbers. The held-out split isn't run:
     the change is decided without it and the zero-difference bar needs no
     tuning.
  2. *The hook replay.* `docs/evidence/2.5/hook-replay.py` at the 2048
     default, with and without session IDs, on builds of the merge base and
     the head, over the real MCP endpoint. Every row must be identical.
  3. *Recall.* The report's recall ranks are part of the comparison; the
     code path is untouched because only the hook sets `Leads`.
- **The either-or matrix**, recorded in the evidence: the Kopia repro and the
  8 `either_or_limit` pairs, in capitalised, lowercase and mixed case, both
  orders, with `or` and `vs`, plus the three-name Beacon prompts in lowercase,
  on 6bed590 and the head.
- **For the requester's private eval**: it runs on the candidate built from
  the merge commit, over a copy of the live signet. Its gate is all 24
  either-or cases in the top 3 and no change on the other sets. Nothing in
  this PR uses or looks for those cases or the sealed probes.
- **Repository checks**: `bin/ci` on the host with `CLAUDE_CONFIG_DIR`
  unset (the container needs a TTY), `bin/sdlc check`, and
  `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`, with
  `GOFLAGS=-count=1`. `bin/check-old-reader` isn't needed: the signet
  layout, intents and sync paths don't change.
- **What the verifier exercises on the real artifact**: the candidate binary
  from the merge commit, serving the synthetic evaluation signet plus the
  Kopia records: the four lowercase prompts and their capitalised forms
  through `memory_context`, `memory_recall` on the same prompts unchanged
  from v2.5.0, and `--version` reporting 2.5.1.

## Rollback

Reinstall 2.5.0 on the server (atomic swap plus `launchctl kickstart -k`).
Nothing is persisted: no signet format, intent, schema or state-file field
changes. The plugin is unchanged. Rolling back brings back only the
lowercase either-or loss. The release notes name 2.5.0 as the rollback
target.
