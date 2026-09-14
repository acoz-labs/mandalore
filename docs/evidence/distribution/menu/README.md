# Guided release installation: native engineering evidence

This is an intermediate contributor checkpoint for draft PR #36, classified as
`new-or-materially-changed-experience`. It is not final-head reconciliation,
independent product acceptance, release nomination or permission to publish.

A [later exact-source checkpoint](../recovery/README.md) records actual partial
installation/recovery, cancellation, offline and invalid-artifact journeys and the
direct-stdin recovery instruction. It does not replace this historical evidence.

## Exact identities and environment

- Implementation: `4c64d9fde2f94b359a27c8cb3f2706ea1d443eb1`.
- Development driver SHA-256: `b49e40bae5c463fde860de51a7a8a7d2e701656994cfdfcafbe28b24ff37df40`.
- Selected candidate source: `2a59cecd92259b22cc219161db0ebec676f61bf7`.
- Candidate manifest SHA-256: `3c2c405c0d68bec6c9c9a23a7677c5cee01f8bdaa95174cff79f7e39e474cbaa`.
- Selected native runtime SHA-256: `4975862877d654f855a733a8c4c35a1753e3dd46b50f94cf94f1436923c67f95`.
- Selected embedded plugin SHA-256: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- Native host: macOS ARM64, Go 1.26.4, designated Herdr test pane, English text,
  keyboard input, 30 terminal rows and recorded 57/24-column widths.
- Codex: 0.153.4 ARM64, SHA-256 `b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`,
  byte-identical executable copied into the temporary fixture, isolated native
  profile and synthetic bank. No authentication was copied or enrolled.

The driver and selected candidate are deliberately different artifacts. The menu
executes the selected candidate for connection planning/application; its own
embedded package cannot stand in for the selected release's package. All candidate
versions here are intended `1.0.0`, not published upgrades. The final completed
implementation still needs its own retained candidate and fresh acceptance.

## Actual recordings and observed results

These are unmodified BSD/macOS `script -qr` recordings, not reconstructed output.
Synthetic paths and identities are intentional. No private memory or credentials
are included. Recordings begin in the child application, not the parent shell.

| Recording | Mode and actions | Observed result |
| --- | --- | --- |
| [default-no.recording](default-no.recording) | 57-column color; j, k, Enter | Default-No selected; destination remained absent |
| [cli-success.recording](cli-success.recording) | 57-column color; Down, Enter, then default unchanged | Owned CLI installed; receipt complete; full launcher shown outside PATH; no native connection update |
| [plain-narrow.recording](plain-narrow.recording) | 24-column plain and NO_COLOR; `:back` | Complete wrapped paths/digests/effects; no prefix created |
| [no-release.recording](no-release.recording) | 57-column plain and NO_COLOR; actual official discovery | No published release; no installation, no false success |
| [native-update.recording](native-update.recording) | 57-column plain and NO_COLOR; approve CLI, choose native preview, accept four explicit fixture defaults, separately approve native apply | Already-current CLI stays complete; selected runtime prepares/applies a new native generation; previous source retained; fresh-session notice |
| [menu-navigation.recording](menu-navigation.recording) | 57-column color; k/Enter into CLI source menu, Escape, gg/G/Enter | Correct source choices; Back returns to main; Exit remains default; no prefix created |
| [foreign-launcher.recording](foreign-launcher.recording) | 57-column plain and NO_COLOR; select fixture with an existing foreign launcher | Refused before confirmation/apply; foreign launcher hash unchanged |

After native update, actual doctor reported 12 structural/inventory checks passing.
Login, hook trust, live MCP, remote freshness and active context remained explicitly
not tested. Before/after SHA-256 inventories matched for every synthetic-bank file
and its binding. Both managed native generations remained present. The isolated
profile had no `auth.json`. CLI and native success were verified separately.

The exact-head full pinned-host CI passed: race tests, vet and four target builds.
Container-first execution was attempted; Docker's daemon was unavailable. Hosted
[CI at this implementation head](https://github.com/acoz-labs/mandalore/actions/runs/34825533899)
also passed. Cross-builds are not native Linux or Intel Mac execution.

## Self-review, limitations and remaining gate

The rendered flow follows the existing console's headings, labels, arrows/Vim keys,
plain mode and default-No behavior. Long paths remain complete at 24 columns but
require substantial vertical space. Receipt fields now say None rather than
leaving absence ambiguous. No approved visual-regression baseline exists.

Automated flow tests separately cover EOF/incomplete consent/output failures,
unavailable/offline previews, CLI partial state, second confirmation, wrong-package
refusal and native partial receipts. These are model-free fixtures, not recordings
of real interrupted installations. Actual subprocess fixtures distinguish a new
embedded package, reject stale/malformed/unbound responses and preserve a typed
partial receipt on nonzero exit. Self-review reproduced an output-limit defect:
an unsuccessful process could mask simultaneous stdout overflow. The failing-first
regression now requires all oversized output to be discarded, including that case.

Scoped contributor verdict: pass for the recorded journeys. PR #36 remains draft.
Still required: remaining failure/recovery rendered cases, final complete-head UI
reconciliation, candidate retention/publication integration and #10 fresh-session,
physical cross-machine and independent exact-candidate acceptance. Screen readers,
alternate fonts/locales, native Linux/Intel Mac and cross-platform recording replay
were not tested. Browser/mobile surfaces do not exist for this CLI.

The older local-artifact connection command intentionally retains its read-only
preparation contract and uses the running toolkit's embedded plugin. It is not a
release selector. Use the new release journey or the verified installed runtime's
typed operations when updating runtime and plugin together. Pending CLI recovery
still uses the saved exact plan; the menu does not invent a replacement plan.

## Replay and integrity

On macOS, use a disposable terminal at the recorded width:

```sh
script -dpq docs/evidence/distribution/menu/cli-success.recording
```

Omit `-d` to preserve original timing. The BSD recording format differs from Linux
util-linux. Playback re-emits terminal queries and can leave replies at a shell
prompt; do not submit them. This limitation is not a claim about live application
terminal restoration. See [the earlier playback notes](../../setup-menu/README.md).

```text
6361420bcbcda19e34eba9149e8b39a5f5bcb8c6fdf3251002236aa705c092fa  default-no.recording
ba9de1259e83f93cd171b12997d28e504c90881d36999a0310d61e5cfff8e75d  cli-success.recording
a588d9bb1e08ffcd8e1bbea89aa0ae21ef5cab984715adbb08d469d05e6e70de  plain-narrow.recording
5ef5d67aa5886b280d7504616cd6e2927928067f5070e40f6bfbff98b54d271a  no-release.recording
62f79be0ea36cc7c8433d392259183634fdd6d5b1c515bda8390b6793a688e59  native-update.recording
369444dc54897720ed777074b4672f90dbbccf0faa6fc24e6a2e8619bb053b22  menu-navigation.recording
2e0682da5e0d4f0d514a3faee7fed7948881e35fe977aa0422844ddf6a0d5efa  foreign-launcher.recording
```
