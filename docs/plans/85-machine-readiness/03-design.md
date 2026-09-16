# Technical Design

## Component And Behavior Flow

`connection assess` and the menu call the same `connection_assess` administration
operation. It resolves bounded local defaults, loads reviewed declarations,
observes selected metadata and classifies each component. Optional prompt
construction uses that fresh report. Native checks remain a separate invocation.

Keep assessment types, declarations, evidence matching, bounded reads and prompt
construction together in `internal/readiness`. Reuse install receipt validation
through a narrow static projection, not Doctor/Prepare/Apply/native inventory.
Reuse binding/memory metadata types and pure field-validation rules, not full
store validation. No new dependency, subprocess runner, network client or service.

## State And Data Model

### Declarations

Embed reviewed format-1 JSON, strict-decoded within 64 KiB and at most 32 evidence
records. Component IDs: `memory-runtime`, `git-sync`, `codex`, `pi`. Declare:

- intended Darwin/Linux amd64/arm64 targets, readable/writable signet formats
  and required memory/native protocols;
- Git features merge-tree --write-tree and commit-tree for synchronization;
- selected native executable, plus Node >=22.19.0 for Pi and the current exact
  implemented Pi native contract 0.85.1;
- evidence ID, component, scenario, platform, exact artifact/package identities,
  applicable native/interpreter identities and versions, verification level
  (engineering or accepted release) and immutable public source pointer.

Development Go/Node pins are not standalone runtime requirements. Codex 0.154.0
is a tested baseline, not an invented minimum/range guarantee. Malformed built-in
declarations fail rather than silently yielding an empty catalog.
Tests bind declaration values to existing package/protocol/native-version
constants so dependency changes cannot silently leave a stale support declaration.

Seed the four 1.0.0 hashes in `docs/releases/1.0.0/public-verification.json` with
their exact native CLI/MCP scenario links, narrower Codex native evidence and
Pi's final implementation receipts from the discovery. Publication proves byte
identity, not native behavior; retain both kinds of reference. Pi is not accepted.

### Report and states

Format-1 report fields: `schema_version`, `complete`, `harness`, `selection`,
`toolkit`, optional `retained`, `declarations`, `components`, `untested`,
`next_action`, optional `prompt`, and `notice`. Selection includes resolved
operator-visible paths and their source. No actor/device labels, signet names or
memory content are returned. Report no aggregate healthy boolean.

| Dimension | States and meaning |
| --- | --- |
| Support | supported / unsupported / unknown contract target |
| Setup | present / verified-static / missing / inconsistent / unknown |
| Evidence | verified / historical / none, qualified by scenario and required identities |
| Completeness | All fixed observations completed; not a readiness verdict |

A verified-static check must name its scope: binding/manifest identity agrees,
or retained bytes match the receipt. It does not verify the record graph, complete
package/cache, active process or provider. Receipt values remain declarations
until measured. Historical/stale evidence is not proof of broken software.

No durable report, database, cache, consent file, lock or signet revision. Shell
redirection is the caller's choice. Filesystem observations are not a readiness
lease against concurrent changes.

### Identity and matching

The CLI entrypoint supplies its existing linker-stamped version/source through a
typed context value. Missing internal context stays unknown; JSON cannot supply
trusted build identity. Running platform and embedded Codex/Pi identities come
from this process. Preserve the old version response, linker stamps and release
manifest/inventory.

Hash the path returned by os.Executable separately as **on-disk**, not attestation
of the loaded image: the file can be replaced during a running process. Evidence
matches measured artifact/scenario, not live execution. Inconsistent known process
source/package identity must prevent calling a disk match running-toolkit evidence.

An explicitly selected retained runtime has a separate measured digest; never
execute it or assign it the current toolkit's version/package. Its receipt can
declare those values but does not establish a currently loaded package identity.

Match every identity required by a record. Missing values are not wildcards.
Exact runtime bytes/platform can match a recorded CLI/MCP artifact scenario.
Native evidence additionally needs its applicable native/interpreter/package
identities. When static observation cannot establish them, show historical
evidence and the missing match. Script/shim bytes do not prove interpreter or
target version. Catalog-known artifact platforms differing from the process are
incompatible selections, not applicable host evidence. No script evaluation,
package traversal or version command fills that gap. A mismatch is historical,
not broken; no evidence is not unsupported.

Do not embed the executable's own final hash into itself. Post-build evidence is
a separate receipt that a later reviewed catalog may describe. Tests supply
synthetic catalogs to the pure matcher; production accepts only bundled records.

## Interfaces And Contracts

### Shared operation and CLI

`connection_assess` is CLI-only, read-only, non-network, idempotent and requires
no bound memory service. Preserve ordinary memory MCP/native tool schemas.

Input: required `harness` (codex or pi); optional `state_dir`, `native_home`,
`native_binary`, `binding`, `connection_root`; optional `include_prompt`.
Every supplied path is absolute, control-free and <=4096 bytes. No caller-supplied
platform, trusted version, evidence, script, raw prompt or previous report.

```sh
mandalore connection assess --harness pi
mandalore connection assess --harness codex --binding /example/binding.json --connection-root /example/owned-generation --prompt
mandalore call connection_assess < assessment-input.json
```

--prompt includes guidance in the JSON report; it does not execute or replace the
report with free text. --read-only is accepted and changes no learning policy.

Resolve defaults inside the shared assessor: platform installation state; native
profile environment variable or ordinary native home; MANDALORE_BINDING or platform
binding; executable through non-executing PATH lookup. Missing executable is a
finding, not an aborted report. No cwd-based memory selection. Invalid/unresolvable
default paths give bounded selection errors; explicit valid paths remain usable.

No automatic connection_root default, generation scan or newest-root selection.
An omitted root means unselected, not absent active registration. Validate any
selected root as an owned generation for this harness/state/profile before following
its retained-runtime path. Compare receipt binding/native paths with operator
selection; do not follow another binding or switch profiles. Preserve unmanaged
sources and report the limit rather than adopting them.

### Fixed observations

1. Process platform, stamped identity and embedded package metadata; separately
   bounded executable-on-disk digest. Physical CPU/translation stay unknown.
2. Native executable and Git PATH presence/type/executable permission; Node presence
   for Pi. Native version, Git features and wrapper target remain untested.
3. Selected native-profile and state directory presence/type only. No native
   settings/authentication/session/cache/plugin scan.
4. Binding JSON and only its referenced manifest/device metadata: schema, root,
   ID/enrollment consistency and legacy-marker presence checks. No records, events,
   sources, foundlings or Git configuration. Reuse pure validation rules without
   Store.Validate, memory tools or a content scan.
5. Optional owned receipt schema/path/declared identities, retained-runtime hash,
   selected native hash and selected binding hash. Receipt package hashes remain
   declared; full package/cache and active native registration need deeper checks.

The absence of a registration is not inferred from an unselected root. Consistency
with a receipt is not publisher trust or authority to execute a discovered path.

### Bounded reads and errors

Keep 32 KiB input and 64 KiB envelope limits; at most 64 findings, 32 evidence
records and 8 KiB generated prompt. Binding <=16 KiB; manifest/device retain the
existing 4 MiB metadata ceiling with limit+1 reads; Codex/Pi receipt bounds stay
32/64 KiB. Runtime/native hashing retains 128/512 MiB limits, streams in bounded
chunks and checks cancellation between reads. Never walk a directory recursively.

Resolve selected executable symlinks for ordinary launcher use, show the resolved
target and inspect only a regular target. Refuse redirected managed metadata and
special files. Use open/fstat and nonblocking/no-follow where available to avoid
opening a substituted FIFO. Compare identity/size/mtime before/after reads; changes
yield unknown/inconsistent observations. No orphan cancellation goroutine remains.
This is not a sandbox against a hostile local user or a hard deadline for blocked
kernel/network-filesystem operations.

Malformed input uses input.invalid; invalid built-in declarations use bounded
readiness.invalid. Missing setup is a successful report with missing findings.
Unreadable/malformed selected metadata yields findings and complete:false while
unrelated observations remain. API cancellation preserves operation.cancelled,
exit 130 and no success report/prompt. No mutation or automatic retry on failure.
If the assembled response exceeds the API envelope limit, return output.invalid;
never silently truncate findings/evidence or call that a complete assessment.

## Authorization And Data Exposure

Operator authority covers selected local setup metadata, not remembered content,
credentials, native sessions or providers. Paths/artifact identities may appear
in local output; labels, raw file content and raw errors do not. Public evidence
uses synthetic fixture paths. Known metadata reads do not authorize enrollment.

Generate guidance from fixed finding/action codes and public declaration labels,
never interpolated private paths, receipt strings, raw errors or arbitrary content.
Include completeness and historical/live distinctions. Ask the receiving agent
to confirm the intended harness/binding, inspect fresh state and use its ordinary
authorization before installs, credentials, sync or configuration changes. Do not
alter ordinary learning policy or interpret this prompt as an instruction to repair.

Native checks remain separately selected, with executable/profile and potential
log/cache effects disclosed. Preserve their existing untested boundaries. Neither
report nor prompt starts repair, sync, login or an agent. CLI automation retains
its existing explicit native-inspection command as the execution choice.

## Failure, Recovery, And Observability

One fixed-priority next step: correct invalid/inconsistent selection; explain an
unsupported contract without forced installation; select missing dependencies;
connect/inspect binding; then explicitly verify native behavior. Unknown/historical
evidence alone requests no repair/upgrade. Partial reports say what is unknown.

The menu requests include_prompt for its report but displays guidance only after
the Show prompt choice. It remains a labeled assessment-time snapshot and asks
the receiving agent to recheck. There is no untrusted report-to-prompt API.

The menu implements the reviewed product flow with Back/Details/Prompt/native-check
choices. Later native results do not rewrite the earlier static snapshot. Rerun
assessment after separate authorized changes. No durable assessment state needs
rollback, and ordinary memory use is never gated by it.

## Design Traceability

| Acceptance group | Implementation boundary | Verification/recovery |
| --- | --- | --- |
| Declarations and evidence | Versioned catalog, exact matcher, independent dimensions | Corruption fails; incomplete identity never matches |
| No-execution default | Fixed reads, no runner/client/service | Trap commands/network, inventories, denied-content traps |
| Missing/stale/wrappers | Shared defaults, explicit retained selection | Typed partial findings; no active-registration inference |
| Toolkit/retained identity | Trusted process context, separately measured disk bytes | Replacement and same-label/different-bytes tests |
| Malformed/unsafe/cancelled | Strict schema, bounded regular reads | Size/FIFO/symlink/permission/race/cancel cases |
| UI/agent parity | One operation and existing console | CLI/menu assertions plus actual recordings |
| Optional scoped prompt | Fixed codes, 8 KiB output | Injection/private-content/no-launch tests |
| Explicit deeper check | Existing doctor behind separate choice | Default-No/cancel/no-dispatch and native fixture handoff |

No signet/connection format migration, release-manifest change, authentication
change or active-session rewrite is included.
