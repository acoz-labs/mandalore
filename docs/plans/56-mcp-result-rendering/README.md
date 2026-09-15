# Solution Design: Faithful MCP result presentation

- **Status:** Draft
- **Issue:** #56
- **Planning PR:** #58
- **Repository basis:** d50d89a986c7fc8ceb4270fd202cceaa726de8ae
- **Execution envelope:** Pending

## Decision

Keep the MCP compatibility contract intact. Make Mandalore's code-mode guidance
select one complete envelope when the returned text and structured data are
verified equivalent, with lossless fallback for unexpected results. Prove the
behavior with synthetic native evidence before claiming an improvement.

## Needs Attention

The roadmap is authorized, but the repository's engineering self-review exception
names only the original MVP. Independent maintainer review/final product approval
is required unless the owner explicitly extends that exception. Proposed envelope:
`implementation`; no personal plugin update or public release is authorized here.
Native acceptance still needs an available designated test pane and an isolated
synthetic connection. Do not interrupt an unrelated active conversation.

## Decision Spotlight

- Preserve both wire fields: removing compatibility data moves the cost to clients
  that need it and violates the intended MCP fallback contract.
- Model-facing guidance, not a new memory service or global harness patch, is the
  first intervention. It is not a deterministic guarantee that all clients dedupe.
- Unexpected, mixed or conflicting results remain visible, even if verbose.
  Saving tokens cannot erase an error, conflict, provenance or continuation.
- No retrieval-budget changes in this issue; measure those separately in #57.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Draft pending the review authority above and approval of the exact planning head.
Passing CI is not approval. The implementation must not begin by treating this
draft or an issue status as authority.
