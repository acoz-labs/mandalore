# Signet format 1

A signet is an explicitly selected directory independent of the calling cwd.
It stores structured evidence, not executable policy, credentials or transcripts.
The current library supports format 1 only. Protocol and release versions are
separate concerns; no migration or downgrade is inferred from similar filenames.

## Layout

```text
signet.json
memory/records/<record-id>/<revision-id>.json
memory/sources/<source-id>.json
memory/events/<UTC-year>/<UTC-month>/<event-id>.json
provenance/devices/<device-id>.json
foundlings/registrations/<foundling-id>/<revision-id>.json
.mandalore/                 # ignored machine-local locks/state
```

Creation publishes a new directory exclusively and never replaces an existing
target. JSON files are strict about unknown fields, trailing JSON, nonregular
files and the 4 MiB file-size ceiling. IDs match `^[a-z][a-z0-9-]{2,127}$` and
must match canonical paths. Files are append-only; new JSON is synced, hard-linked
without replacing an existing destination, then its parent directory is synced.
This is not a multi-file transaction or protection from an actively hostile
local filesystem owner. Symlink checks do not turn the library into a sandbox.

Writes share a nonblocking local file lock. A busy writer reports a retryable
error; it does not overwrite the other writer. Reads do not create locks, local
directories, journals or indexes. A replaced signet ID invalidates existing store
handles. `.mandalore` is absent in a normal Git clone until the first write.

The manifest contains `schema_version`, opaque `id`, and one-line `name`
(1–256 bytes). Device records contain `schema_version`, `id`, and an explicitly
chosen nonempty `label`; the toolkit does not discover or publish a hostname.

## Records, scope and evidence

The [revision JSON Schema](../internal/memory/schemas/memory-revision.schema.json)
is the field/enum authority. Revisions include stable record identity, kind,
scope, summary/body, sensitivity, volatility, recorded/effective time, authorship,
evidence, predecessor IDs and a change reason. Supported kinds are fact,
preference, decision, procedure, project-state, entity, commitment and research-claim.

Scopes are `signet`, `project`, `account`, or `task`. Signet-wide scope must use
this signet's ID. Service operations default to that exact scope, not all scopes.
Scope discovery lists stable IDs/counts without promoting unrelated guidance.
New records get one root. Corrections preserve record identity/kind/scope and
name predecessors; roots, cycles, missing predecessors, unknown devices and
missing/invalid evidence are rejected. A correction cannot take effect before
its predecessor. Two effective heads are a visible conflict, never a winner
chosen by timestamp or lexical score. History preserves all branches.

Authorship records originating device, actor, harness, and optional model/session
identity. Unknown model/session values remain null. Source evidence contains
`schema_version`, `id`, `kind`, concise `summary`, `device_id`, `recorded_at`,
and optional `external_origin`. Sourced writes validate the proposed graph before
writing evidence or revisions. An I/O failure between publication of the two
files may leave orphan evidence; inspect before retrying. No exactly-once claim
is made. Full validation checks unreferenced evidence too.

Service writes require summary 1–256 bytes, body 1–8192 bytes, reason 1–1024
bytes and at most 32 predecessors. Default sensitivity is private, volatility
drift-prone and confidence medium; basis is explicitly supplied. The low-level
schema and 4 MiB file bound remain distinct from service input budgets.

## Journals and retrieval

Journals contain version, ID, kind, summary, UTC recorded time and authorship.
They are semantic summaries, not native transcripts or authoritative current
facts. Service journal kinds are 1–64 bytes, summaries 1–4096 bytes. Journal
queries return newest-first matches with explicit truncation, not a stable cursor.

Scoped lexical recall selects effective unsuperseded heads before ranking, so
an obsolete keyword match does not resurrect superseded guidance. Exact token
matches outweigh a limited English-inflection fallback; identifiers are not
fuzzy-matched. No embedding service or index is used. Query size is at most
2048 bytes; service limits are 1–50 results with a total JSON budget of
1024–32768 bytes. Whole claims are omitted rather than clipped, with matching
counts, conflict counts and truncation disclosed. Empty results do not prove
absence. History/scope pages are bounded to 32 KiB; concurrent mutations can
change offsets, so these are not snapshot cursors.

## Foundling registrations and citations

Registration fields are `schema_version`, `id`, `foundling_id`, `name`
(1–256 bytes), `description` (1–4096), `source`, `pin`, `state`, `recorded_at`,
`authorship`, `supersedes` (at most 32) and `change_reason` (1–1024).
States are active/disconnected. Registration revisions are append-only with one
root, same-foundling predecessors and no cycles; concurrent heads remain present
for explicit resolution. Registration data is excluded from ordinary recall.

`source` contains kind and locator (at most 2048 bytes). Git sources use
credential-free HTTPS/SSH URLs or SCP-style identities. SSH transport usernames
are allowed, HTTPS userinfo and SSH passwords are not. Query strings, fragments,
traversal and machine-local filesystem paths are refused. Local references use
an opaque source ID; the actual path belongs to a future machine-local binding.
`pin` contains algorithm and value: `git-sha1`/40 lowercase hex characters or
`git-sha256`/64 for Git; `sha256`/64 for local sources. Branch names are not pins.

Optional source `external_origin` includes `foundling_id`, immutable
`registration_revision_id`, `source_identity`, `source_pin`, a safe relative
`relative_locator` (at most 2048 bytes), `content_sha256`, and optional
`original_recorded_at`/`original_author` (at most 256 bytes). Unknown original
attribution stays absent, not an invented device enrollment. The citation must
match an existing registration's identity/pin. Disconnecting later preserves
historical citations. Source/revision device and time record incorporation,
separately from original author/time.

These are data-validation guarantees only. No registration fetches, executes or
trusts source content, and a supplied fingerprint is not proof of a successful
source verification. Retrieval, changed-pin observations and selective promotion
are the later foundling workflow. Unevaluated references are neither current
knowledge nor automatically superseded history.

## Compatibility and recovery

Roots containing legacy `bank.json` or `agent.json` are refused, including mixed
formats. Do not rename those files to bypass migration checks. Preserve the
original store and writer until explicit conversion with backups/ID preservation
and rollback is supported. Normal recovery inspects rejected/corrupt files and
history; it never silently drops evidence or picks a conflicting head. Native
Linux behavior, cross-machine bindings and sync require separate acceptance.
