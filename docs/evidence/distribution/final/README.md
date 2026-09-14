# Final PR-head terminal verification

Contributor UI/design self-review under the recorded MVP authorization, not
independent product acceptance. This evidence-only branch preserves the exact
tested implementation head without adding another commit to PR #36.

## Identity and environment

- PR #36 implementation head and candidate source: `af94a679d9bd36bdf2d4f938784149d59abf6c91`.
- Candidate manifest SHA-256: `e4cdb715fb540c9c6223f8add52114f383e8f85885c6d5dd472e842e21213231`.
- Actual ARM64 driver/runtime SHA-256: `636aef6e0f232577f1a7dd688971d96e4782877a80f84bd2a39554d3be54df9d`.
- Embedded plugin SHA-256: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- Intended version 1.0.0; no published release or independent nomination implied.
- macOS ARM64, Go 1.26.4, designated Herdr pane; English, keyboard; 30 rows,
  57-column color or explicitly recorded 24-column plain/NO_COLOR.
- Native handoff: byte-identical pinned Codex 0.153.4, isolated empty native
  profile, synthetic signet and binding. No authentication copied or enrolled.

Two isolated clean-source builds produced identical bytes for all eight files.
The actual candidate executable drove every recording; no development binary
stood in for it. Candidate checksum verification passed again after the tests.
The copied manifest below is the exact tested candidate manifest, not a new build.

## Rendered matrix and observed results

The recordings are unmodified macOS `script -qr` output from the child
application. Paths/data are synthetic. Integrity hashes are in
[SHA256SUMS](SHA256SUMS); [BSD playback limitations](../menu/README.md#replay-and-integrity)
apply. Do not submit terminal-query replies left at a playback shell prompt.

| Recording | Action / result |
| --- | --- |
| [default-no](default-no.recording) | j/k, Enter; default-No returned without creating the destination. |
| [escape](escape.recording) | Escape at approval; stopped without installation. |
| [eof](eof.recording) | Ctrl-D in plain mode; stopped without installation. |
| [partial](partial.recording) | Approve; real owner-write-disabled fixture bin prevents launcher creation after retaining the verified runtime and pending plan. Incomplete receipt and recovery instruction shown. |
| [recovery](recovery.recording) | Restore only fixture bin permission; original executable reads the exact pending record; complete receipt and launcher. |
| [narrow](narrow.recording) | Independent 24-column plain/NO_COLOR permission failure; complete wrapped effects, paths and recovery text. |
| [recovery-narrow](recovery-narrow.recording) | Same-plan recovery for the narrow fixture; complete machine-readable result. |
| [success](success.recording) | Approve fresh CLI installation, retain default unchanged native connections; complete receipt and full outside-PATH command. |
| [native](native.recording) | Already-current CLI; separately select, preview and approve the explicit synthetic native profile. Correct new runtime/plugin, verified receipt and fresh-session notice. |
| [unavailable](unavailable.recording) | Actual public discovery found no published release; no installation or false success. |
| [offline](offline.recording) | Child-only unreachable loopback proxy; request failure with no destination. |
| [corrupt](corrupt.recording) | Same-length one-byte bootstrap change in a separate candidate copy; digest refusal before confirmation. |
| [unsupported](unsupported.recording) | Copied manifest declares an unsupported target; refused before confirmation. Not native Windows execution. |
| [foreign](foreign.recording) | Existing unrelated file named mandalore; ownership refusal, original hash unchanged. |
| [menu](menu.recording) | k/Enter into source choices; Escape back; observed gg/G first/last navigation and Exit. No prefix created. |

The first navigation recording was retained locally: a controller sent a burst
across a screen transition before observing the next screen. The final recording
above repeats the journey with each screen observed before the next selection;
the burst is not counted as passing navigation evidence.

Afterward, filesystem checks confirmed all refused/cancelled destinations absent,
foreign file unchanged, all three successful/recovered launchers byte-identical
to the candidate, expected source metadata, and no pending records. Native doctor
passed 12 structural/inventory checks; its retained runtime matched the candidate.
All synthetic signet file paths/bytes and the binding hash matched before/after;
the isolated native profile contained no auth.json.

## Review and limitations

Verdict: pass for the final-head implementation matrix. Headings, effects,
default-No, keyboard paths, explicit incomplete/failure/success labels, plain mode
and complete narrow content were inspected against the existing console design.
At 24 columns, long paths and digests take substantial vertical space. Raw JSON
recovery output intentionally remains machine-readable. No approved pixel baseline,
screen-reader result, alternate font/locale or native Linux/Intel Mac result is
claimed; this CLI has no browser/mobile surface.

Full local pinned-host CI and
[hosted exact-head CI](https://github.com/acoz-labs/mandalore/actions/runs/34835499342)
passed. Docker was attempted and its daemon was unavailable. Unit/process fixtures
cover corruption, interruption, stale/foreign/concurrent installation, wrong-plugin
delegation, publication guards and ambiguous retry separately from these recordings.

The [fresh native conversations](../native-candidate.md) used source 065b481, not
this candidate. A read-only Git comparison confirmed no changes between that
source and this head in cmd, internal, plugins, packaging, bin, workflows, modules,
toolchain config or VERSION; intervening changes were documentation/plan retirement.
That comparison is not relabeling the earlier session as an exact-artifact test.
This head's native handoff is directly exercised above; #10 independently repeats
the final nominated candidate's model/platform/cross-machine criteria.

The issue and release remain open. No public release, release-policy change,
acceptor configuration, credential copying or real-memory migration occurred.
