# Native lifecycle, refresh and delivery boundaries

Fresh engineering evidence for the [same retained candidate](README.md), source
`bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`. Runtime copies in all newly installed
test connections match retained macOS arm64 SHA-256
`0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`;
Pi package identity is `013dcc492565ba180abd59eb50e3a6fbea2cd8ba045ccf067e17d51a050e7fda`.
Pi 0.85.1, driver Node 24.1.0, macOS arm64. [Verification manifest](lifecycle-verification.json)
binds source/identity, receipt hashes and independently inspected final Git heads.

No implementation change, personal connection update, credential copy, human
acceptance or public release is implied. All memory and local bare origins are
synthetic. Raw native transcripts and machine paths remain private.

## Native lifecycle with a disabled-extension control

Actual native RPC sessions exercised startup, reload, new session, resuming a
session and forking a session. An inert session was constructed using Pi's own
SessionManager solely to give resume/fork a target. Its scripted messages are
not model output or learning evidence. An observation extension recorded native
events, registered tools, skill commands and stored custom-message inventory;
it did not replace Mandalore's implementation. No model was requested here.

The [control](lifecycle-control-result.json), with Mandalore disabled, and the
[candidate](lifecycle-result.json) both emitted:

`startup → reload → new → new → resume → resume → fork → fork`

These repeated native events are not a Mandalore-created duplication. Five
inventories and five shutdown events were observed in each run. At every
candidate inventory there were exactly 18 memory tools and two memory skills;
the control had zero. All tool names remained unique; native read/bash tools
remained available. Neither run accumulated persistent custom messages. Complete
file-byte hash snapshots of both synthetic banks/remotes remained unchanged.
This snapshot claim covers existing file paths/content, not directory mtimes.

## Real model refresh without restarting

A separate actual Pi model process used gpt-6-astra/high, existing native login
in place and only the explicitly selected candidate package/skills. It answered
an ordinary read-only greeting question with **River**. A fixture CLI call then
superseded that same preference with **Willow**, while the process stayed open.
The next identical question returned **Willow**. Native state confirmed an
unchanged session ID. Both model turns preserved bank file-byte snapshots; only
the external fixture calls changed memory.

The [observed context](refresh-result.json) contained one attachment per turn,
2,472 then 2,473 UTF-8 bytes, and zero persistent custom messages. The second
attachment contained Willow and no longer contained River. This is actual
next-turn freshness, not retroactive alteration of already-read reasoning or
a promise that every scoped fact enters the bounded packet. Fixture-authored
changes retain fixture provenance, not model authorship.

## Actual native tool callbacks and cancellation

Two new clones/devices and owned Pi connections shared a local bare remote.
Two actual Pi RPC processes loaded an observation wrapper around the unchanged
candidate extension. The wrapper forwarded registration, used Pi's native
argument validator and invoked the registered callbacks. These are actual
native schema/transport checks, **not** model tool choice, rendered tool UX or
an Escape-key interruption test.

The [18-call receipt](native-delivery/result.json) establishes:

- Cancellation already signaled before dispatch returned `operation.cancelled`
  with no possible-write flag, preserving the complete bank file snapshot.
- Compatible journal writes in separate clones both survived integration.
- Unavailable-origin combined save retained its durable ID and reported pending
  delivery; explicit recovery found exactly one entry with that ID.
- Interruption during an observed fetch retained outer save success and nested
  cancellation, checkpointed/pending phase, possible-write flag and explicit
  inspection requirement. The Git child was reaped. Explicit recovery delivered
  the original entry without another save.
- Different revisions of the same baseline survived integration. Delivery
  succeeded with one semantic conflict; recall exposed no authoritative current
  winner. Independent inspection matched local/remote/receipt head
  `585ea42acc3fdb85c99dc3d460c8fdbee3e3ea6b`.

For cancellation, a fixture-only Git wrapper delegated ordinary calls to system
Git. It paused at `ls-remote` after writing its complete PID line. The observer
required that complete marker and a live process before aborting the native
tool signal, then verified the child no longer existed. No observation timeout
was mistaken for successful interruption. Every callback returned one complete
text envelope and only operation/ok metadata, including partial receipts.

## Contention and ambiguous push acknowledgement

A separate fresh fixture supplies the [seven-call receipt](native-boundaries/result.json).
An owned process acquired the actual writer lock and confirmed it was live.
Both combined save kinds and standalone sync returned `store.busy`, with no
possible-write flag. Bank file hashes and actual local/remote heads remained
unchanged. Releasing that lock allowed a later explicitly requested save and
delivery. This tests contention **before** publication; the between-save-and-sync
lock boundary remains separately covered by API tests, not this native fixture.

For acknowledgement ambiguity, a fixture wrapper performed the real local-origin
push successfully, then returned exit 128 to its caller. Mandalore preserved the
saved ID but truthfully reported pending push/unconfirmed delivery. It did not
infer success from local persistence. The controller inspected the actual remote
before recovery, proving that the pushed head was already present. It read the
journal by ID and performed one standalone sync, never repeating the save.
Semantic file hashes stayed unchanged through recovery; actual head was
`beaefcb0014d7d914d139ca731ee455cc6395a2e`.

This is a controlled Git acknowledgement failure, not a live-network incident or
a dropped native-client response. It proves conservative reporting and safe
explicit recovery in the tested condition, not automatic retry behavior.

## Remaining boundaries

These results add exact-byte lifecycle/callback evidence; they do not complete
the original issue matrix. Rendered interruption/partial-result UI, additional
installation/update/repair cases, quota/readiness terminal matrices, broader
foundling quality and isolated published-1.0.0-to-candidate recovery remain.
The earlier 512-byte repeated-preview finding remains retained. No cross-platform
native, screen-reader, alternate-locale or statistical model-reliability claim is
made. #55 O1/O2 automatic lifecycle synchronization and authorized human product
acceptance remain separate unresolved work.
