# Technical design

## Boundaries and commands

Provide typed bound operations for Git initialization, checkpoint, sync and
read-only sync status, surfaced by the existing API catalog/CLI/MCP. Initialization
is explicit and never implied by recall, remember or server startup. Set up origin
through native Git/local installer configuration, not portable signet JSON.

The synchronizer pins root and signet ID from the already selected memory service.
Use the same nonblocking lock as memory writes. Require a real .git directory,
matching top-level/worktree and main branch, with no merge/rebase/cherry-pick
state or linked/common repository indirection. Refuse unsafe index/worktree state
before staging; never discard staged content or partially staged edits.

Scope automatic checkpoints to validated signet files. Reject unknown tracked or
untracked material and edits/deletions of existing evidence (memory, provenance,
foundling registrations) before git add. Preserve local files/index on rejection.
Existing signet identity cannot change. Modified human README metadata is not a
license to execute hooks or include unrelated files. Existing valid pending signet
records can be committed; exactly-once checkpointing is based on Git content.

## Git process policy

Remove environment variables that redirect Git root/index/config/objects, specify
the intended repository explicitly, disable hooks, interactive prompting, editor
and signing for toolkit checkpoint commits. Preserve native credential helpers
and SSH agent/host trust, not arbitrary portable helper scripts. Do not change
global Git/shell configuration. Git author defaults are synthetic toolkit identity
unless explicitly configured locally; record-level device/actor remains authority.

Accept explicit local file, SSH or HTTPS origin forms without embedded secrets;
refuse remote-helper syntax, option-like/control-character inputs and unsupported
transport. Limit child-process output and duration, cancel the process group and
bound pipe cleanup. Raw stdout/stderr stays out of public receipts/logs.

## Reconciliation

Checkpoint valid pending local memory first. Fetch only origin main into an
explicit remote-tracking ref; distinguish an empty remote from unavailable auth.
Use exact object IDs rather than a mutable branch name for candidate validation.
Require connected histories; do not auto-merge unrelated signets.

When both histories advanced, use merge-tree and commit-tree to construct a
candidate without mutating the worktree. Validate bounded regular data files in
a disposable snapshot: reject symlinks, special paths, changed signet identity,
invalid format/graph, unknown content and rewritten durable evidence. Validate
against both histories so the candidate cannot drop either side's evidence.
Integrate only after checks, preserving user work if files/index changed during
the operation. No force update, destructive reset or automatic conflict editing.

Push current verified main normally. A concurrent remote writer can reject it;
keep local history and return pending. Receipt fields distinguish state,
local checkpoint/head, observed remote head/check time, successful delivery and
semantic conflict count. Git transport success does not resolve competing heads.

## Local status and cancellation

Persist a bounded machine-local last-sync receipt under ignored .mandalore state,
not in portable memory. Status does not create state or run network calls. Compare
current head/worktree with the receipt before describing it as last synchronized.
Return local-only/pending/conflicted/synchronized with freshness qualifiers.

Use cooperative cancellation boundaries before mutation and context-bound Git
processes. A cancelled or failed push can be ambiguous; preserve commits and
report the observed phase. Do not claim remote rollback or retry-safe writes.
Local reads must observe subsequently integrated records without reopening the
MCP server, while already-read model context remains the native agent's concern.
