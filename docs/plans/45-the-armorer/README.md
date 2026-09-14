# Solution Design: The Armorer

- **Status:** Final
- **Issue:** #45
- **Planning PR:** #46
- **Repository basis:** 5c61d3587703555401092dbd070e162967e37014
- **Execution envelope:** implementation

## Decision
Add an on-demand administrative skill over existing typed operations. Preserve the passive memory skill and MCP surface.

## Needs Attention
A replacement immutable candidate and fresh affected acceptance are required. The current owner-acceptance exception names only the old candidate; it is not silently extended.

## Decision Spotlight
Generated local context supplies the verified connection's retained runtime and exact paths; the public skill contains no machine paths. CLI armorer is a human-facing alias of the stable connection_doctor operation. Inspection does not imply repair or authenticated health.

## Plan Map
[Context](01-context.md), [Decision](02-decision.md), [Design](03-design.md), [Verification](04-verification.md), [Handoff](05-handoff.md).

## Final Gate
Contributor self-review is authorized for this owner-requested MVP addition under AGENTS.md. No independent approval or release authority is inferred.
