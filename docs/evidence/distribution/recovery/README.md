# Installer failure and recovery: native engineering evidence

This extends the [guided installer checkpoint](../menu/README.md) for draft PR
#36. Classification: `in-pattern-visual-change` for the recovery instruction,
within the installer’s previously designed experience. These are contributor
self-review results, not independent acceptance or permission to publish.

## Exact source and artifact

- Implementation and selected candidate: `065b481a3dbdf66e731bf1558bc5c5b4302fc0fb`.
- Manifest SHA-256: `e8562e2938ebfefa0aebcad18f2859b804c8b2947d2b4d2887ae99fcab3effd4`.
- Actual menu driver / installed ARM64 binary SHA-256:
  `63f9ab8200fa36e1e8c184b8c699130ffb8838f82b8d7c9625d824ab16dbb4f4`.
- Embedded plugin SHA-256: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- Intended version: `1.0.0`, not a published release or nominated final candidate.
- Native environment: macOS ARM64, Go 1.26.4, designated Herdr pane, English,
  keyboard input, 30 rows and recorded 57/24-column widths.

The clean source was built once. The UI driver is the candidate binary itself,
not a development build. All candidate checksums were checked again after the
tests. Negative artifact tests use separate, deliberately altered copies; the
original candidate remains unchanged. No real signet, credentials, shell profile,
native installation or global network configuration was selected or modified.

## Actual recordings

Unmodified macOS `script -qr` recordings begin inside the child application.
All recorded paths identify synthetic fixtures. See the earlier checkpoint for
[BSD replay instructions and terminal-query caveats](../menu/README.md#replay-and-integrity).

| Recording | Action and result |
| --- | --- |
| [partial.recording](partial.recording) | 57-column color; Down/Enter approves the reviewed plan. A real unwritable fixture `bin` causes launcher creation to fail after retaining the runtime and pending plan. The receipt says incomplete and shows the retry command. |
| [recovery.recording](recovery.recording) | Resolve only the fixture permission problem; the original candidate reads `pending.json` directly through `release apply`. Complete receipt; same plan digest; connection unchanged. |
| [plain-narrow-partial.recording](plain-narrow-partial.recording) | Independent 24-column plain/NO_COLOR fixture; `2`/Enter; same real permission failure. Recovery instructions, paths and effects remain complete. |
| [plain-narrow-recovery.recording](plain-narrow-recovery.recording) | Same-plan recovery of that independent fixture; complete machine-readable receipt. JSON output intentionally remains machine-readable rather than console-formatted. |
| [escape.recording](escape.recording) | 57-column color; Escape at default-No confirmation; stopped without creating a destination. |
| [eof.recording](eof.recording) | 57-column plain/NO_COLOR; Ctrl-D at confirmation; stopped without creating a destination. |
| [offline.recording](offline.recording) | A child-process-only unreachable loopback proxy makes the actual public request fail; no destination created. No live credential or global proxy change. |
| [corrupt.recording](corrupt.recording) | One same-length comment byte changed in a copied bootstrap asset; actual candidate verification rejects its digest before confirmation or destination creation. |
| [unsupported.recording](unsupported.recording) | Copied manifest declares an unsupported CLI target; parser refuses before confirmation or destination creation. This is a synthetic manifest rejection, not native Windows execution. |

For each real partial install, checks outside the recording confirmed: pending
record present; launcher absent; retained binary byte-identical to the candidate.
After restoring only the fixture `bin` permission, direct-stdin recovery cleared
pending state, produced a receipt and activated the exact retained runtime.
Invoking the new launcher reported the expected source/platform/plugin identity.
Reapplying the saved record returned `already_current: true`,
`destination_changed: false`, and `connections: unchanged`.

The permission failure used ordinary filesystem behavior, not a fake API or
production fault-injection switch. Fixture directories retain their identities
when their owner-write permission is restored; apply still checks the actual
pending record, retained bytes, directory identities and old/new ownership state.
The parser does not establish ownership by itself or allow replacement plans.

## Review and limits

Scoped verdict: pass for these journeys. The new instruction closes a real gap:
menu users already had a saved exact plan inside the pending record, but could
not previously pass that record directly to the CLI. The bounded human-CLI input
adapter now accepts it; there is no new memory API or native connection behavior.

Failing-first tests cover the missing instruction and pending parser, malformed
records and digest disagreement; a path containing a quote is safely shell-quoted.
Full pinned-host CI passed, including race tests, vet and target builds.
Container-first execution was attempted but the Docker daemon was unavailable.
[Hosted CI for this source also passed](https://github.com/acoz-labs/mandalore/actions/runs/34833484102).

Visual inspection covered headings, explicit failure/incomplete labels, default-No,
keyboard cancellation, plain mode and complete narrow-width content. Long paths
and digests use substantial vertical space at 24 columns. There is no approved
pixel baseline, screen-reader result, alternative locale/font coverage, native
Linux/Intel Mac result or browser/mobile surface. Cross-builds do not supply these.

The earlier checkpoint retains success, owned/foreign paths, menu navigation and
separate native handoff evidence. A [subsequent fresh native-session check](../native-candidate.md)
uses this same candidate. Final-head reconciliation and #10 independent
exact-candidate/physical cross-machine acceptance remain separate work. These
recordings must not be relabeled as that acceptance or as a later source build.

## Recording integrity

```text
916ac1970f6216e67d4a283b342485098ee6dafeef9ce1025616430a29ba250b  corrupt.recording
1091de5b8fe76fb8ece30b4826794a4a4f2259a4db32589b728e38cabed1df3a  eof.recording
2cb9246a4c6cf6b4235f3cae3e36fdda512cbfd58c5df3eda941185960937fd1  escape.recording
dde1ec33e3666eb8f06212b2b3b262574a63a72505822fb2dc9fba0c3215acb4  offline.recording
7ec2fc71c5876cd0e5eb138574e21384a520253ca6c337cb6751a465d1d0bd83  partial.recording
a0e64d6577b23511c478fa714718db370a6d79433990834e69255245600266c9  plain-narrow-partial.recording
2cf39887e129b7b32673b2447523c4e13e4e95916ab4f717ec293568c7ff4295  plain-narrow-recovery.recording
4eb18ddd8f6bc1710019a61be7003534456f27f080c65cf782efd87c903c12dd  recovery.recording
69d170f925732afe0cde6cd9e5420d42d5078092aaa123b3fef5ee113e8e6e60  unsupported.recording
```
