# Technical design

## Presentation contract

The server continues returning the shared `protocol_version`, `ok`, and
`result`/`error` envelope in both fields with consistent `isError`.

When orchestrating Mandalore through code mode:

1. Await the tool result once; do not repeat the operation to choose a rendering.
2. If structured content is a valid Mandalore envelope, the wrapper contains only
   the expected compatibility fields, and the sole unannotated text block is an
   equivalent serialization, render the structured envelope once.
3. Preserve the outer error flag whenever it conveys information not already
   represented consistently by the envelope. A contradiction is a fallback case,
   never a successful result.
4. With no structured content, preserve the original text-only result and error
   state. If already unwrapped, render the envelope as received.
5. For distinct/multiple/non-text content, unknown metadata, malformed or
   inconsistent values, preserve the full result. Do not guess equivalence or
   silently drop resources, annotations or error details.

Equivalence checking must not coerce values, reorder arrays, round numbers or
discard unknown properties. If the client cannot establish equivalence safely,
fall back. No need to generalize beyond Mandalore's actual bounded JSON shape.
Do not use `text(r.result)` or assume `r.structuredContent ?? r` is sufficient
for arbitrary results.

## Ownership and unchanged state

`internal/mcp` owns the wire contract and server instructions. The bundled Codex
skill owns agent usage guidance; the native client owns actual model rendering.
No bank migration, configuration flag, credential, extra network call, journal,
sync or mutable server session is introduced by presentation selection.

Preserve conflicts, truncation, pagination, source pins and citation identifiers
inside the selected envelope. Failure to optimize output is safe verbose fallback,
not a memory operation failure or permission to retry a write.
