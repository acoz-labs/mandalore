# Technical design

## Binding

Versioned local JSON records signet_id, canonical absolute root, device_id and
actor. Its destination must resolve outside the signet even through existing
symlink ancestors; reject dangling-path ambiguity, unknown fields/versions,
trailing data, oversized files and replacement of an existing binding.

Enrollment generates a fresh opaque device ID with a supplied label; never infer
a hostname. Binding publication is immutable/no-replace. Reading a binding must
not enroll, migrate, synchronize, repair, or modify the signet. Revalidate the
bound identity/device when opening and let store-handle checks reject replacement
during subsequent operations. A second binding/clone preserves prior provenance.

## Shared interface

Define typed inputs for recall, scopes, history, journal search/append and
remember, plus compact write receipts and read-only inspect/validation. Document
create/bind setup schemas for noninteractive administration. A versioned
discovery command lists implemented operations, their schemas, read/write nature,
defaults, limits and retry notes. Do not list unavailable sync as implemented.

Use bounded strict JSON input for machine calls/writes. Reject unknown/duplicate
fields, trailing objects, invalid types and oversized data before writes. Keep
sensitive input values out of error envelopes. Stable error categories distinguish
usage/input, binding/store state, contention, I/O and cancellation; transport
adapters expose the same semantic error code rather than native parser strings.

Return structured local write receipts with immutable IDs and synchronization
not-requested. An error after a write might have occurred must say inspection is
required. No implicit duplicate retry, remote sync or journal from a read command.
Read-only mode must refuse mutations before side effects and be testable.

## CLI and MCP

CLI flags retain human discoverability; a machine-call route uses the same typed
catalog so agents need no menu automation. Explicit binding overrides environment,
which overrides the platform default path. Never discover a bank from cwd.

The MCP server advertises only operations against its fixed bound signet and uses
the official SDK. Tool annotations describe read-only/mutating, non-idempotent
writes and closed/local effects accurately. Bounded semantic payloads remain the
same across CLI/MCP; document transport framing/schema overhead separately.
Inspect actual SDK validation behavior instead of assuming Go structs make every
input strict. Server startup does not install a native plugin.

Signal cancellation and MCP cancellation are propagated to the shared boundary.
Check cancellation before mutation; do not claim an already published file was
undone. Context-aware traversal/transport bounds need explicit tests where added.
