# Retained-candidate Pi display and narrow setup journeys

Fresh contributor-rendered verification for #13 on September 16, 2026. This
repeats the remaining display/narrow/EOF cases from the original
[Pi menu matrix](../../../pi/menu/README.md), not human product acceptance.
The other candidate installation/recovery journeys remain in the separate
[installation checkpoint](../pi-install/README.md).

## Identity and native environment

Source: `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`.
Manifest SHA-256:
`ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`.
Actual retained macOS arm64 executable SHA-256:
`0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`.
Embedded Pi package SHA-256:
`013dcc492565ba180abd59eb50e3a6fbea2cd8ba045ccf067e17d51a050e7fda`.

Environment: macOS arm64, Pi 0.85.1, Node 24.1.0, gpt-6-astra/high, English,
keyboard input in the dedicated Herdr pane. Two fresh synthetic signets and Pi
profiles were installed through candidate plan/apply and independently inspected.
Each preserves an unrelated relative package/filter, theme and empty synthetic
auth/session sentinels. No provider credential is copied into those profiles.

The model display process inherits existing authentication in place, disables
ambient extensions/skills/context files/templates/themes and sessions, and loads
only the selected owned candidate package and its two skills explicitly. This
establishes native rendering, not another implicit-profile-loading result; the
installation checkpoint separately verifies real profile registration. It does
not claim complete native-home immutability or alter a personal connection.

## Actual recordings

| Recording | Journey and observed result |
| --- | --- |
| [Narrow plain/no-color](narrow.recording), 32×30 | The Armorer selects Pi, reviews explicit fixture paths, and receives explicit confirmation before native inspection. All structural/version checks pass; login, live tools, remote freshness and active context remain visibly untested. Full wrapped connection identity is retained. Harness Back returns correctly. Connection setup is then entered and EOF at its first runtime prompt exits without planning/applying a change. |
| [Native tool results](tools.recording), 100×34, color | Actual Pi loads the two candidate skills and extension. One explicitly requested read-only inspection displays a healthy envelope. The next task explicitly authorizes one journal-and-sync call. It displays one result body containing a durable local save and nested failed/pending delivery at checkpoint. The model clearly distinguishes those outcomes and does not retry or initialize Git. |
| [Unavailable-memory warning](warning.recording), 100×34, color | The controller changes only the separate fixture binding's actor before startup, invalidating its pinned digest. No model prompt is submitted. Pi shows a concise unavailable-memory warning with Armorer inspection guidance, no raw child diagnostics, and no automatic repair/sync. It remains usable and exits normally. |

Contributor rendered judgment: **pass for these journeys**. Pi's normal tool
surface shows complete wrapped JSON and no second metadata/result copy per
displayed call. This is a visual judgment of the recorded interface, not a count
of raw terminal redraws or a wire/model-context benchmark. Native UI usage/cost
labels are not a billing claim. All three applications exited 0. Terminal modes
matched before/after excluding transient PENDIN; wrapper restoration also
verified original dimensions.

## Effects and corrections

Full inventories cover signet, binding, owned installation state, synthetic
profile and unrelated package, including paths/types/modes/mtimes/bytes. Narrow
inspection changed only the profile root directory mtime, not any file or owned
installation state. Its protected empty auth and session sentinels are unchanged.
The subsequent read-only model turn preserved the complete post-inspection
fixture inventory.

The write turn added exactly one event:
`event-69c03f634aaac8fc9d99d75290577722`, with Pi authorship and summary
“Synthetic native display test: local save and delivery are separate outcomes.”
Only its new date directories and the events directory mtime accompany that
append. Every previous memory file, binding, profile, unrelated package and owned
runtime/package state is unchanged. No knowledge record or Git repository was
created. The absence of Git explains checkpoint failure; local durability is
verified from the saved event, not inferred from a partial delivery response.

The warning fixture preserves its deliberately changed binding for inspection;
it is not silently restored to look healthy. Its complete post-controller
inventory is unchanged by startup/exit, and every other protected path also
matches its original setup inventory. It requested no model work.

Two observer corrections are retained: a controller assertion stopped because a
20-line screen read omitted the main-menu heading, although the visible choices
proved the correct screen; the same recording continued without repeated
navigation. The recording checker initially stripped CSI but not OSC hyperlink
resets, interrupting a textual warning match. It now strips both for text checks
while scanning **raw** output for private paths/tokens. Original recording bytes
and native runs are unchanged; no application fix or rerun was needed.

## Evidence and boundaries

[Manifest](evidence.json) binds all three original recordings, candidate hashes,
dimensions, outcomes, fixture effects and limitations. SHA-256:
`de80fec9308c470cd71fdbed547ffc70b889b3948c710df55eadfc124831795b`.
Recordings contain only owned synthetic temporary paths, not private profile
paths, credentials or personal memories. Open BSD recordings with macOS
`script -dpq FILE` in a disposable terminal; replayed terminal queries can
produce replies. No screen-reader, alternate locale/font or non-host-platform
verdict is implied.

These are deliberately scripted display requests, not another proof of passive
learning. Ordinary learning, continuity, native lifecycle and failure receipts
have their own candidate evidence. Human acceptance, publication and personal
activation remain separate gates; Claude still waits for Pi acceptance.
