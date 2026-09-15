# Solution decision

## Alternatives

1. Remove text or structured data at the server: mechanically reduces bytes but
   breaks backward compatibility or the declared output schema. Rejected.
2. Truncate or summarize the payload: loses exact evidence and mixes this issue
   with retrieval design. Rejected.
3. Mandalore-scoped presentation guidance with protocol regression tests: lowest
   implementation cost, retains clients and memory semantics. Selected first.
4. Change the native harness renderer: potentially deterministic and broader, but
   outside this repository and the authorized scope. If guidance fails supported
   native acceptance, retain a minimal public synthetic reproduction and return
   the decision to review rather than silently patching the harness.

## Selected approach and tradeoffs

Use a short instruction visible with Mandalore's MCP server instructions and
memory skill. Explain that the complete envelope, not only `.result`, is the unit
to render. Supply one small code-mode example with conservative fallback in a
linked skill reference if the full example is too costly for the startup budget.
Do not introduce a new package, JavaScript runtime dependency, daemon or tool.
Keep always-loaded guidance short (at most 120 added words per instruction
surface); load detailed fallback examples only when needed. Measure added
instruction/example cost as well as response savings so a large selector does
not become a new context problem.

Selection is conditional on the known Mandalore wire contract and equivalent text.
Unknown wrapper fields/content, annotations carrying distinct information,
malformed data and contradictory error status must not be silently removed.
Model adherence must be measured through actual native runs, not inferred from a
test that merely searches the skill for a phrase.

The guidance is client-side and best-effort; a protocol regression test is
deterministic. Keep those evidence classes separate in documentation.
