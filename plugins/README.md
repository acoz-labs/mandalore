# Native integrations

Each harness gets a thin adapter over the same Mandalore memory engine.
Do not duplicate storage, retrieval or supersession semantics.

- [Codex](codex/README.md): first integration, released.
- [Pi](pi/README.md): released after Codex acceptance, issue #13.
- [Claude Code](claude-code/README.md): development integration after Pi
  acceptance, issue #14; acceptance and publication remain separate.

All integrations reuse the shared runtime, explicit signet binding and memory
semantics. Each has native setup, inspection and recovery paths. All plugins use
the display name Mandalore; native identifiers use `mandalore`, and the principal
memory skill is `this-is-the-way`. Native models, authentication and tools remain
with the selected harness.
