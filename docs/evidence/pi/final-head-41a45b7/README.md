# Exact implementation-head Pi rendered verification

Tested implementation: `41a45b730c74070391af0305a1283e136b9ae8de`, PR #72.
This evidence-only branch retains captures without moving the tested PR head.
It is engineering self-review under ADR 0003, not independent product acceptance,
artifact nomination, release or personal activation.

## Artifact and environment

- Local immutable build source: the exact implementation commit above.
- Manifest SHA-256: `e6a9919de1ddbad3d3514a7c5519336505057929ad83ea009b851d57ac7addc1`.
- macOS arm64 executable SHA-256: `a1311edf613c175c8e4dec40bab254d9f9391fc4f708a57879b38b13048dadb0`.
- Embedded Pi SHA-256: `79d2491186a6da434bd1e7fd8ea212136595247e90162c5bb3beabc6639614f8`.
- All manifest payload hashes were checked before the fixture was created.
- macOS arm64, Go 1.26.4, Node 24.1.0, Pi 0.85.1, dedicated Herdr test pane,
  English and keyboard input. This build's tracked 1.0.0 label does not identify
  or replace the published immutable v1.0.0.
- A new synthetic signet/profile/state, unrelated relative package with empty
  resource filters and theme, and a separate foreign-package profile were used.
  No private memory, native authentication or thread history was copied.

These are original BSD `script -qr` recordings. Replay with macOS `script -p`;
format portability to other platforms' script tools is not promised.

## Actual scenario matrix

| Recording | Environment and observed result |
| --- | --- |
| [Color preview](color.recording) | 80×30, color. Arrow and vim navigation, Esc Back, learning-enabled preview, explicit None for absent prior generation, default-No, normal exit. Full bank/profile hashes unchanged and installation state absent. |
| [Install and inspect](install.recording) | 80×30, plain/no-color. Explicit confirmation performed real native install. Receipt verified/installed, uncertainty false, fresh-session requirement visible. Armorer distinguished structural passes from untested authentication/tools/remote/context. Typed doctor healthy; original memory and unrelated settings preserved. |
| [Narrow inspection and EOF](narrow.recording) | 32×30, plain/no-color. Healthy Armorer, full wrapped paths/status words, explicit harness Back, EOF at input. Full installed-fixture snapshot unchanged. |
| [No-color Back and Ctrl+C](no-color.recording) | 80×30, interactive/no-color. h Back, arrow selection, Ctrl+C at runtime input. Full fixture unchanged; shell canonical input, echo and signal handling restored. |
| [Stale preview](stale.recording) | 80×30, plain. Changed only disposable native wrapper bytes after preview, then confirmed. Refused as stale, no retry or fixture changes. Wrapper bytes restored. |
| [Foreign registration](foreign.recording) | 80×30, plain. Refused the separate unowned package before confirmation; displayed structured reason and preservation guidance. Foreign files unchanged, no foreign state created. |
| [Interrupted update and repair](recovery.recording) | 80×30, plain. Observed complete live PID and actual removal before interrupting the synthetic install gate. Receipt retained install-started, uncertainty true, installed false; Armorer reported disconnection. Explicit retained-receipt repair selected a fresh generation and passed rendered/typed inspection. |
| [Native tool results](tools.recording) | 100×34, color, actual gpt-6-astra/high model. One read-only inspection, then a separately authorized single journal/save-and-sync. One complete envelope per tool; local durability and nested pending delivery remained distinct. No retry. |
| [Unavailable warning](warning.recording) | 100×34, color, no model request. Changed fixture binding bytes before startup. Concise warning/Armorer guidance, no raw child errors, normal exit. Binding restored exactly. |

Interruption targeted only the synthetic native gate after proving it live; it
does not claim a user Ctrl+C at that install phase. Real Pi handled remove and
subsequent recovery install. Verification proved child reaping, lock cleanup,
unchanged memory, retained old generations and preserved binding/access mode.
The later native display test intentionally adds exactly one synthetic journal.

## Review and limits

Rendered headings, status words, wrapped identifiers, default-No, explicit
effects, partial receipts and recovery guidance were inspected in the live pane.
Native JSON wraps but retains its complete content; it is not a custom summary
or duplicate metadata body. Model answers separated successful saves from pending
delivery. Warning remained concise without claiming repair or synchronization.

Full pinned host CI passed 22 Node tests plus Go race tests/vet and four target
builds. Docker was unavailable. Exact-head hosted [CI 35041355284](https://github.com/acoz-labs/mandalore/actions/runs/35041355284) passed.
Cross-builds are not native Linux/Intel evidence. No browser/mobile surface
exists; screen readers, other locales/fonts and a pixel baseline are unverified.
No raw process errors, browser console or external-provider installation requests
were introduced. Native model authentication was inherited in place with ambient
resources disabled and the explicit synthetic package/skills selected.

Verdict is recorded in the exact-head PR review after checking the completed
recordings and fixture results. Candidate product acceptance must independently
repeat its applicable matrix on its nominated immutable artifact.
