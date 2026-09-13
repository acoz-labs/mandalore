# Memory-engine extraction

Source: public My Friday commit
`f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8`; MIT attribution is retained in
[NOTICE](../NOTICE) and [LICENSE](../LICENSE). No predecessor Git history or
private-agent configuration was imported.

| Source component | Selected implementation | Deliberately excluded |
| --- | --- | --- |
| `portable/store.go` | JSON publication, locks, device/source records, schema loading, traversal | Assistant creation/identity, capabilities, sync configuration |
| `portable/memory.go` | Revision graph, scopes, history, effective heads, lexical ranking | Assistant-wide namespace; replaced explicitly with signet scope |
| `portable/bank.go` | Sourced writes, journal reading, root validation | Legacy bank identity projection and mixed writer support |
| `portable/events.go` | Semantic journal record/write | Capability execution/checks |
| `portable/changes.go` | Authorship validation concept | Git source-change ledger and checkpoint observers |
| `portable/rename_*` | Exclusive publication on Darwin/Linux | Claims of native tests on untested platforms |
| `memorybank/service.go`, `pages.go` | Validated writes, compact recall, page/byte budgets | Sync method and portable-package dependency |
| Selected memory/retrieval/service tests | Graph/history/scopes/budgets/correction fixtures | Git-sync/native installation fixtures belonging to later issues |

The module has two direct dependencies: JSON Schema validation and `x/sys` for
exclusive native rename. `x/text` remains an indirect schema dependency at the
pinned source version. TUI, MCP, OAuth, provider and assistant dependencies were
not copied. `go mod tidy` generates the dependency closure and checksums.

## Deliberate changes from the characterized source

- New manifest and scope namespace; explicit legacy refusal, not a compatibility
  claim. No real memory has been rewritten.
- Local state is optional on reads, supporting fresh clones without creating files.
- Referenced sources receive semantic validation, not just ID existence checks.
- Full validation also checks orphan sources/devices and canonical revision paths.
- Nested managed symlinks/nonregular entries are refused before writes.
- Signet-wide records must identify the selected signet.
- Data-only foundling registration and original/incorporation provenance contract.
- Writes larger than the reader's 4 MiB limit are refused before publication.
- Both sourced documents are encoded/size-checked before either is published;
  invalid input leaves no evidence, unlike a genuine partial I/O failure.
- Journal order compares actual instants, including timezone offsets, not strings.

## Evidence and outstanding work

Failing-first tests established missing engine symbols, then identified concrete
fresh-clone, corrupt-evidence, journal-symlink, foreign-scope, orphan-validation
and canonical-path failures. Fixes preserve the original graph/ranking tests.
Current native checks use synthetic data and the designated testing terminal.
Race tests, vet and four OS/architecture library builds are automated in `bin/ci`.

The engine regression suite also exercises bounded/continued pages, invalid
registration graphs and versions, immutable publication failures, and a real
permission-denied second-file write. The latter test runs unprivileged and
explicitly skips under root; no skipped test is native permission-failure evidence.
The partial-write test verifies valid orphan evidence, unchanged prior memory,
and no dangling revision. These checks do not simulate every filesystem failure
or a power loss.

This is the library outcome planned in #17, not end-to-end completion of #2/#3.
Engineering self-review/reconciliation and exact-head CI evidence live in #20.
Bindings,
Git synchronization, CLI/MCP, reference retrieval/promotion, native integrations,
real two-machine tests and immutable candidate acceptance belong to later slices.
Native engine checks now run on local macOS/arm64 and hosted Linux/amd64.
Missing Actions runs recovered after fresh/reopened PR events; historical cause
remains separately tracked in #19. No public release or production migration has
occurred.
