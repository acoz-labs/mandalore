# Local-first Git synchronization

Mandalore saves memory locally before network work. In 2.0 the
[server](server.md) is each signet's only writer and owns its Git backup (see
[Server backup](#server-backup)). The Git toolkit below is shared by that backup,
the explicit CLI operations and the guided menu. It does not create hosting
accounts or store credentials.

The 1.x transport, where each device synchronized its own clone around agent
sessions, left the build with 2.0. The CLI operations and the menu's
synchronization below remain for 1.x clones until [the cutover](migration-2.0.md).

## Setup and commands

After creating and binding a new signet, initialize Git explicitly:

```sh
mandalore memory git-init --binding /example/local/binding.json
mandalore memory checkpoint --binding /example/local/binding.json
```

With no origin, synchronization is local-only. Configure the desired private
remote with native Git, then synchronize it from the guided menu (`mandalore
menu`, "Synchronize with its configured remote"):

```sh
git -C /example/signet remote add origin git@example.invalid:owner/signet.git
```

These are placeholder paths/hosts. Use an existing private repository and native
machine-local authentication; do not put tokens into URLs, signet records or
portable configuration. HTTPS, SSH and explicit local/file remotes are accepted.
Unsupported remote-helper schemes, embedded passwords, multiple origin URLs and
option-like/control-character endpoints are refused. The resolved fetch origin
is also the push target; a separate push URL is not followed silently.

Equivalent bound tools are `memory_git_init` and `memory_checkpoint`;
`mandalore call` accepts their cataloged JSON inputs. Read-only mode refuses all
mutations before execution. Native hooks/plugins are not installed by these
commands.

## Server backup

In 2.0 the [server](server.md) is each signet's only writer. It never fetches on
reads; after writes it checkpoints and pushes `main` without forcing, and refuses
a remote that is ahead until `mandalore server reconcile` fast-forwards it. The
1.x `memory_sync`, `memory_sync_status` and save-and-sync tools are removed, so
agents have no synchronization tools. The guided menu still synchronizes a
selected 1.x clone directly until its 1.x journeys are removed.

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

## Opt-in format transitions

Format2 adds portable visibility events and append-only upgrade receipts.
Ordinary sync does not opt a format1 clone into that format. After fetching an
upgraded remote, it reports `upgrade-required` and preserves the local bank/HEAD.
The user must stop writers, preview and explicitly apply that clone's local
upgrade, then separately synchronize. Runtime installation is not bank migration.

The only allowed protected-manifest edit is the validated format1-to-format2
transition with the same signet identity/name. Each receipt must identify an
ancestor format1 Git commit, match its original manifest and raw portable
inventory hashes, and preserve all original protected evidence. Unknown edits,
missing/forged proof, unrelated base commits and downgrades refuse integration.
Validation of historical base bytes is bounded to 64 MiB. Independent upgrades
can converge while retaining both receipts; one receipt never authorizes another
clone's local upgrade.

Visibility events merge as append-only evidence. Delivery can succeed while
concurrent visibility heads or unreviewed content remain withheld conflicts.
Neither sync nor a successful push selects a semantic winner or restores a
withdrawn record. See [the format](signet-format.md) for causal visibility rules
and [the runbook](runbook.md) for stopped-writer recovery. Old offline clones and
already-read model context remain outside revocation.

## Reading the result

| State | Meaning |
| --- | --- |
| `local-only` | Local checkpoint exists; no remote delivery was established |
| `pending` | Work remains local, the remote is unavailable, delivery is unconfirmed, or local state changed |
| `conflicted` | A Git/data candidate needs reconciliation, or delivered memory has competing semantic heads |
| `upgrade-required` | A format1 local clone observed an upgraded remote; explicit local upgrade is required before adoption |
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
Initialization/checkpoint/status have a ten-second budget, and the menu's
synchronization has ten seconds. Synchronous filesystem validation/publication is
cooperatively bounded, not instantaneously interruptible or transactional.

The signet lock coordinates Mandalore writers. Ordinary concurrent Git edits are
checked and preserved, but this is not a sandbox against a hostile filesystem
owner racing paths or configurations. Power-loss recovery is not fully simulated.
Cancellation fixtures suspend at Git command phase boundaries, not at every
internal filesystem instruction. An interruption inside Git itself can leave an
index lock or incomplete checkout requiring inspection; committed heads remain
recovery evidence. Do not infer atomic multi-file checkout or remove a lock while
its owner is alive. Installation recovery guidance belongs to #7.

## Verification

Tests use disposable local remotes and independent clones, preserving original
device/harness and correction history through long-lived service handles. They
cover competing semantic heads, file conflicts without merge-state damage,
unavailable/controlled auth failure, racing pushes, cancellation receipts, unsafe
candidate content, read-only status hashes, server push-only backup, remote-ahead refusal and
fast-forward reconcile.
Local clone tests are not two physical-machine native agent acceptance. Native
journeys and platform-specific execution evidence must identify the exact
candidate and tested versions; cross-builds alone do not establish either.

## Canon reference freshness

Canon reference fetching is distinct from the primary signet's backup.
Registered canon sources are read-only inputs: the server refreshes them at
startup and on a schedule, never pushes upstream and saves no copied semantic
knowledge. Every client reads the latest verified snapshot; reads never fetch.

Inspect each reference's state with `foundling_status` or `mandalore server
status`, separately from the signet's backup state. A successful push does not
prove reference freshness, and a successful reference fetch does not back up
primary writes. Stale-cache fallback identifies its observed commit and time;
absent or ineligible snapshots remain unavailable. See the
[foundling contract](foundlings.md) for eligibility and provenance.
