# Pi engineering evidence

This records development verification, not independent product acceptance,
publication or personal activation. Issue #13 / implementation PR #72 remains
in progress. Owned installation/recovery and the full native model/candidate
matrix must still be completed. Skills and release-stamping engineering evidence
is recorded below; it is not completion of that matrix.

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

## Skills and distribution slice

Source `983dbf47cc875e1b7ae85c3308923ecebe8e19bc` adds native memory/Armorer
skills and embeds Pi's stamped version during the isolated release build. Full
host CI passed: 19 Node tests, Go race tests/vet and four target builds. Docker's
daemon remained unavailable. Both skills passed the skill-creator validator in
an existing isolated PyYAML 6.0.2 environment; no global Python change was made.

Two separate clean builds produced byte-identical inventories of eight files.
Manifest SHA-256:
`4d5e5a004f1c36fdf3a9c0595fff659e14d47129c264c6e6d33572024448d91e`.
The retained macOS arm64 candidate executable SHA-256 is
`a7da45f127f70c2a86f54b6e9749ce98d2fb63ebb91a1f88bb4734339397f166`.
Its Pi package SHA-256 is
`85dfcdaea565bee3226debdd61d5b400bd3220398ec65a531d4af82ec3d813af`.
These local engineering candidates use the current tracked VERSION, `1.0.0`;
they are **not** the published v1.0.0 bytes and were not nominated or published.

The actual published-source installer (`f899cf6a2a255f3b6b35dcd778c672f799c65eb2`)
planned and installed this candidate in a new disposable prefix, passing its
strict version verification. Generic version fields remained unchanged; separate
Pi metadata reported the release stamp and ten embedded files. No live launcher,
native profile, authentication or personal signet was changed.

A second disposable native Pi profile loaded a package whose bytes were checked
against this exact candidate's embedded digest. Both skills were automatically
discovered with zero native diagnostics and exposed their native slash commands;
Pi's initial skill prompt contained discovery metadata, not the full bodies.
All 18 memory tools loaded. Native registration was removed after the check.
The test made no model request and does not establish conversational behavior.
See the [semantic result](skills-distribution-result.json).

The first candidate fixture reconstruction missed Go's HTML escaping in the
manifest. The identity guard refused it; correcting only the fixture's escaping
made its bytes match the embedded digest. No runtime verification was weakened.
Pi also routes extension console output to stderr and can print a command error
while exiting zero. The retained probe was corrected to inspect its explicit
semantic success result, not just exit status or a shell completion marker.
Future owned installation must copy embedded bytes directly, not reconstruct
manifests from parsed JSON. These are engineering findings, not acceptance.
