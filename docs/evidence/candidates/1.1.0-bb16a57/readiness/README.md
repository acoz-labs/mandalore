# Exact-candidate readiness terminal checkpoint

Contributor verification for #85/#88, using the [same nominated candidate](../README.md)
without source or runtime changes. Actual retained macOS arm64 executable SHA-256:
`0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`,
source `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`, version 1.1.0.
The [evidence manifest](evidence.json) binds actual recording hashes, dimensions,
actions, effects and limitations. This is not human product acceptance, a release
or a personal installation update.

## Method and fixture

Actual retained CLI menu in the dedicated Herdr terminal, macOS arm64, English,
32 rows and 80/32 columns. Native wrappers target Codex 0.154.0 and Pi 0.85.1;
their owned test profiles/connections use the retained candidate. Driver Node
24.1.0. Authentication is not copied or tested by readiness. No browser/mobile
surface is involved. Full terminal recordings are unmodified BSD `script -qr`
files; replay with macOS `script -p <recording>`. They contain only synthetic
temporary locations/identities, not real user/profile paths or secrets.

A fresh synthetic signet and explicit local binding are separate from the test
profiles. A Node dependency trap would create a marker if executed; no such
marker appeared. Missing selections remain nonexistent. The foreign-artifact
case uses preserved, digest-verified published 1.0.0 Linux amd64 bytes in an
expressly fabricated owned-metadata fixture. Those foreign bytes were never
executed on macOS, and the fixture is not a native registration.

## Native recordings and rendered judgment

| Recording | Actual scenario and result |
| --- | --- |
| [Alias-path observation](codex-color.recording) | 80-column color; `/tmp` binding refused as unsafe metadata, partial report and sanitized prompt. Default-No did not run native checks. Editing the binding to its canonical path made binding metadata verified-static; the remaining aliased selection still failed retained ownership matching. Includes a controller navigation error, disclosed below; not a clean configured pass. |
| [Configured Codex](codex-canonical.recording) | 80-column color, canonical paths and explicitly selected retained root. Static completion kept support/setup/evidence distinct; details and optional prompt showed limits. Default-No preserved fixtures. Explicit Yes passed native structural checks, while login/hook trust/live MCP/remote freshness/context stayed not-tested. Escape/Back returned to the main menu and Exit. |
| [Configured Pi](pi-plain.recording) | 32-column plain/NO_COLOR, numbered selection. Stacked support/setup/evidence remained legible, details and prompt retained full information. Default-No preserved fixtures; explicit Yes passed native registration/version/identity checks without claiming authentication or active context. |
| [Missing selection](missing-narrow.recording) | 32-column TUI/NO_COLOR, Vim/arrows; missing native/profile/state/binding observations and separately authorized next-step guidance. Default-No left missing locations absent. Ctrl+C restored terminal state. |
| [Malformed binding](malformed-eof.recording) | 80-column plain/NO_COLOR; partial/inconsistent binding with useful bounded guidance. Raw malformed content was not printed. EOF exited and restored terminal state. |
| [Unsupported retained bytes](unsupported.recording) | 80-column color; foreign Linux artifact remained verified-static as bytes but unsupported for the host. Guidance did not equate unsupported/historical evidence with a required upgrade. No foreign executable was run. |
| [Changed native fingerprint](stale-native.recording) | 80-column TUI/NO_COLOR; deliberate fixture-wrapper comment changed its hash. Static retained selection became inconsistent. Explicit native inspection reported failed native-executable identity, retained the original assessment snapshot, made no repair/retry and exited 1. |

Contributor rendered judgment: the exercised canonical flows, incomplete states,
separate native-inspection handoff and cancellation paths behaved as intended.
Status labels remained visible without color; narrow reports stacked fields
rather than splitting status words into columns. Narrow TUI choice/help labels
are ellipsized by the existing console; full descriptions are in surrounding
reports and plain mode. Paths/hashes wrap without being omitted in details.
This is review of actual rendered terminal output, not an approved pixel baseline
or an independent acceptance verdict. Screen readers, other fonts/locales and
other native platforms remain unverified.

All recordings report restored terminal modes. The wrapper restored the original
terminal settings/dimensions; its mode comparison explicitly excludes Darwin's
transient PENDIN bit. Intentional stale-inspection failure exits 1; other captured
menu journeys exit 0. EOF/Ctrl+C are normal menu exits here, not successful
interruption of an in-flight native network operation.

## Typed static matrix and effects

[Initial ten-case receipt](static-initial.json) covers configured, unselected
retained root, missing and malformed states for both harnesses, plus unsupported
retained bytes and an aliased binding. [Stale receipt](static-stale.json) adds
changed native identity. The actual retained binary's `connection_assess` handled
all eleven cases. Every call preserved complete protected fixture inventories:
paths, modes, sizes, mtimes, symlink targets and file hashes. No missing locations
were created, no Node trap fired, and optional prompts contained no fixture paths
or raw malformed content. This is not system-wide network isolation evidence.

Static menu assessment and default-No also preserved the initial inventories.
After all separately approved native checks, independent aggregate comparison
found only the intentional wrapper mutation and mtime changes to a native
temporary directory and the Pi profile directory. Modes, sizes and contents of
those directories were unchanged; all other protected entries, including memory,
bindings and managed generations, were unchanged. No repair or synchronization
was performed. Native inventory is explicitly allowed to make such bookkeeping
changes; it is not the same operation as the non-executing static assessment.

## Retained findings and controller corrections

The first fixture launcher used `/tmp`, a macOS alias of `/private/tmp`, while
setup recorded canonical paths. The static assessor deliberately refuses aliased
binding metadata and mismatched retained selections; it reported
`metadata-unsafe` / `retained-receipt-or-selection-invalid`. Correct canonical
selections passed. **Diagnostic usability remains a limitation:** the short
finding does not directly explain that an alias is involved. Do not call the
initial configured attempt a pass, silently repair paths or relax metadata guards
to make it pass. Keep this observation available for product judgment.

During that same first recording the controller attempted unsupported Herdr
`home` keys, then sent path text without a successful navigation transition.
This was a driver error, not a product-key failure. The run was stopped; the
protected inventory was verified unchanged. Later runs use supported keys and
check each transition. The original recording is retained rather than replaced
with the cleaner run. In the Pi run, an immediate observation arrived before
native inspection completed; the same live operation subsequently finished.
No restart, repeated native check or timeout-as-success inference occurred.

## Remaining gates

This checkpoint supplies fresh retained-byte readiness evidence, not the entire
candidate matrix. Quota presentation requires separately labeled source-bound
synthetic renderer evidence; install/update/repair/recovery and the remaining
foundling-quality matrix still need their own results. Original issue criteria,
human acceptance authority, publication and live-installation gates remain.
