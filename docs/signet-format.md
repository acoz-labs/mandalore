# Signet formats

A signet is an explicitly selected directory independent of the calling cwd.
It stores structured evidence, not executable policy, credentials or transcripts.
The library reads formats 1 and 2. New signets still start in format 1; format 2
requires an explicit local upgrade for withdrawal and restoration. Protocol and
release versions are separate concerns; no migration or downgrade is inferred
from similar filenames or ordinary startup, recall or synchronization.

## Layout

```text
signet.json
memory/records/<record-id>/<revision-id>.json
memory/sources/<source-id>.json
memory/events/<UTC-year>/<UTC-month>/<event-id>.json
memory/visibility/<record-id>/<event-id>.json  # format 2 only
provenance/devices/<device-id>.json
provenance/upgrades/<upgrade-id>.json         # explicit transition evidence
foundlings/registrations/<foundling-id>/<revision-id>.json
.mandalore/                 # ignored machine-local locks/state
```

Creation publishes a new directory exclusively and never replaces an existing
target. JSON files are strict about unknown fields, trailing JSON, nonregular
files and the 4 MiB file-size ceiling. IDs match `^[a-z][a-z0-9-]{2,127}$` and
must match canonical paths. Evidence files are append-only; new JSON is synced,
hard-linked without replacing an existing destination, then its parent directory
is synced. The explicit format transition is the narrow manifest-replacement
exception described below, not a general evidence-edit mechanism.
This is not a multi-file transaction or protection from an actively hostile
local filesystem owner. Symlink checks do not turn the library into a sandbox.

Machine bindings are versioned local files outside this tree; they pin signet
identity, canonical path, enrolled device and actor. They never travel with Git.
See [the interface contract](interface.md#binding-and-provenance) for selection,
new-device enrollment, validation and no-overwrite publication.

Writes share a nonblocking local file lock. A busy writer reports a retryable
error; it does not overwrite the other writer. Reads do not create locks, local
directories, journals or indexes. A replaced signet ID invalidates existing store
handles. A changed format also invalidates cached store handles. `.mandalore` is
absent in a normal Git clone until the first write.

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

Both documents must fit the serialized file limit before either is published;
an oversized revision is invalid input, not an allowable orphan-evidence case.

Service writes require summary 1–256 bytes, body 1–8192 bytes, reason 1–1024
bytes and at most 32 predecessors. Default sensitivity is private, volatility
drift-prone and confidence medium; basis is explicitly supplied. The low-level
schema and 4 MiB file bound remain distinct from service input budgets.

## Format 2 visibility and preserved history

The content supersession graph and visibility decisions are distinct. Existing
format1 revisions retain their original bytes after upgrade. New format2
revisions use revision schema 2 and require `visibility_refs` (an explicit array,
possibly empty), recording the visibility heads observed by that write.

Visibility events use event schema 1 and contain `id`, `record_id`, `action`
(`withdraw` or `restore`), `observed_content_heads`, `parent_visibility_heads`,
`reason`, `recorded_at` and `authorship`. Event and revision references are typed,
must belong to the same record, and form one validated acyclic causal graph.
Missing references, duplicate references, cycles and unknown devices refuse
validation. Format1 banks cannot contain visibility events.

No visibility events means ordinary visibility. A single withdrawal head
withholds the record. Multiple visibility heads are a visibility conflict,
including two concurrent restores; there is no timestamp winner. Restoration
covers its reviewed content ancestry and later revisions that acknowledge it.
Concurrent unreviewed content remains withheld. Content conflicts are preserved,
not resolved by restoration.

The service compares both current content heads and visibility heads under the
shared writer lock before appending a decision. Stale input refuses. A correction
does not implicitly restore a withdrawn record. Publication or cancellation may
leave a durable event even when the call fails; inspect its receipt and history
before retrying. Local durability and Git delivery are separate outcomes.

Recall and context exclude withheld records before ranking or returning content.
Scope counts disclose their state. Explicit content/visibility history retains
the evidence. Default export withholds it; historical disclosure requires both
`include_history` and `include_withdrawn`. Journals and external foundling files
remain independent evidence. Re-promoting the same exact withdrawn foundling
origin is refused, but this is not semantic duplicate detection or erasure.

## Explicit format transition

Upgrade preview requires an explicit binding and clean Git checkpoint. It pins
the binding, root identity, manifest, HEAD and raw portable inventory without
writing. Apply requires the exact plan and actual stopped writers. Machine-local
preparation is retained under `.mandalore/format-upgrade/`; publication places
append-only transition evidence before atomically replacing the manifest.

Upgrade evidence records schema 1, identity, `from_version: 1`, `to_version: 2`,
`original_manifest_sha256`, `base_head`, `portable_sha256`, time and authorship.
It is not permission to upgrade another clone or proof of delivery. Format2
requires valid transition evidence. A format1 manifest with already-published
evidence is a pending transition requiring explicit recovery, not normal use.

Recovery continues the retained preparation and evidence identity, refuses
changed pins and cannot start a new upgrade. Apply/recover do not checkpoint,
synchronize, downgrade or erase history. Original signet identity, revision and
journal bytes survive. Synchronization validates each transition against its
ancestor format1 commit and original protected evidence, not just receipt syntax.
Independent local upgrades may converge with both receipts preserved.

An old client refuses format2. A compatible client with a still-format1 local
clone reports upgrade-required when the remote is upgraded; it does not silently
adopt it. Every clone requires explicit opt-in. Offline copies and content already
read into a conversation remain outside revocation. Installing a compatible
runtime is separate from upgrading a bank; see [the interface](interface.md) for
upgrade operations and the released 1.1.0 updater compatibility limitation.

## Journals and retrieval

Journals contain version, ID, kind, summary, UTC recorded time and authorship.
They are semantic summaries, not native transcripts or authoritative current
facts. Service journal kinds are 1–64 bytes, summaries 1–4096 bytes. Journal
queries return newest-first matches with explicit truncation, not a stable cursor.
Ordering uses actual timestamps, so different timezone offsets do not reorder
events incorrectly.

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
an opaque source ID; the actual path belongs to an ignored clone-local connection
at `.mandalore/foundlings/<foundling-id>.json`, not portable registration metadata.
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
source verification. The separate [foundling workflow](foundlings.md) implements
bounded local verification, changed-pin observations and selective promotion;
ordinary supplied citations do not acquire that verification guarantee merely
by passing schema validation. Unevaluated references are neither current
knowledge nor automatically superseded history.

## Compatibility and recovery

Roots containing legacy `bank.json` or `agent.json` are refused, including mixed
formats. Do not rename those files to bypass migration checks. Preserve the
original store and writer. The [explicit memory-only converter](migration.md)
preserves IDs and history in a new bundle; assistant repositories remain outside
that supported format. Normal recovery inspects rejected/corrupt files and
history; it never silently drops evidence or picks a conflicting head. Native
Linux CLI/plugin behavior, cross-machine bindings and sync require separate
acceptance; library tests run natively on macOS/arm64 and Linux/amd64.
