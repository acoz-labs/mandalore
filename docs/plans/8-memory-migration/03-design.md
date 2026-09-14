# Technical design

## Ownership and interface

Add internal/migration and CLI-only migration_preflight/migration_apply operations
to the existing dispatcher. The normal memory MCP remains unchanged. Input paths
are explicit absolute paths with no overlaps between source, binding and output.
No old environment variable becomes an implicit new binding.

Inventory only bounded regular files without symlinks/devices or traversal.
Reject mixed bank/agent/signet roots and unsupported top-level content. Recognize
known portable memory, devices, source-change evidence, README and ignore rules.
Never copy .git (history/config/hooks/filters), .my-friday (local credentials,
transport state/locks), OS metadata or an outside machine binding. List these
exclusions and their recovery implications rather than implying a full clone.

Bound each file to the engine's 4 MiB limit and the total portable snapshot to
64 MiB/10,000 files; exceeding this is a visible unsupported-size result, never
silent truncation. Bound returned inventories separately with counts and digest.
Use strict JSON and canonical ID/path checks. Explicitly report observed semantic
conflicts without choosing a head. Preserve original file bytes in original/.

## Validation and transformation

Only bank.json becomes signet.json and revision scope.kind assistant becomes
signet after verifying that its scope ID equals the original bank ID. Preserve
all other semantic fields, including extensions and unknown model/session values,
without substituting the importing machine for an original author. Project,
account and task scopes are unchanged. Source/device/journal IDs and timestamps
remain unchanged. Generate only the required empty signet directories and new
ignore rules; exclude operational change observations from normal memory.
Transform only the structural manifest/scope members, never global text matches.
Retain raw JSON values for untouched members so extension numbers do not lose
precision through a float64 round trip; validation must not become reauthoring.
Add a separately identified conversion device and one semantic migration journal
entry using the caller's explicit machine label/actor and CLI harness. This entry
records source content identity and conversion summary, not machine-local paths.
These two new objects identify the conversion without reauthoring old objects;
receipts distinguish preserved from newly added objects. Later binding enrollment
remains a separate operation.

Share the engine's schema/graph validation where practical, using a bounded
in-memory snapshot validation entry point if needed, rather than importing the
predecessor assistant package or creating a competing memory engine. Preflight
must perform real structural/graph checks without writing temporary files; apply
also uses the canonical on-disk Store.Validate before publication. Preserve
conflicting heads and orphan source evidence when they satisfy current invariants.
If stricter current invariants reject legacy data, name the category/locator and
refuse; never repair history as a side effect of import.

## Writer observations and publication

Inspect an existing legacy lock without creating/truncating it. A held exclusive
writer lock blocks conversion. Hold a compatible shared lock during apply when
the existing lock is available, and recheck its identity. An absent lock is not
proof that an idle session or another machine cannot write later.

Optional native inventory uses the already bounded native-command adapter and
reports enabled legacy/current memory integrations as potential duplicate writers.
It does not infer active process state from plugin installation or parse sessions,
auth files or arbitrary commands. Omitted inventory is explicitly not-tested.
Require a quiescence acknowledgement and compare the source fingerprint before
and after snapshot acquisition; changed inputs require a new preflight.

Create a new sibling staging directory owned only by this operation. Write
original/, converted signet/ and a versioned receipt with input/output digests,
counts, exclusions, conversion time and explicit conversion provenance. Local
paths in the receipt stay outside the Git-backed signet. Original device/actor
fields remain untouched; converter identity is recorded separately, not invented
as original authorship. Sync files/directories, validate, then use no-replace
directory publication. Refuse any existing output, including an empty directory.
An I/O failure may leave owned staging; return its location and publication state
for inspection. Never erase source data or a previously published bundle.

The snapshot is an additional local data copy, not an off-device backup or a
copy of Git history. Retain the source repository and its Git history/config
for rollback. Do not copy Git remotes into the converted signet: initialize and
configure its private Git repository through explicit existing operations later.
