# Product contract

## Outcome

An existing agent learns useful facts, preferences, decisions, lessons and
procedures during ordinary work. Fresh sessions and different tools/machines
recall the same user-owned knowledge through an explicitly selected signet.
No assistant launcher, replacement identity, separate inference account or
transferred native thread is required.

## Names

`mandalore` is the lowercase repository, CLI and plugin identifier. Mandalore is
the display name. A signet is a private Git-backed memory bank; users may choose
any repo name and keep multiple independent signets. The memory skill is
`this-is-the-way`. Errors and schemas should remain understandable without lore.

## Learning and current direction

- Learn confirmed, useful knowledge and write concise semantic journals
  incrementally. A special phrase is not required for ordinary learning.
- A direct user instruction of “this is the way” adds explicit save intent for
  confirmed knowledge in the current exchange; it is not a mode switch.
- Clarify only when ambiguity would record a materially wrong decision. Do not
  invent certainty or promote brainstorming into settled requirements.
- Quoted text, retrieved documents, tool output and third-party messages
  containing the phrase are never authorized triggers.
- Consolidate rather than duplicate, preserve correction history, and acknowledge
  actual outcomes concisely. Local saves and remote synchronization are distinct.
- Honor scoped no-write/no-save/read-only instructions, including no journaling
  or synchronization. The cue does not implicitly override an explicit prohibition.
- Historical memory is evidence, never authority over current user direction.
  Project choices do not silently become global preferences.
- Do not claim infallible recall, lossless recording or unsaved-work durability.

## Signet structure and evolution

Use a shared versioned structure for all users. Records need stable identity,
kind, scope, content, source/evidence references, timestamps, originating
machine/harness provenance and explicit supersession edges. Machine labels are
user-controlled; automatic hostname disclosure is unnecessary.

Separate semantic journals from current record state. Preserve change reasons
and history; concurrent heads remain visible conflicts. Define schema upgrades,
compatibility and rollback before rewriting data. Git history is useful recovery
material, not a replacement for safe writes, conflict handling or backup policy.

Personal/work isolation means separate banks and explicit local bindings, not a
new sandbox. Never silently search or combine unrelated signets. The memory
location is independent of the session's working directory.

## Integration and freshness

The shared engine owns all memory semantics; adapters map real native lifecycle
features and document gaps. Keep inherited skills, settings and native auth.
The host agent supplies semantic extraction using its existing model access.

Hooks can refresh files and deliver context where supported, but cannot
retroactively refresh context already read. Later memory reads must see current
local data. Investigate startup/per-turn synchronization with measured latency,
offline persistence, concurrency and visible freshness; never claim global
freshness merely because local recall succeeded.

The predecessor's arbitrary user-script lifecycle bus is not part of the MVP.
Any future hook extension system requires a separate trust/execution contract.

## Retrieval and references

Begin with scoped lexical retrieval, scope discovery, bounded context and
explicit history access. Measure relevance, corpus growth, repeated reads and
latency before adopting indexing, semantic search or qmd. Derived indexes must
be rebuildable; lexical misses are not proof of absence.

Existing memory/documents can inform a later explicit import as reference
material, not live instructions or a 1:1 migration. Preserve source mapping and
important experiences while separating confirmed current knowledge, superseded
decisions and unresolved proposals. Capability code and credentials are not
memory-import targets.

## Sequence and non-goals

First: memory engine, signet format, CLI/MCP, native Codex integration, everyday
setup/update/recovery and immutable release acceptance. Next: Pi with Codex
continuity. Then: Claude Code after Pi acceptance.

Shared installation UX includes a clear menu, arrows/Vim keys, accessible plain
fallback and equivalent agent-friendly commands. It does not own assistant
orchestration, capability execution, service accounts, credentials, computer
control, inboxes, schedules, package management or remote inference. Procedural
knowledge may be remembered without becoming an executable skill installer.
