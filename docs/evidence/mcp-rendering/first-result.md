# First-result sequencing follow-up

Engineering verification for #56 / PR #92, September 16, 2026. Returned design
PR #91 reviewed `7703fdab7fd0a30b6d6a4ef20debbc9726b56f10` and merged as
`f3d33e5386fd0d06a3b2fbc189d820aca84626f6`. This is not independent product
acceptance, a new immutable candidate, live activation or release.

## Retained counterexample and correction

The [candidate checkpoint](https://github.com/acoz-labs/mandalore/blob/c5507c1e4ea89e6d75b4963aeeccbe0de5752c8e/docs/evidence/candidates/1.1.0/continuity-rendering.md)
retains the red test: one fresh error-first session printed its invalid-input
response as a full duplicate wrapper before loading the selector reference.
Later responses were selected correctly. The complete error, valid recovery and
read-only state were preserved; uniform deduplication failed.

The follow-up moves conditional preparation ahead of recall in the memory skill
and clarifies first-call/reuse sequencing in its reference. Selector code, MCP
wire fields, memory semantics and native configuration are unchanged. The main
skill grows by 35 words; the full helper is still conditional, not startup text
for clients that already render one representation. This follows skill-authoring
guidance to correct the demonstrated failure without adding unrelated rules.

## Identity and method

Tested skill source: `e45dd54e6e0ee5b5e83dd175d3ee41fcd482005c`.
`SKILL.md` SHA-256: `48ccb1c93c68e6f5af355629d753e7fd72e4757644b399af489ec09c751cbcf1`.
Reference SHA-256: `4c801bf9e4eda9d0d7ec1abe80bbf3ff5d806a9ddddc9478775b064fd652bfe3`.
Subsequent ancestry reconciliation/documentation does not change these skill bytes.

Codex 0.154.0, macOS arm64, gpt-6-astra/high, code mode. Each fresh synthetic
workspace received the revised skill explicitly. The unchanged retained runtime
was source `6f9bb5e4d3b79a1c5cf4947b428dcd5b31da0a41`, SHA-256
`5f19988a729637eea300949bce603796da831c9c1365c9f8dea043f4c0391d89`.
This deliberate runtime/skill combination tests guidance, not a rebuilt packaged
candidate. Existing native authentication was used in place, not copied; the
personal plugin was disabled for each test invocation, with explicit synthetic
read-only MCP bindings. No personal connection was replaced.

The two error-first prompts exactly repeat the failed request: invalid recall
with limit -1 once, report its error, then valid read-only project recall. Ordinary
recall asks current release day and its two predecessors. Historical consultation
asks the project name, reason for changing deployment and old-source citation.
None instructs the model how to render results or names the expected selector.

Full native session code inputs/outputs were matched to session ID/cwd. Completed
reference reads preceded the first MCP call event. Every selected model-visible
envelope was compared in full with the actual corresponding wire result, not
merely checked for a missing wrapper key. Complete fixture inventories compare
paths, types, modes, mtimes and bytes, including the historical source.

## Native outcomes

| Case | MCP calls | Duplicate wrappers | Observed behavior |
| --- | ---: | ---: | --- |
| Error-first A | 3 | 0 | Invalid call once, complete input.invalid error, valid Monday recall. |
| Error-first B | 3 | 0 | Same complete error/recovery in another fresh session. |
| Ordinary history | 3 | 0 | Monday current; Thursday then Friday preserved in order. |
| Historical/truncated reference | 8 | 0 | Orchid Beacon, Saturday 03:00 UTC; backup conflict distinguished from old proposal, operations.md cited. |

All four read-only inventories were unchanged. Historical search returned an
incomplete 512-byte preview, followed by 820 bytes at offset 512. Combined source
bytes match the 1,332-byte document hash with no overlap. Registration revision,
source identity/pin, content hash, historical label and continuation/truncation
metadata remained intact. An excerpt need not be marked a complete document
just because its tail reaches EOF; coverage was verified from the actual ranges.

[Structured results](first-result.json) retain operation sequences, metrics and
synthetic provenance. Only the final historical answer's local citation target
is normalized to `operations.md`; original output bytes drive metrics. Raw native
sessions, workstation paths and authentication are not published.

The exact reference selector passed all 21 existing synthetic checks, including
pending/conflict/truncation and distinct-content fallback; its code is byte-identical
to the retained candidate's selector. Skill validation passed in an isolated
PyYAML-equipped tool environment after the default Python lacked that dependency.
No production Python/JavaScript dependency was added.

## Costs, limits and rollout

Error A/B memory presentation was 1,223 bytes each; ordinary history 3,632;
historical consultation 9,979. All code-mode output was respectively 14,452,
14,517, 51,637 and 69,445 bytes. Helper/discovery output is included, not subtracted.
Ordinary final-request input was 28,340 tokens, versus 28,181 for the preceding
candidate and 27,482 for published 1.0.0 on that small fixture. This is not a
whole-task cost optimization claim. Aggregate native usage is not retained
context or billing. Individual observations do not prove deterministic adherence
across clients, models, platforms or long sessions.

Contributor judgment: pass for this targeted sequencing matrix. No graphical
or terminal layout changed (`no-rendered-impact`); this is actual model-facing
trace evidence, not a screenshot baseline. Full pinned CI remains required on
the reconciled head. A changed packaged skill needs a new retained candidate,
nomination and exact-candidate checks; the old candidate's failure and evidence
must not be relabeled. All human acceptance and release gates remain intact.
