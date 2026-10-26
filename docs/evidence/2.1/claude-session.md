# Session de-duplication, native Claude Code checks (#228)

Engineering evidence for the plugin sending `session: "${session_id}"`,
recorded 2026-10-26 with Claude Code 2.1.310 on macOS (arm64). It is not
release acceptance. Every run used a synthetic signet with one seeded fact in a
new temporary directory and a temporary `HOME`, linked back to the real home
only through the login keychain so Claude Code could sign in. The test server
was signed out of Claude Code before and after.

The servers were `mandalore` 2.1.0 and 2.0.3, built from the `v2.1.0` and
`v2.0.3` tags ([current](claude-session/server-current-version.json),
[older](claude-session/server-older-version.json)), with `loopback-trust`.
Claude Code reached them through
[`claude-session-proxy.py`](claude-session-proxy.py), which forwards every
request unchanged and logs each `memory_context` call's input and reply, since
the server doesn't log hook input. [`claude-session-native.sh`](claude-session-native.sh)
reproduces the runs:

```sh
docs/evidence/2.1/claude-session-native.sh /tmp/mandalore-2.1.0 /tmp/mandalore-2.0.3 /tmp/session-evidence
```

It installs the plugin from this checkout's marketplace and asks the same
question in each run with `claude -p`, resuming the first session where the
table says so. Between `resumed` and `revised` it revises the fact (Tuesday →
Friday) with `memory_remember` straight to the server, and before
`after-restart` it restarts the server. Servers and the proxy are stopped by
PID. [`claude-session-summarize.py`](claude-session-summarize.py) reduces each
run to the JSON files in [`claude-session/`](claude-session/):
`session_is_this_session` compares the hook's `session` input with the
session ID in Claude Code's own stream.

| Run | Session | `session` input is the session's ID | Injected | Result |
| --- | --- | --- | --- | --- |
| [`first`](claude-session/first.json) | A, new | yes | 385 bytes | The seeded fact and the scopes. The model answered from it. |
| [`resumed`](claude-session/resumed.json) | A, `--resume` | yes, the same ID | 0 | Nothing: the revision was injected earlier in the session. The model answered from the earlier turn. |
| [`revised`](claude-session/revised.json) | A, `--resume` | yes | 325 bytes | The revised fact, without the scopes, which the session already has. |
| [`revised-again`](claude-session/revised-again.json) | A, `--resume` | yes | 0 | Nothing. |
| [`second`](claude-session/second.json) | B, new | yes, B's ID | 384 bytes | The revised fact and the scopes: B has its own state. |
| [`after-restart`](claude-session/after-restart.json) | A, `--resume`, server restarted | yes | 384 bytes | Injected again: a restart forgets every session. |
| [`older-server`](claude-session/older-server.json) | C, new, 2.0.3 server | yes | 0 | `input.invalid` ("Invalid memory_context input."). The hook failed open: `UserPromptSubmit` outcome `error`, and the model answered without memory. |

Criteria on #228:

1. Claude Code substitutes `${session_id}` in the hook's input: every call
   carried the run's own session ID, the same across `--resume`. A revision
   injected earlier in the session isn't injected again (`resumed`,
   `revised-again`), and a revised record is (`revised`).
2. Against 2.1.0 the resumed session kept its state while the server ran
   (`resumed`, `revised-again`), and lost it at a restart (`after-restart`),
   as [documented](../../server.md#tools).
3. Against 2.0.3 every prompt loses its memory and shows the hook error line,
   as the [plugin README](../../../plugins/claude/README.md#server-version)
   says.

In `revised`, the model saw both rotation days, the old one from session A's
earlier turn and the new one from the hook, and said they disagree. The hook
injects the new revision but doesn't say it replaces an earlier one; the
record ID it carries lets the model check with `memory_history`.
