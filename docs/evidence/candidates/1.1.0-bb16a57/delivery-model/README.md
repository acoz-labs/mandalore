# Fresh native delivery and restriction matrix

Engineering observations for #61/#65 on September 16, 2026, using the retained
candidate in the [parent checkpoint](../README.md). These nine fresh Codex
conversations repeat outstanding scenarios from the original
[consolidation](../../../lifecycle-sync/consolidation.md) and
[inline delivery](../../../lifecycle-sync/inline-delivery.md) evidence. They
are not human product acceptance, publication or a live installation update.

## Identity, fixtures and verification

Eight runs use source `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`, retained
macOS arm64 runtime SHA-256
`0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`, and
the extracted candidate skill SHA-256
`48ccb1c93c68e6f5af355629d753e7fd72e4757644b399af489ec09c751cbcf1`.
The manifest identity remains
`ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`.

The fresh historical baseline uses executable SHA-256
`b0f972e94b122b37fd14ddf4799ba20a8fe3dada88418fdd7ca50303c9d82327`
and the preserved pre-companion skill SHA-256
`7a830a6b14543d50ff3107767cb62d587c908d7a978515ca3146d959042b2ab6`.
Earlier evidence associates that executable with
`e829a6c7e11e04eb4901583fff8735f63855abd5`; fresh version inspection reports
`0.0.0-dev` and no source stamp. The hash is verified; source association is
documentary, not embedded attestation. This is not a new baseline build.

All runs use Codex 0.154.0, gpt-6-astra/high, code mode and Node 24.1.0 in the
dedicated test pane. Each has a separate signet, explicit binding, copied skill,
workspace and local bare Git origin. Initial state is delivered, then synthetic
Quartz Robin knowledge is saved and committed **without** delivery. Every run
starts with a clean worktree and distinct local/remote heads. Offline fixtures
move their own bare origin to a retained sibling location before launch.

The selected MCP server is writable with invocation-local approval for the five
save/delivery operations. Thus no-sync/read-only outcomes test agent behavior,
not a server refusing mutations. The missing-connection run explicitly disables
that server while keeping the same skill. The installed personal plugin is
disabled for each invocation; authentication is inherited in place, not copied.
Git uses isolated configuration for these local fixtures. No hosted signet
remote or personal bank is involved. Native session files may still be written
by Codex; this is not whole-home isolation.

Verification inspects actual native calls, saved IDs and content, Git heads,
unchanged prior semantic files, binding and skill inventories, and final answers.
It rejects administrative calls, direct memory-file/CLI fallback, repeated saves
and sync after a combined attempt. All observed delivery requests explicitly use
three seconds. Existing semantic files retain bytes, modes and mtimes. See the
observer limitation below before interpreting whole-Git-tree inventory results.

## Native outcomes

| Case | Observed memory operations | Verified result |
| --- | --- | --- |
| Ordinary learning, candidate | Two recalls; remember-and-sync | One Birch Loop fact, no journal; receipt/local/remote heads equal. No pre- or post-save standalone sync. |
| Same ordinary prompt, historical baseline | Two recalls; remember; sync | One Birch Loop fact, no journal; receipt/local/remote heads equal. Save and sync occur in separate model-authored code batches. |
| Journal only | Journal-append-and-sync | Exactly one Birch Loop session event, no knowledge record; delivered. |
| Useful learning plus direct cue | Recall; remember-and-sync | One fact delivered, no filler journal or redundant sync. |
| Direct cue plus no-sync | None | No new content; earlier pending local/remote heads remain distinct and unchanged. |
| Direct cue plus read-only | None | No save, journal, checkpoint, repair or sync; earlier pending work untouched. |
| Direct cue, no useful new content, unavailable origin | One sync | No new content or commit; pending/not delivered reported without retry. Retained origin unchanged. |
| Direct cue, missing connection | None | Missing connection reported; no installation, credentials, alternate bank or shell fallback. |
| Ordinary learning, unavailable origin | Recall; remember-and-sync | Exactly one fact saved and committed locally; nested delivery pending, no retry or duplicate save. Retained origin unchanged. |

All nine native runs completed. Exact prompts and selected final answers are in
[the results receipt](results.json). Successful saves were verified by receipt ID
against actual record/event files, not only answer wording. Successful delivery
requires actual receipt/local/remote head equality. Offline comparison uses the
retained origin, not an assumption about an unavailable remote. No native tool
refusal was observed. Quoted/retrieved cue and no-sync local learning evidence
remain in the earlier candidate continuity/retrieval checkpoints; these runs do
not relabel those as new observations.

All nine candidate memory calls printed one complete canonical envelope each,
including pending delivery, with no extra wrappers. The old baseline printed one
extra envelope/raw wrapper; that observation remains in the receipt. No savings
from #56 are counted again as an inline-delivery improvement.

## Operation count and context tradeoff

The ordinary prompts are identical. The candidate removes the model decision
between publication and the delivery attempt: one companion call instead of a
local save followed by a later sync call. **Total model rounds did not decrease**
in this pair, and total/native context counters increased.

| Counter | Fresh historical baseline | Retained candidate |
| --- | ---: | ---: |
| Mutation/delivery requests | 2 | 1 |
| All memory requests | 4 | 3 |
| Model requests / code-mode calls | 6 / 5 | 6 / 5 |
| Canonical memory-envelope bytes | 1,916 | 1,937 |
| All tool-output text bytes | 45,407 | 50,735 |
| Model-authored tool-code bytes | 1,235 | 2,004 |
| Final-request input tokens | 26,111 | 27,725 |
| Aggregate input tokens | 142,420 | 148,406 |
| Aggregate cached input tokens | 127,488 | 131,840 |

The intended narrower decision-gap/operation-count improvement is established,
not a universal token, latency or reliability improvement. Both variants made two
recalls; discovery/guidance and other candidate changes differ. One observation
per variant is not causal isolation or a statistical benchmark. Wider tool output
includes schema discovery and skill reads; aggregate input repeats context across
requests and is not retained context size or a billing estimate.

Fresh executable catalog inspection confirms the complete legacy local remember
and journal operation objects are unchanged. Both companions advertise mutation
and network access. Their input/output schemas remain 1,631/1,949 and 431/1,949
bytes; complete operation objects are 4,050 and 2,872 bytes. The full CLI catalog
is now 41 operations / 85,452 bytes versus baseline 32 / 60,837; unrelated added
administration operations contribute to this difference. It is not the bound
MCP tool count or text automatically injected into every model conversation.

## Observer correction retained

The strict whole-inventory assertion failed in all three negative cases on
`.git` and, in two cases, `.git/index` mtimes. Their bytes, types, sizes, modes,
heads and every semantic file matched. The changed timestamps predate native
session startup by minutes: the setup observer captured the inventory and then
ran ordinary `git status`, which can refresh optional index metadata.

A fresh no-agent control reproduced exactly that directory/index metadata effect
with unchanged index bytes. An otherwise equivalent control with
`GIT_OPTIONAL_LOCKS=0` produced no inventory differences. The receipt retains both
controls and the original negative-case differences. The original runs are not
claimed as clean whole-inventory passes, and no agent run was repeated to hide
the observer problem. Their no-save/no-delivery behavior and unchanged semantic
content are verified separately. Future no-effect observers should disable
optional Git locks or finish status reads before taking the comparison snapshot.

The evaluator also initially required an explicit `error: null` in native tool
events; successful events omit that optional field. Accepting absent or null
corrected the parser, not product behavior. Neither correction changed runtime,
prompt, bank data or completed native transcripts.

## Remaining boundary

The receipt SHA-256 is
`8e58255cfcc3290d134ff61ec359569de10d0ad6a8a5f77e60653392d1baeac5`.
Only synthetic measurements, selected answers and control metadata are published;
no raw transcripts, credentials or native workstation paths are included.
Local bare origins are not hosted-authentication or cross-machine-network proof.

The original Pi rendered success/partial-delivery/warning and remaining narrow/EOF
journeys still need exact-candidate evidence. Previously recorded callback
cancellation/contention/ambiguity checks are not relabeled model-selected recovery.
Human acceptance, release and personal activation remain separate; #55 lifecycle
O1/O2 and the broader roadmap are not completed by this checkpoint.

With all new evidence staged, full pinned `mise exec -- bin/ci` passed: public
content checks, 22 Pi tests, Go race-enabled checks, vet and four target builds.
The documented host fallback was used because Docker is unavailable. The initial
run exposed the preceding terminal receipt's wording/scan-coverage issue, corrected
and retained in its [validation note](../retrieval-ui/README.md); it was not a
product or native scenario failure. Cached Go checks are not fresh model runs.
