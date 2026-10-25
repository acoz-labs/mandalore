# Mandalore

Your lore, across time and space. This is the way.

Automatic, durable memory across agents and tools. Bring your own agent;
Mandalore supplies structured, local-first, Git-backed memory through a shared
engine, CLI, MCP server and thin native harness plugins.

## Status

This is the memory-only successor to [My Friday](https://github.com/acoz-labs/my-friday).
The repository contains the memory engine, a CLI/local MCP server,
Git synchronization, native lifecycle integration and memory/administration skills, guided setup/local-artifact update/inspection/repair, explicit
memory-only migration, foundling consultation/selective promotion, retrieval
evaluation and synthetic regression tests. Versioned candidate builds, guided
CLI installation/update and guarded same-byte release tooling are implemented.
[Mandalore v1.3.0](https://github.com/acoz-labs/mandalore/releases/tag/v1.3.0)
is published from the independently accepted retained candidate. See the
[release record](docs/releases/1.3.0.md) for provenance, verification and limits.

`main` is now 2.0 development: the single-writer [Mandalore server](docs/server.md)
(`mandalore serve`) replaces the stdio MCP server, lifecycle hooks, session
synchronization and Razor Crest, and one [Claude plugin](plugins/claude/README.md)
replaces the 1.x Codex, Pi, Claude Code and Hermes plugins. The rest of this
section describes the released 1.x line; [Install](#install) covers 2.0.

Claude Code, Codex and Pi integrations are released in 1.x, where each device
synchronized its own Git clone; the [release records](docs/releases/1.3.0.md)
keep that history. In 2.0 agents read and write through the server, which is
the only writer and backs each signet up to its Git remote. If the server is
unreachable, agents carry on without memory (fail open).

Hermes support was released in [1.6.0](docs/releases/1.6.0.md).
It adds portable signet memory alongside Hermes's native memory, with native
CLI and persistent-service acceptance. Install and activate a reviewed connection
for the selected profile; publication does not automatically upgrade operators.

## A small vocabulary

- **Mandalore**: the application, CLI (`mandalore`) and native plugin name.
- **Signet**: your private Git-backed memory bank. `signet` is a suggested repository
  name, not a required name or a public repository.
- **This is the way**: the memory skill (`this-is-the-way`) and an optional explicit
  cue to remember confirmed knowledge from the current conversation.
- **Foundlings**: linked historical references, kept separate from current memory.
- **The Armorer**: the skill (`the-armorer`) that diagnoses a client's connection
  to the server.
- **Server**: `mandalore serve`, the 2.0 [single-writer MCP server](docs/server.md)
  that replaces the Razor Crest candidate.

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
- [Mandalore server (2.0 development)](docs/server.md), its [design](docs/design/2.0.md) and the [cutover from 1.x](docs/migration-2.0.md)
- Deploying the server: [work laptop on loopback](docs/deploy/local.md) and [Cloudflare Tunnel](docs/deploy/cloudflare-tunnel.md)
- [Privacy, retention and the limits of forgetting](docs/privacy.md)
- [Guided setup and CLI installation](docs/setup.md) and [historical foundlings](docs/foundlings.md)
- [Claude plugin](plugins/claude/README.md) and its [native evidence](docs/evidence/2.0/claude-plugin.md)
- [Migration and predecessor disposition](docs/migration.md)
- [Roadmap](docs/roadmap.md) and [issues](https://github.com/acoz-labs/mandalore/issues)
- [Development](docs/development.md), [releases](docs/deployment.md), [runbook](docs/runbook.md)
- [Contributing](CONTRIBUTING.md) and [security/privacy](SECURITY.md)

Run `mise exec go@1.26.10 -- bin/ci`, or `bin/container bin/ci` with Docker.
The managed workflow uses issues, design/implementation PRs, recorded engineering
review, exact-candidate acceptance and GitHub Releases. The current MVP permits
[engineering self-review](docs/decisions/0001-mvp-self-review.md), not fabricated
independent acceptance. See [SDLC](docs/operations/sdlc.md).

## Install

Mandalore 2.0 is in development on `main` and not released yet; the latest
release is [1.6.1](docs/releases/1.6.1.md), whose release records describe the
1.x installation. In 2.0 one [server](docs/server.md) owns each signet and agents
connect to it over MCP:

1. **Get `mandalore`.** Until 2.0.0 is published, build it from this repository
   with `mise exec go@1.26.10 -- go build -o mandalore ./cmd/mandalore`, or build
   the server image with `docker build -f packaging/server/Dockerfile .`. From
   2.0.0 each release carries the binaries and the image
   `ghcr.io/acoz-labs/mandalore:<version>`.
2. **Prepare the signet.** The server needs its own checkout. Coming from 1.x,
   follow [the cutover](docs/migration-2.0.md).
3. **Configure and run the server.** Write its [configuration](docs/server.md#configuration),
   then run it as a service: `mandalore server install-launchd --config FILE`
   on macOS, or the [Docker image](docs/server.md#docker).
4. **Choose where it listens.** A [work laptop](docs/deploy/local.md) keeps it
   on loopback; a [home server](docs/deploy/cloudflare-tunnel.md) puts it behind
   Cloudflare Tunnel for Claude on every device.
5. **Connect Claude.** Install the [Claude plugin](plugins/claude/README.md) with
   `/plugin install mandalore --marketplace acoz-labs/mandalore`, or add a
   connector in the Claude app, and give it the server's MCP URL,
   such as `http://127.0.0.1:8730/mcp` locally or `https://memory.example.com/mcp`
   through a tunnel. With OAuth, approve the client with your owner passkey.

Independent community project; not affiliated with or endorsed by Lucasfilm or
Disney. The theme is naming and prose; no official artwork is bundled.
