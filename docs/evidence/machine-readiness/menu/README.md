# Readiness menu: rendered engineering checkpoint

Issue #85 / draft PR #87. Classification: `new-or-materially-changed-experience`.
These are actual macOS `script -qr` recordings from the designated test pane,
with synthetic selected paths only. They are not reconstructed output, final
implementation-head acceptance, product acceptance or permission to release.

## Source and artifact identities

| Source | Development executable SHA-256 |
| --- | --- |
| `b757751886fc3e19a0bb4edeab6b8ead98d6fb29` | `e24befa47b54fee7486bc73875dca35db4eb3d47310d932423ce77adc591992d` |
| `e981d32b76e78234ec9840e043953678934b63ef` | `aedb0af697d4a698f3a8e1e0531534eacb39d527c4155cc252157ffc3a28e703` |

Each executable was built from its clean named source with Go 1.26.4 and stamped
with that source and version `0.0.0-dev`. These are not release candidates or the
published 1.0.0. Environment: macOS arm64, English, native terminal, keyboard
input. Inner PTYs were 80 or 32 columns by 32 rows; the outer Herdr viewport was
310 by 67. Application report width leaves one column free. The test launcher
used explicit nonexistent state/profile/native/binding paths and PATH containing
only system tools. No native harness, provider or memory tool was invoked.

## Actual recorded journeys

| Recording | Observed result |
| --- | --- |
| [Color, 80 columns](color-80-b757751.recording) | Source b757751. Vim navigation to Armorer; Codex missing-setup assessment; Details and scoped artifact evidence; explicit prompt; arrow navigation to native disclosure; default No; Back to Exit. No native dispatch. Full terminal mode comparison passed. |
| [Initial narrow output](plain-32-before-fix.recording) | Source b757751, 32 columns, plain and NO_COLOR. Pi report and optional prompt, Back, then EOF at an untouched setup text prompt. Compact component rows split status words; this is retained failure evidence. The initial terminal comparison also flagged only Darwin's transient PENDIN state, investigated separately. |
| [Corrected narrow output](plain-32-e981d32.recording) | Source e981d32, same width/plain/no-color fixture. Pi component statuses now appear on distinct Support, Setup and Evidence lines with intact words. Back and EOF return to the shell without creating setup state. Stable terminal modes/control characters match. |
| [No-color Back and cancellation](no-color-cancel-e981d32.recording) | Source e981d32, 80 columns and interactive NO_COLOR. Vim navigation, Escape from harness selection, reentry, arrow selection of Pi, Ctrl+C at assessment selection. No assessment, success report, prompt or native command followed cancellation. Returned normally under the menu's existing cancellation convention. Stable terminal modes/control characters match. |

The selected state/profile/binding remained absent after every journey. Retained
files consist only of the two development binaries, fixture launcher and
recordings. Their existence is test setup, not an assessment side effect.

The narrow finding produced a failing regression at 31 usable columns before the
width-aware presentation correction. Normal-width rows remain compact. The EOF
fixture originally compared the entire `stty -g` string: only local-mode bit
`0x20000000` changed, which the Darwin SDK names `PENDIN` (pending input retype
state). Retests exclude only that bit from comparison, preserving checks on all
other flags and control characters. The fixture restores its original terminal
configuration on exit. No product terminal handling was changed for that finding.

## Recording inventory

| File | SHA-256 |
| --- | --- |
| color-80-b757751.recording | `c2eeec9fac3f21e0a2214d6ceb267f41d367353cf42e5ba5298bffbefeeb5c53` |
| plain-32-before-fix.recording | `13825c23e98afc6ae080e14ce53845b7b3ecb90193c2daf4d84cb1f8e76a9511` |
| plain-32-e981d32.recording | `9ad6f02970d029a852d0864acd3c555dec58ab46991c51955e75234a172da248` |
| no-color-cancel-e981d32.recording | `c5ff2fa2787ab8afd924a5286b3a38d00a6b9aeeadb02e12ec1e0d23dec90fd9` |

Replay with macOS `script -p <file>`; portability to other script implementations
is not claimed. Recordings start inside the fixture child, not the user's shell.
Color-free tests still contain terminal-control framing; NO_COLOR means no
semantic ANSI color, not removal of interactive cursor control.

## Scope and remaining gates

Contributor self-review: corrected missing-setup/narrow/back/cancellation
journeys pass for their named sources. Complete configured, stale, malformed,
partial, unsupported and native-handoff journeys remain pending, as do final
documentation promotion, exact-final-head reconciliation and hosted checks.
The whole feature remains draft. These recordings are not relabeled as a later
head's evidence. No browser/mobile surface or approved pixel baseline exists;
screen readers, other locales/fonts and native non-host platforms are unverified.
