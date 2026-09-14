# Solution Design: foundling reference workflow

- **Status:** Final
- **Issue:** #12
- **Planning PR:** #31
- **Repository basis:** ad621e58ba65eaf2cfcb66a73b3801ff4ccebb8d
- **Execution envelope:** implementation

## Decision

Make the existing portable registration/citation model usable through bounded
read-only source adapters, shared agent operations and the management menu.
Consultation is reference evidence; incorporation is an explicit semantic write
performed by the host agent under the current user's authorized learning scope.

## Decision Spotlight

- Preserve the existing version-1 registration and external-origin schema. Local
  checkout paths live in ignored clone-local state, never portable registrations.
- Support explicit existing Git checkouts and local text directories, without
  cloning, fetching, executing source code or copying sources into the signet.
- Git pins identify commits; inspect the pinned tracked text against local files.
  Changed, missing, disconnected or conflicting references do not masquerade as
  available pinned evidence. Untracked/binary content is not implicitly consulted.
- Add separate bounded foundling list/inspect/search/read/promotion tools; ordinary
  memory recall remains current signet evidence only. Registration/path management
  also has typed CLI administration and a small in-pattern menu flow.
- Promotion verifies a selected citation and uses the existing sourced-memory
  writer. The host agent handles meaning, disagreement and scoped supersession;
  no rule engine, inference service or per-fact approval ceremony is introduced.
- Disconnect appends registration history. It does not delete sources, local
  bindings, prior citations or promoted knowledge.

## Needs Attention

Native behavioral and actual terminal evidence are required before engineering
readiness. Independent exact-candidate acceptance remains #10/#11. Real historical
sources, snapshots, bulk import and later harnesses remain outside this change.

## Plan Map

- [Context](01-context.md)
- [Decision and product design](02-decision.md)
- [Technical design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)
