# Final candidate acceptance review

Review date: 2026-09-14 (America/New_York). Status: **pending owner verdict**.
This is an evidence-backed recommendation for the initial Codex-first MVP, not
an acceptance status, release receipt or assertion that every combination was tested.

## Candidate and authority

- Source: `f899cf6a2a255f3b6b35dcd778c672f799c65eb2`.
- Identity: `mandalore:f899cf6a2a255f3b6b35dcd778c672f799c65eb2:sha256:c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea`.
- Retained build: [34886390526](https://github.com/acoz-labs/mandalore/actions/runs/34886390526), artifact `10365425624`.
- Archive SHA-256: `424c4151887d57cfebd667233aa9cb9fffa3304dd449c6c842b91bd8f83b311d`.
- Nomination: [34886694068](https://github.com/acoz-labs/mandalore/actions/runs/34886694068).
- Owner-review eligibility: explicitly extended through #48/#50, control merge
  `d20eaf680ef7555473445f13122352dbb3aad161`; product bytes were not rebuilt.

Live audit found the artifact unexpired with the expected source/archive identity,
the current candidate nominated on every issue #2–#12 and #45, and every linked
implementation PR merged and contained in this candidate. The normal read-only
release gate passed. Configured owner, authenticated actor and acceptor allowlist
agree; their private values are not reproduced here. No acceptance status or
public release existed at review time. The next workflow submission requires an
actual owner verdict; eligibility and successful tests do not provide it.

## Fresh retained-binary platform results

These are actual native CLI/MCP executions, not cross-builds or native Codex
conversations. The same retained archive was verified before extraction on each
host. The model-free candidate child received no provider credentials.

| Platform | Results | Execution environment |
| --- | --- | --- |
| macOS ARM64 | [Receipt](platforms/darwin-arm64/summary.json) | Local native ARM64 terminal; actual retained binary |
| macOS AMD64 | [Receipt](platforms/darwin-amd64/summary.json) | Standard Intel macOS hosted runner; translation refused |
| Linux AMD64 | [Receipt](platforms/linux-amd64/summary.json) | Standard Ubuntu x86_64 hosted runner |
| Linux ARM64 | [Receipt](platforms/linux-arm64/summary.json) | Standard Ubuntu ARM64 hosted runner |

Every platform passed same-record correction/history, real stdio MCP discovery
and recall, read-only journal refusal with unchanged file hashes, local bare-Git
delivery and fresh-clone recall, owned CLI installation with exact launcher
identity, synthetic migration with preserved original and exact historical JSON
number, and unrelated working-directory preservation.

Remote collector [PR #51](https://github.com/acoz-labs/mandalore/pull/51), head
`e12b44876d0540921bf33f97b6c8b14366e18171`, reuses #41's bounded collector.
[Run 34916124004](https://github.com/acoz-labs/mandalore/actions/runs/34916124004)
passed CI and all three platform jobs. The PR merge checkout is collector
context, not the product source. It is not intended for product merge.

Downloaded evidence archives were checked against live successful-run metadata,
exact archive hashes and a ten-file regular JSON inventory before extraction.
The source, manifest, platform and executable identities were checked again;
clone/installed receipts match their originals. Published JSON is unchanged.

| Target | Evidence artifact | ZIP SHA-256 |
| --- | --- | --- |
| Linux AMD64 | 10376112681 | `4cd36980482513418032f2dde4ff1756f9530fc51b3158257b9301367e93ff12` |
| Linux ARM64 | 10376252137 | `62c7f01a77e1478d1fb9bde669e414ee3a00694f30746fb3768793ebeb3c96f9` |
| macOS AMD64 | 10376347015 | `0d7cede5aa0fcc818ee4037378f9e7f3e45d147c3378e32f95081754e10abad4` |

## Current candidate owner walkthrough and rendered checks

The owner personally submitted natural health, repair-preview, exact-plan apply
and fresh-session recall prompts. [Sanitized observations](https://github.com/acoz-labs/mandalore/blob/06f2dede753ec4065941017033cdbcafd1ba68c9/docs/evidence/armorer/retained-candidate.md)
record those sessions and independent preservation checks. The exact missing-only
repair passed all twelve structural checks. Fresh native Codex 0.153.4 selected
only This Is the Way for ordinary recall and retrieved the current name, concise
update preference and previously adopted historical checklist in two reads.
Memory, binding and runtime stayed unchanged; prior generation and backup survived.

Fresh rendered evidence uses this candidate's Darwin ARM64 executable, actual
terminal capture, synthetic absent-profile paths, English and NO_COLOR. No browser
or mobile UI exists. Expected missing-profile diagnosis is an intentional failure
fixture, not a configured-installation regression.

| Recording | Matrix and observed result |
| --- | --- |
| [Normal](ui/normal.recording) | 96x28 PTY; gg/j/Enter inspection; Armorer labels and pass/fail/not-tested boundaries visible; Exit |
| [Narrow](ui/narrow.recording) | 40x28 PTY; repair input, Escape cancellation, diagnostic inspection, Exit; long labels elide and reports wrap |
| [Plain](ui/plain.recording) | 96x28 PTY; numbered input, repair `:back`, diagnosis, Exit; no color reliance |

All absent binding/state/profile destinations remained absent. The original PTY
mode and 96x28 dimensions were restored. These are fresh contributor-operated
captures for owner review, not an independent visual verdict. There is no approved
pixel baseline; screen readers, alternate fonts/locales and other-platform native
Codex interfaces were not freshly exercised. Console output contains expected
fixture diagnostics, not an unexpected product failure.

## Issue-by-issue review and evidence reuse

Exact-source CI/build checks remain separate from executing retained artifacts.
Compared with preceding candidate `5b3c7b2`, the memory, sync, migration, MCP,
foundling, API, lifecycle, distribution, memory-skill, hook and bridge source
subtrees have no changes. The product delta is the Armorer skill/context, CLI
alias, menu/help labels and plugin description. The associated owner and rendered
checks above are fresh. Older results below are deliberately not relabeled as
executions of the replacement bytes; this is a disclosed change-impact review.

| Issue | Basis for the proposed acceptance decision |
| --- | --- |
| #2 Engine | Exact-source regression/race CI, extraction/license/dependency inventory, fresh retained CLI/MCP checks; assistant-platform modules remain excluded |
| #3 Signet format | Schema/provenance/graph/binding tests, fresh correction/history and migration receipts; immutable IDs and separate machine-local binding preserved |
| #4 CLI/MCP | Fresh four-platform stdio and typed CLI behavior, read-only refusal and cwd checks; unchanged strict/error/cancellation regressions |
| #5 Sync | Fresh four-platform local delivery and clone reads; historical two-physical-Mac continuity, conflict/offline/uncertain-push tests on unchanged sync code; no universal freshness claim |
| #6 Codex | Owner core-memory walkthrough and new candidate's fresh native recall/health/repair checks; prior lifecycle/inheritance/compaction/warning/interruption evidence on unchanged adapter/memory skill |
| #7 Setup | Current native connection/repair and exact CLI install results; fresh Armorer rendered checks; prior full install/update/refusal matrix retained as historical unchanged-flow evidence |
| #8 Migration | Fresh four-platform synthetic import, exact history and original preservation; prior lock/coexistence refusals on unchanged migration code; no real-bank migration |
| #9 Retrieval | Existing quality/scaling evaluations and prior retained 20-sample measurements on unchanged retriever, plus current native two-call recall; lexical/scale limits explicitly retained |
| #10 Integrated MVP | Owner walkthrough, affected-path retests, current four-platform execution, verified provenance and explicit limits in this review |
| #11 Distribution | Verified retained build/nomination, all platform installs and exact payload checks; same-byte publication and public download verification remain the separate release phase |
| #12 Foundlings | Owner source-authority/promotion/disconnection walkthrough on prior candidate; current fresh recall preserves the adopted lesson after disconnection; unchanged source engine plus exact-source regressions |
| #45 Armorer | Owner natural selection, local context, read-only diagnosis/preview, authorized exact repair and fresh reload; alias/context/package tests and bounded development setup preview evidence |

[Earlier complete engineering matrix](https://github.com/acoz-labs/mandalore/blob/70ba7988b80b4b0f1bfbf61be11062e0f6a51f97/docs/evidence/mvp-candidate/acceptance-audit.md)
retains the original platform/lifecycle/synchronization/installer/retrieval details.
Its former independent-account blocker is superseded by decision 0002 and #50;
its old candidate identities and test limitations remain historical facts.

## Limitations to accept deliberately

- Codex is the only native harness in this release. Pi (#13) and Claude Code
  (#14) are deferred. Native conversations are verified on macOS ARM64, not on
  every CLI-supported platform. Four-platform CLI support is not that claim.
- Retrieval is scoped lexical search, not guaranteed recall of everything.
  Historical unchanged-retriever measurements reached roughly 1.6 seconds p95 at
  10,000 single-revision records. Those timings are not a new-candidate benchmark,
  cold-disk guarantee or isolated model-token cost.
- Offline/local-only/pending/conflicted states are real outcomes. Remote delivery
  does not establish every peer's freshness or reload already-read model context.
  The owner test bank intentionally has no remote and a pending disconnection.
- Not every historical native fault and model interaction was repeated on this
  replacement artifact. Escape was not proof of child termination, and controlled
  push-response loss was not a real packet-loss experiment. Native behavior is
  probabilistic; bounded observations do not promise universal wording/routing.
- Foundlings are unreviewed references, not current authority or automatic bulk
  imports. Synthetic migration is not real-memory adoption or hostile-source immunity.
- Public download/bootstrap/update cannot be verified before an actual release.
  Publication requires separate owner authority; anonymous download, clean install,
  update/rollback and final release-ledger verification remain post-publication
  gates. No issue is represented as released or fully closed by this review.

## Proposed disposition

Recommend accepting the initial Codex-first candidate for the release phase with
the evidence-reuse and limits above, subject to the owner's explicit judgment.
The owner may instead request changes or additional exact-candidate scenarios.
Record approved/changes-required per issue only after that verdict, using the
exact identity and current implementation sets. Do not infer approval from this
report, prior "done" messages, or the owner's eligibility authorization.
