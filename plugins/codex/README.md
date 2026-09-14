# Codex integration

The development `mandalore` plugin contains the `this-is-the-way` skill, one
local stdio MCP connection and read-only `SessionStart`/`UserPromptSubmit` hooks.
It preserves native identity, authentication, skills, settings and project cwd.
Issue #6 tracks native evidence; #10 tracks exact-candidate acceptance. This is
not a released or independently accepted artifact.
See the [native engineering receipt](../../docs/codex-native-evidence.md) for
actual tested artifacts, scenarios and remaining gaps.

## Connect an explicit test installation

Build the CLI and create/bind a synthetic signet using [the CLI guide](../../docs/interface.md).
Keep the binding outside the signet. Before installation inspect `codex plugin
list`: do not activate Mandalore alongside an old memory plugin targeting the
same bank. Removing or disabling another installation is a separate ownership
decision; setup must not silently remove it.

This directory is the marketplace root. From this repository:

```sh
codex plugin marketplace add ./plugins/codex
codex plugin add mandalore@mandalore
```

Start a fresh native Codex session with absolute `MANDALORE_BIN` and
`MANDALORE_BINDING` values for your selected runtime and local binding. These
are machine-local configuration, not portable bank content. Without the runtime
override, the bridge uses `mandalore` on the process PATH; without a binding
override, it uses the CLI's platform-config default. Setup #7 will provide the
guided, pinned connection experience. Launch from any project directory; there
is no assistant-specific launcher or replacement CODEX_HOME.

Review new hooks using Codex's native `/hooks` UI. Installation does not itself
trust them. Do not bypass hook review or overwrite native auth/sessions. Restart
after changing runtime selection, binding or plugin resources; a running MCP
server retains its original binding. Test this directory's installed cache copy,
not just edited source. For local iterations, refresh the plugin version cache
suffix and reinstall from the same confirmed local marketplace.

## What runs and what enters context

Startup/resume/compact sources on `SessionStart` get a short orientation. Each
`UserPromptSubmit` can add up to three signet-wide recall results within a
4096-byte packet and five scope-inventory entries. The hook uses at most 2048
query bytes and accepts at most 64 KiB of native event JSON. Its complete JSON
output is capped at 16 KiB and the native execution timeout is five seconds.
Scope inventory is routing metadata, not unrelated project content. Further
scoped retrieval uses MCP tools. Files are reread; previously loaded model
context is not retroactively refreshed by a Git pull.

Hooks never write, sync, parse transcripts, invoke another model or execute user
scripts. The skill handles relevant confirmed learning and semantic journaling
incrementally, not only at exit. It can request a three-second sync before
cross-machine recall and after useful saves when permitted by the task. Read-only
or no-save tasks forbid journaling/checkpointing/sync too. Local save and remote
delivery are separate receipts; conflicted history is not current guidance.

Direct user “this is the way” adds consolidation intent; quoted/retrieved/tool
occurrences do not activate a command. There is no mechanical phrase detector.
The skill is implicitly discoverable, so normal learning does not require a cue.

## Failure and compatibility

Missing/invalid bindings, malformed hook input, and failed local recall emit
sanitized native warnings without writing or blocking unrelated work. A missing
or older CLI that fails the hook command is caught by the shell bridge; failed
raw output is discarded. The selected executable is trusted local code, not an
untrusted plugin response parser. Native timeout/cancellation remains Codex's
responsibility. No Stop/SessionEnd save is assumed; interrupted work is not
automatically resumed or retried.

Packaging uses the supported `.codex-plugin/plugin.json` compatibility layout
and default `hooks/hooks.json` discovery. It does not require a remote MCP server,
an API key, global instructions, credentials in Git, or a second native login.
Initial native test target: Codex CLI 0.153.4. Do not infer other version/surface,
compaction, interruption or cross-machine acceptance from unit tests.

See [official packaging](https://developers.openai.com/plugins/build/plugins)
and [native hooks](https://learn.chatgpt.com/docs/hooks) for host contracts.
