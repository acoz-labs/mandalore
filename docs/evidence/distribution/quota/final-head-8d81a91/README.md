# Exact-head quota presentation and effect checks

- Source: `8d81a91cee6d493356d2380cad3c5fb791c8c789`, PR76, issue74.
- Environment: macOS arm64, Go1.26.4, English, actual Herdr terminal,
  30 rows, widths and input modes below. No browser/mobile surface.
- Menu test driver SHA256:
  `b219a908cb2f11e27c0b2b3b62a4d37e9f33f0be16c45677a669adc2934c5218`.
- Distribution test driver SHA256:
  `7fdccd84cc2933582f73b151100a1f431a7d8a9637d5437e1bbeb6e1e4538718`.
- Hosted CI: [35044702529](https://github.com/acoz-labs/mandalore/actions/runs/35044702529),
  passed. Full pinned host CI also passed (22 Node tests, Go race tests/vet,
  four target builds). Container daemon unavailable; documented host fallback.
- Scoped engineering verdict: pass. This is not independent acceptance,
  publication, a nominated artifact, real GitHub quota or native Linux evidence.

## Actual scenario matrix

| Recording | Input / result |
| --- | --- |
| [timing](timing.recording) | 80-column color; valid synthetic retry/reset advice, no retry or effects asserted. |
| [unknown narrow](unknown-narrow.recording) | 32-column plain, NO_COLOR; timing explicitly unknown, readable wrapped warning, no automatic retry. |
| [ordinary refusal](ordinary-nocolor.recording) | 80-column NO_COLOR; ordinary refusal is not called rate limiting. |
| [partial](partial.recording) | 80-column color, Down/Enter from default No; simulated partial receipt retains old/new runtime, incomplete status, pending path and original-tool retry guidance. |
| [partial narrow](partial-narrow.recording) | 32-column plain/NO_COLOR, `2`/Enter; same fields, warning and recovery instructions remain present. Long paths/commands wrap over multiple lines. |
| [default No](default-no.recording) | 80-column color; Enter on default No invokes zero applies. |
| [Back](back.recording) | 80-column color; Escape invokes zero applies. |
| [effect checks](effect-checks.recording) | Compiled distribution tests run in terminal: two concurrent operations keep private verified state; fresh/update each plan4+apply9=13, replay0; full verifier13/eight distinct assets; real fixture refusal preserves fresh destination/old launcher and explicit retry succeeds after clearing the refusal. |

## Methods, evidence boundaries and review

Both drivers were compiled once from the clean exact source using pinned
`go test -c`; their hashes were unchanged after all runs. The menu driver uses
the production renderer/navigation with explicitly simulated API responses.
Its native-only scenario gate is test code, absent from the production binary.
It does not contact GitHub or actually install a release. Partial composition
is not evidence of a production network call after activation. The effect test
does execute the distribution/apply code against in-memory HTTP and disposable
filesystem fixtures, but its executable verifier is inert. Do not call this
published-payload/native-install acceptance.

Actual unedited macOS `script -qr` files are retained, not reconstructed output.
Capture calls set `stty cols WIDTH rows 30` inside the child. Color calls unset
NO_COLOR and set TERM=xterm-256color; other cases set NO_COLOR=1. Plain cases
also set MANDALORE_QUOTA_NATIVE_PLAIN=1. The selected scenario is passed as
MANDALORE_QUOTA_NATIVE_SCENARIO, then the driver runs
`-test.run ^TestReleaseQuotaNativePresentation$ -test.v`.

Review inspected actual rendered headings, color-independent FAIL labels,
normal/narrow wrapping, source/receipt consistency, keyboard/default-No/Back,
advisory rather than guaranteed timing, and preserved partial effects. All
scenario assertions passed. No body/header dump or fixture workstation path
appears in these recordings. Earlier preliminary files with machine-specific
temporary paths were excluded and were not redacted/reused as final evidence.

At 32 columns, existing path/command wrapping is readable but the multiline
recovery command should not be copied as a literal shell script; use the canonical
single-line command from the interface/runbook or machine-readable receipt.
This is an existing narrow-terminal presentation limitation, not a new retry
mechanism. No approved pixel baseline, screen-reader, alternate fonts/locales,
native Linux/Intel or exact-candidate product acceptance is claimed.

The evidence branch is a direct child of the tested implementation head and
adds only this directory; it is deliberately not merged back to change the
tested source. Retain the branch/immutable evidence commit with PR76.
For playback use macOS `script -p RECORDING`; BSD recording format and terminal
query sequences require a compatible terminal. The files are not plain logs.

## Recording SHA256

```text
da6c481e628fc7fd2c03b8b060ae07af5b68a634d3380e46fd67a436d7a772d3  back.recording
9dafa1a7dae5af552102eb0e8ca12abff2ee5d0841a31080ea019e246b7997d6  default-no.recording
bc3660e17e3ea013ff84a2451c7bc38cf023294a9ff8bffad827f140f20d0521  effect-checks.recording
339f93c6f904235b39e5be67173c6d6bc9939f9326fae20d118a1f5926d084f9  ordinary-nocolor.recording
82e0c697ec4bc1133e0a43d204d401893881c2dc6950ee866520a528ebb6b995  partial-narrow.recording
9a191e03585fbe79cee70f57adda03116d6316e5bbcf88cbcf2e214bafca111f  partial.recording
1e5b513b669986bd1d04597868b61d91453b4fd3d3986221dc3ea34887ac1c1a  timing.recording
8fda9bb8f93f82825441c9c9cebf323814184006adf30ab89ba0914f83bedafb  unknown-narrow.recording
```
