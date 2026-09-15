# Handoff

## Smallest complete outcome

An agent or human can start with a compact attributed search page, deliberately
reach every matching document and read the needed source ranges without treating
partial text as a complete claim. Current-memory precedence and all source/pin
checks remain intact. User-requested exhaustive work is not prohibited.

## Slices and traceability

- Retrieval engine and tests: bounded defaults, result continuation, actionable
  too-small budgets, strict fresh verification; fulfills efficient initial access
  and complete later access.
- API/CLI/stdio and menu: shared defaults/flags, provenance-preserving serialization,
  explicit paging/expansion and honest errors; fulfills discoverability and both
  agent/human entrypoints.
- Progressive skill, native matrix and durable docs: later passages, negation,
  multiple sources/current distinctions and deliberate deep review; fulfills
  correctness, nonduplicate reading and actual context evidence.

TDD each meaningful slice, open a draft implementation PR early, then reconcile
the exact implementation head with this plan. Promote contracts to
`docs/foundlings.md`, interface/retrieval docs and the on-demand skill reference;
retain sanitized measurement/terminal evidence under `docs/evidence`. Do not
publish private fixtures or native transcript paths. Remove this temporary plan
only after its content is promoted and reconciliation records the destinations.

Engineering self-review/design judgment follows ADR0003, including the changed
in-pattern terminal interaction and exact-head checks. Product acceptance remains
separate. Keep #57 open after merge for new-artifact acceptance/release. O1/O2 of
#55 remain unresolved and are not silently completed by this independent work.

## Reopen design for

Native quality loss, repeated-read cost defeating the outcome, unavailable complete
source traversal, changed ranking/storage/authorization semantics, or a need for
an index/new backend. Do not add a global scan restriction, mandatory summaries,
provider-specific connector behavior, persistent retrieval cache, daemon or new
capability system. Ordinary implementation refinements within the measured bounds
do not require repeated owner decisions.
