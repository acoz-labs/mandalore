# Context and interaction design

The memory engine is merged through #20. People and native agents need the same
memory operations without an assistant identity/launcher or menu-driving scripts.

## Intended CLI journeys

1. Inspect help and machine-readable operation discovery without a binding.
2. Explicitly create a synthetic/new signet at a supplied path with a chosen name
   and device label; never overwrite an existing target.
3. Bind an existing signet to a new machine-local file with explicit actor/label.
   Show the resulting paths and identity. Do not replace an existing binding.
4. Recall from an unrelated project cwd using that binding; inspect scopes first
   when the requested entity's stable ID is unknown.
5. Send structured remember/journal inputs without shell-escaping multiline prose;
   receive compact local-durability receipts, then inspect history after correction.
6. Start a local MCP server on the same binding. No Git remote, credentials or
   native settings are modified merely by startup or read calls.

Help must explain required flags and show synthetic examples. Unknown commands,
unexpected positionals, missing paired scope flags and malformed input fail
clearly rather than fall through to interactive input. JSON discovery and command
execution must not depend on a TTY. stdout carries requested data/protocol only;
diagnostics must not corrupt MCP JSON-RPC. Signals terminate the process cleanly.

## Product-design review boundary

This slice exposes plain CLI/help and structured API data, not a styled menu,
browser page or native settings UI. Review the flows, defaults, output meaning
and error/cancellation behavior above. The #7 installer/TUI needs its own
arrows/Vim, color/plain, narrow-screen, EOF/cancel and ownership acceptance.
