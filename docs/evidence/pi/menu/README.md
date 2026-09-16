# Pi connection menu: rendered engineering checkpoint

Classification: `in-pattern-visual-change`, issue #13 / draft PR #72. These are
actual macOS `script -qr` terminal recordings, not reconstructed output. They
contain only synthetic paths/data and begin inside the child application. This
is contributor verification, not independent product acceptance, nomination or
release. The remaining scenario matrix and final-head reconciliation are open.

## Identity and environment

Fixed implementation source: `2c95d47ba2bb21d5c1b8d6e43522259a027415db`.
Retained local candidate manifest SHA-256:
`2be685827f9fef708ddc0020059df4f709dd74a23ab4ecc3e44f7a51a352ffb2`.
macOS arm64 executable SHA-256:
`502d571d394222aa445811939aad67c40144526a727dbc0fa8dbfe4b85bf031b`.
Embedded Pi package SHA-256:
`79d2491186a6da434bd1e7fd8ea212136595247e90162c5bb3beabc6639614f8`.

The eight-file build was verified against its manifest. Its tracked `1.0.0`
label does **not** make it the published v1.0.0, which remains unchanged.
Environment: macOS arm64, Go 1.26.4, Node 24.1.0, Pi 0.85.1, dedicated Herdr
test pane, English, keyboard input, 80 columns / 30 rows and color enabled.
A regular synthetic-path wrapper delegated native commands to installed Pi;
it did not replace Pi or copy authentication. The native profile and signet were
disposable. A relative unrelated package with resource filters and theme setting
was present before setup. No model request or personal activation occurred.

## Actual recordings

| Recording | Actions and result |
| --- | --- |
| [Initial preview/default-No](preview-before-fix.recording) | On source `318c2bb915e82024da534428a57036062a337f16`, used vim k and arrow navigation, accepted explicit fixture defaults, kept learning enabled, reviewed effects and chose default No. Complete bank/settings snapshots stayed identical and installation state remained absent. Found an ambiguous blank Previous generation label. |
| [Fixed install and inspection](install-inspect-color.recording) | On the fixed source above, repeated keyboard setup, verified Previous generation: None, explicitly approved installation, inspected the connection through the Armorer, then exited normally to the shell. Receipt showed verified/installed, no uncertain effects and a fresh-session requirement. Armorer distinguished structural passes from untested native login, live tools, remote freshness and active context. |
| [Narrow plain inspection and EOF](plain-narrow.recording) | On the fixed source, 32 columns / 30 rows with NO_COLOR and plain mode. Inspected the healthy Pi connection, used the explicit harness Back choice, then entered connection setup and exited at its first text prompt using EOF. Headings, status words and full wrapped identities remained readable. |
| [No-color Back and cancellation](no-color-cancel.recording) | On the fixed source, 80 columns / 30 rows with NO_COLOR in the interactive menu. Used Esc and h to return from harness selection, then selected Pi and used Ctrl+C at the runtime prompt. Returned to the shell with canonical input, echo and signal handling restored. |
| [Stale preview refusal](stale-plan.recording) | On the fixed source, 32-column plain mode. Prepared a learning-enabled update, changed only the disposable native wrapper bytes after preview, then confirmed. The menu refused the stale plan and returned without retry or registration changes. The wrapper was restored to its original bytes afterward. |
| [Foreign registration diagnostic finding](foreign-before-fix.recording) | On the fixed source, 80-column plain mode. A separate synthetic profile referenced an inert, unowned local package named mandalore. Preview correctly refused it before confirmation, but displayed only a generic child-exit error. This is a usability finding, not a passing diagnostic. |

The first artifact's manifest SHA-256 was
`6cb1add4e30037d200d2f708fd420ef5c4601d9990317859cad064ac5bfc1f41`,
and its macOS arm64 executable SHA-256 was
`1a0638825bdbfd91d4c92432e3644d3d214bccd4bad857c13af3d44ecf128432`.
That recording is retained as the actual finding, not mislabeled a fixed pass.
The failing-first menu regression reproduced the blank field before the
presentation-only fix. It now checks both absent and real previous paths in
preview and receipt; no plan, confirmation or registration semantics changed.

After fixed installation, the typed doctor independently reported healthy and
learning enabled, selecting the exact candidate executable. Complete synthetic
bank hashes were unchanged; the unrelated relative package/filter and theme
were preserved. Native credentials were neither copied nor inspected.

The narrow/EOF, no-color/cancellation and stale-refusal journeys preserved all
31 files in the installed synthetic signet, profile and installation state,
verified by complete path/hash snapshots. The foreign profile and inert package
were separate fixtures; no registration or generation was created there.

The foreign-refusal finding prompted a bounded structured-preview diagnostic
fix. It carries only a valid protocol-1 failure's nonempty code and message,
with size limits and rejection of terminal controls; invalid/duplicate JSON and
raw process output remain suppressed. Cancellation/deadline errors retain their
identity. Regression tests cover the decoder and delegated failed previews with
both zero and nonzero exit status. A fresh rendered retest remains required.

## Review, validation and remaining work

Contributor self-review checked actual rendered headings, full wrapped paths
and hashes, default-No, explicit effects, status labels and keyboard return to
the shell. Long identities are complete but consume vertical space; this is not
a pixel baseline or screen-reader verdict. The missing previous path is now
explicit rather than visually ambiguous. Raw native failures and private data
are absent from these recordings.

Full pinned host CI passed 22 Node tests plus Go race tests/vet and four target
builds; Docker was unavailable. Hosted CI for the fixed source passed in
[run 35039496193](https://github.com/acoz-labs/mandalore/actions/runs/35039496193).
Cross-builds are not native Linux/Intel execution.

Scoped verdict: pass for the fixed recorded journeys except the explicitly
retained foreign-diagnostic finding. Still required: the diagnostic's rendered
retest, interrupted registration/recovery and native tool warning/partial-result display at the final
implementation head. Earlier deterministic tests are not rendered evidence.
No browser/mobile interface exists. Screen readers, alternate locales/fonts and
native non-host platforms remain unverified. Replay uses macOS `script -p`;
these BSD recordings are not promised portable to every platform's script tool.
