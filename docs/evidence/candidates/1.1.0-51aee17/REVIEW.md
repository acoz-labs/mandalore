# Exact-candidate product review

Engineering handoff, not a human verdict or release authorization.
Source `51aee17afec015ba2ad44584f8190b4bb6d901a8`, manifest SHA256
`7adacb6e7dc990a12df2cc25e72404dde0156835d771474c9745d3b517a0780c`.
The retained artifact was not rebuilt. See [identity and recovery](README.md).

## What the evidence lets the owner review

| Issues | Behavior | Fresh engineering evidence |
| --- | --- | --- |
| #13 | Native Pi connection, separate banks, correction history and Codex continuity | [Core](engineering/checkpoint.json), [Pi lifecycle](engineering/pi-lifecycle.json), [live context refresh](engineering/pi-live-refresh.json), [install/repair](engineering/pi-install/evidence.json), [Pi displays](engineering/pi-display/evidence.json) |
| #56 | One complete model-visible memory result; errors and provenance remain intact | [Four actual Codex observations and 21 selector fixtures](engineering/rendering.json) |
| #57 | Small initial historical excerpts, deliberate expansion and current-versus-historical distinction | [Six native scenarios and a historical control](engineering/retrieval-and-first-use.json), [historical menu journeys](engineering/behavior-checkpoint.json) |
| #61, #100 | Direct consolidation delivers existing work without filler; explicit prohibitions and quotations remain distinct | [Delivery cases and seven supplemental intent checks](engineering/behavior-checkpoint.json) |
| #65 | Ordinary learning uses save-and-deliver when permitted; local durability is separate from delivery | [Ordinary save](engineering/delivery-model/ordinary.json), [offline save](engineering/delivery-model/ordinary-offline.json), [failure boundaries](engineering/systems-checkpoint.json) |
| #74 | Request reuse, truthful quota timing and partial-state guidance | [Source-bound synthetic request/display cases](engineering/quota/evidence.json) |
| #85 | Static readiness does not execute dependencies; native checks require a separate choice | [Static cases and seven real terminal journeys](engineering/systems-checkpoint.json) |
| #88 | Same immutable payloads through verification, isolated update, recovery and rollback | [Payload verification](verified.json), [isolated upgrade](engineering/upgrade/evidence.json) |
| #93 | First connection succeeds when its selected native profile does not yet exist | [Absent-profile structural regression](engineering/first-use/structural.json), [plain success and controlled failure](engineering/first-use/menu.json) |
| #99 | Cancellation promptly exits blocked plain prompts and restores the terminal | [Nine previously committed exact-artifact journeys](initial-manifest.json) |

Original issue criteria and the [candidate guide](../../../releases/1.1.0-candidate.md)
still apply. This table routes evidence; it does not narrow those criteria.

## Important limitations and choices

- Native execution is macOS arm64, Codex 0.154.0 and Pi 0.85.1 on Node 24.1.0.
  Cross-built platform assets are not equivalent to native Linux testing.
- Ordinary learning remains automatic/model-selected. This candidate does not
  add deterministic compaction/exit synchronization; #55 remains open.
- Fewer memory calls and smaller selected excerpts do not guarantee lower total
  context usage. Tool discovery, skill loading and deliberate full-document
  review still contribute. The delivery control and candidate both used five
  model requests; candidate final input was slightly higher.
- The matched historical-retrieval comparison answered correctly with complete
  relevant-source coverage in both runs. Initial search envelopes were 4,100
  versus 5,052 bytes, but full-task foundling envelopes were 16,818 versus 16,758
  bytes; final model input was 33,945 versus 34,177 tokens. The improvement is
  bounded initial loading and deliberate expansion, not a large measured
  whole-task reduction. Full-log review intentionally read all 12,828 bytes.
- Historical comparison executables were reconstructed from clean pinned old
  sources after the shutdown. Their hashes and provenance are explicit; they
  are not retained old artifacts or rebuilds of this candidate.
- Quota rendering uses source-bound test drivers with controlled API responses,
  not live quota exhaustion. Actual terminal recordings are not screen-reader,
  pixel-baseline, other-locale or other-platform acceptance.
- Synthetic fixtures and local bare remotes were used. No personal runtime,
  signet, credentials or existing native authentication were migrated or replaced.
- Recovery-fixture checksum failures occurred before model execution. They were
  preserved and corrected against original bytes; successful model turns were
  not repeated merely to improve results. Earlier failed candidates keep their
  own immutable evidence and are not silently relabeled.

## Human decision still required

The exact owner-review eligibility extension is merged in [PR #107](https://github.com/acoz-labs/mandalore/pull/107).
Eligibility is not acceptance. The owner must personally review this candidate's
evidence against the eleven nominated issues, raise any remaining concerns and
provide actual verdicts before the guarded acceptance workflow is used.

Previously accepted human scenarios need not be needlessly repeated, but they do
not automatically accept changed candidate bytes. A verdict on this evidence is
separate from permission to publish the release or activate a personal installation.
No acceptance workflow, release publication or personal activation has occurred.
