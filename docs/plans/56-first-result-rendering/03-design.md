# Design

Only bundled Codex `this-is-the-way/SKILL.md` and its `references/code-mode.md`
need instruction changes. The first remains short and conditional: prepare the
selector before the first raw MCP call, apply it to that response and subsequent
responses, and preserve error/provenance/fallback semantics. Define/read guidance
before calling; do not first print a result while loading the reference later in
the same orchestration batch. Existing direct/native single-render clients skip
this step. Tool discovery is not a memory call and may still happen first.

The selector code and MCP response schema are unchanged. No server/session
state, new dependency, tool, permission, hook or persistence change. Safe fallback
may remain verbose when equivalence cannot be proved. Reuse prepared selection;
do not recall/write twice just to change display.
