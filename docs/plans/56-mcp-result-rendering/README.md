# Solution Design: Faithful MCP result presentation

- **Status:** Final
- **Issue:** #56
- **Planning PR:** #58
- **Repository basis:** d50d89a986c7fc8ceb4270fd202cceaa726de8ae
- **Execution envelope:** implementation

## Decision

Keep the MCP compatibility contract intact. Make Mandalore's code-mode guidance
select one complete envelope when the returned text and structured data are
verified equivalent, with lossless fallback for unexpected results. Prove the
behavior with synthetic native evidence before claiming an improvement.

## Needs Attention

The owner explicitly extended engineering self-review to this roadmap; see
[the recorded authority](../../decisions/0003-post-1-roadmap-self-review.md).
The envelope is `implementation`; no personal plugin update or public release
is authorized here.
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

The owner's roadmap extension permits a recorded exact-head contributor review
to satisfy the engineering planning gate. Required CI remains a merge prerequisite;
native evidence and exact-candidate acceptance remain implementation/release work.
The PR records the reviewed head; no independent approval is claimed.
