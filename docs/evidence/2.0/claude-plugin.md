# Claude plugin native checks (#163)

Engineering evidence for the 2.0 Claude plugin, recorded 2026-10-24 and
2026-10-25 with Claude Code 2.1.310 on macOS (arm64). It is not release
acceptance; #166 runs that over the tunnel with real clients. Every run used a
synthetic signet in a new temporary directory and a temporary `HOME`. The only
link back to the real home was the login keychain, so Claude Code could sign in.
No real signet, binding or hostname was involved.

## Step 0: the `mcp_tool` spike

The spike ran before the rest of the work, against `mandalore serve` from `main`
(#167) with `loopback-trust` and one seeded synthetic fact. It also used a small
MCP stub, [`claude-plugin/stub-server.py`](claude-plugin/stub-server.py), for the
reply shapes, stalls and errors the real server doesn't produce on demand. It
found two departures from design section 4, and the lead decided both on the
issue:

| Check | Result |
| --- | --- |
| `${prompt}` substitutes | Yes. The server's `tools/call` received `{"event":"prompt","prompt":"<the exact prompt>"}`. |
| Output reaches context | **Only as JSON.** A plain-text result is recorded as `hook_success` with content `UserPromptSubmit completed` and isn't shown to the model: it answered "I don't know". A `command` hook in the same run printing plain text was injected. When the result's text is `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"..."}}`, Claude Code records `hook_additional_context` and the model answers from it. |
| Stalled server, `timeout: 3` | The hook was cancelled 3.016 s after `tools/call` (`hook_cancelled`, `timedOut: true`, `timeoutMs: 3000`). The server got `notifications/cancelled` and the prompt went through. |
| What the user sees on failure | A non-blocking line under the prompt: `UserPromptSubmit hook error`, then `MCP server 'plugin:mandalore:mandalore' not connected` (server down) or the tool's error text (`isError`). The model still answers. |
| `SessionStart` | **Fails on every launch.** At startup, and on `-p --resume`, Claude Code reports `mcp_tool hooks are not available for the 'SessionStart' hook event (no MCP client context)` and shows `SessionStart:startup hook error`. A `resume\|clear\|compact` matcher doesn't suppress it for `mcp_tool`, though the same matcher on a `command` hook does. On `/clear` it ran and injected context. |

Decisions (design section 4 carries a dated note for each):

- **A.** `memory_context` always returns the hook output envelope.
  `hookEventName` comes from `event` (`prompt` → `UserPromptSubmit`,
  `start` → `SessionStart`), and `additionalContext` holds the text. Nothing
  relevant still gives an empty result. The 4 KB cap and the policy budget apply
  to `additionalContext`.
- **B.** The plugin ships no `SessionStart` hook. The first prompt's hook covers
  launch, and the next prompt's covers `/clear` and `/compact`. The server still
  accepts `event: "start"`.

No `http`-hook fallback was needed. The hook runs through the plugin's own MCP
connection.

## Acceptance runs

[`claude-plugin-native.sh`](claude-plugin-native.sh) reproduces these runs:

```sh
go build -o /tmp/mandalore ./cmd/mandalore
docs/evidence/2.0/claude-plugin-native.sh /tmp/mandalore /tmp/mandalore-evidence
```

It creates the synthetic signet and seeds one fact. It installs the plugin from
this checkout's marketplace with `--config server_url=http://localhost:28730/mcp`
and runs `claude -p` once per scenario. Each server is stopped by its PID, and
the test server is signed out of Claude Code's OAuth store before and after.
[`claude-plugin-summarize.py`](claude-plugin-summarize.py) reduces each run's
stream and transcript to the JSON files in [`claude-plugin/`](claude-plugin/).

| Criterion | Run | Result |
| --- | --- | --- |
| 1. `claude plugin validate --strict` | [plugin](claude-plugin/validate-plugin.txt), [marketplace](claude-plugin/validate-marketplace.txt) | Both pass. The only notice is the directory-listing check for a missing icon, which doesn't affect the verdict. |
| 2. Seeded fact in a fresh session's first prompt | [`seeded.json`](claude-plugin/seeded.json) | `hook_additional_context` 0.009 s after the prompt holds the seeded fact. The model answered from it, without tools. Injected: 385 bytes in total, of which the policy header is 46. |
| 3. Server stopped | [`stopped.json`](claude-plugin/stopped.json) | `hook_non_blocking_error` (`not connected`) after 0.003 s. The prompt was answered (`ok`). |
| 3. Server stalled (`SIGSTOP`) | [`stalled.json`](claude-plugin/stalled.json) | The MCP connection stayed pending. The hook gave up 3.005 s after the prompt, within timeout + 1 s, and the prompt was answered. The spike above covers a stall after connecting (3.016 s). |
| 5. Armorer, server stopped | [`armorer-stopped.json`](claude-plugin/armorer-stopped.json) | "Your Mandalore memory isn't working: I can't reach the server at `http://localhost:28730/mcp`." Then server, URL and tunnel checks, `/mcp`, and the fail-open note. |
| 5. Armorer, unapproved client (OAuth server, never authorized) | [`armorer-unapproved.json`](claude-plugin/armorer-unapproved.json) | "This client isn't authorized yet." Then `/mcp` sign-in and the owner's one-time consent. |
| 5. Armorer, read-only client (`local_scope: read`) | [`armorer-read-only.json`](claude-plugin/armorer-read-only.json) | "Diagnosis: This client is read-only." It called `memory_scopes`, found no `memory_remember`, and pointed to `server clients set-scope`. |
| Control: Armorer, read-write | [`armorer-read-write.json`](claude-plugin/armorer-read-write.json) | "Your Mandalore memory is working. It's connected with read-write access." |

The Armorer runs also show an empty `memory_context` result natively. Their
prompts match no memory, so the hook succeeds and adds nothing to context.

Criterion 4 (no 1.x sync, session, binding or `connection.json` guidance in the
skills) is a grep in `bin/ci`.

## Plugin and connector at one URL (#201)

Recorded 2026-10-25 with Claude Code 2.1.310, by the same script. Account
connectors stay disabled in the temporary `HOME`, so the run stands one in with
`--mcp-config`: a server named `claude.ai My Memory` at the plugin's URL. Claude
Code drops the plugin's server as a duplicate, as it does for a synced
connector.

| Check | Run | Result |
| --- | --- | --- |
| Armorer detects the conflict | [`armorer-connector.json`](claude-plugin/armorer-connector.json) | Only `claude.ai My Memory` is connected, and the hook fails open with `MCP server 'plugin:mandalore:mandalore' not connected`. The Armorer's diagnosis is "The claude.ai connector 'claude.ai My Memory' replaces the plugin's server", and it prints `"deniedMcpServers": [{ "serverName": "claude.ai My Memory" }]`. |
| The workaround restores recall | [`connector-denied.json`](claude-plugin/connector-denied.json) | With that entry in the user settings, `plugin:mandalore:mandalore` is connected and `hook_additional_context` 0.01 s after the prompt holds the seeded fact. |

The other Armorer and hook runs gave the same results as above on this change.

A hook that follows whichever server is connected isn't possible in 2.1.310.
Claude Code resolves an `mcp_tool` hook's `server` by exact name against the
connected servers, with no pattern or fallback list. It substitutes
`${user_config.*}` in MCP server configuration and command hooks, but not in an
`mcp_tool` hook's `server`. The connector's name is chosen by each user, so the
plugin can't name it in advance.

## Notes for operators

- Claude Code keeps MCP OAuth state in the login keychain, keyed by server name
  and URL, and it outlives a `HOME`. An earlier run against the OAuth server
  left a "needs authentication" state that later loopback-trust runs inherited.
  `claude mcp logout plugin:mandalore:mandalore` clears it, and the script now
  runs that before and after.
- `${user_config.server_url}` is substituted in skill text. In a separate run
  with the URL set to `http://localhost:28799/mcp`, the model was asked to quote
  the Armorer's line verbatim. It returned "The server URL from plugin settings
  is `http://localhost:28799/mcp`."
- Without a seeded match, the hook adds nothing, so a model that needs memory
  still calls `memory_recall` itself.
