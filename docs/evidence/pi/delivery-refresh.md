# Delivery, cancellation and live context refresh

Engineering evidence for issue #13 / PR #72. Native checks use the immutable
`e4a8cb3b5dfff62b5370d0ab495d841ba8d6529a` build identified in
[the model baseline](lifecycle-model-baseline.md), on macOS arm64 with Pi 0.85.1
and Node 24.1.0. These checks do not nominate or accept a release, activate a
personal connection or establish Linux native behavior.

## Deterministic native delivery checks

Two new synthetic clones shared one local bare Git remote. Each had a distinct
binding/device, its own owned native Pi connection and an isolated native
profile. Both actual Pi RPC processes loaded an observation wrapper around the
unchanged candidate extension. The wrapper forwarded native tool registration,
validated arguments using Pi's own validator and invoked the registered tool
callbacks. It did not fabricate Go responses, change the runtime/package or
request a model. Thus this is native schema/callback/transport evidence, **not**
model tool selection, rendered tool UX or an Escape-key interruption test.

Eighteen native calls established these outcomes:

| Scenario | Observed result |
| --- | --- |
| Signal already cancelled before a journal save | `operation.cancelled`, no possible-write flag, complete bank snapshot unchanged. |
| Concurrent compatible journal writes in separate clones | Both durable; subsequent explicit syncs integrated both entries with zero semantic conflicts. |
| Combined save with deliberately unavailable remote | Save retained its ID and local durability; nested delivery was pending and not delivered. |
| Explicit recovery after restoring the remote | Delivery completed; exactly one journal entry matched the saved ID. The save was not retried. |
| Combined save interrupted during fetch | Outer save remained successful; nested delivery retained `operation.cancelled`, fetch phase, checkpointed/pending state and inspection requirement. |
| Explicit recovery after interruption | Delivery completed; exactly one journal entry matched the original receipt. |
| Different choices superseding the same baseline on separate clones | Both revisions survived integration; delivered state was `conflicted` with one semantic conflict. Scoped recall had no authoritative current choice and exposed the conflict. |

For interruption, a fixture-only Git wrapper delegated ordinary operations to
the system Git executable. At `ls-remote` it wrote a complete PID line and waited.
The observer required both that complete marker and a live process before
aborting the native tool signal. The actual transport stopped its retained Go
runtime; Go returned the complete partial receipt and reaped the paused Git
child. The observer verified that PID no longer existed. There was no timeout
guess, automatic retry or unobserved process restart. This controlled phase
proves cancellation propagation; it does not simulate a real network service.

Every callback returned one complete text envelope plus only operation/ok
metadata, including the nested cancellation receipt. In this small fixture,
local reads took 23–27 ms, local writes 33–45 ms, and successful local-remote
syncs approximately 0.69–1.57 seconds. These are single-run measurements from
the native callback boundary, not service-level objectives or internet latency.

## Actual model sees an update in an already-open session

A separate actual Pi RPC model process (`gpt-6-astra`, high reasoning) inherited
existing native authentication in place, with only the candidate package/skills
explicitly loaded. The fixture authored a bank-wide greeting preference through
the CLI. The first ordinary read-only question returned **River**. While that
same Pi process remained open, an external fixture CLI call superseded the same
record with **Willow**. The next identical question returned **Willow**. Native
state responses verified a nonempty, unchanged session ID.

An observation-only extension after Mandalore recorded the context supplied to
both real turns:

| Turn | Suffix bytes (UTF-8) | Current fact present | Superseded fact present | Attachment copies | Persistent custom messages |
| --- | ---: | --- | --- | ---: | ---: |
| Before external change | 2472 | River | No | 1 | 0 |
| After external change | 2473 | Willow | No | 1 | 0 |

Each model turn preserved complete snapshots of both synthetic banks, including
Git files. Only the deliberate external fixture calls changed the preference.
The fixture writes have `fixture` provenance, not invented model authorship.
This proves actual next-turn refresh in this native version without restarting,
reloading or accumulating context. It does not promise that already-read
reasoning changes mid-turn, or that every scoped fact enters the bounded packet.

## Regression coverage and review

Two compiled-runtime adapter tests retain the critical boundaries: an open
connection replaces superseded evidence without writing, and cancellation before
dispatch has no effects while interruption after a real save preserves its
receipt and reaps the Git child. The latter waits for a complete live PID,
restores process environment and closes the connection on failure as well as
success. The focused seven-test compiled-connection suite passed. Full pinned
host CI also passed: 22 Node tests, Go race tests/vet and four target builds.
Docker remained unavailable; this uses the documented host fallback.

Contributor self-review checked signal timing, no duplicate save on recovery,
semantic conflict versus delivery distinction, native/model evidence layers and
complete-bank preservation. No product behavior was weakened to obtain a pass.
Hosted CI and exact-head review remain required for the added tests. The final
rendered success/warning/partial-result and menu matrix is still separate work;
these command/RPC probes are not screenshots or independent product acceptance.
