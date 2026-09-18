# Native model observations

Retained candidate source/artifact identities are in [REVIEW](REVIEW.md).
These are synthetic engineering observations, not a human acceptance verdict.

## Codex — passed

Native Codex 0.155.0 with configured gpt-6-astra used a stopped, existing authenticated
isolated test profile. Reviewed candidate connection plan/apply reported installed
and verified. No authentication was copied. Codex package SHA256:
`ad991e55cf3e30479db45cc6a77b67af16c9745ef2b43062c6ca63cae06afa6b`.

The process launched in an empty unrelated project directory with no-alt-screen
and the authorized unrestricted test settings. Trust was accepted for that empty
fixture; this update did not produce a new hook-trust dialog.

1. An ordinary question asking for the fictional route's current name, without
   mentioning Mandalore, loaded the memory skill and recalled Maple Quay.
2. Explicit withdrawal inspected visibility history and wrote an exact-head event.
   It reported local durability and no requested synchronization.
3. That session exited. A fresh process received the same ordinary read-only
   question, with no ID or withdrawal reminder. Recall returned no current record;
   the model did not disclose the withdrawn name as current guidance.
4. Explicit restoration in the fresh session used bounded memory_withheld,
   content history and visibility history, then exact-head memory_restore.
   Latest Maple Quay was restored, not earlier Orchid Pier.

No signet-file scan, journaling, synchronization, configuration mutation or unrelated
record write was observed. Git status showed only the two expected added visibility
events and no tracked-file modifications. Both sessions exited. A30-second observer
timeout during restoration was followed by inspection of the still-running session;
the prompt was not resubmitted.

## Pi — passed

Candidate connection plan/apply reported installed/verified in a new isolated
managed profile. Candidate package SHA256:
`aa2f23dd2c7ccf39ba72e56ff38d7be4da03bb269ebe9f0bec501426f7b6f76e`.
The native process explicitly loaded this package and its two skills with ambient
extension/skill/context discovery disabled, and used native authentication in place.

Native Pi 0.85.1 used gpt-6-astra/high. Ordinary recall returned Maple Quay
without a Mandalore cue. Explicit withdrawal inspected current visibility and
wrote the requested event. Pi confirmed a new session after /new; the observer
reported no lifecycle transition for that slash command, but the native screen
and reset context showed success, so it was not repeated.

In the fresh session, the same ordinary read-only question made two empty recalls
and returned no current name. Explicit restoration loaded the visibility guide,
used memory_withheld, inspected visibility/content history and restored the latest
Maple Quay using exact heads. No signet-file scan or shell listing was used.
No journal, synchronization, configuration or unrelated-record write was observed.

The native process exited. Across both harnesses, Git status showed exactly four
new visibility events and no tracked-file changes. The synthetic content history
and base Git HEAD remained intact. Model/tool usage varies by harness: these are
specific observed flows, not proof for every prompt or model.
