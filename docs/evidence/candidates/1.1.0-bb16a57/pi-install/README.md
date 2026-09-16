# Pi install, replacement and interrupted recovery

Fresh engineering evidence using the **actual retained 1.1 candidate executable**,
not a rebuild or synthetic menu/API renderer. Contributor self-review only; no
human acceptance, personal activation or publication. This extends #13/#88.

## Identity and environment

- Source: `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`.
- Manifest SHA-256: `ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`.
- macOS arm64 executable: `0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`.
- Embedded Pi package: `013dcc492565ba180abd59eb50e3a6fbea2cd8ba045ccf067e17d51a050e7fda`, version 1.1.0.
- Actual native Pi 0.85.1, Node 24.1.0, English, Herdr, 30 rows and widths below.

[Evidence manifest](evidence.json) SHA-256:
`c097e67c82b8440df85cf2c324f28185be90fce1c051085b5ec8d6827b0cdc92`.
The manifest binds fresh unedited recordings, public result projections and
hashes of private raw receipts. The candidate manifest/assets and every selected
retained runtime were rehashed. Source and the user's main worktree stayed clean.

A fresh synthetic signet, binding, profile and installation state were used.
The profile began with a theme, a relative unrelated package with resource
filters, an empty auth sentinel and a session sentinel. A separate foreign
profile referenced an inert unowned package named mandalore. No real credentials
were copied, inspected or enrolled. Learning remained enabled throughout.

## Rendered matrix

| Recording | Actual path and result |
| --- | --- |
| [Declined](declined.recording) | 80-column color TUI, Vim/arrows and explicit fixture defaults. Initial preview shows Previous generation: None. Enter on default No; complete original profile unchanged and installation state absent. |
| [Installed](installed.recording) | 80-column color TUI. Explicit Yes; actual native registration and verification pass; fresh-session requirement and no uncertain effects shown. |
| [Interrupted](interrupted.recording) | 80-column plain/NO_COLOR. Explicit replacement approval; native removal succeeds, then the test gates the install invocation. SIGTERM to the verified live gate process produces an `install-started`, installed-false, uncertain-effects receipt and no automatic retry. |
| [Recovered](recovered.recording) | 32-column plain/NO_COLOR. The Armorer previews recovery from the interrupted retained root; explicit Yes installs a fresh generation. Verified receipt, same binding/learning mode, EOF returns to shell. |
| [Foreign refusal](foreign.recording) | 80-column plain/NO_COLOR. Before confirmation, structured `connection.failed` guidance identifies foreign/filtered/ambiguous registration and asks to preserve it. No foreign state directory is created. |
| [Cancellation](cancel.recording) | 80-column NO_COLOR TUI. Vim/arrows, Escape back, then Ctrl+C at the selected runtime prompt. No install/profile effects. |
| [Stale preview](stale.recording) | 80-column plain/NO_COLOR. A separate wrapper is changed after preview. Explicit Yes refuses the stale plan before registration/receipt changes; the deliberately changed wrapper remains as a test fixture. |
| [Uninterrupted replacement](updated.recording) | 80-column color TUI. A later explicit replacement completes actual native remove/install/verification, preserves earlier generations and retains the same binding and access mode. |

Unmodified BSD `script -qr` recordings; replay with macOS `script -p RECORDING`.
Capture children restored terminal settings; the observed outer size afterward
was 67 rows/309 columns. A verifier parsed output frames, checked critical
preview/receipt/refusal/restoration text, and confirmed no SGR styling in
NO_COLOR/plain recordings. Contributor rendered review inspected headings,
phase/effect truthfulness, default-No, cancellation and narrow wrapping. Long
synthetic paths consume many rows; no approved pixel baseline is claimed.

## Interruption and preservation evidence

The pinned regular fixture wrapper delegates version/remove/install commands to
the real installed Pi. For **one install invocation only**, an enabled fixture
gate paused instead of entering Pi's install implementation. Before termination,
the controller verified the exact gate PID live and the native settings with the
old registration already removed. This is an observed native-process failure
boundary, not a keyboard interruption of Pi's install code or a model turn.

Afterward: one failed attempt, phase `install-started`, uncertain effects; child
absent, installation lock absent, structural doctor explicitly disconnected.
The original generation and protected data were intact. Disabling only the
fixture gate allowed explicit recovery from that failed generation's retained
ownership receipt. Recovery created a fresh generation, not a rewrite of the
failed one. The later normal replacement also passed. Final counts: four native
attempts, three verified and one deliberately failed; no automatic retry.

Final independent inventory checks preserved the initial, failed and recovered
generations, all 35 protected entries, signet/binding identities, access mode,
authentication/session sentinels, theme and unrelated resource filters. Foreign
profile/package data were unchanged. Stale and cancelled journeys preserved the
then-current entire profile/installation-state inventories. Changes to the
separate stale wrapper were deliberate controls, not a preservation claim.

## Fresh native startup and control

Two fresh Pi RPC processes loaded the actual profile registration: once after
repair and once after the successful replacement. Neither supplied an explicit
Mandalore extension override. A test observer inspected native inventory without
a model request: 18 unique Mandalore tools, two skills, native read/bash tools
preserved, zero custom context entries, and clean shutdown. This proves loading,
not execution of every tool or provider authentication; prior candidate native
callback/model evidence remains separate.

The first whole-profile equality assertion **failed** after native startup:
Pi created an empty `models-store.json` and changed the profile root mtime.
A fresh otherwise equivalent no-Mandalore control produced the same empty cache,
with zero Mandalore tools/skills. All existing files and managed state were
unchanged. After the final startup, only the root directory mtime differed;
all file bytes, modes and mtimes plus managed state matched. The final manifest
records these effects explicitly, not a zero-profile-effects claim. An unreviewed
draft receipt's overly broad whole-profile flag was corrected before publication;
the draft is retained privately, not used as passing evidence.

One initial rapid batch of Enter keys advanced only one text prompt. Subsequent
defaults were accepted individually after inspecting each prompt. No changed
selection or unintended apply occurred. No product code changed for these tests.

## Verdict and remaining boundary

Engineering pass for this actual-candidate setup/replacement/refusal/recovery
matrix, with native cache/directory effects explicitly retained. The Armorer's
report still marks native login, tool invocation, remote freshness and active
model context untested; inventory observations do not silently promote all of
those to passes. No provider authentication, model turn or hosted synchronization
was requested here.

Published 1.0 has no Pi connection operations: these are candidate-generation
replacement/recovery tests, not an invented 1.0 Pi upgrade. Other-platform native,
screen-reader, alternate locale/font and human product acceptance remain unclaimed.
The original candidate matrix and broader roadmap are not closed by this result.

Full pinned host `mise exec -- bin/ci` passed after retaining this evidence: 22 Pi
tests, Go race/vet and four target builds, including valid Go cache hits. Used
the documented host fallback for the previously unavailable container daemon.
Public copies were checked against original recording hashes, privacy-scanned,
and reviewed to exclude the uncorrected draft's whole-profile assertion.
