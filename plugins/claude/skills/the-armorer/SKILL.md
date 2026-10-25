---
name: the-armorer
description: Check why Mandalore memory isn't working in this client and say how to fix it. Use when the user asks to check, diagnose or set up Mandalore, or when memory tools are missing or failing. Not for ordinary recall or saving.
---

# The Armorer

Diagnose this client's connection to the user's Mandalore server, without a
shell. Give one line of diagnosis first, then the fix. Change nothing yourself.

The server URL from plugin settings is `${user_config.server_url}`. In the
Claude app the server is a connector instead, under Settings > Connectors.

Work through these in order and stop at the first that applies:

1. **No Mandalore tools, and no `authenticate` tool for it.** Diagnosis: "The
   Mandalore server isn't reachable at <URL>." Ask the user to check that the
   server is running, the URL and any tunnel, then run `/mcp` to reconnect.
   Prompts keep working without memory in the meantime (fail open).
2. **Only an `authenticate` tool, or the server shows "needs authentication" in
   `/mcp`.** Diagnosis: "This client isn't authorized yet." Ask the user to run
   `/mcp`, choose Mandalore and sign in. The owner approves a new client once,
   on the server's consent page. If the owner denied it, the owner has to
   approve it on a new attempt.
3. **A tool call returns 401 or `invalid_grant`.** Diagnosis: "This client's
   sign-in expired or was revoked." Ask the user to authenticate again in
   `/mcp`.
4. **Otherwise call `memory_scopes`.** If it succeeds, the connection works:
   report the signet and its scopes.
5. **Then check the scope.** If there is no `memory_remember` tool, or the
   server says the connection is read-only, the diagnosis is: "This client is
   read-only." The owner can change it on the server with
   `mandalore server clients set-scope --client ID --scope read-write`.
   Otherwise report "Mandalore is connected with read-write access."

Server administration (owner enrollment, listing or revoking clients, migration
from 1.x) happens on the server with the `mandalore server` commands; see the
deployment guide. Point the user there, and don't run them.
