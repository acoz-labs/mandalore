# Recorded native terminal evidence

These files are actual macOS `script -qr` recordings of synthetic menu sessions,
not generated screenshots or reconstructed output. They retain terminal output,
input and timestamps. They are contributor engineering evidence, not independent
product or immutable-candidate acceptance. No private banks or credentials appear.

## Recordings

| File | Runtime source | Scenario |
| --- | --- | --- |
| `menu.recording` | `407cfcf0539bdb657fd83b66b8dfbbefe222f574` | 57-column color TUI: arrows/Vim, default-No cancellation, approved create/bind/Git-init, local-only status, sync preview/cancel, missing-receipt repair refusal, exit |
| `menu-current.recording` | `05e51d9b13ff2368e9d8a0f2b186d799746c8f53` | Repeated color TUI matrix on the corrected runtime: cancel/no files, approved setup, inspection/sync preview/cancel, repair refusal and exit |
| `plain-narrow-fixed.recording` | `05e51d9b13ff2368e9d8a0f2b186d799746c8f53` | 24-column plain/no-color mode: complete paths/hashes, local-artifact connection preview/default-No, missing-profile doctor and distinct not-tested boundaries, exit |
| `partial-no-color.recording` | `05e51d9b13ff2368e9d8a0f2b186d799746c8f53` | 57-column no-color TUI: approved synthetic creation/binding, deliberately unavailable Git, partial-success receipt and retained recovery direction |

The second runtime changes plain choice wrapping plus documentation, not TUI
navigation or the connection engine. Its artifact SHA256 is
`87a7df9efcb558d8dcabddd6e2da58b15feee17d7fdf7188acbb836afb777c19`.
The first artifact SHA256 is
`5178deece0dabbba50150d6ddfbc720ea2cedd06462c42a1e42d0c6d00b49eb5`.
Both were built with Go 1.26.4 on macOS arm64 in the designated Herdr pane.

## Replay and limitations

On macOS, use the native `script` player in a disposable terminal:

```sh
script -dpq docs/evidence/setup-menu/menu-current.recording
script -dpq docs/evidence/setup-menu/plain-narrow-fixed.recording
```

Omit `-d` to preserve the original timing (including pauses while inspecting).
Use 57 columns for the first recording and 24 for the second. The recording
format is BSD/macOS-specific; Linux util-linux `script` is not the same player.

The first recording was replayed in the designated pane and the expected
success/error states were observed. Playback also re-emits native terminal
protocol queries and can leave replies at a shell prompt or keyboard mode changed.
This is a playback limitation, not a failure of the application's tested terminal
restoration. Prefer a disposable terminal, do not submit stray prompt text, and
reset/close that terminal after review. No screenshot or cross-platform video
capture is claimed. These recordings are not an approved visual-regression
baseline, screen-reader verification or evidence for other native platforms.

The plain connection preview deliberately identifies the test Mandalore binary
as the native executable solely to prove preview does not execute it; approval
was declined. It is not a successful Codex installation. Missing-profile doctor
does not invoke that executable. Actual native installation/update/repair and
fresh-session memory verification are in [the managed setup receipt](../../setup-native-evidence.md).

## Review outcome and remaining gate

Observed text/cell layout retained clear headings, numbered/selected options,
default-No and explicit status labels. The initial narrow plain test split words
like paths; that failure prompted the fixed runtime, regression test and second
recording above. Inspection confirmed cancellation created no signet/binding in
the first journey and connection preview created no native profile/state in the
second. The intended synthetic setup remained local-only.

Contributor rendered judgment is recorded in
[the evidence summary](../../menu-engineering-evidence.md); final-head
reconciliation belongs to PR #28. The owner's self-review delegation is not the
release workflow's independent acceptance. Exact-candidate testing remains #10/#11.

The partial scenario restricts PATH only in its child test process. It does not
remove or alter Git on the host. The recorded output states that signet/binding
creation completed, Git initialization did not, and no rollback was performed.
Semantic color sequences were absent while pass/fail labels remained visible.

## Recording integrity

```text
612eac3d4825a6b7015bdc98d0114277dd411df3560401b397db1a70c020414f  menu-current.recording
035e6f4bc24723abf5a676c08048f08f8d3e98812329903c1b5cb1bfc61ac06c  menu.recording
008a393d7675cd6f4e8db9bb6cf3c858a1f7eeb05dcf07b0c5451dc4e7575cad  partial-no-color.recording
433d5baa520a577d7b3215f2293fe6a594512d6f1a139ce287545041795ebe80  plain-narrow-fixed.recording
```
