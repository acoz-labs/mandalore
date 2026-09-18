# Recovery command copyability (#77)

Engineering source: `4425655cd5083a48c332be4df86cb6a218c6c34b`.
Native test executable SHA256:
`0f4f8f7d7b81cc3c7bbcfef2698bade2c4c40814af857c7677793574dc0b3728`.
Driver `cmd/mandalore/release_quota_native_test.go` SHA256:
`5a802f12bfdd27d40d3ab0828c76b1f1b10998395be9d40eb5299fc1691f428e`.

On September 18, 2026, compiled with pinned Go 1.26.4 and exercised in the
designated macOS arm64 terminal lab. Production renderer and keyboard handling
ran with synthetic release API receipts. No network, real recovery, private
pending record, personal installation or memory was targeted.

## Actual rendered matrix

| Surface | Action and result | Recording |
| --- | --- | --- |
| Plain, 32 columns, no color | Select 2; partial receipt and intact command; PASS | [plain32](plain32.txt) |
| Plain, 80 columns, no color | Select 2; same intact command; PASS | [plain80](plain80.txt) |
| TUI, 80 columns | Down/Enter; same command and receipt; PASS | [tui80](tui80.txt) |
| TUI, 32 columns, no color | Down/Enter; readable wrapped explanation, same command; PASS | [tui32](tui32.txt) |
| TUI, 32 columns, no color | Enter selects default No, no apply; PASS | [default32](default32.txt) |
| TUI, 32 columns, no color | Escape returns without apply; PASS | [back32](back32.txt) |

Each run used macOS `script` and set/printed its PTY dimensions with `stty`.
This constrains the renderer, not the enclosing terminal pane's physical size.
The public text captures remove ANSI controls, carriage returns and trailing
whitespace from actual output; they are not raw replayable recordings or pixel
screenshots. Raw captures remain local. Synthetic path/command bytes are unchanged.
All four partial receipts contain exactly one identical 155-byte command line,
without indentation, formatting escapes or inserted line breaks. Narrow receipt
fields remain wrapped; commands are explicitly separate. Existing narrow TUI
choice/help ellipses remain, but the new command and explanatory prose are not
truncated. No error was observed beyond the intentionally simulated quota failure.

## Functional verification and boundaries

The failing-first `TestRecoveryCommandCopyPreservesShellMeaning` reproduced the
old missing intact line. It now extracts the displayed command and evaluates it
with an inert shell function shadowing Mandalore, checking exact arguments and
synthetic pending-file contents. Cases include apostrophes, double spaces,
Unicode and shell metacharacters. No recovery command is executed against the
product. Renderer tests cover exact unstyled bytes at 24/32/80 columns and safe
suppression of control/format characters and invalid UTF-8. Existing width and
sanitation tests for ordinary fields remain unchanged.

No clipboard permission, OSC52, saved command file, automatic retry or new API
is introduced. Terminal copy behavior is not universal: select the whole logical
line rather than reconstructing a command from physical wrapped rows. Tests prove
emitted bytes and shell meaning, not every terminal's clipboard implementation.
Browser/mobile, screen readers, other native platforms and locales are untested.
This is engineering self-review evidence under ADR0003, not nominated artifact
acceptance. Final-head CI and reconciliation are recorded on PR #117.
