# Decision and user journey

An in-place rename is not a migration: it can strand existing bindings and
create mixed writers. A bulk semantic import would confuse historical reference
with current knowledge and cannot preserve exact revision identity. Select an
explicit structural conversion for the actual memory-only format, separately
from foundling consultation/promotion.

## Product-design contract

Use existing JSON CLI patterns: migration preflight and migration apply, with
equivalent typed administration operations in the catalog. No new menu renderer
or bound memory MCP mutation is introduced. Classification: no-rendered-impact;
this slice adds structured receipts and setup documentation, not a new TUI.

The caller explicitly selects a source directory and a new output bundle path,
optionally an existing legacy binding and native inventory profile/binary.
It also supplies an explicit conversion machine label and actor; neither is
inferred from the host name or authenticated provider account.
Preflight names exact local paths, source identity/fingerprint, schema support,
counts, conflicts, exclusions, snapshot limits and observed writer risks. It
does not execute the predecessor or guess paths from cwd, shell aliases or memory.

The apply command consumes the reviewed plan and an explicit writers-stopped
acknowledgement. One deliberate operation creates the output bundle; it does
not enroll a writer or switch native integrations. Explain three next steps:
inspect the converted bank/history, create a new machine binding, then perform
a separately reviewed fresh-session handoff without duplicate memory writers.

Outcomes distinguish refusal, unpublished staging failure and published output.
Partial results retain exact recovery paths; no automatic deletion of a published
bundle or user source. Rollback before activation means continuing with the
untouched original writer/source. After new writes, preserve both histories and
reconcile deliberately; no automatic reverse conversion or merge is promised.

This is format conversion, not semantic endorsement. Existing memory remains
evidence subject to current user direction. People bringing an assistant repo
or arbitrary historical notes use the #12 foundling route, not a renamed manifest.
