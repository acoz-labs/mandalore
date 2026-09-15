# Native lifecycle and model baseline

Engineering verification of source
`e4a8cb3b5dfff62b5370d0ab495d841ba8d6529a`, not product acceptance or release.
The immutable local eight-file build retains the tracked version `1.0.0`; these
are **not** the published v1.0.0 bytes. Nothing was nominated or published.

| Identity | SHA-256 |
| --- | --- |
| Candidate manifest | `280e1147bb30bcdd655827f4d3347beb3872d0ff64e5f76aa900c9988b498bff` |
| macOS arm64 executable | `9e6350e8c0af94f5030ff9573d78930e2c55a5a0d88c9b4160a52cba93d46a95` |
| Embedded Pi package | `79d2491186a6da434bd1e7fd8ea212136595247e90162c5bb3beabc6639614f8` |
| Codex plugin ZIP | `605b32eafff3ae6422a7623cdf235313c1f3e6f3544f25b7b7e054dbc02b393c` |

All candidate assets were checked against the manifest before fixture setup.
Two synthetic signets use local bare Git remotes and separately installed native
Pi profiles. Both have a project scope named `cedar-orbit`, with deliberately
different release days: Thursday in A and Saturday in B. Seed records have
`fixture` provenance, not model-authored learning. Public evidence contains only
synthetic outcomes; workstation paths, authentication and raw sessions stay out.

## Real Pi lifecycle, without a model

On Pi 0.85.1 / Node 24.1.0 / macOS arm64, an observation-only extension recorded
actual native RPC startup, reload, new session, resume and fork. Resume/fork used
an explicitly fabricated inert session fixture, **not** a model conversation.
After each transition the inventory contained 18 unique memory tools, two
skills and the native read/bash tools. No custom context messages accumulated;
both complete bank file snapshots, including Git files, remained unchanged.

The first assertion expected five start notifications but observed eight:
`startup, reload, new, new, resume, resume, fork, fork`. A native control with
Mandalore disabled reproduced the identical sequence and five shutdown events.
Inspection of the pinned upstream
[RPC mode](https://github.com/earendil-works/pi/blob/d981de1229ef899957bbe968bc8dcda02a21f477/packages/coding-agent/src/modes/rpc/rpc-mode.ts)
and [session runtime](https://github.com/earendil-works/pi/blob/d981de1229ef899957bbe968bc8dcda02a21f477/packages/coding-agent/src/core/agent-session-runtime.ts)
showed both replacement-session callbacks and RPC commands rebinding the host.
The observer was corrected to the actual version-specific contract; the failed
observation was retained, not counted as a pass. No product deduplication or
weakened identity checks were needed. The complete native repeat passed.

A focused adapter regression repeats this observed sequence, checks stable tool
registration, closes superseded connections, prohibits incidental memory calls
and verifies the final tool still works. Full pinned host CI with this additional
test passed: 20 Node tests, Go race tests/vet and all four target builds. Docker
was unavailable, so the documented host fallback was used. This is not Linux
native verification or rendered lifecycle acceptance.

## Real Codex model baseline

A fresh native Codex process used the exact candidate's memory skill and MCP
runtime, explicit A binding, existing native authentication in place, ephemeral
history and enforced read-only memory. Ambient user configuration and the live
personal plugin were excluded for this process; no live installation changed.
The model was `gpt-6-astra`, reasoning `high`.

The ordinary question asked the release day for Cedar Orbit without naming
Mandalore or a tool. It also explicitly prohibited writes and synchronization.
The model discovered scopes and recalled `project/cedar-orbit`, returning
Thursday rather than bank B's Saturday. Native events verified only read-only
memory calls, the correct signet identity, successful completion and unchanged
complete snapshots of both banks. The two MCP response text bodies were 228 and
725 UTF-8 bytes; the full JSON bodies were not repeated in result metadata.

Native aggregate usage was 90,207 input tokens, including 63,616 cached tokens,
and 574 output tokens. Those totals include native instructions, skills and all
model requests: they are **not** a measurement of Mandalore-only context cost.
This is a read-only baseline, not passive learning, cross-harness supersession,
Pi model behavior or a complete context-efficiency acceptance result.

## Pi ordinary recall and learning

After the owner completed native Pi authentication, its read-only readiness
check reported ready. Synthetic model runs inherited that native login in place
and explicitly loaded each candidate connection's package and skills, with
ambient resource discovery disabled. Authentication was not copied into the
disposable profiles. This process-specific test attachment is not a permanent
personal-profile installation or a claim of credential isolation.

Separate fresh Pi processes using `gpt-6-astra` / high reasoning answered the
same ordinary read-only question in unrelated working directories. A returned
Thursday; B returned Saturday. Each called the bound `memory_recall`, and both
complete bank snapshots remained unchanged. Each native tool result contained
one full text envelope and only operation/ok metadata, not a duplicate body.

A third ordinary prompt confirmed changing A's release day to Friday, without
naming Mandalore, a tool, or a remembering/consolidation phrase. Pi recalled the
prior decision and used `memory_remember_and_sync`, preserving the record ID and
superseding its previous revision. The receipt confirmed local durability and
delivery to the synthetic local bare remote, with matching head/remote head and
zero semantic conflicts. B remained byte-identical. This proves ordinary model
learning and model-selected combined delivery, **not** lifecycle synchronization.
Pi did not append a journal in this case; journaling and subsequent Codex/Pi
supersession remain separate checks.

## Review and remaining work

Contributor self-review checked source/artifact identity, the native no-model
versus model distinction, the disabled-extension control, complete no-effects
snapshots and aggregate-token limitations. This is not independent approval.
Hosted CI for the exact source passed in
[run 35037748296](https://github.com/acoz-labs/mandalore/actions/runs/35037748296);
later changes still require their own checks. Pi model learning, cross-harness
updates, delivery/concurrency and the final rendered matrix remain in progress.
