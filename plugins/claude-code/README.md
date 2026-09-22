# Claude Code integration

Development integration for [issue #14](https://github.com/acoz-labs/mandalore/issues/14).
It connects ordinary Claude Code sessions to the same signet used by Codex and
Pi through the shared Mandalore MCP server. This development branch is not a
published or accepted release. The initial native contract is Claude Code
2.1.278; binary cross-compilation does not establish native acceptance.

## Connect an existing signet

Install and authenticate Claude Code separately. Mandalore does not obtain model
access, copy credentials or change your subscription. In `mandalore menu`, choose
**Connection**, then **Claude Code**, select the binding and native profile,
and review the installation preview. Learning and enforced read-only connections
are separate choices. Applying requires an explicit confirmation.

The CLI exposes the same workflow:

```sh
mandalore connection plan --harness claude-code \
  --binding /example/binding.json \
  --native-home /example/claude-profile \
  --native-binary /example/bin/claude \
  --state-dir /example/mandalore-state > claude-plan.json
# Inspect the complete plan before applying it.
mandalore connection apply --harness claude-code < claude-plan.json
mandalore connection armorer --harness claude-code \
  --native-home /example/claude-profile \
  --native-binary /example/bin/claude \
  --state-dir /example/mandalore-state
```

The default profile is `CLAUDE_CONFIG_DIR` when set, otherwise the user's native
`.claude` directory. The harness identifier is `claude-code`; the native executable
is `claude`. Binding selection is explicit and independent of the project
directory. Generated connections pin the binding bytes and signet identity.
The connection also checks the retained runtime digest before hook or MCP launch,
using `/usr/bin/shasum` on macOS or `/usr/bin/sha256sum` on Linux. A missing checksum
utility or changed runtime leaves the connection unavailable for inspection.

Use `--memory-read-only` when planning to enforce no memory writes or sync in the
installed connection. The general `--read-only` option instead prohibits mutations
by the particular CLI invocation. An enabled session honors content-level
requests not to remember or journal while software transports existing records.
Existing read-only connections remain read-only until explicitly reconfigured.

## Use memory naturally

The plugin supplies `this-is-the-way` and `the-armorer` through Claude's native
skill discovery, including `/mandalore:this-is-the-way` and
`/mandalore:the-armorer`. Ordinary recall and useful learning do not require
an explicit consolidation phrase. The engine authors `claude-code` provenance
for Claude writes and uses the same scope, revision and visibility rules as the
other harnesses.

Project memory requires a stored scope. An empty unscoped search is not evidence
that project knowledge is absent: discover scopes, then recall in the selected
scope. Historical and withdrawn records are not an automatic fallback for an
ordinary recall miss.

Claude's native auto memory remains a separate system. Mandalore does not disable
it, import its notes or copy signet records into it. The memory skill directs
Claude to recheck overlapping knowledge against current Mandalore evidence and
avoid recreating withdrawn records from native notes. Independently saved native
notes, historical references, offline copies and already-loaded conversation
context cannot be revoked by a Mandalore withdrawal. Explicit inspection of those
separate sources is a different task; withdrawal is not universal erasure.

## Lifecycle and delivery

The native adapter handles synchronous `SessionStart` and `UserPromptSubmit`.
An explicitly enabled session refreshes before assembling bounded memory context;
legacy connections without that policy keep local-only hooks. Hooks never scan
transcripts, extract memories or launch a model. A refresh failure reports pending
or stale state without falsely claiming current remote knowledge.

The shared session API attempts delivery after semantic writes, regardless of
whether the model chooses an ordinary save or a combined save-and-sync tool.
Inspect the separate saved and delivery outcomes. Pending delivery retries at the
next foreground opportunity; never repeat a save to retry transport. The runtime
does not initialize Git, enroll credentials or resolve semantic conflicts.

Native behavior and timings must be verified against the exact release candidate.
Earlier 2.1.278 local-hook evidence observed startup, resume and manual compact
callbacks, but does not certify the changed synchronization callbacks. No exit
or compaction event promises to recover knowledge that was never saved.

To opt out, disable the complete native Mandalore plugin and start a fresh
session. Disabling only skills leaves other plugin components possible, and
already-loaded context cannot be removed retroactively. Requests not to remember
particular content still apply while transport of existing records is enabled.
See [the shared transport contract](../../docs/synchronization.md#session-authorized-transport).

## Update and recover

Connections retain their selected runtime and generated package outside the
signet. Plans pin input identities and reject changed bindings, edited owned
files and ambiguous registrations. Unrelated native settings and authentication
remain Claude-owned.

Exit affected Claude Code sessions before replacing an existing connection, then
apply the reviewed plan with `--sessions-stopped`. That acknowledgement applies
only to the current invocation. Start fresh sessions after installation; a
healthy Armorer report proves structural checks, not active model context.

After a partial update, inspect the returned phase and retained generation before
retrying. Preview repair with:

```sh
mandalore connection repair --harness claude-code \
  --connection-root /example/retained-generation > claude-repair-plan.json
# Inspect the saved repair plan, then apply those exact bytes.
mandalore connection apply --harness claude-code --sessions-stopped \
  < claude-repair-plan.json
```

Repair preserves binding and memory-access mode and creates a new
generation. Missing or edited ownership evidence requires inspection; it does
not authorize silently adopting files or switching signets.

The `repair --apply --sessions-stopped` convenience form prepares and applies
a new repair plan in one invocation. Repeating `repair` generates a new plan;
use the saved-plan form above when approval must bind to the previously viewed
bytes.
