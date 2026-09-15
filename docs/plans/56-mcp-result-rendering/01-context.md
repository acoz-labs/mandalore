# Context

## Problem and outcome

[Issue #56](https://github.com/acoz-labs/mandalore/issues/56) concerns duplicated
model-visible results, not a request to limit user-directed document exploration.
One retrieval should supply its evidence once while preserving all meaning.

## Verified current state

At the repository basis, `internal/mcp/server.go:New` marshals the shared API
envelope into a single text block and also returns that envelope as
`StructuredContent`. `IsError` is the inverse of the envelope's `OK` flag.
`internal/mcp/server_test.go` verifies CLI/MCP contract agreement and read-only
refusal, but not the model-visible cost of printing the entire MCP wrapper.
The Codex memory skill has no result-presentation guidance.

The MCP tools specification recommends retaining serialized text alongside
structured data for backward compatibility. An output schema also requires
conforming structured results. See the
[2025-06-18 tools contract](https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content).

Live investigation observed code-mode calls printing the whole wrapper, including
the two equal representations. That observation is a hypothesis for a synthetic
reproduction, not publishable private evidence or proof about every MCP client.
Official Codex MCP documentation was inspected but did not establish an API for
this plugin to control code-mode rendering deterministically. Do not invent one.

## Journeys and exclusions

Cover normal memory recall, historical foundling evidence, error/refusal,
truncated/continued results, conflicting heads and pending Git delivery. Preserve
read-only boundaries and the distinction between historical evidence and current
guidance. Exclude new retrieval indexes, bank/schema changes, sync automation,
other plugins, generic output caps, production connection edits and release of
new artifacts. Long legitimate tasks may still compact.
