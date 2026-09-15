# Contracts and behavior

## Search

Keep foundling ID, literal query and explicit limit (1–10); API/CLI default limit
becomes three. Add optional `registration_revision_id`, `offset` (default zero),
`excerpt_bytes` (default 512, range 128–1024) and `budget_bytes` (default 8192,
range 2048–32768). Offset is a document-rank index, not a content-byte position;
require nonnegative values no greater than the source's supported file bound.
An offset greater than zero requires the exact registration revision from the
previous result. A supplied revision is validated even at offset zero.

Verify input before source observation. Use existing `verified` with the selected
registration, calculate all matching documents and retain existing deterministic
score/path ordering. Slice from offset, append whole attributed excerpts until
count or serialized-result budget is reached. No silent origin/hash/notice removal
to fit a budget. If no first remaining item fits, return a safe distinct budget
error suggesting an increased budget up to 32768; never return an empty page with
a nonadvancing continuation. Do not count this as no matches.

Keep existing `items`, `matching_count`, `truncated` and `notice`. Add the exact
`registration_revision_id`, selected `offset` and optional `next_offset` for the
next document rank. Matching count remains the full query match count. Truncated
means some matching documents are outside this page, before or after it; omitted
count is derivable from total minus item count. The next offset exists only when
later matches remain and strictly advances after any nonempty page. No matches or
an offset past the end returns no next offset. The notice/schema distinguishes
search result pagination from each item's byte continuation and incomplete text.

The budget applies to the full serialized SearchResult including metadata and
pagination, not the outer API/MCP envelope. Account for final continuation fields
while fitting items. Preserve the existing 32768-byte individual-excerpt ceiling,
UTF-8 alignment and safe source bounds. Larger explicit previews remain limited
to the former maximum; complete document reads remain available separately.

## Read and compatibility

Default API/CLI read content length becomes 1024; explicit range stays 1–8192.
Read output shape and exact-registration/source verification remain unchanged.
CLI adds search `--registration-id`, `--offset`, `--excerpt-bytes`, `--budget-bytes`
alongside existing flags; no flag silently changes meaning for another operation.
Shared generated schemas expose defaults/units and safe errors. Unknown fields,
bad UTF-8, invalid ranges and obsolete pins are refused as before. Strictly
distinguish omitted new budgets from explicitly invalid zero values.

No persisted schema, connection, bank or registration migration. Existing callers
can request their former explicit sizes. New response fields are additive; update
internal strict consumers/tests. Search pagination is tied to a verified immutable
source registration and the repeated query, not a cache or signed authorization
token. A different query denotes a different result ordering. Every new request
revalidates source/connection; no cross-request freshness cache is introduced.

## Agent and menu wiring

Update the on-demand foundling reference and terse tool/schema descriptions with
selection, explicit expansion, both continuation kinds and no-repeat guidance.
Do not copy full schemas into startup hooks. Preserve all attribution fields for
provenance-preserving promotion. The menu holds the selected query/registration
through result paging and uses existing block/select/input helpers; errors never
silently switch the selected source. Only a recognized too-small-budget error may
offer an explicit larger-budget retry of the same page, defaulting to Back.
Read prompts accept explicit content bytes.

## Failure and authorization

All retrieval paths require the bound signet and explicitly chosen foundling.
Read-only mode permits them, never promotion/setup. Source or registration changes
between pages refuse stale continuation; disconnection, missing files, pin/hash
changes, cancellation, path redirection and corruption retain existing failures.
No automatic repair, fetch, retry loop or source execution. Logs/evidence contain
synthetic fixtures only; no source content is persisted as a retrieval cache.
