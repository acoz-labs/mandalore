# Context

Issue #9 asks for evidence that memory stays accurate and lightweight as it grows.
The memory engine, shared CLI/MCP, local-first sync, native hooks/skill and
foundling workflow are implemented at the repository basis. This is the next
engineering slice before distribution and full candidate acceptance.

## Verified current behavior

`internal/memory/memory.go` loads and validates revisions for each recall, selects
effective graph heads before matching, scores each candidate once, and sorts
deterministically. Lexical matching supports a small English inflection fallback
for prose, not synonyms or arbitrary fuzzy identifiers. `Service.Recall` selects
an exact scope, returns whole claims and conflict IDs under a compact JSON budget,
and explicitly reports omissions. Scopes/history load current files too; there
is no persistent index or cross-call cache.

The shared API defaults to five hits and 8192 result bytes. Bound MCP currently
advertises 16 tools with 5966 input-schema bytes, 32678 output-schema bytes and
2488 description bytes, excluding envelopes/transport. `internal/codex/hooks.go`
adds fixed orientation at SessionStart and each UserPromptSubmit; prompt hooks
also recall up to three signet-wide hits under 4096 result bytes and page five
scope entries. The complete emitted hook envelope has a 16383-byte cap.

Foundlings remain separate. Search inspects an explicitly selected pinned corpus
and returns up to five 1024-byte excerpts by default; full source verification
can scan more than the returned excerpt. It is substring-based, unlike ordinary
memory's token/inflection matcher. No-match is not proof of absence.

Actual #12 native runs show successful adaptation/provenance/no-save continuity,
but also a 16000-byte requested recall budget for two records and a full read after
search had already returned the complete reference document. Earlier invalid
empty-query recovery was fixed and retested. These are observations to measure,
not proof that every extra read or larger requested ceiling wastes billed tokens.

## Actors, constraints and unknowns

Users need relevant continuity without stale guidance, project leakage or
unnecessary context. Contributors need repeatable fixtures and raw measurements
that can be compared at a known source head. Native agents retain inference and
semantic judgment; the memory library must not become a model service.

Unknowns are scale curves, dominant filesystem/validation/scoring costs and how
much repeated work matters in real native turns. No private bank or transcript
is required to investigate them. Test counts, schema bytes, result bytes, tool
calls, wall time and model token counters measure different things.

No data format migration, new UI, remote provider, package installation, broad
semantic import or public release is included. Ordinary memory remains current
evidence; historical/foundling text cannot silently acquire current authority.
