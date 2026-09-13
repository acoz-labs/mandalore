# Transition from My Friday

## Strategy and sources

Create a new public repository with fresh, privacy-reviewed history. Preserve
the predecessor repository, all branches, issues, PRs and releases as a read-only
archive after successor links exist. Do not rename the old repo, erase history,
merge outstanding PRs, or redirect old release assets to a different product.

Managed template basis: `feeb93836b967d6c100e58b4384fc067e73a0ac9`.
Only generic template content is included, not private org configuration.

Public code-port basis:
[`f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8`](https://github.com/acoz-labs/my-friday/tree/f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8).
Later documentation checkpoint:
[`b67b169`](https://github.com/acoz-labs/my-friday/commit/b67b169).
The feature branch holds newer memory code than main; preserve it during archive.

## Selective extraction

Carry forward reviewed memory/journal/provenance/supersession/recall/sync
primitives and their tests, local bindings, typed CLI/MCP operations, memory-only
Codex integration, pinned-runtime connection recovery and menu patterns.
Preserve upstream MIT attribution for any code actually copied.

Do not bulk-copy predecessor history, docs, private-agent narratives, workstation
paths, account rules, capabilities or raw evidence logs. Inspect dependency
closure for memory primitives co-located in `internal/portable` before extraction.

## Existing issues

| Predecessor issues | Disposition |
| --- | --- |
| #94 governed memory | Relevant semantics move into the memory-only engine outcome |
| #117 Pi; #118 Claude Code | Sanitized successor issues, preserving sequence |
| #95/#102 acceptance | Preserve evidence quality, not old cohort requirements |
| #111 timeout; #112 container tooling | Carry lessons into bounded-execution and reproducible-CI tests |
| #51/#74/#83 capability foundations | Historical assistant work, not a Mandalore dependency |
| #81/#92/#93 assistant kernel | Superseded by the memory-only product boundary |
| #99/#101/#108/#113 delivery authority | Historical; use the new managed workflow |
| #104/#106 capability routing | Retrieval-efficiency lessons only, no capability framework |
| #116 agent adaptation | Memory integration host readiness only; arbitrary capability porting excluded |

Predecessor open PRs remain historical proposals, not accepted work. The new
backlog owns current scope. Archive notices must not mark unfinished work as
successfully completed.

## Compatibility and safety

The extracted writer creates only `signet.json` format 1 and refuses any root
containing `bank.json` or `agent.json`, including mixed-format roots. The bank-wide
scope is `signet`; local locks use `.mandalore`. Replacing a bound manifest's ID
invalidates the open store handle. Existing record/revision/source/device IDs
remain opaque. No conversion or dual-writer compatibility is implemented yet;
#8 must provide explicit migration with preservation and rollback evidence.

Existing My Friday installations and private agents remain untouched. Mandalore
uses a separate namespace and explicit signet binding. No launcher replacement,
native-profile mutation, real-memory migration, credential copying or old-cache
deletion is implied by repository setup.

Before importing a memory-only bank: define supported schemas, inspect and back
up, preserve identifiers/provenance/history, test explicit migration/refusal and
rollback. Display-name changes are not a reason to rewrite data IDs or every
internal field. Legacy assistant repositories are not silently treated as signets.

A native plugin switch must avoid two active memory writers. Preview the exact
change and require a fresh-session handoff; never take over unrelated marketplaces.

## Evidence boundary

Prototype synthetic/native tests and macOS user trials are prior evidence only.
Rerun ported tests and reaccept the actual new artifacts. Linux cross-compilation
is not Linux runtime acceptance. Publish sanitized summaries and synthetic
fixtures, not user sessions or machine inventories.
