# Design

Five predicate arguments remain repository, source, issue, artifact and reviewer.
Add one source case with an explicit issue allowlist and exact artifact comparison.
All mismatches still refuse. Repository validation, true-only confirmation and
case-insensitive configured-owner matching remain unchanged.

The recorder continues requiring the normal actor allowlist, valid nomination,
candidate ancestry, linked implementation identity, checks and evidence. Both
author checks share the same predicate; no recorder control flow changes.

No state migration, dependency, credentials, native runtime or application API
changes. Revocation still clears the configured owner. Public documents describe
selectors and roles, never private account names.
