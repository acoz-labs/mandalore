# Solution Decision

Use a dedicated report projection and an operation-owned validated snapshot.
Never serialize the entire signet tree or reuse its manifest as an export header.
Explicitly require at least one record ID, scope or journal event ID. Union record
and scope selections, deduplicate deterministically, and refuse missing explicit
IDs. An empty existing scope may produce an honest empty report; a nonexistent
scope is a selection error. Conflicting current records are counted/identified
without choosing a winner; history opt-in may include their labeled branches.

Rejected alternatives: copying the bank exports too much and implies restore;
regex or model-based secret scanning cannot guarantee safe disclosure; Markdown
adds rendering/embedded-content ambiguity; in-place redaction destroys source
evidence; broad clipboard/menu infrastructure is unnecessary for this operation.

Redaction uses explicit item omissions plus field categories covering every
source-originated report value: identity, content, classification, timestamps,
authorship, citations, change history and extensions. Unknown categories/IDs are
errors. Every item has a report-local ordinal independent of its original IDs.
Identity omission also omits dependent citations/change edges; content omission
also drops source reasons/citations/change reasons. The preview reports effective
closure, not just the originally requested omissions. Fixed explanatory labels,
counts and report-local ordinals may remain; no hidden original identifier or
content digest is retained in a redacted report.

Provenance, citations, arbitrary extensions and change-history detail are omitted
unless explicitly requested by the corresponding detail/history option. Default
report fields are identity/scope, summary/body, classification and timestamps;
authorship requires detail opt-in. Signet name is not needed in the report.
The preview contains exact source IDs/digests for review and is independently
sensitive even if the eventual report omits them.
