# Context

## Problem and desired outcome

Issue #79 delivers the selected #16 privacy/lifetime outcome. Users need a
truthful account of storage, disclosure, correction and deletion before trusting
labels or exporting a bank. The discovery's disposable probe passed.

## Current state and actors

At e78e08329335f753adce821979541bf77d879d91, memory service/store/journal and
synchronization retain append-only evidence; API read-only denies mutation;
memorycontext can expose restricted bank-wide content. Foundling disconnection
does not erase promoted records or source copies. Native/provider retention
lies outside these engines. Existing tests and the discovery probe are the basis.

Users read the guide; agents/operators follow linked runbook guidance; developers
run synthetic regressions. Journeys: ordinary learning/recall, correction,
no-save, deliberate foundling disconnection and accidental secret incident.

## Acceptance, non-goals and constraints

Cover every #79 criterion through durable docs and tests. No private-data scan,
new flags/tools, label filtering, encrypted storage, automatic expiry/export,
history rewrite, migration, provider action or additional model context.

## Evidence and unknowns

Current behavior is source- and probe-backed. Future export consumers, withdrawal
compatibility and journal policy remain #80/#81, not implementation blockers.
Native/provider erasure and physical disk sanitization cannot be certified here.
