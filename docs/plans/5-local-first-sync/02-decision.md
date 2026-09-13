# Solution decision

## Comparison

Plain git pull/push is small but can leave merge state, execute repository hooks,
stage unrelated work or integrate unvalidated changes. It does not distinguish
semantic conflicts from file conflicts or bound the unattended authentication path.

The pinned predecessor already checkpoints, uses merge-tree for divergent heads,
validates a candidate before fast-forwarding and preserves offline commits. Reuse
these characterized ideas, not the portable/assistant package, source-change
observer, custom GitHub source account adapter or portable credential helper code.

A background daemon/remote memory service adds process and deployment ownership
without being necessary for this local-first MVP. Defer it; explicit bounded
operations plus thin native integration are sufficient to prove continuity.

## Selected approach

Use internal/sync with the existing memory validation and a shared exclusive lock.
The engine itself gains only a narrow lock boundary; it acquires no Git/process
dependency. Actual Git commands run against an explicit root under cancellation
and output/time limits. Native machine-local authentication remains native.

Validate a candidate in a disposable data-only snapshot before integration.
Never execute candidate content or install a worktree from unvalidated data.
Refuse modified/deleted durable evidence relative to the local checkpoint; valid
appended records and explicit supersession are the automatic reconciliation unit.

Use append-only merge commits for divergent histories and normal fast-forwards
otherwise. Push refusal is pending, not a reason to force or overwrite. Semantic
conflicts can be delivered successfully while still requiring user resolution.

The ordinary operation timeout is bounded and configurable downward for native
automation. Cancellation reports the known phase/commit, never a fictional rollback.
Automatic retry is not promised for ambiguous outcomes; status/history inspection
and a new explicit synchronization attempt are safe recovery entry points.
