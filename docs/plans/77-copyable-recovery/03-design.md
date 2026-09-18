# Technical Design

Extend console Block with an optional literal command string. Render normal title,
body and fields as before. For a nonempty command, validate UTF-8 and reject
control or invisible formatting characters that could change display/meaning.
If rejected, show a clear command-unavailable message with no executable-looking
substitute. Otherwise append the exact unstyled string as one logical line.
Only trusted application code assembles shell syntax; this renderer never executes.

`releaseResult` retains its existing single-quote escaping of the pending path,
but moves the command out of Fields into this surface. Its prose names inspection,
original executable, unchanged plan, and selecting the full logical line. No new
API, persisted schema, automatic retry, clipboard or command file is introduced.

Output errors continue through the existing menu outputErr path. A successful
receipt with no pending plan emits no recovery command. The typed receipt still
provides the canonical pending path for automation. Existing recovery validation
and partial-state handling are untouched.
