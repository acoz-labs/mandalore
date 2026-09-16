# Machine-readiness implementation reconciliation

Issue #85 / PR #87, selected O1 from discovery #15 / PR #84. Engineering
self-review follows ADR 0003, not independent product acceptance. This map is
part of the implementation; final review binds its then-current exact head in
PR #87. Source-specific recordings are not relabeled as later-head evidence.

## Approved basis and scope

Discovery reviewed head:
[`12418fc7dd9f75a565d0ef2b3fb921d11b646088`](https://github.com/acoz-labs/mandalore/blob/12418fc7dd9f75a565d0ef2b3fb921d11b646088/docs/discovery/15-machine-readiness/README.md).
Product/solution planning reviewed head:
[`0f3a1077e77697fbdafefc1e27fc31d3cade5d65`](https://github.com/acoz-labs/mandalore/tree/0f3a1077e77697fbdafefc1e27fc31d3cade5d65/docs/plans/85-machine-readiness),
merged by PR #86 as `3332938212cc73ee05d03d57e5179d13705a0447`.

The implementation is a non-executing memory-integration assessment shared by
CLI/menu, reviewed support/evidence declarations and optional bounded guidance.
It is not a capability scanner, package manager, background service, live
authentication test, automatic repair or installation. Published v1.0.0 and
personal installations remain unchanged.

## Requirement and evidence map

| Contract | Implementation and verification |
| --- | --- |
| Support/dependencies/immutable evidence remain separate | Strict format-1 `internal/readiness/catalog.json`; catalog tests bind published hashes, native/package requirements and protocol declarations. Pi evidence remains engineering-only. |
| Exact, historical, unknown and unsupported classifications | `evidence.go` compares every scenario identity; missing values are not wildcards. Controlled mutation fails the matcher test. `artifactSupport` distinguishes known foreign targets from unknown bytes; actual incompatible-artifact recording does not execute it. |
| No default execution/network/product writes/content reads | Assessor has fixed metadata/fingerprint reads, no runner/client/service. Reader allowlists exclude remembered content; compiled native/dependency traps and configured-network instrumentation plus complete fixture inventories remain unchanged. This is not a hostile-user or system-wide network sandbox. |
| Useful missing/stale/wrapper/dependency observations | Shared bounded defaults, explicit retained-root selection and typed missing/partial results. CLI parity, relative-PATH, stale binding/native, mode, ownership and absent-root tests; actual missing/configured/stale menu scenarios. Wrapper target/version remains untested. |
| Process and retained identities are distinct | Trusted internal stamps/embedded package, separate on-disk observation and owned retained fingerprint. Replacement pipeline test preserves process metadata while measuring replaced bytes; pure matcher rejects source-only or conflicting-source certification. Native loader behavior is not attested. |
| Malformed/oversized/unsafe/permission/cancel handling | Bounded no-follow regular-file reads, streaming hash limits, duplicate/schema checks, FIFO/symlink/permissions/race/cancellation tests. Invalid input and observed partial state are distinct. Cancelled API returns no success report; menu pre-cancel regression renders nothing. |
| Shared administration with unchanged memory tools | CLI-only/unbound `connection_assess`; typed/human reports compare equal for Codex/Pi. Menu delegates to it and uses the same selection resolver. Catalog/MCP visibility tests preserve bound memory-tool schemas. |
| Sanitized optional follow-up | Fixed-text typed findings/actions only, no private path/raw content interpolation or agent launch. Prompt is explicitly viewed; native malformed-binding recording keeps its raw content out. Guidance grants no install/repair/credential/sync authority. |
| Explicit deeper inspection | Both menu entry and report handoff disclose selected program/profile and native log/cache effects, default No. Both actual native handoffs pass structural checks with live boundaries untested; changed identity fails without retry/repair or replacing the snapshot. |
| No implicit setup or personal activation | Missing fixtures remain absent; configured static inventories are unchanged. Disposable native setup is a separate test preparation step. No live signet, credential copying, provider request, sync or release occurs. |

## Rendered review

Classification: `new-or-materially-changed-experience`. Product design was
reviewed in PR #86 before solution implementation. The existing console and
keyboard conventions remain; no new theme, browser/mobile surface or pixel
baseline is introduced.

The [first menu checkpoint](menu/README.md) covers actual missing setup, details,
prompt/default-No, arrows/Vim/Back, normal/narrow/color/no-color/plain and
EOF/Ctrl+C. It retains the initial status-word splitting finding and corrected
retest separately. The [configured checkpoint](menu/configured-d2ce027/README.md)
covers consistent owned metadata, unknown/historical evidence, both native
handoffs, stale native failure, invalid-path correction, malformed partial state
and an unsupported known retained artifact. Full static fixture inventories
remain unchanged; separately approved native checks change only their native
directory bookkeeping in the observed runs. Terminal-mode checks disclose the
Darwin PENDIN exclusion. Screen readers, other locales/fonts and non-host native
platforms remain unverified.

Final implementation-head rendered repeats and an immutable evidence manifest
remain required before leaving draft. Use a separate retained evidence branch,
as in the Pi/quota deliveries, so publishing recordings does not alter the tested
implementation head. Product acceptance must later repeat the nominated candidate
matrix; contributor self-review and engineering recordings do not supply it.

## Refinements and findings

Ordinary in-scope refinements: expose a private executable-path seam for
replacement tests; share the selection resolver with menu preview; stack status
fields at narrow widths. No public fake-identity/platform/evidence inputs or new
execution/data authority were added. Failing-first checks resolved cancelled
summary rendering, Codex's incorrectly listed Node requirement, relative-PATH
precedence, dangling metadata redirects, retargeted executable links, native mode
changes and missing-profile next-step order. Evidence retains the actual failures
and each correction's scope rather than treating passing retries as an explanation.

## Documentation promotion

| Temporary material | Durable destination |
| --- | --- |
| #15 discovery and #85 context/product flow | `docs/setup.md`, this approved-basis/scope map and source-bound evidence |
| #85 decision/technical selection and side effects | `docs/architecture.md`, `docs/interface.md`, `docs/runbook.md` |
| Schemas, limits, identities and prompt semantics | `docs/interface.md`, `docs/architecture.md`, reviewed catalog and tests |
| Verification, findings and native observations | `docs/development.md`, `docs/evidence/machine-readiness/` |
| Conversational administration | Existing Codex/Pi Armorer entrypoints and administration references; ordinary memory skills unchanged |
| Handoff/recovery/release boundaries | `docs/setup.md`, `docs/runbook.md`, this reconciliation and PR #87 |

The temporary six-file #85 plan and #15 discovery are removed from the
implementation tree after this promotion. Their reviewed history remains linked above;
no durable requirements depend on those mutable temporary paths.

## Validation and remaining gates

Pinned host CI passed on the documented source checkpoints: 22 Pi adapter tests,
all Go race suites, vet, privacy checks and four target builds. Docker was
unavailable; the documented host fallback was used. Both packaged skill validators
and Codex plugin validation passed. Exact-final-head full/hosted CI and a separate
recorded engineering self-review remain PR-readiness prerequisites.

Keep #85 open after any reviewed implementation merge for exact-candidate
acceptance/release. Pi implementation is not Pi acceptance, and Claude still
follows that acceptance. No public release or real-memory activation is authorized
by this reconciliation. The complete post-1.0 roadmap remains broader than #85.
