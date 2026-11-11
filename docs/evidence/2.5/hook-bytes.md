# Injected bytes per prompt, hook replay (#232)

On the 2.5.0 code (b17ceb1), the prompt hook injects 320 bytes per prompt on
average across 208 replayed prompts with Claude Code's session IDs, and 600
bytes without them. The cap #232's triage set is 800 bytes on average,
judged on the run with session IDs. The 2048-byte default stays: at 1536 and
1024 bytes, answers that came whole at 2048 go missing.

This is engineering evidence recorded 2026-11-11 on macOS (arm64) with Go
1.26.11. It is not release acceptance. All data is synthetic.

## Method

The requester ruled on #232
([PR comment](https://github.com/acoz-labs/mandalore/pull/261#issuecomment-6801761482))
that a replay of the plugin's hook counts as measuring bytes "in real
sessions", with no Claude Code credential. The replay,
[`hook-replay.py`](hook-replay.py), does this:

- **The hook as shipped.** It reads the `UserPromptSubmit` `mcp_tool` entry of
  `plugins/claude/hooks/hooks.json`: the tool (`memory_context`), its input
  template (`{"event": "prompt", "prompt": "${prompt}", "session":
  "${session_id}"}`) and its 3 second timeout. It fills `${prompt}` and
  `${session_id}` into the template's strings, as Claude Code does.
- **The real endpoint.** It sends a `tools/call` over MCP HTTP to a loopback
  server built from this tree (`http://127.0.0.1:28741/mcp`), on sessionless
  MCP `2026-07-28`, as Claude Code speaks it to this server. It never calls
  the server in process.
- **The measure.** Bytes are the length of the reply's
  `hookSpecificOutput.additionalContext`, which is what Claude Code adds to the
  turn.
- **The signet.** It serves the synthetic evaluation set
  (`internal/server/testdata/context-eval.json`), written by
  `TestWriteEvaluationSignet` as a checkout with an empty remote, under
  `loopback-trust`.
- **The prompt set** is fixed by the plan
  ([Bytes per prompt](../../changes/232-hook-selection/plan.md#bytes-per-prompt-and-the-2048-byte-default)),
  208 prompts in 9 sessions:
  - S0: the 20 scripted session prompts;
  - S1 to S8: one session per scope, each with that scope's case questions
    and paraphrases, its plain short questions, and every eighth tuning
    no-answer prompt.
- **Runs.** Each budget runs twice on a fresh server: once with a fixed
  synthetic session ID per session, once with the `session` input empty. The
  1536 and 1024 builds differ from the default build only in
  `contextDefaultBytes`, through a Go build overlay, because the hook sends no
  `budget_bytes`.

Reproduce from a checkout, with Go and git on `PATH`:

```sh
python3 docs/evidence/2.5/hook-replay.py /tmp/hook-replay 2048 1536 1024
```

The script stops every server it starts by PID. Its outputs are in
[`hook-replay/`](hook-replay/): one file per run with each prompt's bytes,
injected records and answer line, and `summary.json`.

## Results

| Budget | Session IDs | Average bytes per prompt | Maximum | Prompts over 800 bytes | Answer lines: whole / clipped / given earlier in the session / absent |
| --- | --- | --- | --- | --- | --- |
| 2048 (default) | yes | 320 | 2024 | 31 of 208 | 41 / 0 / 84 / 1 |
| 2048 (default) | no | 600 | 2044 | 49 of 208 | 119 / 0 / – / 7 |
| 1536 | yes | 271 | 1527 | 30 of 208 | 40 / 0 / 85 / 1 |
| 1536 | no | 526 | 1529 | 49 of 208 | 117 / 0 / – / 9 |
| 1024 | yes | 224 | 1022 | 23 of 208 | 41 / 0 / 84 / 1 |
| 1024 | no | 454 | 1022 | 49 of 208 | 116 / 0 / – / 10 |

The answer columns count the 126 prompts with an expected record: 88 case
phrasings and 38 short questions. No call failed or timed out.

- **De-duplication** removes 280 bytes per prompt at 2048 (600 without session
  IDs, 320 with). Most of the saving is the paraphrase after its question: the
  answer came with the question, so the session doesn't get it again.
- **The cap.** 320 bytes is under the 800-byte cap. For scale, the reviewer's
  in-process run of the same layout on 2.4.0 code (plan review round 2, not
  this replay) gave 339 bytes with session IDs and 627 without.
- **The default stays 2048.** The plan's rule lowers the default only if no
  answer that came whole at 2048 is lost or clipped at the lower budget. In
  the runs without session IDs, two answers are lost at 1536 and three at
  1024: "What resolves DNS on the home network now?", "When can seedlings go
  out?" and, at 1024, "Any network calls?". Each was the fourth record,
  after the top 3, which appear whole only when they fit. No answer was
  clipped at any budget.

## Limits

- The signet is synthetic, so its record lengths stand in for real ones, and
  no model reads the context. The requester's private eval on a copy of their
  own memory checks real data on the candidate before publication.
- The replay doesn't run Claude Code. The #228 native checks
  ([2.1 evidence](../2.1/claude-session.md)) showed Claude Code fills
  `${session_id}` with its session ID, the same across `--resume`.
  `hooks.json` hasn't changed since, but those checks weren't re-run, because
  they need a Claude Code credential.
