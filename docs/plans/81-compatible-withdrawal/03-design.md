# Technical Design

## Data and read boundary

Support reading existing format1 unchanged and opt-in format2 banks. Preserve
legacy revision/source/journal/device bytes during transition. Introduce a
versioned visibility-event schema with stable record ID, event ID, action,
content-head references, parent visibility-head references, reason and authorship.
New format2 content revisions require visibility references; legacy revisions
have none and cannot be reinterpreted as having observed a later restore.

Validate both per-type parent graphs and their combined causal edges. Enforce
record ownership, unique IDs, existing typed references, nonempty observed content
for decisions and absence of cycles. Local writes compare expected content and
visibility heads under the shared writer lock. Concurrent valid branches remain
inspectable after union. Event creation never chooses a timestamp winner.

Share visibility evaluation across recall, scope counts, native context and report
projection. Exclude withheld text before scoring or context assembly. Explicit
history retains content and events with visibility status. Current-only export
omits withheld records; historical disclosure requires include-withdrawn as an
additional opt-in. Snapshot/digest validation includes visibility objects.

## Interfaces

Provide typed withdraw/restore operations using record ID, expected content and
visibility heads and reason. Explicit history exposes visibility decisions and
provenance. Receipts distinguish durable local event, resulting visibility,
semantic conflicts and delivery. Enforced read-only denies mutations before
decoding. Existing ordinary remember behavior remains unchanged for format1 and
unaffected format2 records; correcting a withheld record does not restore it.

Retention preview takes explicit record/scope selection, a named timestamp and
absolute cutoff when age is used, plus separately selected journal IDs if needed.
It returns bounded IDs, matched policy, current heads/state and known source/
origin relationships, not bodies. Pin source/binding/policy; fail on truncation
instead of returning a misleading complete plan. No retention-apply operation.

Foundling promotion must check target visibility and exact known origin identity
before creating new guidance. A withdrawn target or re-promotion of the same
known origin needs explicit resolution; this is not semantic duplicate detection.
Journal search/history and original foundling reads retain their independent
disclosure boundaries. No inferred record-to-journal association is created.

## Upgrade and synchronization

Provide explicit upgrade preview/apply preserving the same identity/repository.
Require stopped-writer acknowledgement, exact reviewed source/binding/HEAD and
private preparation under the writer lock. Append validated upgrade metadata
before atomically activating the new manifest. No withdrawal mutation can precede
the format gate. Preserve original evidence and retain inspectable partial paths.

Normal append-only enforcement remains. A narrowly validated monotonic transition
may change the manifest only with preserved evidence, unchanged identity and a
matching upgrade record. No downgrade or general evidence-edit exception.
Remote format2 adoption into a format1 clone requires explicit local consent;
ordinary sync reports the upgrade requirement without silently adopting it.

### Transaction contract

Use a versioned append-only upgrade object under `provenance/upgrades/` containing
its ID, signet ID, from/to versions, original manifest SHA-256, base Git HEAD,
portable pre-upgrade inventory digest, authorship and time. The digest excludes
the new upgrade object itself and machine-local files. Independent upgrades on
two clones may produce two valid receipts; neither receipt is a visibility vote.
The manifest changes only its schema version, retaining all other fields.

The upgrade inventory digest frames each checkpointed portable file as path-byte
length, path, content-byte length and raw content, in lexical path order. It
includes README, `.gitignore` and tracked placeholders, unlike the narrower
report-data digest. Git objects, machine-local state and root `.DS_Store` are
excluded. Verify actual file bytes against the checkpoint's blob IDs; index flags
must not hide edits. Apply/recovery and transition validation must use this same
inventory definition, not substitute the report-data digest.

Preview requires a valid, Git-backed, checkpointed format1 source. It is read-only
and pins the exact binding, root identity, HEAD, original manifest bytes and
portable inventory. Dirty evidence must be explicitly checkpointed first; preview
does not checkpoint on the user's behalf. Apply takes the returned plan and an
explicit stopped-writers acknowledgement. Under the existing shared exclusive
writer lock it rechecks every pin, prepares and validates the candidate, publishes
the upgrade object exclusively, then atomically replaces the manifest with the
validated version2 bytes. Sync/checkpoint are subsequent operations, not hidden
parts of format activation. The receipt reports each completed durable phase.

Keep recovery preparation in ignored, private `.mandalore` storage. A prepared
upgrade object with an unchanged format1 manifest is pending, not active. New
readers may inspect it but refuse further mutations until explicit recovery;
released readers may refuse the unfamiliar portable path, which is safe. Recovery
revalidates original pins excluding only the exact prepared object. Changed
evidence, bindings or HEAD require a fresh diagnosis, not automatic replay. Never
delete evidence or downgrade after activation. Test interrupted preparation,
publication, manifest replacement and directory durability independently.

### Remote consent without an ambient permission flag

Do not persist a broad "allow remote upgrades" preference or silently consume a
fetched format2 candidate. A format1 clone reports `upgrade-required` and preserves
its local evidence. The user explicitly upgrades that local clone through the
same preview/apply contract, then separately requests sync. Its active format2
manifest is the local opt-in boundary. Sync still validates the exact fetched
tree, all append-only evidence and both parents before adoption; local activation
does not approve arbitrary future remote data or resolve conflicts.

This avoids pinning consent to a moving remote branch and preserves concurrent
local content through the existing union/conflict path. Concurrent valid upgrade
receipts can merge; unsupported transitions, altered identities, edited old blobs,
missing transition evidence and downgrades must fail. The sole manifest exception
is a validated 1-to-2 change, not a general exemption for `signet.json`.

## Failure and exposure

Pre-activation interruption cannot claim successful withdrawal. Post-activation
failure reports format state and checkpoint/delivery separately; no automatic
downgrade, rollback or destructive cleanup. Stale plans and ambiguous writes
require inspection, not blind retries. Error output omits sensitive contents.
Metadata, including IDs/reasons, is not inherently safe to publish. Native
previously-read context and offline old clones remain outside revocation.
