# Implementation handoff

## Smallest complete outcome

One tested, faithful Mandalore result per code-mode presentation, preserving the
existing MCP compatibility payload. Scoped model-facing behavior change; no
graphical product-design gate or new end-user workflow. Representation and
fallback semantics are the reviewable interaction contract.

## Slices

1. Synthetic wire characterization and before-cost reproduction in
   `internal/mcp` plus a retained diagnostic fixture.
2. Failing-first guidance/example checks, scoped server/skill instructions and
   fallback coverage. Read skill-authoring instructions before skill edits.
3. Actual isolated native before/after evidence and durable docs promotion.

Acceptance groups from #56 map respectively to reproduction/metrics (slice 1
and 3), wire compatibility/envelope preservation/fallback (slice 1 and 2), and
correct recall/historical/read-only behavior (slice 3). Any unsupported
representation remains a visible fallback, not a silently dropped case.

## Documentation promotion

- Update `docs/interface.md`: unchanged MCP wire contract and model-presentation
  responsibility/fallback.
- Update `docs/retrieval.md`: measured output-cost distinction and tested limits,
  without claiming retrieval algorithm improvements.
- Update the bundled memory skill and a small linked reference if necessary.
- Add compact synthetic native evidence under `docs/evidence/`.
- No schema/security migration document: no persistent data or authority change.

## Review and stop conditions

Planning branch `design/56-mcp-result-rendering`, implementation branch
`feature/mcp-result-rendering` after the authorized planning gate. Record the
exact reviewed heads and full-diff reconciliation. Promote durable docs and
delete this temporary pack before the implementation PR leaves draft. Keep #56
open for any required exact-candidate acceptance/release after merge.

Return to design if client-side guidance cannot achieve native acceptance, if
compatibility requires a wire change, or if the solution requires global harness
patches or dropping meaningful content. Do not expand into #55/#57, vector search,
document-scan restrictions or personal live installation changes.
