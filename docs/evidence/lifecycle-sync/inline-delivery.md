# Save and delivery in one request

Contributor engineering evidence, 2026-09-15, for #65 / PR #67 and #55 outcome
O4. Planning PR #66 reviewed `702967dc51383cdb49f6642d7101750b8e1e1f6c`,
merged as `dcca6d7a55fd0876b3b037b47471a12d70fcf8be`. This is self-review
under ADR0003, not independent product acceptance or permission to release.

## Tested identities and fixture

Codex 0.154.0, macOS arm64, `gpt-6-astra`, high reasoning, code mode. Candidate
runtime and copied workspace skill are from
`c06fd8d9ce372cc643c5c4fa141f35c2d5bc19f7`. Executable SHA-256:
`2bd32e805df3d58bfc83a69f30836be3e8322b4da0cd97146a6b97c0e7acdd5e`;
embedded plugin digest:
`ac17783821a2fdb6607b2e222204a814e8e00147df3e2fe117d1f12b89603a50`.
It was built with Go 1.26.4 from a clean worktree, reports unstamped `0.0.0-dev`,
and is not a nominated artifact.

Baseline uses the retained executable from
`e829a6c7e11e04eb4901583fff8735f63855abd5`, SHA-256
`b0f972e94b122b37fd14ddf4799ba20a8fe3dada88418fdd7ca50303c9d82327`,
and the pre-change skill at the planning head above (including O3 consolidation).
The baseline's 32 serialized operation objects compare equal to the corresponding
candidate objects, including schemas, descriptions and network/read-only flags.
Only the two new companions are added. Runtime source versions are not identical;
this is a measured development comparison, not randomized isolation of all changes.

Each of seven cases has a fresh synthetic signet, explicit binding, copied skill,
workspace and disposable local bare origin. Create/bind and initialize Git through
the CLI, add origin, then synchronize initial state. Save fictional project
Quartz Robin and checkpoint without delivery. Require local HEAD to differ from
origin main before the session; retain both heads and tracked-content SHA-256
inventories. The unavailable-origin case moves only that fixture's bare origin
to a retained sibling path before launch. No hosted repository, private signet,
provider credential fixture or installed plugin is changed.

Run in the dedicated terminal test lab with `codex exec --ignore-user-config
--enable code_mode --skip-git-repo-check --json`, shell read-only and approvals
never; select the model/reasoning explicitly. Unset Mandalore binding/executable
environment overrides, disable the installed Mandalore plugin for this invocation,
and configure the selected test executable's stdio MCP server with the fixture
binding. Give the five save/sync tools invocation-local approval, as in the
[consolidation fixture](consolidation.md). The MCP server remains writable, so
negative cases test agent behavior rather than service-level refusal. Native
authentication/resources can be inherited; this is not native-home isolation.

## Native results

Baseline, candidate ordinary learning and candidate offline receive exactly:

> For future reference, my fictional hiking route is Birch Loop.

Neither the special cue nor a tool name appears. Other candidate prompts ask for
a short Birch Loop journal with no new knowledge records; remember Birch Loop
without synchronizing anything including previous pending work; recall the project
read-only without saving, journaling, checkpointing, syncing or repair; and remember
Birch Loop with a direct consolidation cue but no useful journal entry.

| Case | Memory operations observed | Verified result |
| --- | --- | --- |
| Baseline ordinary learning | Two recalls, local remember, sync | Fact saved; exact local/remote/receipt heads equal |
| Candidate ordinary learning | Recall, remember-and-sync | Fact saved and delivered in one write request |
| Candidate journal | Journal-append-and-sync | One useful event, no additional record; delivered |
| Candidate no-sync | Two recalls, local remember | Fact saved; no new commit or remote delivery, earlier work still pending |
| Candidate read-only | Scopes, recall | Quartz Robin recalled; inventory, clean status and both heads unchanged |
| Candidate unavailable origin | Recall, remember-and-sync | Fact and checkpoint retained; pending/not delivered reported without retry |
| Candidate learning plus cue | Two recalls, remember-and-sync | Fact delivered; no filler journal or redundant sync |

Verification reads actual saved record/event JSON by receipt ID and checks Birch
Loop content. Successful cases require receipt/local/remote head equality and a
changed local HEAD. Offline compares the retained origin's head, not a guessed
remote state. No-sync requires unchanged local/remote heads and the uncommitted
new local record. Read-only additionally checks clean status, which catches new
untracked files omitted by a tracked-file inventory. All seven turns completed;
no native tool refusals occurred in this matrix. Raw logs and workstation paths
remain private.

## Context and decision-gap tradeoff

This single ordinary-learning pair establishes selection and a removed post-save
model request, not a general latency, cost or reliability benchmark.

| Observed counter | Baseline | Candidate |
| --- | ---: | ---: |
| Memory mutation requests | 2 | 1 |
| All memory requests | 4 | 2 |
| Model-generated code-mode calls | 5 | 3 |
| Model request usage records | 6 | 4 |
| Tool-output text bytes across the turn | 45,407 | 46,904 |
| Final model request input tokens | 26,038 | 26,345 |
| Aggregate input tokens across requests | 141,936 | 91,500 |
| Aggregate cached input tokens | 126,976 | 73,600 |

The candidate also chose a different recall query and needed one fewer recall;
do not attribute that reduction to the combined operation. The baseline's remember
and sync occurred in distinct code-mode requests; the candidate requested delivery
within the save operation with no intervening model round. Final request context
increased by 307 tokens despite lower aggregate input. Tool discovery output and
skill/reference reads contribute substantially to these totals. Text bytes are
the sum of string outputs and text blocks in native tool outputs, not serialized
compatibility wrappers or an estimate of tokenizer cost. Aggregate counters reuse
context across requests and are not final context size or monetary cost.

Compact full operation-catalog JSON grows from 60,837 to 67,761 bytes. The new
remember companion has a 1,631-byte input schema and 1,949-byte output schema;
the journal companion has a 431-byte input schema and the same output schema.
Their complete operation objects are 4,050 and 2,872 bytes. These are catalog
measurements, not a claim that every model invocation eagerly receives those
bytes: native MCP rendering and discovery can differ. The MCP tool count grows
from 16 to 18. Keep the detailed delivery guidance in a progressive reference.

Decision: retain the two explicit companions. The intended one-write-request
benefit is observed without changing old local-only contracts or weakening
no-sync/read-only behavior. Accept the measured schema/guidance cost; make no
universal token-saving claim. O1/O2 lifecycle authorization remains unresolved.

## Failure boundaries and validation

Race-enabled API tests exercise both save kinds, actual local-origin delivery,
invalid shape/timeout, read-only and pre-cancelled calls before publication, failed
save without delivery, missing Git/origin, offline state, cancellation after save
and during an observed fetch, lock contention at either stage, later standalone
recovery without duplicated content, two-clone semantic conflict preservation,
and identity replacement between stages without switching banks or pushing it.
Existing transport regressions remain responsible for safe integration, process
cleanup and conservative push ambiguity; no new transport implementation exists.

The compiled CLI journal companion and a real writable SDK stdio remember call
verify serialized saved/delivery receipts against actual local and remote heads.
MCP tests validate generated output schemas and network annotations, including a
successful save with a delivery error. The first TDD run failed because the new
operations were absent. A test-fixture Git environment leak was corrected by
isolating ambient Git configuration, without changing production Git behavior.

At the tested source head, `mise exec go@1.26.4 -- bin/ci` passed on the supported
host fallback (Docker daemon unavailable), including race tests, vet and four
target builds. Hosted CI [35018956553](https://github.com/acoz-labs/mandalore/actions/runs/35018956553)
passed. The edited skill passed its structural validator; native selection above
is the behavioral evidence. Final documentation/plan-removal head must pass its
own checks, recorded in PR #67 reconciliation.

Native cancellation/conflict, other models/clients/platforms, long-running outage
recovery and real hosting credentials were not retested in this matrix. Test
coverage is decomposed deliberately; API evidence is not a native interrupt test.
No-rendered-impact: typed tools, CLI help and agent guidance, not a changed owned
menu or graphical interface. New immutable-artifact nomination, exact-candidate
acceptance, release and live activation remain separate. #65 stays open for those
gates, and #55 still tracks lifecycle authorization/checkpoints.
