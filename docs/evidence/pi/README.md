# Pi engineering evidence

This records development verification, not independent product acceptance,
publication or personal activation. Issue #13 / implementation PR #72 remains
in progress. Installation/recovery, skills, release stamping and the full native
model/candidate matrix must still be completed.

## Native package and callback probe

Tested source: `06bf8c370d3b262e9177061ccce08f62685bef91`.
Source-stamped development executable SHA-256:
`4ba013a1fe9f554fd6fc332dd72c5b911c24d9d2ca38d6395894de2277d815af`.
Embedded/public Pi package SHA-256:
`96282e6d08ce6df1e0388dbe05e944b6341d439e4fd916568549ddf0a1cf683f`.

Environment: macOS arm64, Node 24.1.0, native Pi 0.85.1, dedicated terminal test
workspace. An empty disposable Pi profile and unrelated project directory were
used with a synthetic signet. No native authentication, session or private memory
was copied; no model request was made.

The first invocation registered the local package using real `pi install`, then
started ordinary native extension discovery plus an observation-only command.
It saw all 18 memory tools, retained native read/bash tools, and excluded the
CLI-only context/administration operations.

The second invocation explicitly loaded the same package factory through a
probe wrapper with ordinary native package auto-loading disabled, avoiding a
second duplicate attachment. The wrapper retained the registered tool callbacks
while forwarding registration/events to Pi. Its native command passed arguments
through Pi's actual validator and invoked those same callbacks. It saved a
synthetic preference, superseded it, recalled the current revision and verified
both historical revisions with `pi` authorship. It invoked the registered
per-turn callback with the real extension context, retained the native prompt
prefix, returned 2011 UTF-8 bytes, excluded the superseded value and preserved the
fixture files during that read.

Native removal then succeeded and `pi list` reported no installed package in
that disposable profile. The pane returned to its original shell. See the
[semantic result](native-contract-result.json).

This proves real package loading and native callback/schema compatibility. It
does **not** prove model tool selection, passive learning quality, actual
model-turn event dispatch, rendered tool-result UX, cross-harness model behavior,
Linux Pi execution or product acceptance. A command-only probe is not relabeled
as a conversation. No raw private transcript or workstation paths are published.

## Automated layers

At this source, the full pinned host `bin/ci` passed. Docker's daemon was
unavailable; the documented host fallback was used. Seventeen Node tests cover
the extension controller, bounded shell-free transport, process-group shutdown,
deadline escalation, complete partial receipts, a compiled Go runtime, package
identity, binding/runtime changes, read-only enforcement and provenance. Go race
tests, vet and all four target builds also passed. Cross-builds are not native
acceptance. Hosted checks for this source are a separate requirement.

## Review follow-through

Source snapshots and digests here bind only the named development slice. Later
changes require reconciliation and relevant repeats. Final native model tests
must use the finished skills and setup flow; immutable-candidate acceptance must
repeat its own matrix with fresh evidence under the repository release gates.
