# Plan: bounties, named topics an agent can be primed on in one call

## Issue

Refs #246 (https://github.com/acoz-labs/mandalore/issues/246). Outcome: a user
names a topic in conversation, and any agent on any client finds it by name,
loads it with one call and keeps working in it. Ships as minor release 2.2.0,
server and Claude plugin together.

### Decisions the issue asks for

**(a) A bounty is a named scope, not a new scope kind.** "Bounty" is the
product word; in the data and the tools it is a `project`, `account` or `task`
scope that carries a name and description. Reasons:

- A topic already is a scope in every way that matters: records are saved
  into it, recalled from it and counted in it. A new kind would duplicate that
  plumbing and make every existing scope second-class ("is my project a
  bounty?") when the user only wants to give it a name.
- Scope kind is a closed enum in the revision schema and `validateScope`. A
  new kind would appear in new revision files, and 2.1.2 reads the whole
  record graph strictly on every recall, so one such record would break recall
  in every scope on every older client. That would force the format bump that
  (b) avoids.
- Naming an existing scope needs no migration: records already in it become
  the topic's knowledge the moment it is named.
- The signet-wide scope can't be named. It is the whole bank, not a topic, and
  its unscoped journal is already the default journal.

**(b) No signet format bump. The feature ships as new files in new
directories, and existing data stays valid byte for byte.** New writes go only
under one new directory, `memory/scopes/`:

```text
memory/scopes/<kind>/<scope-id>/labels/<label-id>.json
memory/scopes/<kind>/<scope-id>/events/<UTC-year>/<UTC-month>/<event-id>.json
```

`<kind>` is `project`, `account` or `task`, and `<scope-id>` is the scope's
existing ID; both already match the identifier pattern, so paths stay
canonical and are checked against the document's own scope. No existing file,
schema or directory changes: unscoped journal entries keep schema 1 under
`memory/events/`, records and visibility events are untouched, and the
manifest stays at format 1 or 2. New signets don't create `memory/scopes/`
until the first name or scoped journal entry. Both formats get the feature
the same way.

Why new directories rather than a new field in existing files: every reader is
strict about unknown fields. A `scope` field on journal entries under
`memory/events/` would make a 2.1.2 server fail every `memory_journal` call and
`memory_inspect` on that signet, including after a rollback. Under a new
directory, 2.1.2 never opens those files, so its recall, remember, journal and
context keep working.

What older binaries do with a signet that has the new files (this is the
compatibility cost, and the release notes say it):

| 2.1.2 or older | Behaviour |
| --- | --- |
| Recall, remember, history, journal, context, inspect | Work; names and scoped journal entries are invisible |
| `server reconcile` with no local commits, remote has the new paths | Refuses: "unsafe or partially staged work requires explicit reconciliation; files and index preserved" |
| `server reconcile` with an unpushed local commit | Refuses as diverged ("local and remote both have commits"), before the path check; the commit is kept |
| Menu sync (merge) with an unpushed local commit | Refuses at validate: "Remote candidate could not be safely reconciled… Local work was not discarded"; HEAD, index and the unpushed commit are unchanged |
| Checkpoint with the new files in its own worktree (`validateWorktree`) | Refuses (`sync.failed`); nothing changes |
| Export report (`internal/exportreport`) | Refuses (`export.failed`, sanitized from "unknown portable data") |
| Cutover preflight (`server migrate preflight`) | Refuses: "pending write intents are unreadable" |
| Canon refresh of a source that has the new paths (`internal/foundlings/canon.go`) | Serves it: the snapshot copies every `memory/` file and validation never walks `memory/scopes/`, so records are served and names aren't |
| `memory_journal_append` with `entry.scope` | Strict decoding refuses it as `input.invalid` before any intent or write |
| `memory_name_scope`, `memory_prime` | Not served |

`bin/check-old-reader` runs the 2.1.2 binary built from 79b378e against a
signet this build wrote, on format 1 and format 2, and asserts each row.

This follows the canon-registration precedent in `docs/signet-format.md`: a
new document type that needs compatible readers on every clone that syncs,
without a manifest change. A format bump would make the refusal explicit, but
every clone would need the explicit upgrade (clean checkpoint, stopped
writers, preview and apply), the transition proof code assumes exactly 1→2,
and there is still no downgrade. The gain is a clearer error on old clones;
the cost is a mandatory operation for every user. So: upgrade every clone that
syncs to 2.2.0 before the first naming or scoped journal write. If the plan
review prefers the bump, I stop and the lead proposes the split the triage
describes.

**(c) Tool and parameter names.**

| Tool | Change |
| --- | --- |
| `memory_prime` (new, read) | `scope` (required, `{kind, id}`, project/account/task) and `budget_bytes` (optional, default 8192, range 1024–32768, as recall) |
| `memory_name_scope` (new, read-write only) | `request_id` (required), `scope` (required), `name` (1–256 bytes), `description` (1–1024 bytes), `reason` (1–1024 bytes), `supersedes` (label IDs, required with a revision, at most 32) |
| `memory_scopes` | New optional `query` (at most 2048 bytes); items gain `labels` |
| `memory_journal_append` | New optional `entry.scope` (`{kind, id}`, project/account/task) |
| `memory_journal` | New optional `scope` filter (project/account/task; `signet` is refused, since omitting `scope` already means the whole journal); `query` still applies alongside it; entries gain `scope` when set |

The tools say "scope" because that is what they take; their descriptions say
a named scope is a topic, or bounty.

## Behaviour

**Names and descriptions are labels.** A label document (schema 1) holds
`schema_version`, `id` (`label-…`), `scope`, `name`, `description`,
`recorded_at`, `authorship`, `supersedes` and `change_reason`, the same shape
and rules as foundling registrations: one root per scope, predecessors from the
same scope, no cycles, no missing predecessors, unknown devices refused. The
first label of a scope has empty `supersedes`; a revision supersedes the
current label heads. Two concurrent revisions leave two heads, both shown.
Name and description are checked like record summaries (not blank, no NUL),
and the name also refuses newlines. Labels can't be withdrawn;
withdrawal is out of scope, and a wrong description is corrected by a new
revision (the old text stays in history, as for records). The tool description
says never to put secrets in a name or description.

`memory_name_scope` on format 2 compares `supersedes` with the current label
heads under the writer lock and refuses stale input as `memory.stale_heads`,
like record corrections there. On format 1 it accepts a non-current
predecessor and keeps the fork visible, as records do. A name on a scope with
no records is allowed: naming comes first in the acceptance scenario. Input
errors return `input.invalid` with nothing written; `request_id` idempotency,
`request.conflict`, `store.busy` and `save.incomplete` work as for
`memory_remember`, through the same intent files under `.mandalore/`.

**Write intents.** The intent document (`writeIntent`) gains an optional
`label`, and a scoped journal intent carries its entry with `scope`. Unscoped
journal and remember intents keep today's exact shape and digest input, so a
retry that started under 2.1.2 still matches. `PendingIntents` learns both new
targets (the label path and the scoped events path), so a completed naming or
scoped append is not reported as pending and an interrupted one is. Intent
files are kept after completion, as today.

**`memory_scopes`.** Each item keeps `scope`, `record_count` and
`visibility_counts`, and gains `labels`: the current label heads as
`{id, name, description}`, absent when the scope is unnamed, more than one
entry when they conflict. Scopes with no records but a name or scoped journal
entries are listed with `record_count` 0, so every scope that export can
select is also findable. With `query`, items whose ID, name or description match
(the journal's lexical matcher: exact tokens, then the limited inflection
fallback) are ranked by score, then by kind and ID; without it, order is as
today. Paging keeps its limit (1–50) and its 32 KiB page budget, so output
stays bounded even with 1024-byte descriptions. The store's scope inventory
that `memory_context` reads (`Store.Scopes`) does not change; labels are
joined in the service page only.

**`memory_journal_append` and `memory_journal`.** A scoped entry is journal
schema 2 (schema 1 plus a required `scope`) under the new directory. The
scope is validated as for records but needs no records or name. The
unfiltered query returns unscoped and scoped entries together, newest first,
as today. With `scope`, only that scope's entries come back. Read-only
clients are already refused every journal write. A scoped journal intent's
digest covers kind, summary and scope; an unscoped one covers kind and summary
only, as today.

**Export reports.** The report snapshot reads both new directories. In a
report selection:

- `scopes` also accepts a scope that has only labels or scoped journal
  entries. Selecting a scope selects its records (as today), its scoped
  journal entries and its labels; the existing 128-item caps apply to the
  selected journal entries too, and exceeding them refuses as today.
- `journal_ids` and `omit_journal_ids` reach entries in both directories. A
  scoped entry's `classification` gains `scope`.
- Labels are a new item `type: "scope_label"` that uses the existing field
  categories, so `omit_fields` covers them with no new category: `identity`
  (label ID and scope), `content` (name and description), `timestamps`,
  `authorship` and `change_history` (`supersedes`, `change_reason`). Current
  heads are reported; `include_history` adds superseded label revisions.
  The report keeps `version: 1`; the additions are new optional content in
  the same shape.

**Canon sources.** A signet with files under `memory/scopes/` can be another
server's canon source. A 2.2.0 server fetches, verifies, refreshes and serves
it: canon fetch already copies every file under `memory/`, and 2.2.0's
validation of the snapshot checks the labels and scoped journal entries like
any other evidence. Canon tools don't expose names; `foundling_canon_scopes`
is unchanged.

**`memory_prime`.** One read that returns, in this order:

1. `labels`: the scope's current name and description heads.
2. `state`: current `project-state` heads in the scope, newest first.
3. `current`: other current heads in the scope, newest first.
4. `journal`: the scope's journal entries, newest first.

Selection: prime takes recall's exact-scope effective heads with no limit, as
`Service.Recall` does (`int(^uint(0) >> 1)`), so signet-wide records are not
mixed in and withdrawn records are excluded before selection. It does not use
recall's ranked order: an empty query scores every candidate the same and
sorts by random revision ID. Instead prime splits off `project-state` and
sorts each section by `recorded_at`, newest first, with the ID ascending as
the tiebreak. `recorded_at` is when the knowledge was written; `effective_from`
can be set in the past, so it says less about what changed last. Labels and
journal entries sort the same way by their own `recorded_at` and ID. Records
with several heads are listed in `conflicts` (record ID and head IDs) instead
of being picked. Ranking and `memory_context` code are not touched.

Item shapes: `state` items are whole recall hits, body included. `current`
items are compact so a large topic still fits: revision `id`, `record_id`,
`kind`, `summary`, `recorded_at`, `basis` and `confidence`, with no body. The
tool description says to fetch bodies with `memory_recall` in the scope or
with `memory_history`.

Packing follows recall: the budget covers the whole JSON response; whole items
are omitted rather than clipped; the response carries `matching_counts` per
section, `truncated` and recall's evidence notice. Packing priority, which is
separate from the response's section order: label names (each label is
`{id, name}` first), then each label's `description`, then `state`,
`conflicts`, the newest three journal entries, `current`, then the remaining
journal entries. Sections keep the order above and newest first within each
section. So at the minimum budget the topic is still identified by name when
its description doesn't fit, the latest journal survives a crowded `current`,
and `truncated` and the counts say what was left out. A label whose name alone doesn't fit
(JSON escaping can grow a 256-byte name past 1024 bytes) is omitted and
counted. Read clients get it.

**Guidance.** The server instructions stay under their existing limit:
`TestInstructionsAreShort` requires each to be under 600 bytes, and it doesn't
change. Today the base is 279 bytes, the read-write addition 149 and the
read-only addition 51, so 428 and 330. The final wording, appended after the
existing text:

- Read-write gains 167 bytes, for 595: " For lasting work, find its topic with
  a memory_scopes query, memory_prime it, save and journal in its scope, and
  correct its project-state record rather than add one."
- Read-only gains 81 bytes, for 411: " For lasting work, find its topic with a
  memory_scopes query and memory_prime it."

The server sends these only from 2.2.0, so they never name tools a server
lacks.

**Plugin text that is harmless on 2.1.2.** Installed plugins follow main
(#229), so the skill and README reach users at merge, before 2.2.0 is
released and installed on the live server. The new `this-is-the-way`
paragraph, "Topics (bounties)", therefore starts with the condition: use it
only when the server lists `memory_prime`; otherwise use scopes and recall as
before. It then gives the steps: find the topic with a `memory_scopes` query,
prime on it, name it with `memory_name_scope` if the user named it and it has
no name, save into its scope, correct its `project-state` record rather than
adding one, and journal with `entry.scope`. One sentence covers a new topic:
when no topic matches and the user has named one, use a `project` scope with a
short lowercase ID derived from the name, unless it is clearly an account or a
bounded task, then name it with `memory_name_scope`. This sentence is in the
skill only; the server instructions have no room for it. On a 2.1.2 server the condition is
false, and if an agent sends `entry.scope` anyway, 2.1.2 refuses it as
`input.invalid` with nothing written (proved below). The plugin README says
the same. `references/memory.md` gains the label rules. The plugin bump ships
in this PR; there is no follow-up plugin PR.

## Files that change

- **Labels in the memory store** (new `internal/memory/scope_labels.go`): the
  label type, strict reading of `memory/scopes/*/*/labels`, the graph checks,
  current heads per scope, and the append under the writer lock.
- **Scoped journal** (`internal/memory/journal.go`): journal schema 2 with
  `scope`, written under the new events directory and read alongside
  `memory/events/`; the scope filter; path and duplicate checks across both.
- **Full validation** (`internal/memory/store.go`, `Validate`): validates
  labels and scoped entries; refuses unexpected files under `memory/scopes/`.
- **Service** (`internal/memory/service.go`, `pages.go`,
  `idempotency.go`, `input_error.go`): `NameScope` and its idempotent form,
  the `label` intent and `PendingIntents` targets for labels and scoped
  entries,
  the scoped `AppendJournal`, `JournalPage` with scope, `ScopePage` with query
  and labels, and `Prime`.
- **Catalog operations** (`internal/api/api.go`, `remember.go`):
  `memory_prime` and `memory_name_scope`, new fields on the scopes and journal
  inputs. The CLI reaches the new operations through `mandalore call`, as it
  does every catalog operation; no new CLI subcommands.
- **MCP tools** (`internal/server/tools.go`): `memory_prime` joins the read
  tools; `memory_name_scope` is a server tool with `request_id`, schema byte
  limits through `limit`, added to `isWrite`; descriptions; instructions.
- **Sync, export and upgrade paths**: `allowedPath` in
  `internal/sync/checkpoint.go` admits the two new path shapes. The export
  report snapshot (`internal/exportreport/snapshot.go`, `memory.Snapshot`)
  reads and validates them, and the report projection
  (`internal/exportreport/project.go`) selects and renders them as described
  under Behaviour. The format upgrade proof already treats everything under
  `memory/` as protected append-only evidence, so a format 1 signet with
  labels upgrades with them covered; a test proves it.
- **Docs**: `docs/signet-format.md` (layout, label rules, journal schema 2,
  old-reader table), `docs/server.md` (Tools: the two new tools and new
  parameters, scope lists), `docs/interface.md` (operations and failure
  table, export report items), the plugin README and both skill files.
- **Release preparation**: `VERSION` and
  `plugins/claude/.claude-plugin/plugin.json` to 2.2.0, and
  `docs/releases/2.2.0.md` with what changed, the upgrade-every-syncing-clone
  note, the #229 step (the plugin text is written to be harmless on a 2.1.2
  server; the reviewer verifies that on 2.1.2 with posted evidence before
  merge, and the live server upgrade follows the release), rollout and
  rollback. No new HTTP or OAuth path, so no Cloudflare allowlist change.
  No new secret.
- **Tests and fixtures** in `internal/memory`, `internal/api`,
  `internal/server`, `internal/sync`, `internal/exportreport` and
  `internal/foundlings` (a canon source with the new paths), with small
  synthetic format 1 and format 2 signets written by 2.1.2 under
  `internal/memory/testdata/compat/`, and their committed generator.
- **Old-reader check** (new `bin/check-old-reader` and a test in
  `cmd/mandalore` that skips unless `MANDALORE_OLD_BINARY` names a binary):
  the script builds 79b378e from a temporary Git worktree in a temporary
  directory, runs the test against that binary, and removes both. It isn't
  part of `bin/ci`, which can't assume full history; I run it for the PR and
  the verifier can rerun it.

## Order of work

1. This plan, alone, as the first commit; draft PR; plan review; wait for the
   requester's go.
2. Fixtures: commit the generator, generate the format 1 and format 2
   signets with the 2.1.2 binary (`main` at 79b378e), and commit them with a
   test that validates them and hashes every file.
3. Labels in the store and service, with graph and input tests; then
   `memory_name_scope` over MCP with idempotency and the read-only refusal.
4. Scoped journal in store, service, catalog and MCP, with tests that
   unscoped entries are unchanged and still returned.
5. `memory_scopes` query and labels, with the bounded-page tests and a test
   that `memory_context` output is identical with and without labels.
6. `memory_prime`, with order, budget, withdrawal and conflict tests.
7. Sync, export and upgrade path support, with their tests; then the
   old-reader check against a 2.2.0-written signet.
8. Instructions, skill, docs, version bump and release notes; `bin/ci`,
   `bin/sdlc check`, govulncheck; simplify pass; mark the PR ready.

## Risks

- **Old clones that sync.** A 2.1.2 clone refuses to sync once the new files
  arrive. It loses nothing and recovers by installing 2.2.0. Contained by the
  release note and the table above; the error those clones show ("unsafe or
  partially staged work") is old code we can't change, so the note names it.
- **Rollback after use.** 2.1.2 on a signet with labels serves memory
  normally but can't sync or export until 2.2.0 is back. The new intent shapes
  also stay in `.mandalore/`, and 2.1.2 decodes intents strictly:
  - 2.1.2's `PendingIntents` fails with a decode error on the first label or
    scoped-journal intent. Its only caller is the 1.x-to-2.0 cutover
    preflight, which then reports that error instead of a blocker list. It
    doesn't affect serving.
  - 2.1.2 has no `memory_name_scope`, so a naming can't be retried there.
  - A `memory_journal_append` retry that still carries `entry.scope` is
    refused as `input.invalid` by strict input decoding before any intent is
    read, with nothing written.
  - A retry that drops `entry.scope` but reuses the `request_id` reads the
    new intent, fails to decode it and returns `save.incomplete`, with
    nothing written.

  The Proof section shows each of these on the 2.1.2 binary, and the release
  notes and Rollback carry them.
- **Hook drift (#232).** `memory_context` reads `Store.Scopes`, which doesn't
  change, and prime reuses recall's exact-scope selection without touching
  ranking. A
  test pins `memory_context` output on a signet with and without labels and
  scoped entries.
- **Text treated as instructions.** Names and descriptions are user-authored
  text served in scope listings and primes. They are returned as data fields
  under the existing evidence notice, never in the server instructions, and
  the tool descriptions say they are evidence.
- **Unbounded reads.** Prime and scopes read every label and journal entry, as
  journal and validation already do. Output is bounded; reads are proportional
  to the signet, as today.
- **Plugin rollout (#229).** Installs follow main, so merging the plugin bump
  rolls the new skill text out before the live server runs 2.2.0. The text is
  conditional on the server listing `memory_prime`, and 2.1.2 refuses a stray
  `entry.scope` without writing, so it is harmless there. The merge condition
  is the reviewer's verification of that on 2.1.2, with the evidence posted on
  the PR (the old-reader check's output, `tools/list` without `memory_prime`,
  and the skill diff), not an owner confirmation: the text degrades safely
  whatever the live server runs.
- **Tight instructions budget.** Read-write instructions reach 595 of 599
  allowed bytes, so later additions must shorten something first. The
  existing test catches it.

## Proof

Tests, run with `GOFLAGS=-count=1`:

- Labels: create, revise, concurrent heads kept, cycles and missing or
  foreign-scope predecessors refused, non-canonical paths refused; format 2
  stale `supersedes` returns `memory.stale_heads`; field limits at and one
  byte over, blank, NUL and newline; signet scope refused; idempotent retry
  returns the same receipt and different input returns `request.conflict`.
- Read-only clients don't list `memory_name_scope`, and calling it or a
  scoped `memory_journal_append` returns `access.denied`.
- Journal: unscoped entries keep schema 1, their bytes and their place in an
  unfiltered query; scoped entries come back with and without the filter, and
  `query` narrows a scoped query; `scope: {kind: "signet"}` is refused as
  `input.invalid`; a schema 2 entry under `memory/events/` and a schema 1
  entry under the new directory are refused by validation.
- Intents: after a completed naming and a completed scoped append,
  `PendingIntents` reports nothing; with each target file removed, it reports
  that intent as pending. An unscoped journal intent written by 2.2.0 is
  byte-identical in shape to one written by 2.1.2.
- Scopes: query matches by ID, name and description and ranks the named
  topic first; conflicting labels both shown; a named empty scope listed;
  forty scopes with 1024-byte descriptions page within 32 KiB.
- Prime: section order; each section newest first by `recorded_at`, with
  equal times ordered by ID; more than eight records in the scope all
  selected; a corrected `project-state` shows only its new head; a withdrawn
  record absent on format 2; a conflicting record in `conflicts`; no item is
  clipped. At 1024 bytes with a 1024-byte description: the name is returned
  without the description, `truncated` is true and the counts say what was
  left out; with a short description, the description and state both fit.
- Large topic: a scope with 40 records with 4 KiB bodies and 5 journal
  entries, at the default budget, returns all 40 summaries in `current` or
  discloses truncation, and returns at least the newest 3 journal entries.
  `current` items carry no body; `state` items do.
- Scopes: a scope with only scoped journal entries is listed with
  `record_count` 0.
- Canon: a 2.2.0 server fetches, verifies, refreshes and serves a canon
  source whose commit contains labels and scoped journal entries; a source
  with a malformed label is withheld like any invalid snapshot.
- Export: selecting a scope selects its records, scoped journal entries and
  labels; `journal_ids` and `omit_journal_ids` reach both directories;
  `omit_fields` content removes names and descriptions; a label-only scope is
  selectable; the 128-item cap covers scope-selected journal entries.
- Format: the committed 2.1.2 fixtures validate, recall and journal
  unchanged under the new code. They are synthetic and cover every document
  type each format holds: on format 1, a correction, a content conflict, a
  source, an unscoped journal entry, a device, a legacy foundling
  registration and a canon (version 2) registration; on format 2, the same
  plus visibility events (a withdrawal and a restoration) and the upgrade
  record. The committed generator drives the 2.1.2 binary's CLI, so they can
  be rebuilt from 79b378e; after naming and scoped journal
  writes on a copy, every original file hashes the same and the only new
  paths are under `memory/scopes/`. A format 1 signet with labels upgrades to
  format 2 and syncs. Sync and export accept the new paths and still refuse
  unknown ones.
- `memory_context` output is identical with and without labels and scoped
  entries.
- Instructions stay under 600 bytes (existing test), at 595 and 411.
- Old reader (`bin/check-old-reader`): the 2.1.2 binary built from 79b378e,
  run against a signet 2.2.0 wrote with labels, scoped journal entries and
  their intents, on format 1 and format 2:
  - recall, journal, history, inspect and `memory_context` succeed and show
    no names or scoped entries;
  - 2.2.0 has no client-side sync command; a server reconciles with
    `server reconcile` (fast-forward only) and a 1.x-style clone with the
    menu's sync (merge). 2.1.2 reconcile behind the remote refuses with
    "unsafe or partially staged work"; with an unpushed local commit it
    refuses as diverged; the menu sync with an unpushed local commit refuses
    at validate. In every case HEAD, the index, the files and the unpushed
    commit are unchanged;
  - a 2.1.2 checkpoint with the new files in its worktree refuses;
  - export preview refuses (its message is sanitized), and the same
    selection previews under this build;
  - `memory_journal_append` with `entry.scope` returns `input.invalid` and
    no file appears in the signet or `.mandalore/`;
  - the same `request_id` without `entry.scope` returns `save.incomplete`
    with nothing written;
  - cutover preflight reports the intent decode error;
  - canon refresh of a source with the new paths serves it, without names.

Repository checks: `bin/container bin/ci` needs a TTY, so where it can't get
one I run host `bin/ci` in a Herdr tab and say so in the PR; `bin/sdlc
check`; `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`. Plugin
validation needs `claude` on `PATH`; any Claude Code run uses a temporary
`HOME` and `CLAUDE_CONFIG_DIR`.

For the independent verifier, on the built 2.2.0 binary against a running
server, the issue's acceptance with two installations, A read-write and B
read-only. The main pass uses a synthetic signet upgraded to format 2 before
step 1, so the server keeps running throughout:

1. A names a topic, saves three records in its scope including a
   `project-state`, and appends two scoped journal entries.
2. B, in a fresh session given only the display name, finds it with a
   `memory_scopes` query and primes on it, seeing the description and the
   state record first.
3. A corrects the state and withdraws another record; B's next prime shows
   the new state only and not the withdrawn record.
4. A 1024-byte prime returns the description and state or discloses
   truncation.
5. B is refused naming and a scoped journal write.
6. The format 1 and format 2 fixtures validate and their unscoped journal
   entries are returned.

A second pass on a format 1 signet repeats steps 1 to 5 without the
withdrawal. The verifier also reruns `bin/check-old-reader`. Separately, the requester's private
recall eval on the candidate (no regression against 2.1.2) happens before
nomination, as the triage says.

## Rollback

Reinstall 2.1.2 (`sh install.sh 2.1.2`, or the bare-binary atomic swap) and,
for the plugin, revert its bump on main. Nothing existing is rewritten, so
2.1.2 reads every record and unscoped journal entry as before. If labels or
scoped entries were written, 2.1.2 can't sync or export that signet until
2.2.0 is reinstalled; they can't be removed without editing append-only
history, which is out of bounds. Their intent files in `.mandalore/` make
2.1.2's cutover preflight fail with a decode error, and a journal retry that
reuses such a `request_id` returns `save.incomplete` without writing (see
Risks). Serving is unaffected. Roll back before the first naming or scoped
journal write to avoid that cost entirely.
