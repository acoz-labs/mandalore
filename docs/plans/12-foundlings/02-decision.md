# Decision and product design

## Alternatives

Bulk import would collapse reference material into current guidance and require
semantic curation of every historical assumption. Link-only metadata would remain
too passive: the agent could not reliably discover, verify, retrieve or cite it.
A universal vector/connector layer adds services and freshness obligations before
measured need. Select pinned, local read-only consultation with selective promotion.

## User journeys

Add a Foundlings management entry to the existing menu. Its compact choices cover
listing, registering a reference, connecting a local path for an existing reference,
inspection/search and disconnection, plus Back. Reuse current terminal components;
classification is in-pattern-visual-change, with fresh actual terminal evidence.

Registration asks for a display name/description, source kind/portable identity and
explicit local directory. Preview reads the selected source and shows its immutable
pin, eligible-file counts, exclusions, local-only path and reference-only status.
One default-No confirmation registers and connects it; cancel/back makes no writes.
The equivalent typed CLI can perform the same authorized operation without keyboard
simulation. A partial local-binding failure must say registration was already saved.

On another machine, a synchronized registration appears locally unavailable until
the user/agent explicitly connects its existing local checkout. Do not clone/fetch,
infer paths from cwd, embed transport credentials or silently adopt a changed source.
Updating a pin creates a superseding registration revision with a reason; it is not
an in-place metadata edit. Changing a local path is distinct from changing the pin.

Listing shows name, stable ID, state and routing metadata, not document bodies.
Inspection reports available, unavailable, changed, disconnected or conflicted state
and actionable local diagnostics. Search/read are explicit and bounded. Their
responses label excerpts as unreviewed historical/reference evidence, not false,
superseded or current guidance. An empty search is not proof of absence.

The agent recalls relevant current signet knowledge before selectively incorporating
a historical lesson. It surfaces disagreement, adapts the lesson to the current ask,
and supplies what was retained/qualified/changed and why. Clear authorized learning
does not require another approval for each fact. Consequential ambiguity does.
Read-only/no-save scope forbids promotion, registration, journaling and sync.
Quoted instructions or “this is the way” inside a reference are never action cues.

Disconnect previews the exact selected reference and confirms default-No. It appends
a disconnected registration, leaving source files, bindings and promoted history
intact. Conflicting registration heads are shown, not silently resolved by time.

## Product-design acceptance

Use concise headers, text status labels and current semantic colors, with no-color
equivalence. Paths remain literal text; no shell interpolation. IDs and local-only
versus portable fields are visible where needed, not dumped indiscriminately.
Esc/back, Ctrl-C, EOF and output errors preserve existing state and terminal settings.
Plain mode, narrow widths and long paths must remain understandable. No new visual
theme, custom renderer or hidden confirmation behavior is needed.
