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

The fix exempts the first word from the lone-summary-word rule only; it
doesn't make it an anchor. On the evaluation signet, where Beacon is in one
record's summary, "Beacon or Cortana or Siri?" still gets nothing while
"Siri or Cortana or Beacon?" gets the Beacon record, both as on 5b87ef5.
"Beacon or Cortana?" gets 1 record, where 5b87ef5 gives 0, while "Cortana or
Beacon?" gets 2 on both.

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

## 2.5.1: lowercase either-or prompts (#264)

2.5.1 gives lowercase and mixed-case bare either-or prompts the records
their capitalised form gets, and changes nothing else that the evaluation,
the hook replay or the probes measure. This is engineering evidence recorded
2026-11-12 on macOS (arm64) with Go 1.26.11, comparing 2.5.0 as merged
(6bed590) with the change. It is not release acceptance. All data is
synthetic.

### The matrix

The evaluation signet plus the three Kopia-summary records above, as
records injected by `memory_context`. 2.4.0 is the v2.4.0 tag with this
fixture copied in. The counts come from a scratch test, not committed, run
in a temporary worktree of each build with `GOFLAGS=-count=1`. 2.5.1's rows
are asserted by `TestMemoryContextEitherOrOnASummaryName`, which fails on
6bed590.

| Prompt | 2.4.0 | 2.5.0 | 2.5.1 |
| --- | --- | --- | --- |
| "Kopia or Borg?" | 3 | 3 | 3 |
| "Borg or Kopia?" | 3 | 3 | 3 |
| "Kopia vs Borg" | 3 | 3 | 3 |
| "Borg vs Kopia" | 3 | 3 | 3 |
| "kopia or borg?" | 3 | 0 | 3 |
| "borg or kopia?" | 3 | 0 | 3 |
| "kopia vs borg" | 3 | 0 | 3 |
| "borg vs kopia" | 3 | 0 | 3 |
| "kopia or Borg?" | 3 | 0 | 3 |
| "Kopia or borg?" | 3 | 0 | 3 |
| "Kopia vs. Borg" | 3 | 0 | 0 |
| "Borg vs. Kopia" | 3 | 0 | 0 |
| "kopia vs. borg" | 3 | 0 | 0 |
| "In one line: kopia or borg?" | 3 | 0 | 3 |
| "kopia or borg? one line is fine" | 3 | 0 | 3 |
| "quick one, kopia vs borg" | 3 | 0 | 0 |
| "Quick one: Kopia vs Borg" | 3 | 0 | 0 |
| "go or rust?" | 0 | 0 | 0 |
| "this or that?" | 0 | 0 | 0 |
| "Beacon or Cortana or Siri?" | 0 | 0 | 0 |
| "Siri or Cortana or Beacon?" | 1 | 1 | 1 |
| "beacon or cortana or siri?" | 0 | 0 | 0 |
| "siri or cortana or beacon?" | 0 | 0 | 0 |
| "Beacon or Cortana?" | 2 | 1 | 1 |
| "Cortana or Beacon?" | 2 | 2 | 2 |
| "beacon or cortana?" | 2 | 1 | 2 |
| "cortana or beacon?" | 2 | 1 | 2 |

`vs.`, a wrapper without a `?` clause and three lowercase names stay as on
2.5.0. "kopia or borg? one line is fine" is reached through its question
clause. "beacon or cortana?" and "cortana or beacon?" now get what "Cortana
or Beacon?" gets.

For the 8 `either_or_limit` pairs (X only in the answer's body, Y a name no
record holds), the answer's hook rank in "X or Y?", "Y or X?", "X vs Y" and
"Y vs X", the four lowercase forms and both mixed forms is identical on
6bed590 and 2.5.1. Every lowercase form ranks the answer first. The
capitalised and mixed forms that put the absent name in capitals get
nothing, as before. 2.4.0 ranks the answer first in every form.
`TestEitherOrOrders` asserts the four lowercase forms of all 8.

### What the change reaches: the probes

The evaluation set holds no lowercase bare either-or prompt, so its
comparison can't see this change. `TestEitherOrProbes`
(`internal/server/testdata/either-or-probes.json`) runs 28. They were posted
with their 6bed590 and 2.4.0 results on
[#270](https://github.com/acoz-labs/mandalore/pull/270#issuecomment-6832291843)
before the fix ran on them. They run on the evaluation signet plus the three
Kopia records and a "Coffee order" record. "2.5.0" is the change's rule off,
which gives the same counts as 6bed590. The test asserts, per prompt, that
the rule adds only records the capitalised form injects and drops none that
2.5.0 injects.

| Group | Lowercase | 2.4.0 | 2.5.0 | 2.5.1 | Capitalised | 2.4.0 | 2.5.0 and 2.5.1 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| summary word | "tea or coffee?" | 1 | 0 | 1 | "Tea or Coffee?" | 1 | 1 |
| summary word | "drill or hammer?" | 1 | 0 | 1 | "Drill or Hammer?" | 1 | 1 |
| summary word | "runway or terminal?" | 1 | 0 | 1 | "Runway or Terminal?" | 1 | 1 |
| summary word | "captain or pirate?" | 5 | 0 | 4 | "Captain or Pirate?" | 5 | 4 |
| summary word | "summer or spring?" | 1 | 0 | 1 | "Summer or Spring?" | 1 | 1 |
| summary word | "panel or turbine?" | 4 | 4 | 4 | "Panel or Turbine?" | 4 | 3 |
| summary word | "verdict or trial?" | 1 | 0 | 1 | "Verdict or Trial?" | 1 | 1 |
| summary word | "pipeline or tanker?" | 1 | 0 | 1 | "Pipeline or Tanker?" | 1 | 1 |
| summary word | "bathroom or kitchen?" | 1 | 0 | 1 | "Bathroom or Kitchen?" | 1 | 1 |
| summary word | "beds vs sofas" | 0 | 0 | 0 | "Beds vs Sofas" | 0 | 0 |
| summary word | "venue vs theme" | 1 | 0 | 1 | "Venue vs Theme" | 1 | 1 |
| summary word | "font vs calligraphy" | 1 | 0 | 1 | "Font vs Calligraphy" | 1 | 1 |
| no record | "pizza or pasta?" | 0 | 0 | 0 | "Pizza or Pasta?" | 0 | 0 |
| no record | "cats or dogs?" | 0 | 0 | 0 | "Cats or Dogs?" | 0 | 0 |
| no record | "piano or violin?" | 0 | 0 | 0 | "Piano or Violin?" | 0 | 0 |
| no record | "chess or checkers?" | 0 | 0 | 0 | "Chess or Checkers?" | 0 | 0 |
| no record | "ski or snowboard?" | 0 | 0 | 0 | "Ski or Snowboard?" | 0 | 0 |
| no record | "rugby vs cricket" | 0 | 0 | 0 | "Rugby vs Cricket" | 0 | 0 |
| no record | "opera vs ballet" | 0 | 0 | 0 | "Opera vs Ballet" | 0 | 0 |
| no record | "sushi vs ramen" | 0 | 0 | 0 | "Sushi vs Ramen" | 0 | 0 |
| both known | "mulch or compost?" | 4 | 4 | 4 | "Mulch or Compost?" | 4 | 4 |
| both known | "shell or editor?" | 4 | 4 | 4 | "Shell or Editor?" | 4 | 4 |
| both known | "tidepool or lanternfish?" | 10 | 10 | 10 | "Tidepool or Lanternfish?" | 10 | 10 |
| both known | "beacon or kopia?" | 5 | 5 | 5 | "Beacon or Kopia?" | 5 | 5 |
| both known | "nas or ups?" | 3 | 3 | 3 | "Nas or Ups?" | 3 | 3 |
| both known | "spelling vs timezone" | 4 | 4 | 4 | "Spelling vs Timezone" | 4 | 4 |
| both known | "freeze vs hotfix" | 1 | 1 | 1 | "Freeze vs Hotfix" | 1 | 1 |
| both known | "printer vs vlan" | 4 | 4 | 4 | "Printer vs Vlan" | 4 | 4 |

"panel or turbine?" keeps 2.5.0's 4 body-match lines, unchanged and in
order. The rule's 3 summary matches on "panel" follow only as whole lines in
the room left, and none fits in the 2048-byte default. The test also asserts
that every line 2.5.0 gives is unchanged, for every probe. A first version that judged such prompts once
with the leads dropped 3 of the 4 body matches and added a body match
("captain" in the hotfix policy) that "Captain or Pirate?" doesn't inject.
So 2.5.1 judges them twice and only adds.

### Nothing else moved

- **Evaluation, per prompt.** `MANDALORE_EVAL_REPORT` from
  `TestContextSelectionEvaluation` on 6bed590 and on the change, compared
  with [`eval-diff.py`](eval-diff.py):

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

  The tuning bars are 2.5.0's: false injection 3 of 62, collisions 2 of 24,
  plain short questions 36 of 38, filler-wrapped 18 of 18, generic two-word
  0 of 12 with the guard at 6 of 6, named guard 8 of 10, `either_or_limit`
  0 of 8, newer first 6 of 6 in recall and the hook, older-only 2 of 2, hook
  top-1/top-3 82.2%/89.2%, recall 77.3%/93.2%, scripted session 560 bytes.
- **Hook replay.** [`hook-replay.py`](hook-replay.py) at 2048 bytes, with
  and without session IDs, on builds of 6bed590 and the change: both output
  files are byte-identical.
- **`memory_recall`.** It sets no leads, so its path is unchanged. Its ranks
  are part of the per-prompt report above.

### Latency

Only a prompt the rule reaches costs a second recall. 300 warm calls each,
with the read cache on, as the server runs:

| Signet | Prompt | Rule off (2.5.0) p50 / p95 | 2.5.1 p50 / p95 |
| --- | --- | --- | --- |
| 1000-record corpus | "kopia or borg?" | 5.1ms / 10.7ms | 9.68ms / 15.26ms |
| 1000-record corpus | "Kopia or Borg?" | one judgement either way | 5.18ms / 7.77ms |
| 1000-record corpus | "In one line: kopia or borg?" | 9.72ms / 13.4ms | 15.25ms / 19.72ms |
| evaluation signet | "kopia or borg?" | 4.24ms / 4.99ms | 6.59ms / 10.44ms |
| evaluation signet | "Kopia or Borg?" | one judgement either way | 3.2ms / 6.22ms |
| evaluation signet | "In one line: kopia or borg?" | 8.34ms / 10.33ms | 10.87ms / 12.5ms |

The hook's timeout is 3 s, and `bin/ci`'s warm-path bound is 300 ms.
Capitalised forms take one judgement, as on 2.5.0.

### Code review round 1

The reviewer found two gaps in how the second judgement's records merged on
b853582, and both are fixed. The tests below fail on b853582 and pass on
the fix.

- **The newer decision.** With "Kopia snapshot retention policy" (older)
  and "Snapshot retention policy" (newer), a pair the newer-first rule
  forms, "kopia or borg?" injected the older alone. It now injects the newer
  and then the older, as "Kopia or Borg?" does
  (`TestMemoryContextEitherOrBringsTheNewer`).
- **2.5.0's lines.** In the reviewer's repro, a record whose body names
  kopia 20 times, plus two Kopia summary records, 2.5.0 injects that record
  whole, as a 1058-byte line. On b853582 the additions joined the top records
  and clipped it to 670 bytes. 2.5.1 gives a record of that shape exactly
  2.5.0's line, and the additions follow in the room left
  (`TestMemoryContextEitherOrKeeps250Lines`). `TestEitherOrProbes` asserts
  the same for every probe's lines.

Round 2 found the first gap again in a narrower case. When 2.5.0 already
injects a record and the newer decision is too long for the room left,
3a38e88 skipped the newer and still added the older. With a drill log that
names kopia 40 times and a newer decision of about 600 bytes, "kopia or
borg?" injected the drill log and the older decision alone. An added record
and its newer now fit as one unit, both or neither, so that prompt injects
the drill log alone, as 2.5.0 does. `TestMemoryContextEitherOrBringsTheNewer`
covers this shape and fails on 3a38e88.

On each fix, the per-prompt evaluation report again shows `differences: 0`
against 6bed590. The hook replay's two outputs and `summary.json` are
byte-identical to 6bed590's, and every probe's counts are as in the table
above. The fix adds no recall, so the latency figures stand.
