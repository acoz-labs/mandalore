# Context

Issue #6 connects the merged CLI/MCP and local-first synchronization to native
Codex without replacing native tools, authentication, cwd or session handling.
The current binary has no hook command or plugin package. The installed CLI
reports 0.153.4; that is a test target, not a claim about every Codex surface.

Relevant current ownership: internal/binding, internal/memory, internal/api,
internal/mcp, cmd/mandalore, plugins/codex. Pinned public predecessor
f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8 has internal/memorycodex/hooks.go and
plugins/codex/plugins/my-friday-memory; adapt reviewed memory-only behavior,
not the legacy assistant/capability framework or private installation.

Users should start native Codex in any project, recall prior decisions, learn
confirmed changes incrementally, and see truthful local/delivery status.
Read-only tasks must leave memory and Git state unchanged. Missing connections
must warn without preventing unrelated work. Multiple signets remain isolated.

Official packaging supports the scaffold compatibility manifest and default
hooks/hooks.json discovery. Hook trust is separate from plugin installation.
Current documentation has broader lifecycle support than historical versions;
verify startup, resume, prompt, compaction and interruption on the actual CLI.

Sources: [packaging](https://developers.openai.com/plugins/build/plugins),
[hooks](https://learn.chatgpt.com/docs/hooks). Plugin and skill creator references
guide scaffold/validation; native execution remains acceptance evidence.
