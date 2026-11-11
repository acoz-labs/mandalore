# Either-or questions on a name in record summaries (#232)

The fix for bare "A or B?" and "A vs B" questions changes no other prompt the
evaluation or the hook replay measures. On 2.5.0 as merged (5b87ef5), such a
question got nothing when A is in several records' summaries and B in none,
while "B or A?" got A's records. With the fix, all three phrasings inject A's
records. Every per-prompt result of the evaluation's tuning split and every
replay row is identical to 5b87ef5's.

This is engineering evidence recorded 2026-11-11 on macOS (arm64) with Go
1.26.11. It is not release acceptance. All data is synthetic.

## The repro

The signet holds three records whose summaries name a backup tool, "Kopia",
and none naming "Borg", plus 30 filler notes
(`TestMemoryContextEitherOrOnASummaryName`).

| Prompt | 5b87ef5 | With the fix |
| --- | --- | --- |
| "Kopia or Borg?" | 0 records | 3 records |
| "Kopia vs Borg" | 0 records | 3 records |
| "Borg or Kopia?" | 3 records | 3 records |

The test fails on 5b87ef5 for the first two prompts and passes with the fix.

## The body-only shape, unchanged

The evaluation's `either_or_limit` set holds 8 "X or Y?" questions whose X
is only in the answer's body and whose Y is a name no record holds ("Helix or
Neovim?"). Each phrasing below was built from those 8 pairs and run through
the hook on the evaluation signet. The fix changes none of them:

| Phrasing | 5b87ef5 | With the fix |
| --- | --- | --- |
| "X or Y?" | 0 of 8 | 0 of 8 |
| "X vs Y" | 0 of 8 | 0 of 8 |
| "Should I use X or Y?" | 0 of 8 | 0 of 8 |
| "Do we use X or Y for this?" | 0 of 8 | 0 of 8 |
| "Y or X?" | 8 of 8 | 8 of 8 |
| "X?" | 8 of 8 | 8 of 8 |
| "What about X?" | 8 of 8 | 8 of 8 |

A first version of the fix made "Y or X?" lose 7 of its 8 answers. That's
why `TestEitherOrOrders` now asserts all 8.

## Per-prompt comparison

`MANDALORE_EVAL_REPORT` from `TestContextSelectionEvaluation`, at a clean
worktree of 5b87ef5 and at this change, on the same fixture, compared with
[`eval-diff.py`](eval-diff.py):

```text
Collisions   24 entries, 0 different
EitherOr      8 entries, 0 different
Filler       18 entries, 0 different
Generic      12 entries, 0 different
Guard         6 entries, 0 different
Named        10 entries, 0 different
Newer         6 entries, 0 different
NoMemory     62 entries, 0 different
OlderOnly     2 entries, 0 different
Session      20 entries, 0 different
Short        38 entries, 0 different
Trials      880 entries, 0 different
differences: 0
```

The tuning bars are therefore the same as 5b87ef5's: false injection 3 of 62,
collisions 2 of 24, plain short questions 36 of 38, filler-wrapped 18 of 18,
generic two-word 0 of 12 with the guard at 6 of 6, named guard 8 of 10,
`either_or_limit` 0 of 8, and the newer decision first in 6 of 6 in recall and
the hook. The hook's top-1/top-3 is 82.2%/89.2%, and recall's 77.3%/93.2%.

## Replay comparison

[`hook-replay.py`](hook-replay.py) ran at 2048, 1536 and 1024 bytes, with and
without session IDs. Each of the six runs' 208 rows (bytes, injected records,
answer line) is identical to the committed outputs in
[`hook-replay/`](hook-replay/). At 2048 bytes the hook injects 320 bytes per
prompt with session IDs and 600 without, as in [the replay
record](hook-bytes.md).

Reproduce:

```sh
GOFLAGS=-count=1 MANDALORE_EVAL_REPORT=/tmp/after.json go test ./internal/server -run '^TestContextSelectionEvaluation$'
# the same at a worktree of 5b87ef5, into /tmp/before.json
python3 docs/evidence/2.5/eval-diff.py /tmp/before.json /tmp/after.json
python3 docs/evidence/2.5/hook-replay.py /tmp/replay 2048 1536 1024
```
