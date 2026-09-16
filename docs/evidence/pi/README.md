# Pi engineering evidence

This records development verification, not independent product acceptance,
publication or personal activation. Issue #13 / implementation PR #72 remains
in progress. The full native model/candidate and exact-head rendered matrix must
still be completed. Guided menu/owned installation/recovery, skills and release-
stamping engineering evidence are recorded below; they do not complete that matrix.

The [native lifecycle and model baseline](lifecycle-model-baseline.md) records
actual Pi RPC transitions (with a disabled-extension control), an immutable
engineering build and the first Codex model recall against its synthetic bank.
The [delivery and live-refresh checks](delivery-refresh.md) cover compatible
concurrent writes, unresolved semantic conflicts, partial-save cancellation,
explicit recovery and an actual already-open Pi model seeing superseded memory.

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

## Owned connection workflow

Source `a4c143574744b771b12e138a3b940493895a4c41` adds the owned Pi connection
engine and typed/human CLI. Source-stamped development executable SHA-256:
`7edd53debd8b8375246db0b2f1ea39f6ddd012e8720f012ac596763c956bf63e`.
Embedded Pi package SHA-256:
`d12d46064bad1f1f406e0f204d18fa4a69de53df63e42530fd6a0457a4de785f`.
Package version is `0.0.0-dev`; this executable is not a release candidate or
the previously tested distribution slice. Full pinned host CI passed, with
Docker unavailable. Sixteen new Go tests (including parameterized cases) cover
settings inventory, plans, ownership, apply/partial results, doctor/repair and
typed/human parity. The existing 19 Node tests still pass.

On native Pi 0.85.1 / Node 24.1.0 / macOS arm64, the exact executable performed
initial planning without state creation, actual installation, idempotent repeat,
Armorer inspection, native discovery of 18 tools/two skills, enforced read-only
refusal, update, intentional missing-file detection, and repair into a fresh
retained generation. The test verified unrelated local-package resource filters,
theme and a native-profile sentinel, and a complete synthetic-bank file snapshot.
The missing old file stayed missing: recovery did not rewrite that generation.
It removed only the selected synthetic registration afterward.

A second isolated profile used an explicitly selected wrapper that delegated
ordinary operations to real Pi. After native removal during an update, it paused
the new install before delegation and emitted its PID. The observer required a
complete PID line and verified that process was live, then interrupted the
Mandalore CLI with SIGINT. The returned error preserved `install-started`,
uncertain native effects and the earlier completed-removal receipt. The old
registration was actually absent; the paused child was reaped. After removing
the fixture's pause condition, a fresh repair preview restored a healthy native
registration. Both retained generations and the unchanged bank were verified.
No model request, real account authentication or live connection change occurred.
See the [semantic result](owned-workflow-result.json).

The first cancellation observer incorrectly escaped its newline matcher and
timed out despite the complete PID marker. Cleanup stopped the child and retained
the inactive profile/partial receipt. The observer was corrected and the whole
scenario repeated in a new fixture; the failed observation was not counted as a
pass and did not justify changing runtime cancellation behavior. This evidence
does not replace real model/GUI acceptance, selected-runtime menu delegation,
the broader native event matrix, or immutable-candidate acceptance.

## Guided menu and selected-runtime delegation

Source `a43fae481d118c5b7fe3d0ab26361d0593f06180` adds harness selection to
connection/Armorer journeys and a separately confirmed Pi handoff after CLI
installation. It separates Pi defaults from Codex, displays enforced read-only
mode and partial phase receipts, and delegates planning/apply to the selected
runtime. Repair executes the intact runtime identified by the owned receipt,
not the menu's potentially newer embedded package.

Full pinned host CI passed, including 19 Node tests, Go race tests, vet and four
target builds. Docker remained unavailable. Ten additional race-enabled repeats
verified delegated cancellation with a live child, partial output preservation,
child reaping and owned-repair identity/access-mode refusal cases. Menu tests
cover back, default-No, EOF/incomplete consent, invisible output, partial native
failure, separate release/native success, release identity/version mismatches,
and independent profile selection. The first inspection test expected a long
path on one output line; correcting its assertion to account for actual wrapping
resolved the failure without changing the product renderer or weakening the
profile/no-effects checks. Full validation then passed again at the named commit.

The source-stamped menu executable has SHA-256
`6d4f9fc08a7ceef9d755fb1ca065d83cd97d39b875b6412f03de58561cd00ca0`.
In a fresh disposable macOS arm64 fixture, it drove real Pi 0.85.1 / Node 24.1.0
through the plain menu. The explicitly selected source executable was the older
`a4c1435` owned-workflow build identified above, proving that the menu did not
substitute its own executable. Back/default-No left installation state absent
and native settings byte-identical. Confirmed install, read-only mode, healthy
inspection, intentional missing-file detection, and fresh-generation repair all
passed. Recovery selected the previous generation's retained runtime, preserved
its access mode/binding and left the old missing file untouched. Unrelated
relative-package filters and theme remained intact; a complete synthetic signet
snapshot was unchanged. See the [semantic result](guided-menu-result.json).

An earlier development build also exercised the rendered Armorer journey in the
dedicated terminal pane using vim and arrow navigation, default text entry and
normal exit. Inspection clearly distinguished structural passes from untested
login/live tools/remote freshness/context. This smoke recording is retained
locally, not offered as the final exact-head rendered evidence matrix. No model
request, live personal activation, authentication change or publication occurred.

Engineering self-review of this source checked default-No/no-output/no-effects
paths, selected-runtime and retained-repair identity, cancellation/receipt
handling, Codex regression behavior, and release handoff without interpreting
the format-1 Codex hash as Pi's package hash. No unresolved slice-level finding
remains. This is contributor self-review, not independent acceptance. Final PR
reconciliation, full rendered/model verification and hosted checks for this
source remain required; local CI is not hosted CI or release authority.
