# Explicit visibility decisions

Distinguish correction, withdrawal, restoration and deletion. A changed fact
usually needs a superseding revision. A request to stop using a memory can use
`memory_withdraw`; it preserves evidence and is not secure erasure. Ask only when
the target or intended effect is unclear. Do not withdraw merely because a
record is old, irrelevant to this turn, or absent from a narrow search.

If the record ID is no longer in context, use `memory_withheld` in the intended
scope to discover it for this explicit historical inspection or restoration.
Query matches current-head summaries/IDs, not bodies; an empty query lists the
scope's withheld records. Page as needed and inspect candidates rather than
guessing IDs or scanning signet files. Do not use this route merely to fill an
ordinary recall gap: withheld history is not current guidance.

Inspect `memory_visibility_history` for the selected stable record ID. Supply
its current content and visibility heads to `memory_withdraw` or
`memory_restore`, with a reason that need not repeat the content. Inspect content
history when the user actually needs historical detail; never present withdrawn
history as current guidance. Restore only on explicit direction concerning the
reviewed content. It does not resolve conflicting content revisions.

Stale heads require fresh inspection, not resubmission of guessed heads.
Corrections to a withdrawn record remain withdrawn. Do not re-promote the same
foundling origin or create a new record to circumvent that decision. Journals
and external reference files remain independent evidence, not implicitly erased.

Both decisions save locally without delivery. When allowed, use one bounded
`memory_sync` afterward and distinguish local durability from delivery. On
failure inspect `visibility_result`, the event ID and current visibility history
before retrying; a cancellation can occur after publication. Do not repeat the
decision just to deliver it.

These operations require format2. An upgrade-required response is not permission
to migrate the bank, edit its JSON, or weaken an older client. Use The Armorer
only if maintenance is requested, and explain that already-read context and
offline old copies cannot be revoked. Ordinary learning stays enabled for other
confirmed knowledge; withdrawal is not a global read-only mode.
