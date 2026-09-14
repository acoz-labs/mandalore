# Foundling menu recordings

Contributor engineering evidence for #12 / PR #32. The actual menu journeys
below passed contributor review. Native-agent behavior remains separate, and
this is not independent immutable-candidate acceptance. Classification:
`new-or-materially-changed-experience` within the existing terminal-menu patterns.

## Registration cancellation

`register-cancel.recording` is actual macOS `script -qr` output/input timing from
the designated Herdr test pane, not a generated transcript. Runtime source:
`09e5aa0573db0416132a225211f7fc37453dbfd0`; Go 1.26.4, macOS arm64,
57-column color-capable terminal, English and keyboard input.

Runtime SHA-256:
`cdf492235288d58da7d0b10b58380bd36bcaf472c643f02b6976ad3858cbee75`.
Recording SHA-256:
`e084e27f557c63a26141ec82fb398bf6a7a8928d4853ec4a151c1fc25ccc3aa3`.

The synthetic source has one 312-byte Markdown document. Its selected-content pin:
`a937137277bb40b24143dcf489f6671144b34d7477fd3d1c1d750c660ae9284f`.
The source describes an old fictional project name/process and quotes the manual
trigger phrase; it contains no real user memory or credentials.

Actions observed: Vim `k` and arrows selected Foundlings/Register; a name containing
`jklgG` remained ordinary text. The preview showed the full portable identity/pin,
long local-only path, eligible counts, reference-only boundary and separate durable
steps. Enter on default No returned to the submenu; Back/Exit returned to the shell.
The actual CLI then returned an empty registration list, and `stty -g` matched its
pre-session value. No registration, connection or promoted memory was created.

Contributor judgment for this scenario: pass. Headings, selection and default-No
were clear; paths and hashes wrapped without losing content. The narrow footer and
collapsed previous-input summaries can be ellipsized by existing TUI components,
while the effects preview retains complete values. This is not approval of all
remaining journeys or a visual-regression baseline.

Replay on macOS at 57 columns:

```sh
script -dpq docs/evidence/foundlings-menu/register-cancel.recording
```

Playback uses macOS script's recording format and the viewer's terminal behavior;
other script implementations are not assumed compatible. The file is retained in
Git as an openable local evidence artifact, not a hosted service or private session.
Actual replay reproduced the existing script-player limitation: protocol queries
can leave replies or keyboard mode changes at the shell prompt. Use a disposable
terminal for replay, do not submit stray input, and reset the terminal afterward.
The designated pane's keyboard mode/input were restored after this check. The
application itself had already restored its terminal settings on normal exit.

## Approved, changed and recovery journeys

These additional actual recordings use the same runtime source and binary hash
above, on the same macOS arm64 terminal. The UI code did not change between
recordings. Each link is a retained binary recording, not a reconstructed trace.
All source text, actors and banks are synthetic; no account authentication,
network access, Git remote, source execution or real-memory migration occurred.

| Recording | Rendering | Observed journey and result |
| --- | --- | --- |
| [Approved registration](approved.recording) | 57 columns, color | Reviewed and approved registration/connection; available listing; search for Copper Finch; read `reference.md` with unreviewed label, actual file hash and complete 312-byte excerpt. Ctrl+C returned to the shell. Ordinary recall and journal remained empty. |
| [Moved-source reconnection](reconnect-plain.recording) | Plain, `NO_COLOR`, PTY temporarily 24 columns | Moving the source made its old connection unavailable. Explicitly selecting the moved directory showed the same complete pin and local-only effects. Approval restored access without changing portable registration. Back/Exit restored the shell and 57-column PTY. |
| [Repin and disconnect](repin-disconnect.recording) | 57 columns, `NO_COLOR` | Changing the source to record Silver Heron produced a changed state. Reviewed old/new pins and explicit reason before repinning. A mistakenly submitted blank disconnect reason was refused; Escape from a subsequent valid preview preserved availability. Approved disconnect then showed disconnected. History retained all three registration revisions. |
| [Partial setup failure](partial.recording) | 57 columns, `NO_COLOR` | An intentional regular-file obstruction at the clone-local connection directory allowed registration but prevented connection. Separate “Registration saved” and “Connection not completed” blocks reported partial state without claiming success or rollback. The obstruction was preserved. |
| [Connection recovery](recovery.recording) | 57 columns, `NO_COLOR` | After moving the synthetic obstruction aside, Connect selected the already saved registration, previewed the source and published a durable connection. Listing showed available. CLI history still contained exactly one registration, and recall/journal remained empty. |

Recording SHA-256 values:

```text
99bfb6116b2e022437b5407737301fe85ee24e2fc35b1ce16492f31c5ea8ad29  approved.recording
e2a94bafeeb6aaeb0279816ed9805386cc1133acbea6c0be1de439c65e5abc41  reconnect-plain.recording
84d9e9e7b74402319ffcf15418506a1452c67a6234139142983f6393b1a67932  repin-disconnect.recording
498d983e7fd192ddb9f2c2206d53406ebd66ab898137aad8dbab2491fecb6a66  partial.recording
b86d13805573e228b9334d2ecbc5bf48755fce28c85229640ef9d577d1b72fdb  recovery.recording
```

The modified source is 334 bytes, with selected-content pin
`be3b5eb748b6968024892d7d4ea7d3a5b76adf2dd2d1208efa9426eda5f2e28b`.
Its prior approval-every-task process and quoted trigger remained historical text;
the menu neither executed nor promoted them. The recovery fixture used a separate
signet, not a reset of the first fixture. Terminal settings after recovery exactly
matched the pre-session `stty -g` baseline; dimensions were again 42 rows by 57
columns. The accidental blank reason correctly latched the existing menu's failure
exit status even though later operations succeeded; it was not an intentional
validation input or an unexplained application crash.

## Contributor judgment and limits

Verdict: **pass for these menu journeys**. Effects previews preserve full hashes
and paths; selection, default-No, historical authority and local-vs-portable writes
remain distinguishable without color. Partial completion provides actionable
registration identity instead of encouraging duplicate setup. Empty connection-ID
and path rows in the pre-publication failure receipt are cosmetic, not success
claims. Plain mode avoids dependence on cursor styling at a narrow terminal width.

Functional assertions and CLI post-state checks complement the actual rendered
recordings; they do not replace product judgment. Keyboard checks include arrows,
Vim navigation, literal navigation letters in text inputs, Enter, Escape and Ctrl+C.
EOF/output-write failures remain automated coverage, not claims of additional
recorded sessions. There is no browser/mobile UI or network console in this local
menu. No unexpected source execution or network operation was observed; this is
not a packet-capture claim.

There is no approved pixel/recording baseline, screen-reader acceptance, alternate
locale/font coverage or native Linux UI evidence. Fresh native-agent consultation
and promotion remain a separate #12 test. Final PR reconciliation must bind this
runtime evidence to its exact reviewed head and account for any later UI changes.
Independent acceptance must repeat the relevant matrix on the nominated immutable
artifact under #10/#11; these contributor recordings cannot be relabeled as that
acceptance.
