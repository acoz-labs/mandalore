---
name: the-armorer
description: Set up, inspect, repair or update Mandalore memory connections and manage historical references conversationally. Use for Mandalore installation or maintenance requests, not ordinary remembering, recall or project work.
---

# The Armorer

Help the user maintain Mandalore through its structured CLI, not menu keystrokes.
This is an on-demand administrative workflow; ordinary memory belongs to
This Is the Way. Do not turn a memory lookup into a setup or repair task.

## Select the actual installation

Resolve this skill's full path from the native skill catalog, including its skill
root alias. Relative links below are relative to this SKILL.md, not the workspace.
Do not search unrelated directories for another installation.

Managed connections include a local `../../connection.json` at the plugin root. If present, read it: version 1 provides `runtime`, `binding`,
`state_dir`, `native_home`, `native_binary` and `connection_root`. It describes
the connection that installed this skill, not every connection on the machine.
Use its retained `runtime` even if Mandalore is absent from PATH. Never execute
the JSON as shell code. This context contains paths, not credentials.

An explicit user-selected administrative target takes precedence, but the attached
MCP and hooks remain pinned to their installed binding and runtime. Environment
overrides do not retarget this Claude Code connection. Reconnect explicitly when
changing the signet or writer; never silently redirect a broken connection.

Public, unconnected plugin packages have no local context file. Use an explicitly
supplied trusted executable or resolve `mandalore` on PATH, then inspect its
version/help and ask only for missing target choices. If no runtime is available,
explain the bootstrap requirement; do not invent a download or repair installation
metadata by hand. The plugin must be installed before it can offer this skill.

## Inspect, then act within the request

Read [administration.md](references/administration.md) for the requested operation.
Discover current schemas with the selected executable's `operations` command.
Capture its JSON locally in the tool call, filter `result.operations` by the
needed `name`, and print only those entries; do not print the entire catalog
before filtering it. Use `call OPERATION` with JSON
stdin, explicit paths and returned IDs. Administration is CLI-only, so it needs
shell access even when the memory MCP is attached. Never assume shell exports
from the separate MCP process are available in the agent's shell.

Start with the checks needed for the request. A request to check or diagnose is
not a request to repair, synchronize or install. Describe the selected target and
effects before applying an authorized change; ask only for missing choices or
authority beyond the request. Preview and apply use the same reviewed plan,
without bypassing ownership, stale-plan or read-only refusals.

For new-machine readiness or an initial connection check, prefer the advertised
`connection_assess` operation. It reads setup metadata without running native
programs or reading remembered content. Native inspection is a separate deeper
check that may create native logs/cache; use the reference to distinguish them.

Keep verification proportional to the changed resources: use structured receipts,
selected-bank inspection and scoped diffs. Do not recursively hash native profiles,
session archives or every retained runtime, or read credential files just to prove
they were preserved. Summarize checks and differences instead of printing file
inventories. Retained copies are not additional targets to inspect on every task.

Report what passed, what failed, what was not tested, and the next useful action.
Separate structural health, live access, local saves and remote delivery. Partial
results require inspection before retrying. Stop retries when the cause has not
changed. Native login and hook trust stay with their native mechanisms; do not
copy credentials or claim a fresh session has already loaded an update.

## Native memory coexistence

Claude Code native memory remains a separate store. Do not silently copy Mandalore records to native notes, import native notes automatically, or recreate withdrawn Mandalore knowledge from notes, journals or history. Revalidate overlapping native assertions against current Mandalore evidence and surface conflicts. This cannot revoke independently retained notes or already loaded context. Do not disable native auto memory or migrate it implicitly.
