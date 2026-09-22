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
by the particular CLI invocation. A learning-enabled connection still follows
the user's current task restrictions; model adherence is distinct from the
runtime enforcement provided by a read-only connection.

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

The native adapter handles `SessionStart` and `UserPromptSubmit`. It reads bounded
local context, never a transcript. It does not save memories, initialize Git,
synchronize, launch a model or block unrelated work. Context output stays below
Claude's 10,000-character inline-context threshold; a failure produces a concise
warning so tools can be inspected directly.

Native macOS validation with Claude Code 2.1.278 observed `SessionStart:startup`
and `UserPromptSubmit` on a fresh print session, `SessionStart:resume` when
resuming, and `SessionStart:compact` after an actual manual `/compact`. A resumed
post-compaction session received orientation again and retained the discussion.
Interrupting streamed model text with SIGINT left the synthetic signet and native
memory unchanged. These are observed native behaviors, not guarantees that every
termination runs a hook. Linux adapter-input checks are distinct from a native
Linux Claude session.

Learning and synchronization happen through the shared memory tools during the
conversation. The model is asked to save confirmed knowledge incrementally and
attempt bounded delivery when allowed. Prefer `memory_remember_and_sync` and
`memory_journal_append_and_sync` when both saving and synchronization are allowed;
use their local-only counterparts when synchronization is prohibited. Inspect
both the saved receipt and delivery result. A local-only receipt means delivery
was not requested and must not be reported as synchronized. At the start of ordinary memory work, use one short-budget `memory_sync`
before relying on attached or recalled memory when the task and connection allow
it. This does not require knowing whether another device wrote something: that
change is invisible until delivery is checked. A hook only reflects the selected
local clone. No-sync and read-only tasks keep local evidence and its freshness
limitation; a failed attempt is not an instruction to retry on every lookup. A saved record, successful network
delivery and knowledge already loaded into another conversation are different
states. Offline delivery and interrupted responses require inspection before
retrying a save. No end-of-session or compaction hook promises to recover
knowledge that was never saved. Deterministic lifecycle transport remains tracked
separately in [issue #55](https://github.com/acoz-labs/mandalore/issues/55).

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
