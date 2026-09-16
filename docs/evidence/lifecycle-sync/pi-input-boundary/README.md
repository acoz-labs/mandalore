# Pi input events are not a universal invalidation boundary

Fresh #55 O1 discovery probe, September 16, 2026. This is not a change to
Mandalore, an activation of lifecycle synchronization or candidate acceptance.

## Question and observed result

Can a Pi `input` subscriber invalidate cached task authorization for every new
user instruction? **Not for all native RPC entry points in Pi 0.85.1.**

The actual native process loaded a deliberately minimal observer in a fresh
synthetic profile. It had no Mandalore extension, signet, skills, context files,
prompt templates or provider login. Offline mode and no-session were selected.
The observer handled the ordinary prompt before any model request. It retained
only event/source metadata, never prompt text or credentials.

| Native RPC operation | Observed input events added | Pending messages afterward |
| --- | ---: | ---: |
| Ordinary prompt, handled by observer | 1 | 0 |
| Direct `steer` | 0 | 1 |
| Direct `follow_up` | 0 | 2 |
| Explicit `clear_queue` | 0 | 0 |

Both queued commands received successful native responses. Clearing returned one
steering and one follow-up message. EOF caused `session_shutdown` and exit 0;
stderr was empty. The retained [receipt](receipt.json) records those outcomes.
This was an idle RPC queue probe, not model-active steering or interactive UI
ordering. No claim about network access from an arbitrary future extension follows.

Installed-source inspection explains the observation: `AgentSession.prompt`
emits input before its streaming branch, whereas direct `steer` and `followUp`
expand/queue without that call. Native RPC dispatch selects those direct methods.
The input runner also catches handler failures and continues; the product must
not assume a failed observer necessarily blocks the prompt. Only the queue bypass
was executed here; handler-failure behavior was inspected, not fault-injected.

## Reproduce and identify

Retained original files: [driver](probe.mjs), [observer](observer.ts), and receipt.
With Node 24.1.0 and Pi 0.85.1, copy the observer into a newly created disposable
directory. Run `node probe.mjs DISPOSABLE_DIRECTORY PI_EXECUTABLE`; the driver
creates a fresh profile/project there, refuses existing directories, selects
only that observer and checks each native response. Do not use a personal profile.
Driver and observer are test instruments, not a supported permission mechanism.

SHA-256 identities:

- Driver: `9e31441e335b838410c8717191239159d06ff13ffca0009a1314fec8fe065e0c`.
- Observer: `69af2340a5b2559d65c127f0ca320b8a97e3c9b34eeea8441188f318c222556e`.
- Receipt: `542188522e123954b8ac26ea402e4e765c0d57df54c262172bdb6a07dbfab59b`.
- Executed native bundle: `e6d7fcf36a239cf3746e67ddf4222081ac01a601b85a3ee688bdfe9c161d754c`.
- Installed `dist/core/agent-session.js`: `fb8a3981c20c8c0bbd42231b1c99a10335fb3858b659056b341954de9cfa467f`.
- Installed `dist/core/extensions/runner.js`: `0de12ed1275e02595f92476eec3f61ae1f2e54fd2225ced721ddc90af58a5e61`.
- Installed `dist/modes/rpc/rpc-mode.js`: `e7e4724aa55c5aac73cf36793653b26736200e5c59d58373990fc31028f86477`.

Upstream tag v0.85.1 resolves to
[`d981de1229ef899957bbe968bc8dcda02a21f477`](https://github.com/earendil-works/pi/tree/d981de1229ef899957bbe968bc8dcda02a21f477).
That is a source reference, not a reproducible-build attestation for the installed
package; the executable digest above identifies what actually ran.

## Design consequence and limits

A permission flag invalidated solely by `input` must not authorize generic
lifecycle transport across RPC/SDK entry points. This adds a Pi-specific
counterexample to the earlier Codex freshness findings; it does not show that
every narrower design is impossible. No cached-allow implementation is selected.

Further design would need to enumerate supported entry points, establish
failure-safe invalidation before transport, and prove behavior for queued messages,
extension-generated instructions, failed handlers, compaction, exit and resume.
Simply checking that a queue is empty does not prove current semantic permission.
An explicit narrower scope or upstream event contract would need review before
implementation. Ordinary model-selected save-and-deliver remains available.

No product bytes, native configuration, authentication, memory or public release
were changed. The 1.1 candidate and its separate human-acceptance gate are intact.
