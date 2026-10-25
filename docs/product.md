# Product contract

## Outcome

An existing agent learns useful facts, preferences, decisions, lessons and
procedures during ordinary work. Fresh sessions and different tools/machines
recall the same user-owned knowledge through an explicitly selected signet.
An optional thin launcher selects a named local connection and native harness.
Direct native starts remain supported; no replacement identity, separate inference
account or transferred native thread is required.

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
- Honor requests not to remember or journal particular content. A read-only
  client can't save; the owner sets each client's scope on the server, and
  remembered instructions cannot change it.
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

In 2.0 one [server](server.md) is each signet's only writer, so every client
reads current memory without syncing first. Agents connect over MCP: the
[Claude plugin](../plugins/claude/README.md) in Claude Code, or a connector in the
Claude app. The plugin's hook asks the server for relevant memory on each prompt;
the app has no hooks, so recall there depends on the model calling the tools.
The agent selects useful content to save. Hooks never extract memory from
transcripts.

The server commits and pushes each signet to its Git remote as a backup. A save
is durable once the server confirms it; a failed push never fails a save, and the
receipt reports the backup state. A successful push does not resolve competing
semantic heads.

Fail open, with no client queue: if the server is down or slow, the hook gives
up and the prompt goes through without memory, saves fail with an honest error,
and nothing is stored on the client to retry later. The model must not claim a
save the server didn't confirm.

Disable the plugin or remove the connector to opt out, and revoke the client on
the server to stop its access. Disconnecting cannot remove context already read.
A request to stop Mandalore requires an actual disconnect or revocation, not a
model acknowledgment alone. Installing a runtime does not migrate a signet; the
[1.x cutover](migration-2.0.md) is an explicit admin operation.

The predecessor's arbitrary user-script lifecycle bus is not part of the MVP.
Any future hook extension system requires a separate trust/execution contract.

## Retrieval and references

Begin with scoped lexical retrieval, scope discovery, bounded context and
explicit history access. Measure relevance, corpus growth, repeated reads and
latency before adopting indexing, semantic search or qmd. Derived indexes must
be rebuildable; lexical misses are not proof of absence.

Existing memory/documents can be linked as **foundlings**. **Legacy** foundlings
are explicitly pinned historical reference evidence. **Canon** foundlings track
a Git-hosted signet; the server refreshes them on a schedule and serves the
latest verified commit. Neither mode grants instruction authority or performs a
1:1 migration. The MVP includes a minimal
reference workflow before real-memory adoption (#12). Versioned registrations
and citations belong to the signet; machine-local checkout paths remain local.
Selective promotion preserves original source identity/pin/attribution separately
from the device/time/harness that incorporated the adapted knowledge. Keep
confirmed current knowledge, superseded decisions and unevaluated references
distinct. Capability code and credentials are not memory-import targets.

## Sequence and non-goals

First: memory engine, signet format, CLI/MCP, native Codex integration, everyday
setup/update/recovery and immutable release acceptance. Next: Pi with Codex
continuity. Then: Claude Code after Pi acceptance.

Shared installation UX includes a clear menu, arrows/Vim keys, accessible plain
fallback and equivalent agent-friendly commands. It does not own assistant
orchestration, capability execution, service accounts, credentials, computer
control, inboxes, schedules, package management or remote inference. Procedural
knowledge may be remembered without becoming an executable skill installer.
