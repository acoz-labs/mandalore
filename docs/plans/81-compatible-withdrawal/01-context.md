# Context

Issue81 extends the reviewed #16 O3 contract. Discovery PR121 establishes the
current compatibility boundary and a single-record causal model. The owner
explicitly selected the same-repository opt-in format upgrade on September18.

Current format1 stores append-only content revisions, sources, devices, journals
and foundling registrations. `memory.Open` rejects unknown manifest versions.
`sync.appendOnly` also protects the manifest. `memory.EffectiveHeads` resolves
content branches; `memorycontext.Build` supplies native recall and scope routing.
`exportreport` now implements selected reports and must respect future visibility.

Users need to withhold a record from ordinary guidance, inspect its preserved
history and explicitly restore it. Multi-machine writers must not restore content
by accident. Operators need an explicit reviewed upgrade with truthful partial
receipts and a safe refusal from old clients. Existing format1 users must retain
normal behavior until they choose to upgrade.

The released1.1.0 probe shows direct unsupported-format refusal and refused remote
adoption while preserving old local content. The nine-test design model covers
causal cycles, stale requests and120 delivery permutations. Neither proves the
production file transaction; that remains mandatory implementation evidence.

No erasure, history rewriting, automatic TTL, global phrase blacklist, source or
journal suppression inferred from text, provider-session cleanup or personal
migration is in scope. Labels remain descriptive, not new access controls.
