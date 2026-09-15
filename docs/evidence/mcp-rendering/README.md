# MCP presentation engineering checks

Development evidence for #56, September 15, 2026. Not native product acceptance
or a new release; immutable v1.0.0 is unchanged.

## What was exercised

`TestPresentationWireCompatibility` calls the actual MCP server through the SDK
using a temporary synthetic signet. It covers remember/recall, invalid input,
scope listing, empty foundling listing and read-only refusal. Both representations
retain the same full envelope and matching error state. The initial test compared
nested raw object order and failed because the SDK materializes structured data
as maps; the corrected characterization canonicalizes recursively with
`json.Decoder.UseNumber`. This was a test-comparison correction, not a product fix
or permission to coerce numbers in the presentation example.

The actual example in the bundled memory skill's `references/code-mode.md` was
extracted verbatim and executed against [selector-checks.js](selector-checks.js)
in both Node.js 24.1.0 (one-off diagnostic) and the active code-mode JavaScript
runtime. All 21 cases passed: equivalent envelopes, pending/error state,
conflicts, truncation, citation/pin preservation, text-only and already-unwrapped
responses, extra content/annotations/metadata, contradictory errors, unknown
versions, rounded large numbers, duplicate JSON keys and non-wrapper values.
No production JavaScript runtime dependency was added.

For the fixed selector fixture, JSON serialization measured 639 UTF-8 bytes for
the wrapper and 257 bytes for the selected envelope. These are representation
bytes, not native input tokens or whole-session savings. The example preserves
input objects and calls no tools, writes or network operations.

Run the Go characterization with:

```sh
mise exec go@1.26.4 -- go test ./internal/mcp -run TestPresentationWireCompatibility -v
```

For the selector, extract the `javascript` fenced block from the skill reference,
evaluate it together with `selector-checks.js` in code mode, and call
`verifyMandaloreOutput(mandaloreOutput)`. Do not substitute a copied selector:
tests must exercise the maintained example. The Go checks run in normal CI;
the JavaScript example check is a separately performed diagnostic.

## Remaining evidence

Fresh isolated native sessions must still demonstrate automatic use of the
guidance, correct recall and historical citation, unchanged no-write fixtures,
and before/after model-visible cost including added instructions/example code.
The existing personal session and connection were not replaced for this test.
No claim is made that all clients preserve object serialization order, optimize
results, or avoid future compaction. Safe fallback may retain duplication.
