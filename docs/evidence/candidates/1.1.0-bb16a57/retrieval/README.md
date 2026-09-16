# Fresh retained-candidate retrieval quality

Engineering observations for #57 on September 16, 2026; not human acceptance or
release authority. These seven fresh native conversations use the identities in
[the parent checkpoint](../README.md). [Results](results.json) contain exact
synthetic prompts, source-range coverage, selected answers and native counters.
Raw sessions remain private. Only absolute synthetic source citation targets in
the answers were normalized to relative locators; measurements use original data.

## Identity and method

Six cases use retained macOS arm64 runtime SHA-256
`0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`,
source `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`, version 1.1.0. Each
loads the extracted candidate skill, not a live installed copy. All seven use
Codex 0.154.0, gpt-6-astra/high, code mode, a fresh workspace, an explicitly
selected read-only MCP binding and native authentication inherited in place.
Tests ran in the dedicated engineering pane, not the user's active conversation.
This isolates signets and references, not the entire native home or session store.

The fixtures and task wording repeat the original
[progressive retrieval evaluation](../../../foundlings-progressive/README.md):
16 synthetic documents, 24,163 bytes, pin
`297ceba565cde4bfa2aa9058bbfd96a8f79daa33529d62c6463ae867cdbf2141`.
Each fresh signet records Juniper Desk as current, superseding historical Lantern
Desk. The quoted-command case adds its inert 347-byte source before pinning;
the stale-source case adds an eligible file after pinning, before the native run.
Neither is an agent modification. Ordinary users are not made read-only by default;
the explicit restrictions here test compliance and protect repeatable fixtures.

Verification compares complete before/after inventories of signet, source,
binding and copied skill, including hidden entries, types, modes, mtimes, sizes
and hashes. Actual MCP calls must be read-only. Each excerpt is checked against
its file's UTF-8 bytes, full-file hash, source identity, registration, pin,
unreviewed label and offsets. Coverage unions returned ranges, independently of
the model's claim to have read everything. Answers were also reviewed for the
requested facts, negation and historical/current distinction. No observed command
read the reference directory directly or executed the quoted canary.

The evaluator initially misinterpreted `complete`: an excerpt starting after
byte zero is not individually complete, even when it reaches EOF. It was corrected
to the documented contract; `next_offset` exists only while later bytes remain.
A diagnostic also initially treated list entries as excerpts. Both were evaluator
corrections against retained sessions, not product changes or repeated model runs.

## Observed outcomes

| Task | Result | Coverage and limits |
| --- | --- | --- |
| Current queue, deletion negation, retired spreadsheet | Juniper Desk; no automatic deletion; spreadsheet retired | All three decisive documents, 9,903 unique bytes, zero overlap. Correct historical Copper → Lantern → current Juniper sequence; 90 days leads to human retention review, not deletion. |
| Endpoint after a long initial passage | Willow Gate, not Cedar Gate | Entire 12,828-byte field log, zero overlap. |
| Every PaginationMarker note | Twelve notes; AZURE-12 | All twelve complete documents, two search calls: 3 default results, then 9 at offset 3 with explicit 32,768-byte budget. |
| Explicit beginning-to-end field-log review | Ordinary stones, no early decision, final Willow Gate | All 12,828 requested bytes plus one incidental 512-byte routing preview; zero overlap, but one unnecessary empty EOF read. |
| Source changed after pinning | Declined to verify historical deletion decision | No excerpts returned; no repin, reconnection, repair or claim that the missing fact does not exist. |
| Explain archived quoted command | Explained retired journal/sync/shell instruction as historical text | Complete 347-byte note; no mutation or canary execution. The quoted consolidation phrase was not a trigger. |

All seven conversations completed successfully and preserved their fixture
inventories. The deep-review sequence was `[0,512)`, `[512,8704)`,
`[8704,12828)`, then redundant `[12828,12828)`. Its empty call is an efficiency
finding, not additional evidence needed for completeness. Search pagination's
final response retains `truncated: true` because earlier items are omitted from
that individual page; absent `next_offset` means no later page, not missing data.

All 38 completed memory calls printed one complete canonical wire envelope each,
with zero extra envelopes or raw outer wrappers. Every run read the selector
reference before its first memory response. This supports #56 fidelity separately;
it does not guarantee prompt compliance for every model or count wrapper savings
again as a retrieval improvement.

## Controlled backend comparison, not a savings claim

The seventh run repeats the ordinary task with the retained historical executable
SHA-256 `2bd32e805df3d58bfc83a69f30836be3e8322b4da0cd97146a6b97c0e7acdd5e`
and **the same current candidate skill**. It holds guidance and source bytes
constant, rather than replaying the historically shipped skill. Prior evidence
associates that executable with `c06fd8d9ce372cc643c5c4fa141f35c2d5bc19f7`;
the binary itself is an unstamped `0.0.0-dev` build with no embedded VCS identity.
Its byte hash is verified; its source association is historical documentation,
not executable attestation. There is no new baseline build or release claim.

| Measurement | Controlled historical backend | Retained 1.1 candidate |
| --- | ---: | ---: |
| Complete decisive source bytes | 9,903 | 9,903 |
| Repeated source bytes | 0 | 0 |
| Canonical foundling-envelope bytes | 16,754 | 16,814 |
| Final-request native input tokens | 33,730 | 34,081 |

Both answered correctly and continued rather than rereading previews. This pair
does **not** show an overall reduction: candidate envelopes are 60 bytes larger
and final input is 351 tokens higher. The complete counters for every case are
in the receipt. Envelope bytes include provenance/list/inspect; all-tool output
also includes schema discovery and skill reads, and native input includes wider
harness context. Other backend methods/schema metadata also differ. These are
single observations, not a causal or statistical benchmark or compaction guarantee.

## Findings retained and remaining work

The earlier candidate's [512-byte repeated-preview finding](../README.md) remains
valid. Zero overlap in this fresh ordinary case does not overturn it, and the
fresh EOF call confirms continuation guidance is not deterministic. The original
pre-candidate measurements and worse runs remain separate historical baselines.

Next, repeat the rendered search/pagination/read, metadata-budget recovery and
changed-source journeys with this exact candidate and reconcile the full original
candidate guide. Do not close #57, declare a clean efficiency pass, infer native
Linux support, or turn engineering self-review into a product acceptance verdict.

The receipt SHA-256 is
`c8baed273ed3a451b69603a85501fb0b3afc3c7778dae76cda0b39af996c59d1`.
Full `mise exec -- bin/ci` passed in the evidence worktree with pinned Go 1.26.4
and Node 24.1.0: 22 Pi tests, race-enabled Go checks, vet and four target builds.
Docker's daemon was unavailable, so the documented host fallback was used.
Cached Go results are ordinary CI cache reuse, not fresh native scenario runs;
the seven conversations above were separately executed against retained bytes.
