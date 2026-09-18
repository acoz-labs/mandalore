# Context

The actors are users updating through CLI/menu and agents using the Armorer or
typed operations. Their selected profile may have running sessions holding old
cache paths. Issue #112 records real old-hook failure and surviving MCP state.

The existing installer retains its own bundles but native marketplace removal
deletes a different dependency: Codex's installed cache. Replacing the source
without removal is rejected by native Codex 0.154.0. See the evidence, constraints
and scope in [Technical design](03-design.md).

Desired journeys: safe fresh install, unchanged replay, deferred replacement,
explicit stopped-session replacement and inspectable partial-failure recovery.
No signet rewrite, native authentication change or personal activation is needed.
