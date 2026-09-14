# Transition from My Friday

## Strategy and sources

Transition completed: the predecessor repository is archived and its planning
board is closed. A [pinned transition notice](https://github.com/acoz-labs/my-friday/issues/119)
links to Mandalore. The preserved feature-branch checkpoint remains
`b67b169cd6e2de1681f48470400d7b44f73fecc1`; no branch or release was removed.
This completes source/project retirement, not a runtime or memory migration.

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
remain opaque. The explicit converter below preserves a legacy bank ID, including
its `bank-` prefix. It does not provide dual-writer compatibility.

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

## Explicit memory-only conversion

`migration preflight` and `migration apply` support the pinned public My Friday
`bank.json` schema version 1 only. They do not run My Friday. An `agent.json`
repository, mixed root, arbitrary Markdown tree, unknown field/schema, invalid
graph, unsupported path, symlink or oversized snapshot is refused. Historical
assistant repositories and other resources belong to foundlings (#12), not a
manifest rename or automatic promotion into current guidance.

| Surface | Legacy | Mandalore |
| --- | --- | --- |
| Portable manifest | `bank.json` version 1 | `signet.json` version 1; same ID/name |
| Bank-wide revision scope | `assistant`, bank ID | `signet`, same bank ID |
| Other revision scopes | `project`, `account`, `task` | Unchanged |
| Writer lock | `.my-friday/write.lock` | `.mandalore/write.lock`; created by a later writer |
| Local binding | `bank_id`, root, device, actor | New explicit binding with `signet_id` and newly enrolled device |
| Binding selection | `MY_FRIDAY_MEMORY_BINDING` or old platform config | `MANDALORE_BINDING` or new platform config; no implicit inheritance |
| Native integration | `my-friday-memory` / `my-friday` | `mandalore`; installation is a separate opt-in operation |

Choose absolute paths. The source and output must not overlap. The output must
not exist, even as an empty directory, and its parent must already exist. The
optional legacy binding stays outside both. Neither hostname nor provider account
is inferred for conversion attribution.

```sh
mandalore migration preflight --source /example/old-bank --output /example/migration-bundle --device-label "Migration laptop" --actor Example > preflight.json
```

Preflight writes no source, output, lock or temporary files. The shell redirection
above saves its report where you selected. It validates a bounded in-memory
snapshot using the engine's revision graph and provenance rules. It inspects an
existing legacy lock without creating one, reads the snapshot twice, and reports
source digest, counts, conflicting records and exclusions. A held exclusive lock
blocks the operation. An absent or available lock does **not** prove all sessions
or other machines are idle. Conflicted-record counts include all stored heads,
including future-effective branches; no winner is selected.

Optionally add `--legacy-binding /example/local/old-binding.json` to verify its
source ID/path and original device. Optionally add both `--native-home DIR` and
`--native-binary FILE` to invoke that trusted CLI's bounded plugin listing commands.
Enabled old/new memory integrations are **potential** writers, not evidence of
active sessions or same-bank targeting. Without both flags, native inventory is
not tested. Mandalore does not edit native state; the selected CLI may write its
own logs/caches. It does not inspect authentication or transcripts.

Stop all relevant writers on all machines, review the plan and retain the original
repository and any needed off-device backup. Then explicitly acknowledge this:

```sh
mandalore migration apply --writers-stopped < preflight.json
```

Apply revalidates the exact plan and holds the existing legacy shared lock when
one exists. It refuses changed source, binding, lock observations or native
inventory; obtain a new preflight rather than editing a plan to hide a change.
This cannot stop arbitrary noncooperating writers or a remote machine. The writer
acknowledgement remains necessary even when local observations look clear.

The new bundle contains:

- `original/`: byte-for-byte portable files and their empty directories, including
  README, ignore rules and `provenance/changes` Git-index observations.
- `signet/`: the converted memory, sources, devices and journal. Only the manifest
  filename and bank-wide structural scope kind change. Original IDs, timestamps,
  actor/device/model/session values, evidence, supersession and extension values
  remain intact; untouched JSON values are not round-tripped through floats.
  The normal history/recall reader also retains exact opaque JSON numbers.
- `migration.json`: a versioned local receipt containing selected paths, original
  and converted hashes, preserved counts and separate conversion attribution.

The converted signet receives necessary directories/ignore rules, one new explicitly
labeled conversion device and one migration journal entry. These identify the
conversion without rewriting original authorship. The entry records content identity
and counts, not local paths. A later binding enrolls its own new writer device.
README and Git-index observations remain historical evidence in `original/`, not
new assistant instructions or current memory. Conversion is not endorsement of
historical guidance over current user direction.

`.git`, `.my-friday`, OS metadata and the external binding are excluded. The snapshot
is not a Git clone, credential transfer, archive of old native sessions or off-device
backup. No Git hooks/filters/config/remotes or private runtime code are copied.
Retain the untouched source and its Git history for recovery.

Limits are 4 MiB per portable file, 64 MiB of portable input, 10,000 files,
10,000 directories, 30,000 walked entries and 64 reported exclusions. Paths and
native inventory have separate bounds; the plan is limited to 24 KiB so it fits
the 32 KiB administration input envelope. Limits cause explicit refusal, never
truncation or partial semantic import. Conversion also refuses a transformed
revision exceeding the engine's file limit. Stricter current validation may
refuse an otherwise legacy-shaped source; no repair or loss of history is implied.

## Publication, handoff and recovery

Apply writes a new owned sibling staging directory, syncs files/directories,
validates the on-disk signet, rechecks the source and publishes the whole bundle
without replacing any existing path. A local receipt says prepared/validated;
it does not by itself prove publication. The result reports `published` and
`phase`; a retained staging receipt is not a completed conversion.

An error may leave a staging path, or a published bundle if the final parent sync
failed. The error envelope retains this result and `write_may_have_occurred`.
Inspect it before retrying; never delete the source or blindly rerun a mutation.
There is no automatic cleanup or reverse conversion.

After inspecting the converted history, explicitly create a new binding with
`signet bind`. Configure private Git initialization/remotes and synchronization
through the existing separate operations. Preview the native connection and
review the removal/disablement of a known old writer separately. Start a fresh
session and verify recall and new-write provenance against the intended signet.
This converter never switches plugins, binds, syncs or removes old installations.

Before activation, rollback means continuing with the untouched original source
and writer. After new writes, retain both histories and reconcile deliberately;
switching back to a stale source can lose continuity. Archiving a GitHub project
does not stop running native sessions or make a local snapshot an adequate backup.

## Evidence boundary

See [migration engineering evidence](migration-engineering-evidence.md) for the
tested implementation, synthetic cases, corrections and untested boundaries.

Prototype synthetic/native tests and macOS user trials are prior evidence only.
Rerun ported tests and reaccept the actual new artifacts. Linux cross-compilation
is not Linux runtime acceptance. Publish sanitized summaries and synthetic
fixtures, not user sessions or machine inventories.
