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
- Scope kinds are in the revision schema enum and `validateScope`. A new kind
  would be a new value in existing documents, which strict older readers
  refuse, so it would force the format bump that (b) avoids.
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
| Sync and checkpoint (`allowedPath` in `internal/sync`) | Refuse the new paths as unsafe work, preserving files and index; nothing is lost |
| Export report (`internal/exportreport`) | Refuses "unknown portable data" |

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
| `memory_journal` | New optional `scope` filter; entries gain `scope` when set |

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
`memory_remember`, through the same intent files under `.mandalore/` with a new
intent kind.

**`memory_scopes`.** Each item keeps `scope`, `record_count` and
`visibility_counts`, and gains `labels`: the current label heads as
`{id, name, description}`, absent when the scope is unnamed, more than one
entry when they conflict. Named scopes with no records are listed with
`record_count` 0. With `query`, items whose ID, name or description match
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
clients are already refused every journal write.

**`memory_prime`.** One read that returns, in this order:

1. `labels`: the scope's current name and description heads.
2. `state`: current `project-state` heads in the scope, newest first.
3. `current`: other current heads in the scope, newest first.
4. `journal`: the scope's journal entries, newest first.

It follows recall: exact scope only, no signet-wide records mixed in;
withdrawn records excluded before selection; records with several heads are
listed in `conflicts` (record ID and head IDs) instead of being picked; whole
items are omitted rather than clipped; and the response carries
`matching_counts` per section, `truncated` and recall's evidence notice. The
budget covers the whole JSON response, filled in the order above with
conflicts after `state`, so at the minimum budget the name, description and
state come first or `truncated` says what was left out. Read clients get it.
Selection reuses the store's recall path with an empty query, so ranking and
`memory_context` code are not touched.

**Guidance.** The server instructions stay under their existing 600-byte limit
(`TestInstructionsAreShort`; today 330 read and 428 read-write). Read-write
gains about 160 bytes: "For work that will outlast this chat, find its topic
with memory_scopes, prime with memory_prime, save into its scope, correct its
project-state record rather than add one, and journal with the scope." Read
gains the find-and-prime half. `this-is-the-way` gains a short "Topics
(bounties)" paragraph with the same steps plus naming, and its
`references/memory.md` gains the label rules.

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
  `internal/sync/checkpoint.go` admits the two new path shapes; the export
  report snapshot (`internal/exportreport/snapshot.go`, `memory.Snapshot`)
  reads and validates them. The format upgrade's portable inventory goes
  through `allowedPath`, so a format 1 signet with labels can still upgrade;
  a test proves it.
- **Docs**: `docs/signet-format.md` (layout, label rules, journal schema 2,
  old-reader table), `docs/server.md` (Tools: the two new tools and new
  parameters, scope lists), `docs/interface.md` (operations and failure
  table), the plugin README and both skill files.
- **Release preparation**: `VERSION` and
  `plugins/claude/.claude-plugin/plugin.json` to 2.2.0, and
  `docs/releases/2.2.0.md` with what changed, the upgrade-every-syncing-clone
  note, the owner's #229 step (installed plugins follow main, so the plugin
  merge waits for the owner to confirm the live server runs 2.2.0), rollout
  and rollback. No new HTTP or OAuth path, so no Cloudflare allowlist change.
  No new secret.
- **Tests and fixtures** in `internal/memory`, `internal/api`,
  `internal/server`, `internal/sync`, `internal/exportreport`, with small
  synthetic format 1 and format 2 signets written by the 2.1.2 code under
  `internal/memory/testdata/` (see Proof).

## Order of work

1. This plan, alone, as the first commit; draft PR; plan review; wait for the
   requester's go.
2. Fixtures: generate the format 1 and format 2 signets from 2.1.2 (`main` at
   79b378e) and commit them with a test that validates them and hashes every
   file.
3. Labels in the store and service, with graph and input tests; then
   `memory_name_scope` over MCP with idempotency and the read-only refusal.
4. Scoped journal in store, service, catalog and MCP, with tests that
   unscoped entries are unchanged and still returned.
5. `memory_scopes` query and labels, with the bounded-page tests and a test
   that `memory_context` output is identical with and without labels.
6. `memory_prime`, with order, budget, withdrawal and conflict tests.
7. Sync, export and upgrade path support, with their tests.
8. Instructions, skill, docs, version bump and release notes; `bin/ci`,
   `bin/sdlc check`, govulncheck; simplify pass; mark the PR ready.

## Risks

- **Old clones that sync.** A 2.1.2 clone refuses to sync once the new files
  arrive. It loses nothing and recovers by installing 2.2.0. Contained by the
  release note and the table above; the error those clones show ("unsafe or
  partially staged work") is old code we can't change, so the note names it.
- **Rollback after use.** 2.1.2 on a signet with labels serves memory
  normally but can't sync or export until 2.2.0 is back. Intent files for
  `memory_name_scope` and scoped journal saves are a new kind with a scope in
  the digest; an older binary retrying such a `request_id` doesn't recognise
  the intent and reports `save.incomplete`, writing nothing. The release notes
  carry both.
- **Hook drift (#232).** `memory_context` reads `Store.Scopes`, which doesn't
  change, and prime reuses recall's store path without touching ranking. A
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
  rolls the new skill text out at once. The PR merges only after the owner
  confirms the live server runs 2.2.0; until then the skill would name tools
  the server lacks.

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
  unfiltered query; scoped entries come back with and without the filter; a
  schema 2 entry under `memory/events/` and a schema 1 entry under the new
  directory are refused by validation.
- Scopes: query matches by ID, name and description and ranks the named
  topic first; conflicting labels both shown; a named empty scope listed;
  forty scopes with 1024-byte descriptions page within 32 KiB.
- Prime: section order; newest first; a corrected `project-state` shows only
  its new head; a withdrawn record absent on format 2; a conflicting record
  in `conflicts`; at 1024 bytes the labels and state come first or
  `truncated` is true with counts; no item is clipped.
- Format: the committed 2.1.2 format 1 and format 2 fixtures validate, recall
  and journal unchanged under the new code; after naming and scoped journal
  writes on a copy, every original file hashes the same and the only new
  paths are under `memory/scopes/`. A format 1 signet with labels upgrades to
  format 2 and syncs. Sync and export accept the new paths and still refuse
  unknown ones.
- `memory_context` output is identical with and without labels and scoped
  entries.
- Instructions stay under 600 bytes (existing test).

Repository checks: `bin/container bin/ci` needs a TTY, so where it can't get
one I run host `bin/ci` in a Herdr tab and say so in the PR; `bin/sdlc
check`; `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`. Plugin
validation needs `claude` on `PATH`; any Claude Code run uses a temporary
`HOME` and `CLAUDE_CONFIG_DIR`.

For the independent verifier, on the built 2.2.0 binary against a running
server on a synthetic signet: the issue's acceptance with two installations,
A read-write and B read-only. A names a topic, saves three records in its
scope including a `project-state`, and appends two scoped journal entries; B,
in a fresh session given only the display name, finds it with
`memory_scopes` and a query and primes on it, seeing the description and the
state record first; A corrects the state and withdraws another record (the
signet upgraded to format 2 for that step), and B's next prime shows the new
state only and not the withdrawn record; a 1024-byte prime returns the
description and state or discloses truncation; B is refused naming and a
scoped journal write; the format 1 and format 2 fixtures validate and their
unscoped journal entries are returned. Separately, the requester's private
recall eval on the candidate (no regression against 2.1.2) happens before
nomination, as the triage says.

## Rollback

Reinstall 2.1.2 (`sh install.sh 2.1.2`, or the bare-binary atomic swap) and,
for the plugin, revert its bump on main. Nothing existing is rewritten, so
2.1.2 reads every record and unscoped journal entry as before. If labels or
scoped entries were written, 2.1.2 can't sync or export that signet until
2.2.0 is reinstalled; they can't be removed without editing append-only
history, which is out of bounds. Roll back before the first naming or scoped
journal write to avoid that cost entirely.
