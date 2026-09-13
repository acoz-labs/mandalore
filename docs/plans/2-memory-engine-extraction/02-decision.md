# Solution decision

## Alternatives

| Approach | Benefit | Material cost |
| --- | --- | --- |
| Copy and rename all of My Friday | Fast apparent parity | Retains assistant framework, private-history risk and ambiguous compatibility |
| Rewrite memory from scratch | Clean naming | Discards tested graph/storage/recall failure boundaries |
| Extract characterized memory primitives | Reuses evidence while narrowing scope | Requires deliberate dependency and format review |

Select the third approach. Compile and run failing-first characterization tests
around small extracted components. Keep mechanical moves separate from behavior
changes so differences can be reviewed.

## Proposed format decision for #3

New signets use a versioned `signet.json` manifest. Preserve record/revision/source/
device IDs as opaque stable identifiers, not strings to rename for branding.
New bank-wide scopes use kind `signet`; project/account/task scopes retain their
meaning. New local locks/state use `.mandalore`, explicitly ignored as appropriate.
Do not open or write legacy `bank.json`/`agent.json` stores through the new writer.

This deliberately makes compatibility explicit. #8 must offer an inspectable
import/conversion or clear refusal, preserving IDs and history and translating
scope fields consistently. Never let old and new writers share a store while
using different lock names. Existing users remain on their old runtime until an
explicit migration is supported and selected.

The final review may choose a narrower compatibility envelope, but must resolve
these names before code writes signets. Do not silently alter the contract in
implementation or call an unreviewed format “backward compatible.”

## Other decisions

Preserve current-revision/conflict semantics, bounded lexical recall, structured
sources and semantic journals. No derived index or transcript archive. Device
labels are explicit, not auto-populated with private hostnames. Keep the storage
API independent of native harness integration and external credentials.

## Foundling data decision

Foundling registrations are versioned reference metadata, not memory records.
Keep immutable registration revisions so source pins and labels can evolve or
be disconnected without erasing the meaning of citations already promoted.
Unevaluated material is neither current guidance nor automatically superseded.

Promotion creates an ordinary signet revision with the importing device/harness
as its incorporation authorship. Its source evidence separately identifies the
foundling, registration revision, original content location/fingerprint and
original time/author when known. Unknown original attribution stays unknown.
No source text becomes an instruction merely because it is registered locally.
