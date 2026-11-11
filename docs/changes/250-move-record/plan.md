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
asserts each row against 2.2.0 (0d3c22d) and 2.1.2 (79b378e) built from
source:

| 2.2.0 or 2.1.2 on a signet with a move | Behaviour |
| --- | --- |
| Recall, prime (2.2.0), scopes, context, history, inspect | Work; the record shows in its old scope; moves are invisible |
| Correction with the old scope | Succeeds; once 2.3.0 is back the record is still in its new scope, with that content |
| Correction with the new scope | Refused by graph validation, nothing written |
| Reconcile, menu sync, checkpoint with the new path | Refuse; files, index and unpushed commits kept (as the 2.2.0 table) |
| Export preview | Refuses |
| Canon refresh of a source with a move | Serves the record in its old scope |
| `memory_move` | Not served |
| Cutover preflight with a move intent in `.mandalore/` | Refuses: "pending write intents are unreadable" |
| A `memory_remember` retry that reuses a move's `request_id` | `save.incomplete`, nothing written |

The alternatives are worse. Storing the new scope in post-move revisions
breaks every read on older readers and after a rollback. A format bump would
make old clones refuse explicitly, but costs every user the explicit upgrade
(clean checkpoint, stopped writers, preview and apply) for a feature that
otherwise rolls out like 2.2.0. If the plan review prefers the bump, I stop
and the lead reports.

### Release

2.3.0, the next minor after 2.2.0: a new write tool and new files in the
signet, as 2.2.0 was for topics. The skill text changes, so
`plugins/claude/.claude-plugin/plugin.json` moves to 2.3.0 with `VERSION`
(`docs/repository.md`: a plugin change bumps `plugin.json`, and either bump
means an artifact release). The PR says `Refs #250`.

### Recall paths change

**Yes: `memory_recall`, `memory_context`, `memory_prime` and `memory_scopes`
change**, so the requester runs the private recall eval before publication.
What changes is scope selection only: each reads a record's current scope from
its moves instead of from its revisions. Ranking, term statistics, packing and
the context hook's thresholds are not touched. On a signet with no moves every
output is byte-identical to 2.2.0's; a test pins that on a fixture for recall,
context, prime and scopes.

## Behaviour

**The tool.** `memory_move` (read-write only), a server tool with
`request_id` like `memory_remember`, and a catalog operation for
`mandalore call memory_move` without it, as `memory_name_scope` is:

```json
{
  "request_id": "move-example-1",
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
unsuperseded move is a current move head. One head: its `scope`. Two or more:
a move conflict (below). A move from one clone may have no parents, and so may
a concurrent first move from another; both are valid, so several move roots
are allowed (the two-clone lesson, below).

**Guards**, in order, all under the writer lock, nothing written on refusal:

1. Input shape and limits: `input.invalid`.
2. Record not found: `input.invalid`.
3. More than one content head: `input.invalid`, "record has conflicting
   revisions; settle the conflict with a correction that supersedes every head
   first".
4. Format 2 and the record isn't visible (withdrawn or a visibility conflict):
   `input.invalid`, "record is withdrawn; restore it before moving it".
5. Expected content or move heads differ from the current ones:
   `memory.stale_heads`, on both formats (this is a new tool, so it is strict
   from the start).
6. One move head (or none) and the destination is the current scope: success
   with `moved: false`, no file and no intent. A retry re-evaluates, and the
   heads guard keeps it safe if the record moved meanwhile.

A move conflict is settled by a move whose `expected_move_heads` names every
current move head; it is written even when its destination is one of theirs.

**Receipt:** `signet_id`, `record_id`, `scope` (current after the call),
`moved`, `id` (the move ID, absent on a no-op), and the server's durability
and backup fields as for `memory_remember`.

**Idempotency.** The intent document (`writeIntent`) gains an optional
`move`; the digest covers the whole input. Same `request_id` and input
returns the first receipt; different input returns `request.conflict`;
`store.busy` and `save.incomplete` work as for `memory_remember`.
`PendingIntents` learns the move target, so a completed move isn't reported
as pending and an interrupted one is. Existing intent shapes and digests don't
change.

**Corrections after a move.** `memory_remember` with `record_id` and
`supersedes` compares the input scope with the record's current scope under
the writer lock. A match is accepted and the revision is stored with the
record's root scope, keeping the invariant above; a mismatch is refused as
`input.invalid` naming the current scope ("record was moved to project
example-y; correct it in that scope"). During a move conflict a correction is
refused: "record has conflicting moves; settle them with memory_move first".
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
`memory_scopes` counts it in each destination, as `conflicted` in format 2's
visibility counts; `memory_context` surfaces it as a conflict when relevant,
as for content conflicts. Neither move is lost: history lists both.

**History.** `memory_history` keeps its paged `items` (the stored revisions,
unchanged) and gains `scope` (current scope; absent during a move conflict),
`move_heads`, and `moves`: the record's moves newest first, each with `from`
(derived: the root's scope, or the scopes of its parent moves) and the stored
fields. Moves are packed into the same 32 KiB page budget after the items,
whole moves only, with `moves_truncated` and `move_count` when some don't fit.
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
moves with the rest of the graph. Canon fetch already copies every file under
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
  descriptions.
- **Sync** (`internal/sync/checkpoint.go`): the new path pattern.
- **Export** (`internal/exportreport/snapshot.go`, `project.go`): read moves,
  select by placement, `record_move` items.
- **Canon** (`internal/foundlings/canon.go`): promotion's recall uses the
  record's current scope (`canon.go:713` reads the root revision's scope
  today).
- **Docs**: `docs/signet-format.md` (layout, move rules, the old-reader
  table), `docs/server.md` (tools), `docs/interface.md` (operation, failures,
  history and export shapes), plugin README, both skill files.
- **Repository lessons** (`docs/repository.md`): the two-clone lesson below,
  which the issue cites but which isn't written there yet.
- **Release preparation**: `VERSION` and `plugin.json` to 2.3.0,
  `docs/releases/2.3.0.md` with what changed, the upgrade-every-syncing-clone
  note, the old-reader table, rollout and rollback. No new HTTP or OAuth
  path, state field, configuration or secret.
- **Tests and fixtures**: in `internal/memory`, `internal/api`,
  `internal/server`, `internal/sync`, `internal/exportreport`,
  `internal/foundlings`; a synthetic format 1 and format 2 fixture written by
  2.2.0 under `internal/memory/testdata/compat/` with its committed generator.
- **Old-reader check** (`bin/check-old-reader`, `cmd/mandalore`): also builds
  2.2.0 from 0d3c22d and runs a new `TestOldReaderOnMovedRecord` against both
  binaries, asserting the table above.

## Order of work

1. This plan alone as the first commit; draft PR; plan review; wait for the
   lead's go.
2. Fixtures and the byte-identity test: generate the 2.2.0 signets, pin every
   file's hash, and pin recall, context, prime and scopes output on them.
3. Move document, validation and placement in the store, with graph tests.
4. `Move` in the service with guards and idempotency; `memory_move` over MCP
   and the catalog, with the read-only refusal.
5. The correction scope check; reads by placement; history's new fields.
6. Move conflicts across clones in sync tests; export, canon and upgrade.
7. Old-reader check against 2.2.0 and 2.1.2.
8. Guidance, docs, lesson, version bump, release notes; `bin/ci`,
   `bin/sdlc check`, govulncheck; simplify pass; mark the PR ready.

## Risks

- **Old clones that sync.** A 2.2.0 clone refuses to sync once a move arrives;
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
- **A correction on 2.2.0 after rollback** uses the old scope and lands in the
  record's new scope once 2.3.0 is back. That is the intended reading (the
  move moves the record, not one revision) and the release notes say so.
- **Concurrent move and correction on two clones** both survive: the
  correction's content, in the moved-to scope. Not a conflict, by design; a
  test pins it.
- **Plugin rollout (#229).** Installed plugins follow main, so the skill line
  reaches users at merge. It is conditional on the server listing
  `memory_move`, so it is harmless on 2.2.0; the reviewer verifies that on
  2.2.0 with posted evidence (`tools/list` without `memory_move`, the skill
  diff), as for 2.2.0.
- **History size.** Moves share the 32 KiB page; whole moves are omitted with
  disclosure, never clipped.

## Proof

Tests, run with `GOFLAGS=-count=1`, synthetic data only:

- Validation: moves with missing, foreign or cyclic parents, foreign observed
  heads, unknown devices, a wrong signet ID or a noncanonical path refuse;
  several move roots are valid; format 1 accepts moves.
- Guards: each of the six above, including a stale content head, a stale move
  head, a withdrawn record (format 2), a content conflict and the no-op; field
  limits at and one byte over.
- Idempotency: same `request_id` and input returns the first receipt after a
  later move; different input is `request.conflict`; an interrupted move is
  pending and a completed one isn't.
- Corrections: after a move, the new scope succeeds and the old one is
  refused with the current scope named; the stored revision keeps the root
  scope; during a move conflict a correction is refused.
- Reads: recall, prime and scopes follow a move; byte-identical output on
  signets without moves; a move conflict is listed in both destinations and
  current in neither, and a settling move clears it.
- Two clones (sync tests): concurrent moves to different scopes sync; both
  move heads show in history and as conflicts; a settling move syncs back;
  a concurrent move and correction keep both.
- Export, canon and upgrade: report selection and `record_move` items; a canon
  source with a move is served by placement; a format 1 signet with a move
  upgrades with it protected.
- Read-only clients aren't served `memory_move`; the catalog returns
  `operation.read_only`.
- `bin/check-old-reader` passes against 2.2.0 and 2.1.2.
- `bin/ci`, `bin/sdlc check` and govulncheck pass.

Acceptance the verifier runs on the built binary, against a running server on
a synthetic signet: the issue's six scenarios. Client A saves in project X,
corrects once, moves to Y; recall and prime on Y return the same `record_id`,
on X they don't; history shows both revisions and the move with reason and
authorship. A second correction with Y succeeds and with X is refused. Stale
head, withdrawn record and the repeated `request_id` behave as specified. Two
clones move to different scopes and sync; the conflict is listed and both
moves are kept. A read-only client is refused.

## Rollback

Reinstall 2.2.0. Nothing to undo in the data: moves stay in `memory/moves/` as
append-only evidence and 2.2.0 ignores them, so moved records show in their
old scopes, and that clone can't sync, checkpoint or export until 2.3.0 is
back (the table above). Move intents stay in `.mandalore/`; 2.2.0's cutover
preflight refuses on them, which affects no serving. There is no downgrade
that removes moves, and reinstalling 2.3.0 restores every placement.

## Lesson for `docs/repository.md`

Added with the code: a write that can happen on two clones before they sync
must keep both documents valid after the merge. Allow several roots or heads
and show them as a conflict; never require a single root across clones, or
the second clone can never sync (#246 code review round 1).
