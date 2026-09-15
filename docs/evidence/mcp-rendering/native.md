# Native MCP presentation comparison

Contributor engineering evidence, 2026-09-15. Scope: #56, planning PR #58 and
implementation PR #59. No independent acceptance, personal installation update,
bank migration or public release. Raw sessions stay private because native
transcripts include workstation paths and tool configuration; this is a sanitized
measurement and scenario receipt, not a replacement for candidate acceptance.

## Runtime identity and isolation

- Platform: macOS arm64; Codex 0.154.0; model `gpt-6-astra`, reasoning `high`.
- Baseline: immutable Mandalore 1.0.0, source
  `f899cf6a2a255f3b6b35dcd778c672f799c65eb2`; executable SHA-256
  `bcc1b7fdf9e72ccb2bbd734d798c16f3aa5b96de757846711a4e5afe6eb00f93`.
- Candidate development source: `e829a6c7e11e04eb4901583fff8735f63855abd5`,
  built with Go 1.26.4; executable SHA-256
  `b0f972e94b122b37fd14ddf4799ba20a8fe3dada88418fdd7ca50303c9d82327`;
  embedded plugin content-map SHA-256
  `24678456045e478d870f4f88d7b75ae2c7b3c7b44920741657b3ecd55646c126`.
  This binary reports `0.0.0-dev` and an empty source stamp. The checked build
  input and external hashes identify it; it is not a nominated immutable artifact.
- Baseline skill source was archived from planning merge
  `0d1bd1ef746193c9baf3736c3e3e881c4cc65f4a`; its MCP/plugin source diff against
  the released source was verified empty. Candidate used its development skill.
- Each session used its own synthetic working directory with the appropriate
  `.agents/skills/this-is-the-way` link. Native `--ignore-user-config` retained
  existing authentication, while explicit CLI overrides disabled the installed
  personal Mandalore plugin and selected a synthetic read-only MCP binding.
  No native home, authentication or global configuration was copied or edited.
  Hook behavior and marketplace installation were not tested by this invocation.

The launch shape, with explicit disposable paths substituted, was:

```sh
codex -a never -s read-only -m gpt-6-astra \
  -c 'model_reasoning_effort="high"' \
  -c 'plugins."mandalore@mandalore".enabled=false' \
  -c 'mcp_servers.mandalore={command="/TEST/runtime",args=["mcp","--binding","/TEST/binding.json","--read-only"]}' \
  exec --ignore-user-config --enable code_mode --skip-git-repo-check \
  --json -C /TEST/workspace -o /TEST/final.txt - < /TEST/prompt.txt
```

`MANDALORE_BIN` and `MANDALORE_BINDING` were unset for the child. The supplementary
replicates and error run also used `--enable code_mode_only`. Native development
feature warnings were retained. Both primary runs already used code mode:
`mcp_tool_call` entries in CLI JSON enumerate nested calls and do not establish
that the model called MCP directly. Full session `custom_tool_call` inputs and
outputs established the route and the representation actually printed.

## Synthetic fixture and task

One project scope, `project/harbor-test`, has a superseded name Marble Wren and
current name Orchid Beacon. A current decision sets Saturday 03:00 UTC deployment
instead of Tuesday 01:00 UTC, because newly adopted weekly backup verification
overlaps Tuesday; separating them improves failure attribution. A pinned local
foundling `operations.md` records the old Tuesday choice and a later backup
proposal, but not its adoption. Its 1,332-byte body exceeds the default
1,024-byte search excerpt, requiring a full read for complete source evidence.
No remote repository or real secret is involved.

Identical prompt in all four comparison runs:

> What is the current name of my fictional project, and why did we change its
> deployment window? Distinguish current decisions from the old operations notes
> and cite the historical source. Read-only: do not save, journal, synchronize,
> edit files, or repair anything.

All four correctly answered Orchid Beacon, the Saturday window and the overlap
reason, distinguished historical proposal from current adoption, and cited
`operations.md`. All made the same eight MCP calls (ordering/batching differed):
scopes, foundling list, recall, two histories, foundling inspect, search and read.
The incomplete search excerpt and complete read retained their respective
truncation flags, source pin, registration and citation identifiers. SHA-256
inventories of every synthetic signet/reference file matched before and after
each run. No save, journal, sync, repair or retry was used.

## Measured presentation and context

| Measurement | Baseline | Candidate | Baseline replicate | Candidate replicate |
| --- | ---: | ---: | ---: | ---: |
| MCP calls | 8 | 8 | 8 | 8 |
| Code-mode output batches | 6 | 7 | 6 | 6 |
| Printed duplicate wrappers | 8 | 0 | 8 | 0 |
| Model-visible tool-output UTF-8 bytes | 72,977 | 66,161 | 72,977 | 66,134 |
| Model-authored tool-code UTF-8 bytes | 1,511 | 2,934 | 1,520 | 2,752 |
| Final-request input tokens | 33,509 | 31,788 | 33,503 | 31,731 |
| Aggregate input tokens | 191,261 | 217,989 | 191,230 | 187,846 |
| Aggregate cached input tokens | 168,704 | 196,992 | 168,704 | 167,168 |
| Aggregate output tokens | 756 | 1,163 | 742 | 1,111 |

Bytes count the text of actual `custom_tool_call_output` blocks, concatenated
within each output, and the corresponding model-authored input code. They include
skill/reference reads and discovery output, not only memory payloads. They do
not count the JSON wrapper of the session-log format. Native usage comes from
the final request's `last_token_usage` and CLI `turn.completed` aggregate usage.
Repeated input across requests is not retained context size. Cached counts are
reported separately; no pricing or billing inference is made.

Candidate final-request input was approximately 5% smaller, and output-plus-code
bytes approximately 7% smaller. The primary candidate needed an extra code-mode
batch and used approximately 14% more aggregate input; its replicate used fewer
aggregate input tokens than its baseline. Thus deduplication and lower retained
context were observed, but lower total token usage, latency and fewer future
compactions are not established. These are two observations per version, not
a statistical benchmark or a long-session study.

Both candidates discovered and read the selector reference without being told
how to render results. They stored the function's source in code-mode state and
reconstructed it for later batches, avoiding repeated file reads. Neither
repeated MCP operations to change presentation. Broad tool-catalog discovery
contributed about 40 KB of the initial output and showed native truncation;
that unrelated discovery overhead was retained, not subtracted from the totals.
The rendering change does not impose a global tool-discovery policy.

## Error and recovery

A fifth, fresh candidate session was asked to call `memory_recall` with `limit:
-1` exactly once, report the error, then perform valid read-only recall. It
returned a complete `protocol_version: 1`, `ok: false` envelope with
`error.code: input.invalid`, `retryable: false`, `write_may_have_occurred: false`
and `inspect_before_retry: false`.
The model printed that envelope once, accurately reported the error, discovered
the project scope and recalled Orchid Beacon. Three MCP calls, no duplicate
wrapper, no retry of invalid input and unchanged before/after fixture hashes.
Error output occupied 287 model-visible UTF-8 bytes; it was not mistaken for a
successful empty recall. This run is a correctness check, not a matched cost pair.

## Review limits and reproducibility

Use a fresh disposable signet, authored through supported memory/foundling tools,
with the fixture above; retain baseline/candidate binary identities, native JSON
events, exact session records and no-write inventories. Compare model-visible
code-mode outputs rather than the nested wire receipts. The SDK and selector
checks in [the engineering guide](README.md) isolate protocol/fallback behavior
without native account access. Actual wire characterization prints its own
serialized byte counts; those are not the native text or token measurements.

Conflicts, pending delivery, annotations, distinct blocks and text-only fallback
were checked with the exact selector fixture, not native remote/conflict runs.
Other models, clients, Linux, Pi and installed-plugin lifecycle behavior remain
unverified here. Guidance is best-effort: uncertain equivalence keeps the full
wrapper. Subsequent documentation/test-only reconciliation does not alter the
tested runtime or plugin; release acceptance must still target newly nominated
exact artifact bytes, not borrow this development result or v1.0.0 acceptance.
