# Plan: move a record to another scope without losing its id or history

## Issue

Refs #250 (https://github.com/acoz-labs/mandalore/issues/250). Outcome: a
read-write client moves a misfiled record to another scope with a reason; the
record keeps its `record_id`, kind, content and history, the move is new
append-only evidence, and reads follow it. Ships as minor release 2.3.0,
server and Claude plugin together.

### First question: no signet format bump

**The move ships without a signet format bump, and existing data stays valid
byte for byte.** A move is a new document in a new directory that no released
reader opens:

```text
memory/moves/<record-id>/<move-id>.json
```

No existing file, schema or directory changes. Revision schemas 1 and 2,
visibility events, journals, labels and the manifest (format 1 or 2) are
untouched, and `memory/moves/` is created only by the first move. Both formats
get the feature. This is the 2.2.0 precedent for `memory/scopes/`
(`docs/signet-format.md`, "Scope labels and scoped journals").

The design keeps one existing invariant on purpose: **every revision of a
record keeps the scope its root was saved in**, including corrections made
after a move. The record's current scope comes from its moves. Evidence from
the code:

- Graph validation refuses a successor whose scope differs from its
  predecessor's (`internal/memory/memory.go:260`, "successor … changes record
  identity, kind, or scope"), and every read validates the whole record graph
  (`Store.graph` in `internal/memory/cache.go`, through
  `validateGraphState`). If a post-move correction stored the new scope,
  2.2.0 and 2.1.2 would refuse the whole graph, and recall would fail in every
  scope on that signet, including after a rollback. The 2.2.0 plan rejected a
  new scope kind for the same reason, as one that would force a format bump.
- Released readers never open an unknown directory under `memory/`. `Validate`
  walks only records, sources, devices, foundling registrations, labels and
  journals (`internal/memory/store.go:453-495`); the layout check requires
  known directories to exist but refuses no extra one (`checkDirectoryLayout`,
  `store.go:288`). So 2.2.0 keeps serving such a signet.
- Released sync, checkpoint and export refuse an unknown path, so an old clone
  can't pull a move in without refusing first. Sync admits only the
  `portablePaths` patterns (`internal/sync/checkpoint.go:142-151`); the export
  report snapshot refuses anything else as "unknown portable data"
  (`internal/exportreport/snapshot.go`, the `default` case).

So an older reader sees a moved record in its old scope, with any later
corrections, and can't sync until it runs 2.3.0. This table is the
compatibility cost; the release notes say it, and `bin/check-old-reader`
asserts each row against 2.2.1 (95bbf17, the current release, which the live
server runs) and 2.1.2 (79b378e) built from
source:

| 2.2.1 or 2.1.2 on a signet with a move | Behaviour |
| --- | --- |
| Recall, prime (2.2.1), scopes, context, history, inspect | Work; the record shows in its old scope; moves are invisible |
| Correction with the old scope | Succeeds; once 2.3.0 is back the record is still in its new scope, with that content |
| Correction with the new scope | Refused by graph validation, nothing written |
| Reconcile, menu sync, checkpoint with the new path | Refuse; files, index and unpushed commits kept (as the 2.2.0 table) |
| The server's backup after a write (checkpoint, then push) | The write saves; the backup's checkpoint refuses on every retry, and status reads `pending`, remote `unreachable`, "push not confirmed; retrying" |
| `server status` | Works, and shows that pending backup: it reads the saved backup and remote state without checking the worktree |
| The menu's status (sync inspection) | Refuses: "Cannot read local synchronization state" |
| Format 1→2 upgrade preview, and apply of a plan from 2.3.0 | Refuse; the signet stays format 1, with no retained preparation |
| `server migrate init` from a remote with a move | Clones and validates, moves the checkout to root and enrolls the server device, then refuses at the enrollment checkpoint; the configuration is unchanged and the partial-write error names the checkout (recovery in `docs/migration-2.0.md`) |
| `server migrate verify` | Saves a canary record and withdraws it, then refuses at its checkpoint; the canary stays in the checkout, saved, withdrawn and uncommitted, and the partial-write error names that stage |
| Export preview | Refuses |
| Canon refresh of a source with a move | Serves the record in its old scope |
| `memory_move` | Not served |
| Cutover preflight with a move intent in `.mandalore/` | Refuses: "pending write intents are unreadable" |
| A `memory_remember` retry that reuses a move's `request_id` | `save.incomplete`, nothing written |

The rows come from a sweep of every caller of the sync path check
(`allowedPath` and `validateWorktree`) at both commits, which have the same
callers: checkpoint (`internal/sync/checkpoint.go`), the clean-tree check that
reconcile and fast-forward run (`remote.go:182`, `backup.go`), candidate
validation for reconcile, menu sync, checkpoint and the upgrade proof
(`candidate.go:59` and `:98`), sync inspection (`status.go:168`), and the
upgrade source (`upgrade_source.go:62` and `:84`). Server code reaches them
through the post-write backup (`internal/server/signet.go:344`), reconcile
(`admin.go:125`) and `server migrate init` and `verify`
(`internal/server/migrate.go:210` and `:360`). `migrate init` passes its
`Validate` (`migrate.go:146`), which ignores `memory/moves/`, so it refuses
late, at its checkpoint, not before cloning. One more use, listing ignored
portable files (`status.go:243`), only selects paths and refuses nothing.

The alternatives are worse. Storing the new scope in post-move revisions
breaks every read on older readers and after a rollback. A format bump would
make old clones refuse explicitly, but costs every user the explicit upgrade
(clean checkpoint, stopped writers, preview and apply) for a feature that
otherwise rolls out like 2.2.0. If the plan review prefers the bump, I stop
and the lead reports.

### Release

2.3.0, the next minor after 2.2.1 (the go1.26.11 patch release; 2.2.0
before it): a new write tool and new files in the
signet, as 2.2.0 was for topics. The skill text changes, so
`plugins/claude/.claude-plugin/plugin.json` moves to 2.3.0 with `VERSION`
(`docs/repository.md`: a plugin change bumps `plugin.json`, and either bump
means an artifact release). The PR says `Refs #250`.

### Recall paths change

**Yes: `memory_recall`, `memory_context`, `memory_prime` and `memory_scopes`
change**, so the requester runs the private recall eval before publication.
What changes is scope selection only: each reads a record's current scope from
its moves instead of from its revisions. Ranking, term statistics, packing and
the context hook's thresholds are not touched. Canon recall and withheld-record
discovery switch to placement the same way.

On a signet with no moves, recall, context, prime, scopes, canon recall and
withheld-record discovery output stays byte-identical to 2.2.0's. The golden
outputs were captured from main (e3c9712, 2.2.0's code plus its release
record) before any code change, in their own commit (ca4980e). The generator
is the test itself (`TestNoMoveReadsMatchGoldens` in `internal/server`, run
once with `MANDALORE_WRITE_NOMOVE_GOLDEN=1`): it builds the fixtures under
`internal/memory/testdata/compat/v2.2.0/` from the 2.1.2 ones with main's
code, and writes the outputs under `internal/server/testdata/nomove-golden/`.
This PR never regenerates them; the test compares 2.3.0's output with them.
They stand for 2.2.1 too: its diff from 0d3c22d changes only toolchain pins
and no read code.
Canon recall is memory recall on a verified snapshot, and canon promotion's
head lookup is an exact-scope store recall, so both are pinned through those
calls. History pages are pinned on their `items`, `next_offset` and
`truncated` only.

`memory_history` is the exception: its output changes on every record, moves
or not, because it always gains `scope`, `move_heads` and `moves` (below).
`docs/interface.md` and the release notes say so.

## Behaviour

**The tool.** `memory_move` (read-write only), a server tool with
`request_id` like `memory_remember`, and a catalog operation for
`mandalore call memory_move` without it, as `memory_name_scope` is:

```json
{
  "request_id": "move-example-request-1",
  "move": {
    "record_id": "record-example",
    "scope": {"kind": "project", "id": "example-y"},
    "expected_content_heads": ["revision-example"],
    "expected_move_heads": [],
    "reason": "Filed under the wrong project."
  }
}
```

`scope` is the destination: project, account, task or signet-wide (the
signet's own ID). The two head lists are the compare-and-swap input, as
withdraw takes content and visibility heads: each 0–32 IDs, no duplicates,
content heads at least 1. `reason` is 1–1024 bytes. Both lists come from
`memory_history`, which gains `scope` and `move_heads` (below).

**The move document** (schema 1): `schema_version`, `id` (`move-…`),
`record_id`, `scope` (destination), `observed_content_heads`,
`parent_move_heads`, `reason`, `recorded_at`, `authorship`. Validation: the
path matches the record and ID; the record exists; observed content heads
exist and belong to the record; parent moves exist, belong to the record and
form no cycle; the device is known; a signet-wide scope uses this signet's ID.
A move does not depend on visibility events and is valid on format 1.

**Current scope.** No moves: the root's scope, as today. Otherwise every
unsuperseded move is a current move head. If every head names the same scope,
that is the current scope, with one head or several. Heads that name
different scopes are a move conflict (below). A move from one clone may have
no parents, and so may a concurrent first move from another; both are valid,
so several move roots are allowed (the two-clone lesson, below).

Two clones that both move a record to the same scope therefore don't
conflict: they agree, nothing is ambiguous, and refusing corrections until
someone settles a disagreement that doesn't exist would only get in the way.
Both moves stay in history as evidence. The next move must still name every
move head in `expected_move_heads`, so it supersedes both.

**Guards**, in order, all under the writer lock, nothing written on refusal:

1. Input shape and limits: `input.invalid`.
2. Record not found: `input.invalid`.
3. More than one content head and no move conflict: `input.invalid`,
   "record has conflicting revisions; settle the conflict with a correction
   that supersedes every head first". With a move conflict too, the move is
   let through when it settles the move conflict (step 5 then requires every
   move head and every content head), so a record with both conflicts can be
   settled: the move first, then a correction.
4. Format 2 and the record isn't visible (withdrawn or a visibility conflict):
   `input.invalid`, "record is withdrawn; restore it before moving it".
5. Expected content or move heads differ from the current ones:
   `memory.stale_heads`, on both formats (this is a new tool, so it is strict
   from the start).
6. No move conflict and the destination is the current scope: success with
   `moved: false` and no move file. With a `request_id`, the intent records
   this no-op result (below).

A move conflict is settled by a move whose `expected_move_heads` names every
current move head; it is written even when its destination is one of theirs.

Two clones that each correct and move the same record before syncing leave
both a content conflict and a move conflict. Neither tool is blocked by the
other: the settling move goes first (guard 3), and once the record has one
scope, a correction in that scope that supersedes every content head settles
the content conflict.

**Receipt:** `signet_id`, `record_id`, `scope` (current after the call; on a
retry, the first call's),
`moved`, `id` (the move ID, absent on a no-op), and the server's durability
and backup fields as for `memory_remember`.

**Idempotency.** The intent document (`writeIntent`) gains an optional
`move`, holding either the move document or, for a no-op, the no-op result
(`record_id` and `scope`, no document). The digest covers the whole input.
Same `request_id` and input returns the first result without rechecking the
guards: a real move republishes its document if it is missing and returns its
receipt; a no-op returns `moved: false` with the scope it recorded, even after
a later move elsewhere. So a retry after a later move never turns into
`memory.stale_heads`, and the receipt is the first call's, not a fresh read.
Different input returns `request.conflict`; `store.busy` and
`save.incomplete` work as for `memory_remember`.

`PendingIntents` learns the move target, so a completed move isn't reported as
pending and an interrupted one is. A no-op intent has nothing to publish and is
never pending; today an intent with no target counts as pending, so that case
is handled explicitly. The existing kind checks in `RememberIdempotent`,
`AppendJournalIdempotent` and `NameScopeIdempotent`
(`internal/memory/idempotency.go:54`, `:120` and `:157`) learn the new field,
so reusing a move's `request_id` for any of them is `request.conflict`, and
reusing theirs for a move is too. Existing intent shapes and digests don't
change.

**Corrections after a move.** `memory_remember` with `record_id` and
`supersedes` compares the input scope with the record's current scope under
the writer lock. A match is accepted and the revision is stored with the
record's root scope, keeping the invariant above; a mismatch is refused as
`input.invalid` naming the current scope ("record was moved to project
example-y; correct it in that scope"). During a move conflict a correction is
refused: "record has conflicting moves; settle them with memory_move first".
That never deadlocks, because the settling move is allowed during a content
conflict (guard 3).
Foundling promotions that correct a record go through the same path. A new
record (no `record_id`) is unchanged.

**Reads.** The store resolves each record's placement next to its visibility
state, in the same validated graph read, and the read cache drops it on every
write as today. Recall, context, prime, scopes, withheld-record discovery,
canon recall and canon promotion select by placement, and returned hits and
receipts carry the current scope. A moved record keeps its own `recorded_at`,
so prime orders it by when its content was written, not when it moved.

**Move conflicts.** Two clones that move the same record to different scopes
and sync leave two move heads. The record is never current guidance while
they stand: recall and prime in each destination scope list it under
`conflicts` with its content heads and the competing scopes (`move_scopes`);
`memory_context` surfaces it as a conflict when relevant, as for content
conflicts. Neither move is lost: history lists both.

`memory_scopes` counts it in `record_count` of each destination scope, and
each scope item gains `move_conflicts`, the number of its records in a move
conflict, on both formats. The field is omitted when zero, so scope output
without moves doesn't change. Format 1 has no other count for it. On format 2
the record also counts in `visibility_counts`: as `conflicted` when it is
otherwise visible, so visible, withheld and conflicted still add up to
`record_count`; a withdrawn record stays `withheld`, as today.

**History.** `memory_history` keeps its paged `items` (the stored revisions,
unchanged) and gains, on every record:

- `scope`: the current scope, absent during a move conflict;
- `move_heads`: the current move heads, `[]` when never moved;
- `move_count`: how many moves the record has;
- `moves`: the record's moves newest first, each with `from` (derived: the
  root's scope, or the scopes of its parent moves) and the stored fields.

`moves` is filled on the first page (offset 0) only, and is `[]` on later
pages; `scope`, `move_heads` and `move_count` are on every page. The revision
items keep 2.2.0's 32 KiB page budget, measured on the page of revisions
alone, so every page holds the same revisions as on 2.2.0, moves or not
(plan review round 2, L3). The moves have their own 16 KiB budget on the
first page; the whole response stays under the 64 KiB output limit. If the
moves don't fit, the newest that fit are kept, whole, and `moves_truncated`
is set; `move_count` says how many there are.
The tool description says a revision's `scope` is the scope the record was
saved in, and `scope` at the top is where it is now.

**Export reports.** The snapshot reads and validates `memory/moves/`.
Selecting a scope selects the records currently in it. A record item's
`identity.scope` is the current scope; with `include_history`, moves are
reported as `type: "record_move"` items in the existing field categories
(`identity`: move and record IDs; `content`: scope and reason; `timestamps`;
`authorship`; `change_history`: observed and parent heads). Report `version`
stays 1.

**Sync, canon and upgrade.** `portablePaths` admits
`^memory/moves/<id>/<id>\.json$`, and sync's candidate validation checks
moves with the rest of the graph. Synchronization's `conflicted` state and
`semantic_conflicts` count include move conflicts, each record once, though
it is listed in every scope its moves name. Canon fetch already copies every file under
`memory/`, and 2.3.0's snapshot validation checks moves. The format 1→2
upgrade proof already protects everything under `memory/` as append-only
evidence; a test proves a format 1 signet with a move upgrades with it intact.

**Guidance.** The server instructions have no room: read-write is 595 bytes
of the 599 `TestInstructionsAreShort` allows. So the line goes in the
`this-is-the-way` skill, conditional like 2.2.0's topics paragraph: "To
re-home a misfiled record, use `memory_move` (only when the server lists it)
rather than saving a copy and withdrawing the original." The server-side
`memory_remember` description gains the same pointer, which is safe because
only a 2.3.0 server sends it. `references/memory.md` gains the move rules.

## Files that change

- **Moves in the store** (new `internal/memory/moves.go`): the move type,
  strict reading of `memory/moves/`, validation, placement resolution and the
  append under the writer lock.
- **Graph and cache** (`internal/memory/memory.go`, `cache.go`): placement in
  the validated graph read; `Scopes` and `recallHeads` select by placement;
  `Validate` checks moves.
- **Service** (`internal/memory/service.go`, `pages.go`, `prime.go`,
  `idempotency.go`, `sourced.go`, `visibility_discovery.go`): `Move` and its
  idempotent form, the correction scope check, history's new fields, prime and
  withheld discovery by placement, the `move` intent and its
  `PendingIntents` target.
- **Snapshot** (`internal/memory/snapshot.go`): `Moves` in the snapshot and
  its validation.
- **Catalog and MCP** (`internal/api/api.go`, `remember.go`;
  `internal/server/tools.go`): `memory_move` as a catalog operation and a
  read-write server tool with byte limits through `limit`, added to `isWrite`;
  descriptions. The CLI's generic `memory <name>` mapping reaches it as
  `memory move` (`cmd/mandalore/main.go` usage only); the context hook's
  conflict line names a move conflict (`internal/server/context.go`).
- **Sync** (`internal/sync/checkpoint.go`, `remote.go`): the new path
  pattern, and counting a move-conflicted record once.
- **Export** (`internal/exportreport/snapshot.go`, `project.go`): read moves,
  select by placement, `record_move` items.
- **Canon** (`internal/foundlings/canon.go`): promotion's recall uses the
  record's current scope (`canon.go:713` reads the root revision's scope
  today).
- **Docs**: `docs/signet-format.md` (layout, move rules, the old-reader
  table), `docs/server.md` (tools), `docs/interface.md` (operation, failures,
  history and export shapes), plugin README, both skill files.
- **Repository lessons** (`docs/repository.md`): the two-clone lesson and
  the #229 ruling with its limit (both below), added with the code because
  the plan commit holds only this file.
- **Release preparation**: `VERSION` and `plugin.json` to 2.3.0,
  `docs/releases/2.3.0.md` with what changed, the upgrade-every-syncing-clone
  note, the old-reader table (every row, including backup, status, upgrade
  and `migrate init`), the note that `memory_history` output changes on
  every record, rollout and rollback. No new HTTP or OAuth
  path, state field, configuration or secret.
- **Tests and fixtures**: in `internal/memory`, `internal/api`,
  `internal/server`, `internal/sync`, `internal/exportreport`,
  `internal/foundlings`; a synthetic format 1 and format 2 fixture written by
  main's code under `internal/memory/testdata/compat/v2.2.0/`, and the golden
  read outputs under `internal/server/testdata/nomove-golden/`, both written
  by `TestNoMoveReadsMatchGoldens` on main before any code change.
- **Old-reader check** (`bin/check-old-reader`, `cmd/mandalore`): also builds
  2.2.1 from 95bbf17 and runs a new `TestOldReaderOnMovedRecord` against both
  binaries, asserting every row of the table above.

## Order of work

1. This plan alone as the first commit; draft PR; plan review; wait for the
   lead's go.
2. Fixtures and golden outputs, before any code change: generate the 2.2.0
   signets, pin every file's hash, and capture recall, context, prime,
   scopes, canon recall and withheld-record discovery output from main on
   them. Commit them in their own commit; later steps never regenerate them.
3. Move document, validation and placement in the store, with graph tests.
4. `Move` in the service with guards and idempotency; `memory_move` over MCP
   and the catalog, with the read-only refusal.
5. The correction scope check; reads by placement; history's new fields.
6. Move conflicts across clones in sync tests; export, canon and upgrade.
7. Old-reader check against 2.2.1 and 2.1.2.
8. Guidance, docs, lesson, version bump, release notes; `bin/ci`,
   `bin/sdlc check`, govulncheck; simplify pass; mark the PR ready.

## Risks

- **Old clones that sync.** A 2.2.1 clone refuses to sync once a move arrives;
  nothing is lost, and installing 2.3.0 recovers it. Contained by the release
  note and the table.
- **Hidden recall drift.** Scope selection moves from the revision to the
  placement map on every read path. Contained by the byte-identity test on
  signets without moves, unchanged ranking code, and the requester's private
  eval.
- **Two meanings of `scope`.** A stored revision's `scope` is where the
  record was saved; the current scope comes from moves. An agent reading raw
  history could correct with the old scope. Contained by history's top-level
  `scope`, the refusal message that names the current scope, and the tool
  descriptions.
- **A correction on 2.2.1 after rollback** uses the old scope and lands in the
  record's new scope once 2.3.0 is back. That is the intended reading (the
  move moves the record, not one revision) and the release notes say so.
- **Concurrent move and correction on two clones** both survive: the
  correction's content, in the moved-to scope. Not a conflict, by design; a
  test pins it.
- **Plugin rollout (#229).** Installed plugins follow main, so the skill line
  reaches users at merge, while the live server still runs 2.2.1 (95bbf17)
  until it upgrades after publication. The requester ruled, through the lead,
  that for this PR the reviewer's posted evidence on 2.2.1, the version the
  live server runs, satisfies the #229
  merge rule: `tools/list` without `memory_move` and the skill diff. The
  wording is inert there, because it applies only when the server lists
  `memory_move`, so there is no failed call and no changed behaviour. The
  limit: this covers only wording-only plugin changes gated on a tool the
  server lists. A plugin change that alters hooks or `.mcp.json`, or calls a
  tool unconditionally, still waits for the requester's confirmation after
  the server upgrade. The ruling and its limit go into `docs/repository.md`
  next to the #229 lines.
- **History size.** Moves share the first page's 32 KiB budget; whole moves
  are omitted with disclosure, never clipped.
- **History output changes for every caller.** The new fields appear on every
  record. They are additive; the docs and release notes say so.

## Proof

Tests, run with `GOFLAGS=-count=1`, synthetic data only:

- Validation: moves with missing, foreign or cyclic parents, foreign observed
  heads, unknown devices, a wrong signet ID or a noncanonical path refuse;
  several move roots are valid; format 1 accepts moves.
- Guards: each of the six above, including a stale content head, a stale move
  head, a withdrawn record (format 2), a content conflict without a move
  conflict (refused), a settling move during both conflicts (allowed only
  when it names every move and content head) and the no-op; field limits at
  and one byte over.
- Idempotency: same `request_id` and input returns the first receipt after a
  later move, for a real move and for a no-op (the no-op retry returns
  `moved: false` and the recorded scope, not `memory.stale_heads`); different
  input is `request.conflict`; an interrupted move is pending, and a
  completed move and a no-op intent aren't; `memory_remember`,
  `memory_journal_append` and `memory_name_scope` reusing a move's
  `request_id` are `request.conflict`, and so is a move reusing theirs.
- Corrections: after a move, the new scope succeeds and the old one is
  refused with the current scope named; the stored revision keeps the root
  scope; during a move conflict a correction is refused.
- Reads: recall, prime and scopes follow a move; on signets without moves,
  recall, context, prime, scopes, canon recall and withheld-record discovery
  match the golden outputs byte for byte; a move conflict is listed in both
  destinations and current in neither, with `move_conflicts` counted on
  format 1 and format 2 and format 2's visibility counts still adding up;
  a settling move clears it.
- History: `scope`, `move_heads` and `move_count` on every page; `moves`
  only on the first page, newest first; at the 32 KiB boundary, a record
  whose revisions fill the page holds the same revisions with 12 moves as
  2.2.0's page function gives, and keeps the newest whole moves with
  `moves_truncated`, under 64 KiB in all.
- Two clones (sync tests): concurrent moves to different scopes sync; both
  move heads show in history and as conflicts; a settling move syncs back.
  Concurrent moves to the same scope sync without a conflict, both kept in
  history, and a correction is accepted. A concurrent move and correction
  keep both. Each clone corrects and moves the same record to different
  scopes before syncing: after sync the record has a content conflict and a
  move conflict; a correction is refused, a move that misses a content head
  is refused, the settling move that names every head succeeds, then a
  correction superseding both content heads in the new scope succeeds; after
  syncing back, both clones show one scope and one head.
- Export, canon and upgrade: report selection and `record_move` items; a canon
  source with a move is served by placement; a format 1 signet with a move
  upgrades with it protected.
- Read-only clients aren't served `memory_move`; the catalog returns
  `operation.read_only`.
- `bin/check-old-reader` passes against 2.2.1 and 2.1.2, including the
  backup, sync inspection, upgrade preview and apply, `migrate init` and
  `migrate verify` rows.
- `bin/ci`, `bin/sdlc check` and govulncheck pass.

Acceptance the verifier runs on the built binary, against a running server on
a synthetic signet: the issue's six scenarios. Client A saves in project X,
corrects once, moves to Y; recall and prime on Y return the same `record_id`,
on X they don't; history shows both revisions and the move with reason and
authorship. A second correction with Y succeeds and with X is refused. Stale
head, withdrawn record (on a format 2 signet; the test server's fixture is
format 1, so the automated withdrawn case runs at the service level) and the
repeated `request_id` behave as specified. Two
clones move to different scopes and sync; the conflict is listed and both
moves are kept. A read-only client is refused.

## Rollback

Reinstall 2.2.1. Nothing to undo in the data: moves stay in `memory/moves/` as
append-only evidence and 2.2.1 ignores them, so moved records show in their
old scopes, and that clone can't sync, checkpoint, back up after writes,
show the menu's status, upgrade or export until 2.3.0 is back (the table
above). On the server this means writes still save locally but their backups
never complete (`server status` shows them pending), so reinstall 2.3.0
rather than run 2.2.1 for long. Move intents stay in `.mandalore/`; 2.2.1's cutover
preflight refuses on them, which affects no serving. There is no downgrade
that removes moves, and reinstalling 2.3.0 restores every placement.

## Additions to `docs/repository.md`

Added with the code:

- A write that can happen on two clones before they sync must keep both
  documents valid after the merge. Allow several roots or heads and show
  them as a conflict; never require a single root across clones, or the
  second clone can never sync (#246 code review round 1). When two
  conflicts can coexist on one record, check that each one can be settled
  while the other stands (#250 plan review).
- Next to the #229 lines: a wording-only plugin change gated on a tool the
  server lists may merge on the reviewer's posted evidence from the live
  server's version (the tool missing from `tools/list`, and the diff),
  because it is inert there. A plugin change that alters hooks or
  `.mcp.json`, or calls a tool unconditionally, still waits for the
  requester's confirmation after the server upgrade (#250).
- A new portable path is refused by every caller of the sync path check, not
  only sync: sweep `allowedPath` and `validateWorktree` callers at the
  released commits for the old-reader table (#250 plan review).
