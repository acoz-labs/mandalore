# Retained-candidate foundling terminal matrix

Fresh contributor-rendered verification for #57 on September 16, 2026, following
the [original terminal matrix](../../../foundlings-progressive/README.md#rendered-terminal-journeys).
This is engineering evidence, not human product acceptance. The actual retained
1.1.0 executable, not a source-built test double, ran in the dedicated Herdr pane.

Source is `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`; manifest SHA-256 is
`ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`;
macOS arm64 executable SHA-256 is
`0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`.
No executable, live installation or published artifact was modified.

## Actual journeys

| Recording | Controls and observed result |
| --- | --- |
| [Color](color.recording), 80×40 | Vim and arrows select a reference. Default Back declines further search pages; a repeated search follows three Next selections through all twelve documents. The last page returns directly to Reference actions. Default read reports 1,024 bytes, incomplete, with continuation; explicit 8,192 returns the entire 1,717-byte final note and END-NOTE-11. Escape and default Back return through the menus. Exit 0. |
| [Plain/no-color](plain.recording), 24×40 | Numbered controls reach all four pages and expand the final note. Complete hashes and locators wrap without disappearing. An 8,193-byte request is refused with the supported range; a subsequent `:back` cancels the read. The earlier validation failure correctly leaves process exit 1. |
| [Budget recovery/no-color](budget.recording), 80×40 | Metadata-heavy first item exceeds the default 8,192-byte result budget. The warning explicitly says this is not a no-match result. Enter on default Back declines expansion; repeating the search and selecting 32,768 explicitly returns the attributed preview. Exit 0. |
| [Changed source/no-color](stale.recording), 80×40 | After the first page is displayed, the controller adds one eligible synthetic file. Next refuses changed content, does not expose that file, and offers no misleading budget retry. No repin, repair or reconnection. Expected exit 1. |

The plain-mode controller expected cancellation to return to the main menu. Its
assertion stopped before sending further input; inspection showed the correct
parent Foundlings menu. The controller continued the **same** recording via Back.
This was an incorrect test expectation, not an application failure or a hidden
rerun. An immediate read after the color process exited was initially blank;
the same pane was reread to confirm completion rather than restarting it.

## Fixture integrity and rendering judgment

Each journey has a fresh synthetic signet, binding and source. The twelve ordinary
notes retain the original pin
`6646dc92ab5a39c30998c00d561b32184f0748bb9d1285ddda34c5daa90368f6`.
The metadata-heavy source retains
`32defb28e947d3af016cc172053404812b733df61bc4932f88428d1562d36d1c`.
Old fixtures and recordings were not reused as new evidence or overwritten.

Complete before/after inventories include hidden files, type, modes, mtimes,
sizes and content hashes. Every signet and binding stayed unchanged. Sources
stayed unchanged except the explicitly controller-added stale file and its parent
directory mtime. Existing stale-source files were unchanged; the post-controller
inventory remained identical through Next/refusal/exit. Terminal mode and window
dimensions matched the original values after each capture wrapper restored them.

Recorded output confirms all twelve document locators and full hashes, page
ranges 1–3/4–6/7–9/10–12, completeness and continuation labels. The narrow output
wraps fields rather than eliding them. Color capture contains SGR styling; the
three no-color captures do not. Next/Back, default refusal, full-read expansion,
budget refusal and changed-source errors remain distinguishable. Contributor
rendered judgment: **pass for this matrix**, with the controller correction above.
This does not promise copyability of arbitrary wrapped recovery commands (#77).

## Retained evidence and limits

[Evidence manifest](evidence.json) binds all four original binary recordings,
dimensions, application outcomes, pins and inventory checks. Its SHA-256 is
`ec41127e31b337f8d2e962e04eaddeb99f2bf1d7a36fdff54aed727e1ba3c81a`.
Recordings contain only owned synthetic temporary paths; no native user profile,
authentication, personal memory or model transcripts are included. They preserve
original bytes and can be opened with macOS `script -dpq FILE`; replay in a
disposable terminal because recorded terminal queries may produce replies.

This repeats the original rendered retrieval matrix with exact candidate bytes;
it does not erase the [native repeated-preview/empty-EOF findings](../retrieval/README.md),
prove context savings, or complete the wider candidate/roadmap. Human acceptance,
other platforms, screen readers, alternate locales/fonts and an approved pixel
baseline are not established. Reconcile remaining original candidate-guide gaps
before presenting a release decision.

The initial pinned `mise exec -- bin/ci` passed before the new receipt was tracked;
its tracked-content privacy scan therefore did not cover that receipt. The next
CI run caught a slash-separated explanatory phrase that resembled a home path.
Inspection established a wording false positive, not a private path or credential.
Only that limitation sentence was reworded; recordings and all measurements are
unchanged. The old receipt digest `62d81e26af3ca2744fa93e3292cc76e246466f8fcdb3ba393fe7f909c6ceaf11`
remains bound to the earlier commit, not to the corrected receipt above.

Validation must run with all evidence files staged/tracked so the privacy scan
covers the complete change. Docker is unavailable; use the documented pinned host
fallback. Cached Go results are not fresh native runs; the four recordings were
executed separately against the retained release-candidate executable.
The corrected staged checkpoint passed full host CI, including the public-content
scan, 22 Pi tests, Go race-enabled checks, vet and four target builds.
