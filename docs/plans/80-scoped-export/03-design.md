# Technical Design

## Interfaces and flow

Add strict CLI-only typed `export_preview` and `export_apply` operations plus
`mandalore export preview|apply` JSON-stdin wrappers. Preview accepts an explicit
binding path, destination directory, record IDs/scopes/journal IDs, opt-in history
and detail flags, and omission policy. Apply accepts only the complete reviewed
preview. Read-only API mode denies apply before decoding; preview never writes,
checkpoints, journals or fetches. No corresponding everyday MCP tool is added.

Resolve and hash the selected binding, open its guarded signet, validate memory,
and build a bounded read-only snapshot. Read source twice and compare deterministic
raw-file digests to reject a changing snapshot without creating a bank lockfile.
Validate the exact decoded snapshot used for projection with engine validators;
validation of separately reread live files is not proof about projected bytes.
Bound source reads to 128 MiB, each existing file to the engine's 4 MiB limit,
selection input to 128 record IDs/16 scopes/128 journal IDs, and total projected
revisions plus journals to 128 items. Limit report bytes to 4 MiB and preview
JSON to 24 KiB so it remains usable within the 32 KiB apply input budget.
Bound cumulative reads and selected output; fail with a scope-narrowing message
instead of returning an incomplete usable plan. Re-read identity and source after
projection. The report is derived only from validated selected data, never from
unknown files, Git objects, ignored state or raw harness logs.

Preview contains format version, binding/source digests, canonical source/dest
coordinates, exact selected record/revision/journal IDs, unresolved conflicts,
omission counts/categories, effective redaction policy, exact report byte count
and SHA256, and fresh-directory/file effects. It does not return source bodies,
device labels, reasons or citation content. Serialize deterministic lists and reject a
preview too large for the apply input budget. Exact source digests make drift
fail closed, including a changed selection after new records are added.

Apply independently rebuilds the preview from current binding/source/destination
and compares all fields before writing. It never trusts a caller-supplied report
or expected digest without reconstruction. Recheck before publication and refuse
changed binding, source, parent directory identity or overlap. No automatic retry.

## Selection and report shape

Use engine-owned validation/selection access, not a second looser JSON reader.
Current heads are derived from the existing supersession graph. No timestamp
winner. History opt-in contains all revisions of explicitly selected records,
with current/conflicted/historical status. Journals require exact IDs, never a
query or inferred scope. Missing explicit IDs fail. Sources/devices are included
only as opt-in details reachable from selected entries; no unrelated provenance
or foundling registration/source documents are copied. Historical origins remain
citations, not active instructions or complete original evidence.

Report header uses `kind: mandalore-memory-report`, version 1 and fixed boundary
notices. Group each projected item into omission categories, with report-local
ordinals. Report metadata contains only deliberate selected disclosure; plan
hashes/absolute paths and original digests stay outside the report. All categories
can be omitted; detail/history opt-ins do not override an omission policy.
No unredacted report is written as an intermediate step. The receipt states that
this operation wrote only the selected report and did not inspect or remove prior
exports elsewhere.

## Filesystem and authorization

Destination must be a fresh directory beneath an existing, nonoverlapping real
parent. Resolve ancestor aliases and compare filesystem identity; reject dangling
links, symlink destinations, existing entries, bank/ignored-state overlap,
binding-directory overlap and overlap with every configured local foundling root,
including disconnected sources. Malformed/uninspectable connection metadata
fails closed. Do not read foundling content merely to validate a destination.
Guard the parent with a directory handle and recheck its identity before writes.
Create a randomly named private sibling staging directory with mode0700 and
report.json exclusively mode0600; write only the final redacted bytes and sync.
Revalidate source/binding/parent and publish the directory using the existing
platform no-replace rename pattern (Darwin RENAME_EXCL, Linux RENAME_NOREPLACE).
A concurrently created destination is preserved, never merged into or overwritten.
Use directory-relative handles for writes/publication and inspect identity changes;
do not reopen a replaced staging path and write into someone else's directory.
Never automatically delete partial output. Do not claim a sandbox against an
unrestricted filesystem owner modifying already-produced output.

The result/error receipt names any staging or published directory/file and phase,
including cancellation/partial writes. On post-write source drift, the partial
artifact remains explicitly unconfirmed; no success/rollback claim. A retry needs
a fresh destination and a newly reviewed plan. No source lockfile, receipt,
journal, checkpoint or memory mutation is performed by export itself. Failures
do not print raw sensitive field contents or provider errors.
