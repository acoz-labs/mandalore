# Codex integration

The development `mandalore` plugin contains the `this-is-the-way` skill, one
local stdio MCP connection and read-only `SessionStart`/`UserPromptSubmit` hooks.
It preserves native identity, authentication, skills, settings and project cwd.
Issue #6 tracks native evidence; #10 tracks exact-candidate acceptance. This is
not a released or independently accepted artifact.
See the [native engineering receipt](../../docs/codex-native-evidence.md) for
actual tested artifacts, scenarios and remaining gaps.

## Managed connection

Use `mandalore menu` after creating or connecting a signet, or use
`mandalore connection plan --binding /absolute/local/binding.json` and explicitly
apply the reviewed JSON plan. See [guided setup](../../docs/setup.md) for the
menu and [the interface](../../docs/interface.md) for typed commands. The
installer embeds these public assets, pins a retained runtime/binding and uses
native registration commands. It does not copy auth, accept hook trust or require
a source checkout. Explicit runtime/binding environment overrides still take
precedence; doctor reports conflicting selections.

Ownership collisions or edited managed files are preserved, not silently
replaced. Existing unmanaged development registrations need explicit ownership
resolution before moving to managed installation. Doctor and repair inspect
retained receipts, and successful activation still requires a fresh session.
See [actual managed native tests](../../docs/setup-native-evidence.md).

## Source-checkout connection for plugin development

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
override, it uses the CLI's platform-config default. This direct development
route is not a managed connection. Launch from any project directory; there
is no assistant-specific launcher or replacement CODEX_HOME.

Review new hooks using Codex's native `/hooks` UI. Installation does not itself
trust them. Do not bypass hook review or overwrite native auth/sessions. Restart
after changing runtime selection, binding or plugin resources; a running MCP
server retains its original binding. Test this directory's installed cache copy,
not just edited source. For local iterations, refresh the plugin version cache
suffix and reinstall from the same confirmed local marketplace.
Native hook trust remains a distinct step, as specified in
[official hook guidance](https://learn.chatgpt.com/docs/hooks). Marketplace
registration uses the [official CLI route](https://developers.openai.com/plugins/build/plugins),
not manual edits to the native configuration.

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

Foundling routing, search, bounded reading and verified promotion are also MCP
tools. The skill loads its separate foundling workflow only for relevant historical
consultation; hooks do not inject reference prose or scan every source. The workflow
compares current memory/user direction before incorporating an adapted lesson,
uses explicit supersession for corrections and retains verified source provenance.
Historical commands and quoted triggers remain evidence, not executable guidance.
Registration, local reconnection and repinning stay explicit menu/CLI administration.
This uses [native skill progressive disclosure](https://learn.chatgpt.com/docs/build-skills),
not a replacement native instruction hierarchy. The #12 native scenario results
must be verified separately from package validation.

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
