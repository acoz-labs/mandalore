# Native retained Linux AMD64 execution

Contributor engineering evidence, not independent acceptance or a native agent
conversation. The exact Linux executable was downloaded from the already-retained
candidate and executed on the repository's existing public Ubuntu x86_64 runner.
No candidate was rebuilt, no model was launched and no account was enrolled.

## Identity and reproducible collector

- Candidate source: `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`.
- Manifest: `c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
- Linux AMD64 executable: `983297e20d0faa919c76d5cd8b124e5b2b5202082a6d14870532ee62b2c75c89`.
- Collector head: [`86dee8814bf82cae77b34cde70d2a9afc378bd50`](https://github.com/acoz-labs/mandalore/tree/86dee8814bf82cae77b34cde70d2a9afc378bd50/verification/retained-candidate).
- Verification-only [PR #40](https://github.com/acoz-labs/mandalore/pull/40), not an implementation merge or nomination.
- Successful [run `34849801016`](https://github.com/acoz-labs/mandalore/actions/runs/34849801016), attempt 1; PR merge checkout `7786d3ff2b449ad28158717a752fa0fecf58900b` is collector context, not the tested binary's source.
- Evidence artifact `10349726461`, 5,605 bytes, raw ZIP SHA-256 `f6d223c381bfc505148b6c64ab0b27375b0361d954fbee703148d5ecad63e7b7`, expires 2026-12-13.

The collector uses the trusted transport verifier to refresh successful-run and
artifact provenance, checks the exact complete candidate ZIP and all payload
hashes, then runs the retained Linux executable with an empty credential
environment. Read-only Actions authentication belongs only to transport fetching
and verification. The normal Go source suite runs separately; its compiled test
binaries are not substituted for the retained executable.

The evidence archive was downloaded in the designated testing pane, its live
successful-run metadata and archive digest checked, and its ten-member regular
JSON inventory verified before extraction. The JSON files here are unchanged
artifact bytes, with [per-file hashes](receipt-hashes.json). Administrative
bindings and ephemeral runner paths are excluded; all memory is synthetic.

## Actual results

The [summary](summary.json) and receipts establish:

1. Native version reports the expected source, OS and architecture.
2. Copper Finch is corrected to Silver Heron within one record, retaining the
   exact predecessor and two revisions. Recall selects the new effective name.
3. Actual stdio MCP initializes, exposes 16 tools, recalls the name, rejects a
   journal write with `operation.read_only`, and exits cleanly on EOF. Signet
   file hashes match before/after the read-only process. The tools/list response
   is 44,087 serialized bytes, not a token estimate.
4. Local Git delivery reports synchronized with matching head/remote head;
   fresh-clone recall is byte-identical to original recall. This is a local bare
   endpoint, not a second physical Linux machine or provider authentication test.
5. CLI installation into an owned temporary prefix succeeds, and the resulting
   launcher reports byte-identical version metadata.
6. Explicit synthetic legacy conversion publishes successfully and preserves
   its complete original snapshot. Historical JSON retains the exact large
   decimal `9007199254740993123456789.123456789` without numeric rounding.
7. The unrelated working directory remains empty.

The first [run `34849600239`](https://github.com/acoz-labs/mandalore/actions/runs/34849600239)
completed native probe steps but failed at the evidence privacy scan because the
runner did not provide `rg`. It uploaded no evidence and is not counted as a
completed verification. The collector was corrected to use portable quiet
`grep`; the passing run used a fresh disposable fixture. No product defect or
candidate change was involved, and the original failure remains inspectable.

## Limits

This is native Linux AMD64 CLI/MCP evidence only. Linux ARM64 and macOS AMD64
execution, Linux native Codex conversations, alternate terminal rendering and
independent acceptance remain unverified. Public bootstrap download cannot be
tested successfully before an authorized public release exists. The temporary
collector does not change default runner configuration, release policy or main.
