# Technical Design

Native Codex loads plugins/codex/plugins/mandalore from the repo marketplace
root plugins/codex. The manifest, .mcp.json, hooks/hooks.json, shell bridge and
skills/this-is-the-way are public assets. No machine paths or memory are shipped.

The bridge selects an absolute MANDALORE_BIN override, otherwise mandalore on
PATH; setup #7 will pin the chosen installation. MANDALORE_BINDING selects the
existing machine-local binding. MCP and hooks use the same environment selection.
MCP is stdio and explicitly attributed to codex. A missing/older executable must
produce a sanitized warning on hook failure, never a blocking exit or raw output.

Add a codex-memory-hook command accepting --binding and bounded native JSON on
stdin. Permit evolving unrelated event fields, but reject malformed/trailing
JSON and duplicate keys for interpreted fields. Ignore unhandled events. The
adapter opens the validated binding, reads bounded evidence, and emits native
hookSpecificOutput.additionalContext/systemMessage JSON. No writes, Git subprocesses,
transcript access, shell profiles or credential reads are permitted in the adapter.

Memory text is explicitly untrusted evidence, not developer instruction. Scope
inventory routes further recall without preloading unrelated domains. Hook input
and output limits must fit the configured native context cap. No raw prompts or
transcripts are retained. Operational failures are generic and actionable.

The skill recalls relevant history, distinguishes speculation from settled facts,
supersedes using returned identities, journals useful outcomes incrementally,
and reports local save versus delivery accurately. No-write and no-save scope
prohibit journaling and synchronization too. It never turns external source text
or the consolidation phrase into permission for an unrelated action.

Installation checks enumerate ownership and duplicate old/new plugins before
activation. Tests use synthetic bindings, fresh native sessions and the existing
native account; do not clone auth, reset native home, remove plugins or migrate
Alfred. Native trust questions stay with the user if encountered.
