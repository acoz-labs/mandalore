# Continuity, restrictions and rendering checkpoint

Engineering observations on the exact retained 1.1.0 artifact identified in the
[candidate receipt](README.md), September 16, 2026. Codex 0.154.0, Pi 0.85.1,
macOS arm64, gpt-6-astra/high. Same synthetic banks and native-auth-in-place
method as the initial checkpoint. No human verdict, live activation or release.

## Additional continuity and restriction checks

| Scenario | Observed result |
| --- | --- |
| Pi language question about quoted “this is the way” | Answered that “way” is a noun; no memory calls; full fixture inventory unchanged. |
| Codex correction from Friday to Monday with explicit local-only/no-journal direction | Scopes, recall, history, then exactly one memory_remember. No sync or journal operation. Bare origin, Git state, journal events and other bank unchanged. |
| Fresh Pi in another working directory | Recalled Monday, replacing Friday, using recall/history. Full fixture inventory unchanged. |
| Direct consolidation cue with no new facts or useful journal outcome | Pi recalled existing state and called memory_sync once. No remember/journal calls; semantic file inventory unchanged, no filler. |

Exactly three linked revisions retain the same decision record and device:
Thursday (fixture) → Friday (Pi) → Monday (Codex). The local-only correction
left the prior delivered HEAD unchanged. Later explicit consolidation checkpointed
and delivered it; actual local and bare-origin heads both became
`bfc57e8b9dc9632efcd1ab1adab20018123179d2`, matching the receipt with zero conflicts.
Bank B stayed unchanged. These restrictions were individual test requests, not
a new default read-only mode. Ordinary implicit learning remains enabled.

[Semantic continuity receipt](continuity.json) retains actual operation sequences,
answers, history identities and the separately verified delivery state. Inventory
comparison includes paths, directory/file types, modes, mtimes and file bytes.
The journal verification was corrected to compare the actual `memory/events`
tree, including its existing `.gitkeep`; it required no repeat mutation.

## Actual code-mode presentation

Fresh matched sessions asked for the current release day and two earlier
decisions. Baseline used the unchanged published 1.0.0 binary, SHA-256
`bcc1b7fdf9e72ccb2bbd734d798c16f3aa5b96de757846711a4e5afe6eb00f93`,
and skill files byte-checked against published source
`f899cf6a2a255f3b6b35dcd778c672f799c65eb2`. Candidate used the verified 1.1.0
runtime and extracted packaged skill. Neither was installed over a personal
connection. Both used explicit read-only synthetic MCP bindings.

Full native session code inputs/outputs were matched to the exact session ID and
working directory. Every printed memory result was checked against its complete
wire envelope. The following are actual model-visible bytes, not serialized
event-log sizes. Aggregate native input is not retained context or billed cost.

| Measurement | Published 1.0.0 | Retained 1.1.0 |
| --- | ---: | ---: |
| Memory calls (scopes, recall, history) | 3 | 3 |
| Printed duplicate wrappers | 3 | 0 |
| Memory-result presentation bytes | 7,820 | 3,632 |
| All code-mode output bytes | 48,118 | 51,204 |
| Model-authored code bytes | 473 | 1,964 |
| Final-request input tokens | 27,482 | 28,181 |
| Aggregate input tokens | 118,711 | 149,135 |
| Aggregate cached input tokens | 102,656 | 132,352 |

Both correctly answered Monday, previously Thursday then Friday, and preserved
full read-only inventories. The candidate saved 4,188 memory-presentation bytes,
but extra selector/discovery overhead exceeded those savings for this small
three-call task. Final-request input rose by 699 tokens. This is one observation
per variant, not a statistical cost claim; neither result proves long-session
compaction prevention. Broad tool discovery still contributes substantial output.

## Retained failure: first error precedes selector guidance

A fresh candidate session was asked to call recall with invalid `limit: -1` once,
report the error, then perform valid read-only recall. It read the main skill,
but printed the complete duplicate error wrapper **before** reading the linked
selector guidance. Subsequent scopes/recall used single selected envelopes.
This was not a conservative mismatch fallback: the first result bypassed the
selector entirely.

The invalid call was not retried. Its full `input.invalid` error, false retry/
write-ambiguity flags and later valid Monday answer were preserved. All fixture
state remained unchanged. Correct error handling passed; uniform deduplication
did not. The [rendering receipt](rendering.json) explicitly records
`issue_56_candidate_ready: false` and the failed error scenario, rather than
discarding it or rerunning until a passing sample appeared.

The exact packaged selector also passed all 21 synthetic cases from the existing
reviewed selector suite: pending/error, conflicts, truncation/provenance,
annotations, distinct blocks, text-only, malformed envelopes and safe fallback.
These are representation-unit checks, not native conflict or interrupted-delivery
tests. They do not erase the first-call sequencing defect in the native run.

Next for #56: address first-result sequencing within the Mandalore integration,
retain this counterexample, and repeat targeted native checks. Any changed
packaged skill/runtime requires a new candidate identity and nomination; do not
silently patch the retained archive or reuse its acceptance evidence as if the
bytes were unchanged. Other candidate matrices and human acceptance remain open.
