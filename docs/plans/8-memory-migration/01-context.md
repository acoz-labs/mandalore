# Context

#3/#4/#6/#7 provide a signet engine, shared interface, native memory plugin and
managed setup/recovery. Existing bank.json and agent.json roots are deliberately
refused by the normal engine. Archive status does not convert data or disable
the predecessor's running binaries, native plugins or sessions.

Public source inspection uses My Friday commit
f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8, specifically portable/{bank,store,memory,
events,changes}.go, its revision schema, and memorybank/{binding,service}.go.
No personal source repository or memory is required for implementation.

The old memory-only manifest uses schema_version/id/name with a bank- ID.
Its revision schema differs from Mandalore primarily in the bank-wide scope
enum (assistant versus signet) and schema URI. Records, sources, journal entries
and devices use compatible structural fields. The old local binding uses
bank_id rather than signet_id. Its writer lock is .my-friday/write.lock.

The old bank can also contain provenance/changes: observations of Git index
changes, explicitly excluding memory authorship. These must not silently become
new memory or disappear from the migration's retained evidence. Git history,
README, ignore rules and local sync/native configuration have different ownership
and execution implications from immutable memory records.

Mandalore has stricter semantic validation and size/identity checks. A legacy
format label alone does not establish that all contained data is valid or fits
current limits. Refuse unsupported/corrupt inputs explicitly; do not truncate,
invent provenance, pick conflict winners or import an old assistant framework.
