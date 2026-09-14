# Menu engineering evidence

This is contributor engineering verification, not independent product acceptance
or the completed rendered-evidence gate. PR #28 remains draft.

Runtime source: `bae7d81547c711dbdd26110e6811075cf1ccc8a9`.
Compiled macOS arm64 artifact SHA256:
`2eb2dff392e65632544fbe181243924755ab591bf3dbb6340fcd0125343ebff3`.
Build and tests used Go 1.26.4 in the designated sibling Herdr pane. The native
terminal was 57 columns wide, English, with a color-capable theme. No private
bank, live predecessor migration or provider credentials were involved.

## Automated evidence

Failing-first menu tests initially reported an unknown menu command. Subsequent
tests exposed approval despite a failed output stream and report overflow at
eight columns. Both were corrected; the final full local `bin/ci` passed race
tests, vet and macOS/Linux amd64/arm64 cross-builds. Cross-builds are not native
acceptance. Hosted CI for this runtime is run `34796947583`; inspect its actual
conclusion rather than inferring success from the local result.

Covered behavior: complete-line input, default-No, EOF after an unfinished
confirmation, :back, process cancellation, binding collision before creation,
successful create/bind/Git-init, independent enrollment into an existing bank,
partial Git failure preserving binding, bounded input, failed display refusal,
local-artifact preview without execution, partial native phase reporting and
readable nested status fields. Console tests cover arrows/Vim navigation, text
entry, sanitation, narrow reports, Unicode, literal path preservation and color.

## Native interaction observations

The first worktree build (before the final status-prose refinement) exercised
the actual terminal, not a fake input adapter:

- Arrows and j/k navigated; gg selected the first option.
- A name containing `jklgG` remained ordinary text in a text field.
- The full effects preview retained both synthetic paths at 57 columns.
- Enter on highlighted No returned to the menu. Direct inspection verified no
  signet or binding existed afterward.
- Repeating with explicit Yes created the bank, outside-Git binding and local
  Git history. The menu labeled remote synchronization as not configured.
- Inspect reported a clean, local-only bank. Ctrl-C returned to the shell.

The exact committed runtime above repeated read-only inspection and Ctrl-C exit
against that synthetic bank after the status-prose refinement. Hashing all bank
file paths/contents (including Git and local receipt files) before and afterward
matched. This does not cover directory metadata or prove remote freshness.
The terminal's `stty -g` value also matched before startup and after Ctrl-C exit.

## Remaining before readiness

Repeat the complete journey matrix on the final implementation head, including
plain/no-color mode and recovery/error previews. Retain
openable rendered evidence and record product judgment under the delegated
self-review policy; these text observations are not screenshots or visual
approval. The current tool session has no usable screenshot capture backend.
Do not replace the missing evidence with a fabricated rendering.

Promote/reconcile the six-file #7 plan into durable operations/architecture docs,
remove the temporary plan, self-review the exact final head and require hosted
CI before readiness. Published update discovery and immutable-candidate physical
machine acceptance remain #11/#10; the local menu does not close those gates.
