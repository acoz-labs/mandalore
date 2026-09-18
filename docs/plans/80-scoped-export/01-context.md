# Context

#80 activates O2 from the reviewed #16 discovery at
`b0e6880bc5f1e6f2085e26a093625ba69e2a0abb` (PR #78). Its prior parked sequencing
is superseded by the current owner goal, not by implicit scope expansion.
`docs/privacy.md` records the implemented O1 boundary: memory/provenance/journals
are plaintext, historical evidence persists, and labels are not access control.

Today `internal/memory` validates append-only revisions, sources, devices,
journals and foundling registrations. `internal/binding` selects one clone and
author. `internal/api` has strict bounded typed operations and read-only denial.
No export operation exists. Foundling source paths live only in ignored local
connection files; these require overlap checks but never belong in a report.

Consumers are a person reviewing selected knowledge or an agent preparing a
derived document. The readable first format is indented JSON with an explicit
report version/kind, not a restorable bank or executable Markdown/HTML. Current
memory recall remains unchanged. No real signet is inspected for these tests.

Acceptance covers exact selection, metadata preview, unchanged reviewed apply,
private fresh output, provenance-aware omission, and explicit partial receipts.
No secret detection, global erasure, backup restoration, Git history, transcript,
credential, foundling-source copy, native plugin change or network is in scope.
