# Mandalore

Your lore, across time and space. This is the way.

Automatic, durable memory across agents and tools. Bring your own agent;
Mandalore supplies structured, Git-backed memory through a shared engine, a
single-writer MCP server, an admin CLI and a thin Claude plugin.

## Status

This is the memory-only successor to [My Friday](https://github.com/acoz-labs/my-friday).
`main` is 2.0 development. The [Mandalore server](docs/server.md)
(`mandalore serve`) is each signet's only writer and backs it up to its Git
remote. Agents read and write through it over MCP: the
[Claude plugin](plugins/claude/README.md) in Claude Code, or a connector in the
Claude app. If the server is unreachable, agents carry on without memory (fail
open). The repository also contains the memory engine, the admin CLI and guided
menu, the [1.x cutover](docs/migration-2.0.md), explicit memory-only migration,
foundling consultation/selective promotion, retrieval evaluation, versioned
candidate builds with guarded same-byte release tooling, and synthetic
regression tests.

The latest release is [2.0.3](docs/releases/2.0.3.md).
In 1.x each device synchronized its own Git clone around agent sessions,
through a stdio MCP server, lifecycle hooks and native Claude Code, Codex, Pi
and Hermes plugins; the [release records](docs/releases/) keep that history.
2.0 removes them, along with Razor Crest. Codex, Pi and Hermes plugins are
paused until they're ported to the server model.

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
- [Admin CLI, operation catalog and machine bindings](docs/interface.md)
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

The latest release is [2.0.3](docs/releases/2.0.3.md); the
[release records](docs/releases/) describe the 1.x installation. In 2.0 one [server](docs/server.md) owns each signet and agents
connect to it over MCP:

1. **Get `mandalore`.** Install it with the release's bootstrap
   ([Installing 2.0](docs/releases/2.0.0.md#installing-20); for an existing
   server, see [Upgrading to 2.0.3](docs/releases/2.0.3.md#upgrading-a-running-launchd-server)):

   ```sh
   curl -fsSLO https://github.com/acoz-labs/mandalore/releases/download/v2.0.3/install.sh
   sh install.sh 2.0.3
   ```

   Or build the server image locally from [packaging/server](docs/server.md#docker).
   A published image will follow
   ([#188](https://github.com/acoz-labs/mandalore/issues/188)).
2. **Prepare the signet.** The server needs its own checkout. Coming from 1.x,
   follow [the cutover](docs/migration-2.0.md); for a new signet, see
   [A new signet](docs/server.md#a-new-signet).
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
   Using both? A connector with the plugin's URL replaces the plugin's server in
   Claude Code and stops its recall hook; deny it there as the
   [plugin guide](plugins/claude/README.md#plugin-and-connector-together) shows.

Independent community project; not affiliated with or endorsed by Lucasfilm or
Disney. The theme is naming and prose; no official artwork is bundled.
