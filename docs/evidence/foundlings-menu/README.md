# Foundling menu recordings

Contributor engineering evidence in progress for #12 / PR #32. This is not a
completed UI matrix or independent immutable-candidate acceptance.

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

Remaining actual scenarios: approved registration, availability/changed/missing
states, search/read, reconnect, explicit pin update, disconnect and partial failure;
plain/no-color/narrow rendering, additional cancellation, and fresh native-agent
behavior. Automated tests do not substitute for those rendered checks.
