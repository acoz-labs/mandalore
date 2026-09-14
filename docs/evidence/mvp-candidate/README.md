# Retained-candidate native verification — first pass

Classification: contributor verification, not independent acceptance. The complete
MVP is **not yet accepted or released**. This evidence-only branch preserves the
tested product commit; it does not rebuild or modify its retained payloads.

## Candidate and environment

- Source: `0fe7e0e1eb175943995b0d0d5e54130ff65b074e` (merged #36).
- Manifest: `829be4285324e00ab6c2fdde482c1471df13c74f8facab0e6fdcdba0ed3c09df`.
- [Hosted build](https://github.com/acoz-labs/mandalore/actions/runs/34836436353),
  retained artifact `10344107712`, expiry 2026-12-13T11:05:51Z.
- Raw ZIP: `7338f85bd305cc34a20ed0f57315226b6f4ae8f5a4ab4594c44ed135b1aead7d`.
- [Verified nomination for #2–#9/#11/#12](https://github.com/acoz-labs/mandalore/actions/runs/34837370613),
  retained verification receipt `10344932288`.
- Actual macOS ARM64 CLI: `b959ccc66d518c151a2efc1111bf2739736b3980308a4ee43889bd99b151b819`.
- Embedded plugin: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- Codex 0.153.4, native binary SHA-256
  `b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`;
  observed model `gpt-6-astra`, high effort, ordinary existing authentication and
  inherited native resources. The offered CLI update was skipped to preserve
  the test version. No model or credential configuration was changed.

The designated native testing pane ran candidate verification, CLI installation,
native activation and both conversations. A fresh synthetic signet and local bare
Git remote were used; these are not a physical second machine or off-device backup.
The old synthetic bank was fingerprinted and remained unchanged. Its connection
generation and runtime remain retained.

The actual candidate's CLI prepared/applied its own connection. Doctor reported
12 structural passes and five appropriately untested live boundaries. Subsequent
native use separately verified login, MCP and hook behavior; it does not make
doctor itself a live acceptance test. The two Mandalore hooks (SessionStart and
UserPromptSubmit) were inspected as active/trusted in `/hooks`; the unrelated
existing native hook remained active. No blanket hook-trust bypass, auth copying,
global shell edits or hand-edited marketplace was used. The managed connection
retains its deterministic generation cachebuster; candidate plugin bytes were
not rewritten for testing. Installed-plugin validation passed using PyYAML 6.0.2
in an isolated validation environment.

## Observed scenarios

| Scenario | Actual result |
| --- | --- |
| Download and install | Raw Actions archive verified against fresh provenance and every payload hash; native version matched source/manifest. Read-only preview created no destination. Explicit isolated CLI apply completed; identical-plan replay was already-current with unchanged file inventory/bytes. |
| Ordinary learning | A new fictional project, Lantern Otter, and two-short-sentence status preference were saved without a remember phrase. Scope discovery found no stored scopes, then the agent deliberately created `project/lantern-otter`. |
| Current-user correction | Rename to Maple Lynx retained the record and scope, explicitly superseded its predecessor, and kept the formatting preference. No competing new project identity was invented. |
| Explicit consolidation | Direct “This is the way” added decision-then-one-brief-reason formatting and one concise semantic journal entry. The two earlier turns saved records but did **not** journal; this is not evidence of journaling every learning turn. |
| Git delivery | Each learning turn requested short-budget sync; committed local and bare-remote heads matched. Delivery was to a local Git remote, not a claim of independent machine continuity. |
| Actual compaction | Native `/compact` completed in about 43 seconds. The next read-only turn made one scoped memory recall and returned the correct name and both preferences. No Mandalore PostCompact hook is claimed. |
| Fresh session and cwd | A second session in a different empty project recalled Maple Lynx and both formats with one scoped recall. It did not reuse the prior conversation or guess a new scope from cwd/display name. |
| Existing-session external update | Without closing the second native session or reinstalling its connection, an explicit fixture command used a separate candidate CLI process/binding to record Cedar Wren as another superseding rename. Its output exposed only command completion, not the new name. The following fresh MCP read immediately returned Cedar Wren and retained formats. MCP process-ID continuity was not separately captured, so this is existing-session freshness evidence, not an assertion that no automatic server restart occurred. |
| Interruption | A single synthetic `sleep 45` was started, then Escape interrupted the agent turn. The native background command remained alive afterward and later finished. The next read-only turn neither resumed/retried it nor saved/journaled/synchronized, and correctly recalled Cedar Wren. Agent-turn interruption is not proof of subprocess cancellation. |
| Preservation and status | All signet file paths/bytes, including local/Git files, matched across post-compaction, cold-read, external-read and post-interruption read-only windows and exits. Both project directories stayed empty. The older bank remained unchanged. Final status correctly reported `pending`/dirty after the intentionally unsynchronized external write, despite retaining the earlier successful delivery receipt. |

The separate CLI writer's device label and actor are synthetic, not a claim that
a second physical machine participated. Its `external-cli-test` harness origin
is distinct from the native `codex` records. Original authorship is retained
through the three-revision chain. Unknown model/session fields remain null;
observed test-session metadata is not retroactively fabricated as record origin.

## Evidence and context measurements

- [Verified candidate manifest and transport](candidate.json).
- [Whitelisted native call/latency/byte ledger](native-summary.json).
- [Exact synthetic three-revision history](history.json),
  [current recall](current.json), and [semantic journal](journal.json).
- [Final local sync status](sync-status.json), including the explicitly historical
  last-delivery receipt. It is not proof of current remote freshness.
- [Integrity hashes](SHA256SUMS).

Actual recall text envelopes were 1,487 UTF-8 bytes for post-compaction/cold recall
and 1,527 bytes after the external correction. Each of these read-only turns made
one memory recall, with no write/sync operation. These are envelope byte counts,
not token counts or the doubled MCP transport wrapper. Native aggregate counters
include inherited resources, repeated model calls and compaction; they must not
be advertised as isolated Mandalore overhead. User-turn durations include model
work and native tooling, not just storage latency.

Raw local recordings and native rollouts were reviewed but are not published:
they contain native account/workstation context. This sanitized ledger and the
synthetic memory artifacts are not a substitute for fresh independently judged
rendered terminal evidence. No earlier local-candidate recording is relabeled as
this candidate's acceptance.

## Regression follow-up — not a clean full-suite pass

The evidence worktree's full pinned-host `bin/ci` run failed in
`internal/api/TestSyncCancellationExposesPartialWriteEvidence`. Other completed
package results passed, but the failed suite stopped CI before vet/cross-build
completion. Container validation was attempted first; the Docker daemon was
unavailable. Foundation/public-content checks passed before the test failure.

The original assertion printed an opaque error pointer. A private Go test overlay
that changed **only failure reporting** passed ten focused repetitions. A second
controlled overlay added a two-second delay before the wrapper's first Git call,
without changing product code or the test's one-second operation deadline. It
reliably produced [this failure receipt](cancellation-delayed.log): cancellation
in `checkpoint`, with `checkpointed: false`, `write_may_have_occurred: true`,
`inspect_before_retry: true` and intact phase evidence. The test wrongly assumes
that its whole-operation deadline necessarily expires after reaching `fetch`.

This establishes a timing-dependent test assumption, not proof of the opaque
original failure's exact phase or a lost runtime receipt. The follow-up must wait
for an observed fetch-phase marker before testing explicit cancellation, retain
separate deadline/early-phase coverage and improve failure diagnostics. A later
passing rerun alone must not erase the original failure. These overlays neither
modify the candidate nor constitute a merged test fix. No clean current full-suite
or complete-candidate acceptance verdict is claimed.

## Remaining acceptance

First resolve the cancellation-test follow-up above. Still required:
exact-candidate failed-hook and quoted-trigger negatives;
foundling registration/stale/missing/promotion/source-preservation scenarios;
ambiguous-write/corruption/offline/concurrent recovery; full rendered
install/update/repair matrix; second physical machine and honest native
OS/architecture coverage; independent authorized product acceptance; and explicit
publication authority with verified release prerequisites. Cross-builds and
source-unit tests do not replace those observations. See
[the verification protocol](https://github.com/acoz-labs/mandalore/issues/10#issuecomment-5663140800).

The evidence branch changes no runtime, plugin, schema or release-control code.
If testing finds a product defect, use a reviewed fix and nominate a new immutable
candidate; do not patch or rebuild accepted bytes in place.
