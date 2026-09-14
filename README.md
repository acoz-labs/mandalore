# Mandalore

Your lore, across time and space. This is the way.

Automatic, durable memory across agents and tools. Bring your own agent;
Mandalore supplies structured, local-first, Git-backed memory through a shared
engine, CLI, MCP server and thin native harness plugins.

## Status

This is the memory-only successor to [My Friday](https://github.com/acoz-labs/my-friday).
The repository contains the memory engine, a development CLI/local MCP server,
explicit Git synchronization, a development Codex plugin with read-only lifecycle
hooks and memory/administration skills, guided setup/local-artifact update/inspection/repair, explicit
memory-only migration, foundling consultation/selective promotion, retrieval
evaluation and synthetic regression tests. Versioned candidate builds, guided
CLI installation/update and guarded same-byte release tooling are implemented.
Independent exact-candidate acceptance and the first public release remain.
No Mandalore release has been published.

Codex is the first integration. Pi follows accepted Codex support, then Claude
Code follows Pi. Prior prototype evidence informs the port; it does not certify
newly extracted code or renamed artifacts.

## A small vocabulary

- **Mandalore**: the application, CLI (`mandalore`) and native plugin name.
- **Signet**: your private Git-backed memory bank. `signet` is a suggested repository
  name, not a required name or a public repository.
- **This is the way**: the memory skill (`this-is-the-way`) and an optional explicit
  cue to remember confirmed knowledge from the current conversation.
- **Foundlings**: linked historical references, kept separate from current memory.
- **The Armorer**: the on-demand setup and maintenance skill (`the-armorer`),
  including read-only connection inspection (`connection armorer`).

Learning should work without the cue. Memory is revisable evidence, not rules
that override current user direction. Journals are semantic summaries, not a
promise to preserve every word.

## Boundaries

Remember decisions, preferences, facts, lessons and procedures with scope,
source, time, device/harness provenance and explicit supersession. Keep native
tools, skills, model access, authentication and working directories intact.

This is not an assistant builder, credential vault, capability marketplace,
account-role framework, scheduler or computer-control system. Native thread
portability is not included.

## Documentation

- [Product contract](docs/product.md)
- [Architecture and implementation status](docs/architecture.md)
- [Signet data format](docs/signet-format.md) and [source extraction](docs/memory-extraction.md)
- [Development CLI, local MCP and machine bindings](docs/interface.md)
- [Git synchronization, offline work and freshness](docs/synchronization.md)
- [Guided setup and CLI installation](docs/setup.md) and [historical foundlings](docs/foundlings.md)
- [Codex plugin](plugins/codex/README.md) and [native engineering evidence](docs/codex-native-evidence.md)
- [Migration and predecessor disposition](docs/migration.md)
- [Roadmap](docs/roadmap.md) and [issues](https://github.com/acoz-labs/mandalore/issues)
- [Development](docs/development.md), [releases](docs/deployment.md), [runbook](docs/runbook.md)
- [Contributing](CONTRIBUTING.md) and [security/privacy](SECURITY.md)

Run `mise exec go@1.26.4 -- bin/ci`, or `bin/container bin/ci` with Docker.
The managed workflow uses issues, design/implementation PRs, recorded engineering
review, exact-candidate acceptance and GitHub Releases. The current MVP permits
[engineering self-review](docs/decisions/0001-mvp-self-review.md), not fabricated
independent acceptance. See [SDLC](docs/operations/sdlc.md).

For prerelease evaluation, use a reviewed local candidate from the
[pinned builder](docs/development.md#local-distribution-candidates), then run its
platform executable with `release install --candidate /absolute/candidate-dir`.
The installer previews its destination and defaults to No. CLI installation and
connecting a chosen signet to Codex require separate confirmation; no shell,
credential or memory migration is implicit. There is no published version to
download until the release prerequisites are met.

Independent community project; not affiliated with or endorsed by Lucasfilm or
Disney. The theme is naming and prose; no official artwork is bundled.
