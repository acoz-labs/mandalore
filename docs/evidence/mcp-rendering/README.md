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

## Native engineering evidence and remaining acceptance

The [first-result follow-up](first-result.md) retains a later candidate sequencing
failure and four fresh targeted runs of the corrected guidance. It requires
preparing selection before the first raw call, including the first error. Neither
that improvement nor the earlier measurements guarantees lower whole-task cost.

The [native comparison](native.md) records four fresh baseline/candidate sessions
and a candidate error/recovery session. The candidate selected complete envelopes
without a presentation-specific user prompt, preserved correct recall and
historical citations, and left fixture hashes unchanged. Instructions and helper
cost are included; reduced final context did not consistently reduce aggregate
input tokens. These are contributor observations, not product acceptance.

Coverage is deliberately layered: the SDK test exercises actual success/error
wire results; native runs exercise current/history, truncated foundling excerpts
and error recovery; synthetic selector cases cover conflicting heads, pending
delivery and unexpected wrappers. The latter are not claims of native conflict
resolution or real remote-delivery acceptance. No Git remote was configured for
the native fixture. This is narrower than repeating every wire scenario in the
plan; the shared envelope contract and existing service tests remain unchanged.

The existing personal session and connection were not replaced for these tests.
No claim is made that all clients preserve object serialization order, optimize
results, or avoid future compaction. Safe fallback may retain duplication.
New artifact nomination, exact-candidate acceptance and release remain required;
the native development executable was not stamped as a release candidate.
