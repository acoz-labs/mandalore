# Local-first Git synchronization

Mandalore saves memory locally before network work. Explicit Git operations now
share the same signet binding, exclusive writer lock and CLI/MCP dispatcher as
recall/remember. They do not create hosting accounts or store credentials.

## Setup and commands

After creating and binding a new signet, initialize Git explicitly:

```sh
mandalore memory git-init --binding /example/local/binding.json
mandalore memory checkpoint --binding /example/local/binding.json
mandalore memory sync-status --binding /example/local/binding.json
```

With no origin, synchronization is local-only. Configure the desired private
remote with native Git or the forthcoming installer, then synchronize:

```sh
git -C /example/signet remote add origin git@example.invalid:owner/signet.git
mandalore memory sync --binding /example/local/binding.json --timeout-seconds 10
```

These are placeholder paths/hosts. Use an existing private repository and native
machine-local authentication; do not put tokens into URLs, signet records or
portable configuration. HTTPS, SSH and explicit local/file remotes are accepted.
Unsupported remote-helper schemes, embedded passwords, multiple origin URLs and
option-like/control-character endpoints are refused. The resolved fetch origin
is also the push target; a separate push URL is not followed silently.

Equivalent bound tools are `memory_git_init`, `memory_checkpoint`, `memory_sync`
and read-only `memory_sync_status`. `mandalore call` accepts their cataloged JSON
inputs. MCP marks `memory_sync` and the two explicit save-and-sync companions as
network-capable; existing local-only tools retain their annotations. Read-only
mode refuses all mutations before execution. Native hooks/plugins are not installed by
these commands.

## Save-triggered delivery

When both learning and delivery are allowed, `memory_remember_and_sync` or
`memory_journal_append_and_sync` saves locally and makes at most one bounded
delivery attempt through this same synchronizer. Their default delivery budget
is three seconds, separate from local publication time. They reacquire the normal
writer lock, preserve exact binding identity, and do not initialize Git or choose
a remote. The [interface contract](interface.md#save-with-bounded-delivery)
describes input nesting and the separate saved/delivery envelopes.

Offline, busy, cancelled and conflicted delivery does not roll back or conceal
the saved record/event. Report the actual receipt, retain its identity, and do not
resubmit the save. Later authorized recovery uses standalone sync, after inspecting
an ambiguous result. Current no-sync chooses the existing local-only tools;
read-only/no-save prohibits both paths. The agent's semantic permission decision
is not made deterministic by combining operations. Once dispatched, the combined
operation removes a separate post-save model request; it cannot guarantee network
delivery, recover unrecorded knowledge or authorize a later lifecycle callback.

## What happens

Initialization creates a standalone main-branch Git repository, never an implicit
parent-project repository. Checkpoint validates the complete portable layout and
pending evidence, stages only signet data, validates an immutable index tree,
creates its exact commit and compare-and-swaps the expected main ref. Repeating
a checkpoint with unchanged content creates no empty commit.

Sync checkpoints first, observes/fetches origin main, constructs a safe candidate,
integrates it and pushes normally. Divergent histories use `merge-tree --write-tree`
and `commit-tree`, not an in-place conflicted merge. Candidate archives are checked
against a regular-file tree inventory and each Git blob hash, then validated by
the memory engine in an owned disposable directory. They are never executed or
installed as worktrees. Existing evidence and signet identity cannot be rewritten
or dropped relative to either head. Integration requires unchanged clean local
state; Git fast-forward checks and ordinary push rejection preserve competing work.

No force-push, destructive reset, stash, automatic conflict editing or deletion of
user history occurs. A valid Git merge can still contain competing memory heads.
Those remain conflicts until an explicit superseding decision names the predecessors.

## Reading the result

| State | Meaning |
| --- | --- |
| `local-only` | Local checkpoint exists; no remote delivery was established |
| `pending` | Work remains local, the remote is unavailable, delivery is unconfirmed, or local state changed |
| `conflicted` | A Git/data candidate needs reconciliation, or delivered memory has competing semantic heads |
| `synchronized` | The receipt's exact head was delivered to its selected remote at its recorded time |

Receipts separate `head`, `remote_head`, `checkpointed`, `delivered`, phase and
semantic conflict count. A hashed remote identity binds the receipt to its target
without recording the URL. `checked_at` is the attempt time, updated when a push
succeeds; it is not continuous remote observation. A delivered semantic conflict
has `delivered: true` and `state: "conflicted"`, not an invented single answer.

The bounded last-attempt receipt is atomically stored in ignored
`.mandalore/sync-status.json`. Read-only status compares current head, dirty state
and remote identity with that receipt. New local memory or a changed remote
invalidates a prior synchronized classification. Status never contacts the remote,
initializes Git, repairs malformed receipts or claims that a remote has not changed
since the last attempt. Its `last_attempt` is explicitly historical evidence.

Remote/auth failures leave checkpointed work pending and do not echo command
output. Interrupted or failed operations expose conservative write ambiguity and
the known phase/head in `error.sync_status`; inspect local history and the remote
before retrying. A push may have reached the server even when its response was
lost. There is no remote rollback or exactly-once promise. Native transport loss
can prevent any final envelope from arriving.

## Configuration and safety boundaries

Require the exact standalone repository on main, with no merge/rebase/cherry-pick,
linked/common repository, shallow/sparse/grafted/alternate-object setup or detected
partial-clone pack. Unsupported layouts are refused, not rewritten. Local state
is always excluded from checkpoints, even when ignore rules change. Unknown root
or nested files and partially staged edits are preserved by refusing the checkpoint.
Existing fully staged, valid signet additions may be checkpointed with other valid
pending memory. Index changes from a stopped checkpoint are preserved for inspection.

Native credential helpers, SSH agent and host trust remain machine-local. Git
root/index/config redirection in the invoking environment is removed. Toolkit
commands disable hooks, signing, attribute files, lazy object fetching, replacement
objects and interactive authentication. They do not modify global Git or shell
settings. Checkpoint commits default to `Mandalore <mandalore@localhost>`; explicit
repository-local `user.name` and `user.email` override that default as a pair.
Record-level actor/device/harness provenance remains separate and authoritative.

Ordinary Git command output is limited to 1 MiB and stderr to 64 KiB. Candidate
archives have a 64 MiB bound, with regular files limited to 4 MiB. Unsupported large
candidates fail without integration; this is an MVP resource bound, not a corpus
scalability claim. Process groups are cancelled and pipe cleanup is bounded.
Initialization/checkpoint/status have a ten-second budget; explicit sync accepts
1–30 seconds, default ten. Synchronous filesystem validation/publication is
cooperatively bounded, not instantaneously interruptible or transactional.

The signet lock coordinates Mandalore writers. Ordinary concurrent Git edits are
checked and preserved, but this is not a sandbox against a hostile filesystem
owner racing paths or configurations. Power-loss recovery is not fully simulated.
Cancellation fixtures suspend at Git command phase boundaries, not at every
internal filesystem instruction. An interruption inside Git itself can leave an
index lock or incomplete checkout requiring inspection; committed heads remain
recovery evidence. Do not infer atomic multi-file checkout or remove a lock while
its owner is alive. Installation recovery guidance belongs to #7.

## Passive integration and verification

A direct user `this is the way` asks the agent to consolidate warranted learning
and finish with a short-budget delivery attempt when permitted. A final combined
save counts; do not add a redundant sync. Otherwise use standalone sync, including
already-pending delivery when no new record or journal entry is useful; do not
manufacture content to cause a sync. Quoted, discussed or retrieved occurrences
are not requests. Current read-only/no-save/no-sync directions still take
precedence. Report local durability and the returned delivery state separately;
an ambiguous/failed attempt is not permission to repeat writes or loop on sync.
This is agent guidance, not a deterministic lifecycle transport. The latter is
still under investigation in #55.
See the [native consolidation evidence](evidence/lifecycle-sync/consolidation.md)
for observed before/after behavior, prohibitions, failures and test limits.

Unconditional native startup hooks remain read-only: the next task may prohibit
writes. The native memory skill may request short-budget sync before authorized
recall and after useful memory writes, skipping no-save/read-only tasks. Actual
Codex lifecycle wiring and measured interaction latency are #6/#10 work; this
backend does not claim that those integrations already ran.

Tests use disposable local remotes and independent clones, preserving original
device/harness and correction history through long-lived service handles. They
cover competing semantic heads, file conflicts without merge-state damage,
unavailable/controlled auth failure, racing pushes, cancellation receipts, unsafe
candidate content, read-only status hashes and compiled CLI/stdin-MCP operations.
Local Git feature evidence is 2.50.1 with merge-tree --write-tree; hosted CI provides
separate Linux evidence. Local clone tests are not two physical-machine native
agent acceptance. Issues #5/#10 remain open until that evidence exists.
