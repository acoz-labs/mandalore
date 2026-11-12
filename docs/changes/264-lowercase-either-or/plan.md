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

### Course change at code round 0: the rule only adds

Two changes to the plan below came from the probes, both ruled by the lead
on 2026-11-12 before code review.

- **The probe bar has two halves.** The bar as first planned (a lowercase
  form injects no record its capitalised form doesn't) fails on 6bed590
  itself. "panel or turbine?" injects 4 body matches on "panel", and "Panel
  or Turbine?" injects 3 others, because "Turbine" is then a name no record
  holds ([before-fix results](https://github.com/acoz-labs/mandalore/pull/270#issuecomment-6832291843)).
  That is the lowercase body-only shape this plan keeps. The bar, asserted
  per probe in `bin/ci`: (a) every record the head injects that the rule-off
  run (2.5.0 behaviour) doesn't is injected by the capitalised form; (b)
  every record the rule-off run injects is still injected on the head.
- **The hook judges a reached prompt twice.** With the leads passed in a
  single judgement, 2 of the 28 probes failed. In "panel or turbine?" the
  rescued summary matches outscored the body matches, and 3 of 2.5.0's 4
  fell under the hook's relative cut (0.65 of the best), failing (b). In
  "captain or pirate?" the rescued summary match lifted the best over the
  bar, and a body match on "captain" (the hotfix policy) rode in under the
  relative cut, which "Captain or Pirate?" doesn't inject, failing (a). The
  lead chose option 1 of three: the rule only adds. A query the rule reaches
  is recalled once as 2.5.0 does and once with its leads. The hook injects
  everything the first judgement keeps, in its order, then the second
  judgement's records (and conflicts) that match the prompt in a summary or
  identifiers. Both halves are still asserted per probe, since (a) doesn't
  hold by construction: a summary match the capitalised form cuts could
  still be added.
- **Code review round 1: how additions merge.** Two fixes, both found by
  the reviewer:
  - An added older decision now brings the newer record `choose` paired
    with it, just ahead, so the hook never gives the old outcome alone.
  - Additions no longer join the top records' packing, where they took an
    even share of the room and clipped a 2.5.0 line. When 2.5.0 injects
    anything, its output is built as before, and the additions follow as
    whole lines in the room left, ahead of the scopes. When it injects
    nothing, the additions are packed as usual.

  Half (b) of the probe bar now also asserts that every line 2.5.0 gives
  is in the head's output, unchanged and in order.
- **Code review round 2: a pair fits whole.** An added record and the newer
  record ahead of it are one unit in the room 2.5.0 leaves, injected
  together or not at all. Before, a newer too long for the room was skipped
  and its shorter older went in alone.

## Files that change

- **Hook selection** (`internal/server/context.go`): `lead` stays as it is.
  A new `eitherOrLeads` returns the rule's leads: when the prompt is a bare
  either-or, every term in it that isn't a name. It returns nothing when
  `lead` already gives the same single lead, which is every capitalised
  form. `relevant` recalls each query (the whole prompt and each question
  clause) as 2.5.0 does. Only when `eitherOrLeads` reaches one of them does
  it recall that query again with the leads and add from that second
  judgement as the [course change](#course-change-at-code-round-0-the-rule-only-adds)
  says. A package switch, `contextEitherOrLeads`, turns the rule off for the
  probe test's 2.5.0 run, as `memory.RankNewerFirst` does for newer-first.
  - *Bare either-or*: the prompt's words, with trailing `?`, `.` or `!`
    dropped, alternate between a word and a connective, `or` or `vs`, in any
    case, starting and ending with a word, with at least two words. Each
    word must give exactly one content term. "kopia or borg?", "borg vs
    kopia" and "kopia or borg or restic?" qualify. "should i use kopia or
    borg?", "kopia versus borg", "go or rust?" ("go" gives no term) and
    "this or that?" (stop words) don't, and keep 2.5.0's behaviour.
  - *`vs.` is outside the shape* (the lead's ruling at plan review).
    `names` takes a word ending in `.` as a sentence end, so in "Kopia vs.
    Borg" the word "Borg" isn't a name, and 2.5.0 injects nothing.
    Accepting `vs.` would change that capitalised form, so "X vs. Y" and
    "x vs. y" stay exactly as in 2.5.0, and the release notes say so.
  - *Why capitalised forms can't change*: in an all-capitalised bare
    either-or with `or` or `vs`, every word after the first is a name, so
    the only non-name term is the first word, which today's rule already
    makes the lead. `eitherOrLeads` then returns nothing, so the prompt
    takes the single 2.5.0 judgement. Every other capitalised prompt, `vs.`
    included, isn't the bare shape and takes it too.
  - *Wrapped prompts.* The rule judges the whole prompt and each question
    clause. "In one line: kopia or borg?" is fixed through its clause
    "kopia or borg?". A wrapper without a `?` clause ("quick one, kopia vs
    borg") isn't the bare shape and keeps 2.5.0's behaviour. The matrix
    measures both, and the release notes state the boundary.
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
    adds the four lowercase prompts, "Borg vs Kopia", the two mixed-case
    forms and "In one line: kopia or borg?", each asserting the 3 Kopia
    records. It's the tripwire, and fails on 6bed590 for the lowercase,
    mixed and wrapped rows. "Kopia vs. Borg" and "Borg vs. Kopia" are
    asserted at 0 records, as on 6bed590, so `vs.` stays outside the fix.
  - `TestEitherOrOrders` (`internal/server/context_calibration_test.go`)
    adds the lowercase forms of all 8 `either_or_limit` pairs, "x or y?",
    "y or x?", "x vs y" and "y vs x", each asserting the answer in the top 3.
    This fails if the fix makes the lowercase second word an absent name, as
    the rejected title-casing would.
  - `TestContextLead` stays unchanged. A new `TestEitherOrLeads` covers
    the rule: every non-name word is a lead in "kopia or borg?" (`kopia`,
    `borg`), "borg vs kopia", "kopia or borg or restic?" (all three), "Kopia
    or borg?" (`Kopia`, `borg`) and "restic or Borg?" (`restic`). It returns
    nothing for the capitalised forms ("Kopia or Borg?", "Kopia vs Borg",
    "Beacon or Cortana or Siri?"), which shows they take the single
    judgement. It also returns nothing for "Kopia vs. Borg", "kopia vs.
    borg", "kopia versus borg", "go or rust?", "this or that?", "kopia?", a
    wrapped whole prompt ("In one line: kopia or borg?"; its clause is
    judged on its own) and a sentence ("should i use kopia or borg?").
  - *Lowercase either-or probes* (new, in `internal/server/context_test.go`,
    with the prompts in `internal/server/testdata/either-or-probes.json`):
    synthetic lowercase bare either-or prompts, each paired with its
    capitalised form. They run on the evaluation signet plus the issue's
    three Kopia records and a "Coffee order" record, loaded separately, so
    the evaluation fixture's statistics don't move. Three groups, written
    and posted on the PR before the fix runs on them:
    - 12 no-memory prompts whose one known word is in a record's summary and
      whose other word is in no record (the "tea or coffee?" shape);
    - 8 no-memory prompts whose words are in no record ("pizza or pasta?");
    - 8 pairs of two known words, each in a different record's summary.

    The test asserts both halves of the bar on every prompt (see the
    [course change](#course-change-at-code-round-0-the-rule-only-adds)),
    running each lowercase prompt with the rule off and on.
  - A unit test in `internal/memory` that a lowercase lone summary match is
    kept when it's in `Leads` and dropped when it isn't, with an absent
    other term.
- **Docs**: `docs/retrieval.md` (the either-or limit paragraph and the
  2.5.1 notes), `docs/releases/2.5.1.md` (the fix; the intended change that
  a lowercase prompt on a word in a record's summary, such as "tea or
  coffee?", now injects what its capitalised form does; and the limits:
  `vs.`, wrappers without a `?` clause, words that give no content term),
  `docs/evidence/2.5/either-or.md`
  (a 2.5.1 section with the matrix and the per-prompt comparison), and
  `VERSION` to 2.5.1. README and install pointers move in the
  post-publication record PR, as for 2.5.0.
- Not changed: the evaluation fixture (`context-eval.json`), every tuning
  floor in `assertEvaluation`, the signet layout, intents, sync, the state
  file, and every file under `plugins/` (`plugin.json` stays 2.3.0).

## Order of work

1. **Tripwire and probes first.** Add the test rows and the probe set
   above and run them on unchanged code: the lowercase, mixed-case and
   wrapped rows of `TestMemoryContextEitherOrOnASummaryName` fail, and the
   `vs.` rows and the lowercase rows of `TestEitherOrOrders` pass. Post the
   probe prompts and their per-prompt results on the merge base, and on
   v2.4.0 for reference, as a PR comment before the fix is written. Save
   the baseline evaluation report with `MANDALORE_EVAL_REPORT` on the merge
   base.
2. **The fix**: `eitherOrLeads`, the second judgement in `relevant`,
   `Query.Leads`, the `bm25` membership check, and the unit tests. The
   tripwire and both halves of the probe bar pass.
3. **No-regression evidence** (see [Proof](#proof)): the per-prompt
   comparison, the hook replay comparison, the probes on the head, and the
   either-or matrix.
4. **Docs, release notes, `VERSION`**; `bin/ci`, `bin/sdlc check`,
   `govulncheck`. Simplify, then mark the PR ready.

Each step leaves the branch building and reviewable; `bin/ci` passes from
step 2.

## Risks

- **A lowercase bare either-or that names nothing in the signet injects.**
  This is an intended change, and the release notes name it. A lead word's
  lone summary match now counts, so with a "Coffee order" record, "tea or
  coffee?" injects it (1 record) where 2.5.0 injects nothing (measured by
  the reviewer on 6bed590). "Tea or Coffee?" already injects it on 2.5.0.
  The existing fixture has no lowercase bare either-or prompt, so its
  zero-difference comparison can't see this. The probe set measures it
  per prompt and bars the rule from adding any record the capitalised form
  doesn't inject. The requester's private eval covers real data.
- **More records on a prompt the rule reaches.** The second judgement only
  adds, so a prompt can carry 2.5.0's records plus the added ones. "panel
  or turbine?" has 7 relevant records: 2.5.0's 4 body matches first, in
  their order, then 3 summary matches on "panel". Packing keeps the top 3
  and then only what fits whole in the 2048-byte budget, so it injects 2.5.0's
  4 and drops the 3 added. The byte budget bounds every prompt as before.
- **A second recall on reached prompts.** Only a bare either-or (whole
  prompt or question clause) that `eitherOrLeads` reaches pays it. Measured
  warm, with 300 calls each: about 2 to 6 ms more at the median on the
  1000-record benchmark corpus and the evaluation signet, inside the hook's
  3 s timeout and `bin/ci`'s 300 ms warm-path bound. Capitalised forms keep
  one judgement.
- **Breaking the lowercase body-only shape.** Contained by keeping leads out
  of absent names, and asserted for all 8 pairs in four phrasings.
- **Changing capitalised forms.** Contained by the construction above, with
  `vs.` left outside the shape, and asserted by `TestEitherOrLeads`
  returning nothing for them, the unchanged `TestContextLead`, the `vs.`
  tripwire rows and the zero-difference comparison.
- **Three-word lowercase prompts.** Every word becomes a lead and none
  anchors. The matrix shows "beacon or cortana or siri?" still gets nothing,
  as on 2.5.0. Order effects with three names remain a stated limit.
- **Main moves under this PR** (#259 is in flight). The baseline report is
  regenerated on the merge base used for review, and main is merged in, never
  rebased or force-pushed.

## Proof

- **The contract's four shapes**: `TestMemoryContextEitherOrOnASummaryName`
  in `bin/ci`, 3 records each, failing on 6bed590, with `vs.` asserted
  unchanged.
- **What the change reaches**: the lowercase either-or probes, in `bin/ci`,
  with the bar above. Reported per prompt on the merge base, the head and
  v2.4.0 (its tag in a temporary worktree, with the probe test copied in).
  Each prompt whose injection differs from the merge base is listed in the
  PR and the evidence.
- **Latency of the second judgement**: warm p50 and p95 over 300 calls each,
  rule off and on, for a bare prompt, a capitalised form and a wrapped
  clause, on the 1000-record benchmark corpus and the evaluation signet,
  reported in the evidence.
- **No per-prompt regression on any existing set**, three ways. The
  fixture has no lowercase bare either-or prompt, so these show that
  nothing else moved; the probes above cover what the change reaches.
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
  orders, with `or`, `vs` and `vs.`, plus the three-name Beacon prompts in
  lowercase, and wrapped forms: "In one line: kopia or borg?", "kopia or
  borg? one line is fine", "quick one, kopia vs borg" and "Quick one: Kopia
  vs Borg". Measured on the merge base and the head. The release notes'
  limits come from it: `vs.`, wrappers without a `?` clause, and words that
  give no content term ("go or rust?", "this or that?") keep 2.5.0's
  behaviour.
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
