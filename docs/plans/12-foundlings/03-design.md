# Technical design

## Ownership and state

Keep version-1 portable registrations and citations in internal/memory. Add a
service-level paginated routing view over registration heads; all revisions still
undergo validation. One head is active/disconnected; multiple heads are conflicted,
never an arbitrary winner. Add authored registration/disconnection service methods
using existing immutable writes and graph validation.

Add internal/foundlings for local bindings and source adapters, with no dependency
from the memory engine back to it. A local binding under
`.mandalore/foundlings/<foundling-id>.json` identifies signet ID, registration revision,
source identity/pin and canonical absolute root. It is ignored clone-local state,
not portable metadata. Normal reads never create this directory or repair a binding.
Explicit writes validate ownership, use atomic publication and preserve unknown or
changed bindings rather than overwrite them silently. A deliberate path replacement
requires the expected prior binding identity; missing paths can be reconnected.

Bindings must match the selected signet ID and registration; never fall back to
another bank's connection. Sources cannot overlap the selected signet/local state.
Another historical memory directory is consulted only when explicitly registered
as a reference, never through discovery of another bank's binding. Refuse
symlinks, traversal and special files; return bounded diagnostics, not secrets or
raw subprocess output. Separate signets have separate registrations/connections.

## Local source verification

Support regular UTF-8 Markdown, text and JSON documents. Do not execute scripts,
load native skills, interpret JSON as code, include binary content or automatically
consume raw transcript formats. Report eligibility/exclusions and limitations; no
heuristic can guarantee a selected document contains no sensitive prose. Instructions
must prohibit saving secrets or raw transcripts and require appropriate source choice.

Local directories use a deterministic SHA-256 over canonical selected file paths
and bytes. Git sources use a pinned commit and credential-free portable locator,
with an explicitly connected existing standalone checkout. Use bounded read-only Git
commands to resolve the selected commit/tree and compare eligible tracked files with
their exact blob identities and the observed HEAD/source identity. Disable lazy fetch,
replacement objects, fsmonitor, hooks and interactive transport; never run filters,
checkout, status refresh, submodules or source programs. Exclude untracked/binary files
explicitly rather than labeling them as part of the pin. No full checkout-cleanliness
or remote-freshness claim is made. Missing objects or incompatible Git fail visibly.

Inspect compares actual selected content, identity and pin without updating it.
Missing/unavailable, changed, disconnected and conflicting states withhold normal
pinned retrieval/promotion. Reconnection/repinning is explicit. Recheck identity and
content before returning evidence or writing a promotion; source changes invalidate
previous evidence rather than acquiring the old pin's authority.

Bound enumeration and reads: initially 10,000 eligible files, 64 MiB total eligible
bytes, 4 MiB per file and bounded directory/metadata enumeration. Git commands have
deadlines, output caps and process-group cleanup. Implement shared low-level helpers
only where real reuse warrants them; do not depend on the migration converter.

## Interfaces and authority

Typed CLI administration covers preview, register/update, local bind/rebind and
disconnect. These operations never use an unrelated implicit signet. Bound agent
operations cover list, inspect, search, read and verified promotion. Exact command
spelling can follow existing conventions; all routes use the shared dispatcher,
strict schemas, no-write enforcement and bounded envelopes. No connector silently
fetches, writes the source or synchronizes the signet.

List/inspect disclose routing/state, not whole documents. Search uses scoped lexical
matching within an explicitly selected foundling and returns a small deterministic
set of excerpts. Read selects an exact safe locator and bounded UTF-8 excerpt range.
Results contain source identity, registration revision, pin, relative locator and
actual file SHA-256 plus excerpt/completeness markers. Clip only labeled reference
excerpts, never present a clipped claim as complete current knowledge. Apply an
overall serialized byte budget and report truncation/empty-result limitations.

Promotion selects a previously observed locator/hash and current registration plus
a normal memory Write. Reverify that source, generate ExternalOrigin from verified
metadata, preserve optional explicitly known original author/date, then delegate to
the existing sourced-memory writer. Incorporation device/time/harness remains the
current binding's authorship. Do not infer document authorship from a checkout owner
or Git commit author. Preserve a reason explaining retention/qualification/change.

Ordinary memory_remember retains its existing ability to store supplied citations;
a structurally valid supplied citation remains distinct from observed verification.
The dedicated promotion route verifies the source at incorporation, not forever.
The host agent is responsible for semantic comparison/deduplication and current-user
authority: recall existing records first, use explicit record IDs/supersedes for
corrections, and clarify consequential ambiguity. Storage does not choose a winner
or automatically promote instructions because a lexical match exists.

## Failures and recovery

Portable registration and local binding cannot be one cross-filesystem transaction.
Report completed registration and failed connection separately; inspect before retry.
A local source can disappear without harming signet reads or historical citations.
Disconnecting retains all evidence. Concurrent registrations require explicit head
selection/supersession; a stale update may create a visible conflict, not a silent win.
Failed promotion creates no revision with missing evidence; existing sourced-write
partial-I/O semantics remain in force. No automatic deletion, rollback or source repair.
