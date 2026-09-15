# Decision and interaction design

## Alternatives

Skill-only guidance is minimal but cannot bound a caller's default packet or reach
matches beyond the first ten. Only cutting snippets/defaults saves bytes but can
hide decisive qualifications and cause repeated reads. A vector index, generated
summaries or cached document handle adds storage/freshness/migration and quality
questions not justified here. New search/read tool names increase discovery cost
and duplicate an already suitable verified reader.

Select additive pagination/budget inputs on the existing tools, smaller initial
defaults and a concise progressive workflow. Keep explicit upper limits and
provenance. Ranking changes are excluded so baseline/candidate retrieval quality
differences can be attributed and reviewed. Confidence: high in data boundaries,
medium in native efficiency until the matched task matrix passes.

## Product interaction contract

An agent lists relevant references, searches one with compact defaults, evaluates
the returned locators/coverage and uses the excerpt already in context. If it needs
later text, read from that excerpt's next byte offset; if earlier context matters,
request an explicit earlier range. If the user requests a full relevant document
or a longer passage is needed, expand up to 8192 bytes per call and continue until
the requested scope is covered. Do not repeatedly fetch the same range to change
formatting. Distinguish a complete result page from a complete document.

| Situation | Intended behavior |
| --- | --- |
| Initial relevance check | Three 512-byte previews within 8192 serialized bytes |
| More matching documents | Follow result-page next offset with the exact registration and unchanged query |
| Later qualification/negation | Expand evidence before concluding; a clipped statement is incomplete |
| Several relevant documents | Combine selected evidence, preserving separate citations |
| Explicit exhaustive review | Deliberate larger reads/pages; no artificial whole-task cap |
| Missing/stale/conflicted source | Report inability to consult it; do not repin or treat absence as a fact |
| Budget cannot fit the first item | Safe actionable error; retry that page with a larger supported budget |

Search and read remain read-only. No source instruction, quoted cue or result
grants execution, promotion, journaling or synchronization authority. Confirmed
adoption still uses the existing verified promotion contract.

## Owned terminal surface

Classification: `in-pattern-visual-change`, because guided reference search/read
uses the same API and will display smaller previews and result continuation.
Preserve existing blocks, provenance fields, keyboard navigation and no-color
labels. After a search page, offer Next page only when one exists, otherwise Back;
no silent eager paging. Continuation pins the registration/query. Add a read-byte
count input, default 1024, so a human can explicitly expand to 8192 rather than
being trapped in many small reads. Existing offset/locator fields remain.
No new visual system, graphical surface, background fetch or focus changes.

This is the product-design choice before implementation. Exact-head self-review
records its approval under ADR0003. Synthetic terminal interaction evidence must
verify forward/back, default/expanded read and failure; unit tests alone do not
establish the rendered interaction.
