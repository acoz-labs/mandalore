# Published 1.0 → retained 1.1: owned upgrade/recovery checkpoint

Actual published and retained executables, not a rebuild or API double. Fresh
synthetic prefix, Codex profile, installation state, binding and signet. No
personal activation, public release or human acceptance. The original candidate
guide and issue criteria remain applicable beyond this checkpoint.

## Identities and environment

- Candidate source: `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`.
- Candidate manifest: `ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`.
- Candidate macOS arm64 executable: `0679b27eaefddb842ec54e03d966dc8369b39c63842bd7ddd74e2f058505c37e`.
- Published source: `f899cf6a2a255f3b6b35dcd778c672f799c65eb2`.
- Published manifest: `c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea`.
- Published macOS arm64 executable: `bcc1b7fdf9e72ccb2bbd734d798c16f3aa5b96de757846711a4e5afe6eb00f93`.
- Environment: macOS arm64, Codex 0.154.0, Node 24.1.0, dedicated Herdr terminal.

Downloaded all eight 1.0.0 files afresh from the official published release URLs
using unauthenticated HTTPS. Checked each against the pinned
[publication receipt](../../../../releases/1.0.0/public-verification.json), then
used those verified local files for installation. This does **not** exercise a
live GitHub release-API apply. Rechecked candidate manifest and all six manifest
assets against their retained identity; prior transport/nomination evidence
remains in the parent checkpoint. Foreign-platform binaries were not executed.

[Evidence manifest](evidence.json) SHA-256:
`4c700a405ef2881be8dead40ad00d4a457ae92c28968d7c1954dec2e8ef56349`.
It contains public projected assertions, identities, structural-check results,
recording hashes and hashes of the retained private raw administrative receipts.
Raw native executable paths and profile inventories are not published.

## Actual sequence and effects

1. The downloaded published 1.0 CLI created a synthetic signet/binding and one
   canary record, then installed the published runtime in an owned prefix.
   Initial native Codex plan/apply and structural inspection passed using an
   empty synthetic auth file, session sentinel and unrelated-file sentinel.
   No provider login or model request occurred in this profile.
2. The published CLI planned installation of the exact candidate files. After
   preview, changed only the fixture launcher directory to mode 0555. Actual
   apply retained the candidate runtime and exact pending plan, then failed at
   launcher creation with phase `pending-recorded`. The old launcher remained
   active. The receipt correctly reported possible writes and inspection before
   retry; the entire native installation-state inventory stayed unchanged.
3. Inspected the saved pending record, restored only that directory to 0755,
   and passed the unedited pending record to the original published CLI's
   `release apply`. It completed with the **same plan digest**, cleared pending
   state and activated the byte-identical 1.1 runtime. Its actual `version`
   response matched candidate source/package/protocol/schema identities.
4. Replayed the completed plan once: `already_current: true`,
   `destination_changed: false`; complete prefix inventory unchanged. CLI
   installation did not update the native connection implicitly.
5. Explicitly planned/applied the installed candidate's Codex connection.
   Actual native registration and structural inspection passed. The new
   connection retained the same signet and binding, selected the candidate's
   package/runtime, and reported that a fresh session is required. The old
   generation remained byte/type/mode/mtime-identical.
6. The **retained 1.1 executable** drove its real rollback menu. Enter on default
   No left complete CLI-prefix/native-state inventories unchanged. A second
   journey in narrow plain/no-color mode selected the reviewed retained 1.0
   manifest and explicitly applied it. Declined the optional native handoff.
   Actual launcher hash became the published 1.0 hash; native state remained
   on 1.1. Then the published CLI explicitly reactivated the retained 1.1 runtime.
7. Final verification rehashed both distributions, both retained native runtimes
   and the active launcher; compared all original CLI-generation entries and
   the original native generation; confirmed no pending record/install lock;
   and compared all 32 protected inventory entries. Memory, binding, synthetic
   authentication/session sentinels and the unrelated file were unchanged.

The original 1.0 executable is the updater/recovery driver in steps 1–4; the
actual retained candidate executes native update and the rollback menu. This is
not evidence that 1.0 supports Pi: it does not. Pi installation, generation
update and interrupted recovery require their own candidate matrix.

## Openable rendering evidence

| Recording | Surface and result |
| --- | --- |
| [Rollback declined](rollback-declined.recording) | Retained 1.1, 80 columns/30 rows, color TUI. Source/effects/retained generation visible; Enter on default No; stopped without effects. |
| [Rollback applied](rollback-applied.recording) | Same executable, 32 columns/30 rows, plain/NO_COLOR. `2`/Enter explicitly applies; complete receipt, no pending record; Enter keeps native connections unchanged. |

Unedited BSD `script -qr` recordings; replay with macOS `script -p RECORDING`.
The capture child restored terminal settings, with final outer dimensions
67 rows/309 columns. The verifier parsed frames and checked critical preview,
receipt, completion, restoration and no-color behavior. Contributor rendered
review inspected default-No, narrow labels, phase/previous-target truthfulness
and the separate native handoff. Long synthetic paths wrap over many lines;
the #77 narrow copyability limitation remains, not a new accepted baseline.

## Controller finding and limits

The initial driver assertion looked for `source.manifest.version`, but the public
schema is `source.manifest.manifest.version`. Initial installation, native setup
and upgrade planning had already succeeded. Corrected the assertion, inspected
the saved receipts, then continued from the existing state **without reapplying
those operations**. This was a test-driver mistake, not a product failure or a
discarded failing installation.

Engineering verdict: pass for the recorded CLI/Codex upgrade, partial activation,
recovery, replay, rollback/re-upgrade and preservation boundaries. Structural
inspection explicitly leaves native login, hook trust, live MCP, remote freshness
and active-session context untested. This fixture does not silently promote those
checks to passes; previous candidate model tests remain separate evidence.

No human product verdict, provider authentication, hosted synchronization,
screen-reader result, alternate locale/font or other-platform native result is
claimed. Candidate source and the user's main worktree were unchanged. The
published v1.0.0 release, personal runtime and native profiles were not modified.

Full pinned host `mise exec -- bin/ci` passed after adding this checkpoint (22 Pi
tests, Go race/vet and four target builds; valid Go cache hits retained). Used the
documented host fallback for the previously unavailable container daemon. Public
recordings were compared with original hashes and the published evidence was
scanned for private home paths and credential markers.
