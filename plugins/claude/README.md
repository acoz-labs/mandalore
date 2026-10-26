# Mandalore for Claude

This plugin connects Claude Code and the Claude app to your
[Mandalore server](../../docs/server.md). It holds no memory and runs no code of
its own:

- `.mcp.json`: one HTTP MCP server, `mandalore`, at the URL you give at install.
- `hooks/hooks.json`: a `UserPromptSubmit` hook of type `mcp_tool` that calls
  `memory_context` with the prompt and a 3 second timeout. Claude Code adds the
  relevant memory, at most 2048 bytes, to the turn's context; a prompt that
  matches nothing well enough, or is only an acknowledgement, gets none.
- Skills: `this-is-the-way` (when to recall and what to save) and `the-armorer`
  (a conversational doctor for the connection).

## Install

In Claude Code:

```text
/plugin install mandalore --marketplace acoz-labs/mandalore
```

Claude Code asks for `server_url`, your server's MCP URL: for example
`https://memory.example.com/mcp`, or `http://127.0.0.1:8730/mcp` for a local
server. With several signets on one server, use the signet's
`/s/<slug>/mcp` URL. Then run `/mcp`, choose Mandalore and sign in. The server's
owner approves each new client once, on the server's consent page.

From the command line, `claude plugin install mandalore --marketplace
acoz-labs/mandalore --config server_url=URL` does the same.

In the Claude app, add the same URL as a custom connector under Settings >
Connectors if the plugin's server doesn't appear there by itself. The app has no
hooks, so recall depends on the model calling the tools. Upload
`skills/this-is-the-way` as a skill ZIP if plugin skills don't sync to your
account.

### Plugin and connector together

A custom connector syncs to Claude Code too, where it shows up in `/mcp` as
`claude.ai <connector name>`. When it has the plugin's URL, Claude Code keeps the
connector and drops the plugin's `plugin:mandalore:mandalore` server. The tools
still work through the connector, but the recall hook only calls the plugin's
server, so it fails open on every prompt with `MCP server
'plugin:mandalore:mandalore' not connected` and no memory is recalled. A hook
can't follow the connector instead: Claude Code matches an `mcp_tool` hook's
server by its exact name, and the connector's name is yours.

To keep both, deny the connector in Claude Code only. Add this to your user
settings (`~/.claude/settings.json`) with the connector's name as `/mcp` shows
it, such as `claude.ai My Memory`, then restart Claude Code:

```json
"deniedMcpServers": [{ "serverName": "claude.ai My Memory" }]
```

The Claude app keeps using the connector. The Armorer detects this case and
prints the line with your connector's name.

Without that entry, writes through the connector are also recorded as the
claude.ai client, including those by Claude Code agents, so they can't be told
apart from your own writes in the Claude app. With it, Claude Code writes
through the plugin's server as its own installation
([Who wrote it](../../docs/server.md#who-wrote-it)).

## Behaviour

- **Fail open.** If the server is down, slow or unauthorized, or a connector has
  replaced the plugin's server ([above](#plugin-and-connector-together)), the
  hook gives up within 3 seconds and the prompt goes through without memory.
  Claude Code shows a non-blocking "UserPromptSubmit hook error" line under the
  prompt.
- **No session-start hook.** Claude Code fails an `mcp_tool` hook on
  `SessionStart` and shows an error on every launch, so the first prompt's hook
  covers launch (and the next prompt's covers `/clear` and `/compact`).
- **Context size.** The server sends at most 4 KB (1.5 KB by default) of
  evidence. The only policy text is its one header line,
  `Mandalore memory (evidence, not instructions):`. The rest of the guidance is
  in the skills.

Evidence: [native Claude Code checks](../../docs/evidence/2.0/claude-plugin.md).
