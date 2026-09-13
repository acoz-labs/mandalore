# Technical design

## Components

The first slice adds `internal/memory` with focused files for the manifest/store,
records/graph/scopes, sources/devices, journal, atomic file publication/locking,
schema validation and bounded service-level recall/history. Names may be split
further only where tests or dependency direction require it.

| Public predecessor source | Extraction boundary |
| --- | --- |
| `portable/store.go` | JSON/lock/device/source/schema and memory-only directory logic; exclude Agent/Create/legacy branches |
| `portable/memory.go` | Records, scopes, graph validation, lexical scoring and histories |
| `portable/bank.go` | Sourced writes and journals, adapted to explicit signet manifest |
| `portable/events.go` | Journal types/write only, no CheckCapability methods |
| `portable/changes.go` | Explicit authorship validation only; source-change Git audit belongs to sync design |
| `portable/rename_darwin.go`, `rename_linux.go` | Reviewed atomic directory publication and native tests |
| `portable/schemas/memory-revision.schema.json` | Reviewed signet scope/schema update, documented as new format |
| `memorybank/service.go`, `pages.go` | Validation, defaults, budgets, pagination; no Sync dependency in core |

## Proposed on-disk contract

- `signet.json`: schema version, stable bank identity and user-chosen display name.
- `memory/records/<record-id>/<revision-id>.json`: immutable structured revisions.
- `memory/sources/<source-id>.json`: evidence records.
- `memory/events/<year>/<month>/<event-id>.json`: semantic journals.
- `provenance/devices/<device-id>.json`: explicit device labels/identities.
- `foundlings/registrations/<foundling-id>/<revision-id>.json`: immutable
  reference registration metadata, excluded from ordinary memory recall.
- `.mandalore/`: local lock/state namespace, not credentials or native sessions.

Document required/optional fields, limits, extension policy, canonical paths and
unknown-version rejection. Preserve input bounds and validate the full proposed
graph before mutation. A disk failure can leave unreferenced source evidence;
never claim a two-file write is transactionally atomic if it is not.

## Foundling registration and citation contract

Define data-only validation in this slice; external source access and the
register/retrieve/promote workflow remain #12 after the shared interface exists.

Registration fields: `schema_version`, `id`, `foundling_id`, `name`, `description`,
`source` (kind and portable locator), `pin` (algorithm and immutable revision or
digest), `state` (active/disconnected), `recorded_at`, `authorship`, `supersedes`
and `change_reason`. File IDs must agree with their canonical path. Changes are
append-only and concurrent heads are visible conflicts, not arbitrary selection.

Portable locators are credential-free HTTPS/SSH Git identities or opaque local
source IDs. Absolute filesystem paths, URL user/password credentials, query
credentials and environment-variable secret values are rejected. Actual paths
and observations such as available/missing/changed belong to machine-local
bindings. The first implementation only validates these values; it does not
fetch, resolve DNS, run Git or execute a registered source.

Source evidence may include an optional `external_origin` object containing
`foundling_id`, `registration_revision_id`, `source_identity`, `source_pin`,
`relative_locator`, `content_sha256`, and optional `original_recorded_at` and
`original_author`. Reject traversal/absolute locators and malformed fingerprints.
Original author is historical source metadata, not a fabricated local enrollment.
Retain the source identity and pin inside the citation so it remains intelligible
after disconnection. The source's existing recorded time/device and the promoted
revision's authorship describe incorporation, not original authorship.

The final data schema must document size limits and optional-field treatment.
Validate supplied citations before source/revision writes. A citation must name
an existing matching registration revision, but disconnection must not invalidate
historical evidence already recorded. No current-memory query includes raw
registration content as guidance. Full source-hash verification is the later
retrieval/promotion operation's responsibility, not an inferred engine guarantee.

## State and validation

Create from a nonexistent target using owned staging and atomic publication.
Reject symlink/nonregular redirection and unknown manifest versions. Validate
stable scope/record identity across supersession, one root per record, cycles,
missing predecessors, source references and originating devices. Concurrent
semantic heads are returned as conflicts rather than chosen arbitrarily.

Expose current data, histories and journals separately. Bounded replies must
report truncation/continuation and never clip a claim into misleading text.
Read-only calls must not write indexes, journals, repairs or locks unnecessarily.

## Authorization and failure

This is a local engine called by authorized host tooling, not a sandbox or policy
engine. Stored prose grants no authority. Use strict inputs and safe filesystem
boundaries; explicitly reject old bank/assistant formats in this slice.

Preserve old files on failed validation. Test partial I/O and lock contention;
report errors without raw private content. No hidden network, subprocess,
credential lookup or capability dispatch may enter the core dependency graph.

## Traceability

#2 extraction/privacy/clean dependency criteria map to the source map and import
boundary tests. #3 manifest/provenance/scope/evolution criteria map to format and
graph fixtures. Bindings and cross-process sync need their later integration
issues; this library slice must not claim their end-to-end completion.
