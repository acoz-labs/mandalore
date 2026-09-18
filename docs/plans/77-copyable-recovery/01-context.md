# Context

`menu.releaseResult` currently puts the POSIX-quoted pending-plan command in a
normal console Field. `RenderBlock` inserts newlines and indentation inside long
values. At 32 columns this changes a command when copied literally (#77).

Users must inspect partial installation state and explicitly retry the same plan
using the original executable. Agents already have machine-readable receipts.
The fix must retain that authority boundary, quote spaces/apostrophes exactly,
remain understandable without color, and avoid clipboard or file-write effects.
Current prefix validation already forbids control characters; the renderer must
also fail safely for malformed data passed through an error/test seam.

No installation/recovery behavior, memory semantics or network access changes.
