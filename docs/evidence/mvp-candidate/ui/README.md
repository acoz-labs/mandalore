# Exact-candidate rendered installer and repair checkpoint

Contributor engineering verification for #10/#11; not independent product
acceptance or release authority. These are fresh recordings of the retained
candidate, not relabeled implementation screenshots.

## Identity and environment

- Source: `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`.
- Candidate manifest: `c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
- Menu driver and selected/installed runtime: `eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
- Embedded plugin: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- Codex 0.153.4 macOS ARM64 binary: `b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.
- Actual rendering: macOS ARM64, designated Herdr pane, English, keyboard input,
  30 rows, 57 columns except the 24-column narrow scenario.

The retained distribution was copied, not rebuilt, into a fresh synthetic fixture.
All candidate checksums were checked before and after. A byte-identical Codex
executable and an isolated unauthenticated profile tested native installation.
The synthetic signet and external binding were prepared explicitly. No real bank,
native profile, account, shell configuration or global trust setting was changed.

## Actual scenario matrix

The files below are unmodified macOS `script -qr` recordings beginning inside the
child application, not reconstructed output or native model transcripts. Their
absolute temporary paths deliberately identify only synthetic test fixtures.

| Recording | Interaction and observed outcome |
| --- | --- |
| [default-no](default-no.recording) | 57-column color; j then k returned to No, Enter cancelled. Prefix absent. |
| [cli-success](cli-success.recording) | Down/Enter approved CLI installation. Receipt complete; exact launcher shown outside PATH. Default choice kept native connections unchanged. |
| [native-update](native-update.recording) | Plain/NO_COLOR; explicit CLI approval then optional native handoff and separate approval. Missing native profile caused preflight failure; CLI remained complete. |
| [native-ready](native-ready.recording) | After creating only the empty fixture profile, repeat explicit approvals and fixture defaults. CLI already-current; native phase verified, fresh-session/hook-trust notice shown. |
| [doctor-repair](doctor-repair.recording) | NO_COLOR TUI; Vim navigation to doctor, healthy checks plus separate not-tested items. Explicit retained-root repair and default-No approval generated a new connection; subsequent doctor healthy. |
| [partial](partial.recording) | Color; actual unwritable fixture bin directory caused launcher publication to fail. Runtime/pending plan retained; incomplete status and exact-plan recovery direction visible. |
| [recover](recover.recording) | Restore only fixture owner-write permission; original candidate reads the retained pending record directly. Same-plan activation completes. Output remains machine-readable JSON. |
| [narrow](narrow.recording) | 24-column plain/NO_COLOR; complete wrapped identities/effects, `:back` cancels. Prefix absent. |
| [escape](escape.recording) | Color; Escape at default-No confirmation. Prefix absent. |
| [missing-hook](missing-hook.recording) | One cached hook file moved to a retained external backup. Doctor reports incomplete cache; explicit repair from owned receipt followed by doctor restores healthy state. |

The missing-profile failure was investigated using the isolated native CLI: Codex
explicitly refused a nonexistent `CODEX_HOME`. No authentication was needed for
installation once the empty directory existed. This is retained as an actual
failed first attempt, not hidden as immediate success. The main menu's doctor
distinguishes missing profile, but the delegated handoff's failure text is less
specific: it reports preflight/incomplete rather than the underlying native cause.

## Verification beyond the screen

Both complete and recovered launchers returned the exact candidate version and
plugin identity. Recovery cleared pending state; replaying the saved plan reported
already-current/no destination change. All regular-file hashes and the launcher
target matched across that replay. No new plan was substituted.

Final doctor reports 12 passing structural/inventory checks and five not-tested
checks: login, hook trust, live MCP, remote freshness and active context. Actual
plugin validation passed. Native installation is not a model-session acceptance.

The original and both repair generations remain retained:

- `8fc0fe74638f3b8f8a599da6d89ecd1ac87cdc46fffdc745fd00e708aec3fb0e`
- `26fa5b37fe773f58a35b6ce3d357d00e3e8189642028efe519edaeadc05b0f9e`
- `cb064c23a068f377bffb2ec995bec1c0ec3d72f307ed841883b1442ac81c7b4a`

Repair restored a hook file byte-identical to the saved copy. The native cache can
be replaced by normal plugin installation; retained managed source generations
are separate from that cache. The missing-hook menu invocation encountered a real
failure before recovery; its accumulated failure exit status does not negate the
later healthy doctor result.

All synthetic signet-file and binding hashes remained unchanged across the whole
journey. The unrelated earlier native-test bank also matched its prior baseline.
The isolated profile contains no `auth.json`; no native session or login was
launched here. Existing authenticated native-session evidence stays separate in
[native.md](../native.md) and [live-recovery.md](../live-recovery.md).

## Rendered judgment, replay and remaining scope

Contributor self-review observed clear headings, explicit pass/fail/not-tested
states, default-No choices and complete identities. Long paths and hashes consume
considerable vertical space at 24 columns. The native receipt leaves an absent
previous source blank; the CLI receipt renders absence as None. These are not
claims of independent product approval or an approved visual-regression baseline.

On macOS, replay in a disposable terminal at the recorded width:

```sh
script -dpq docs/evidence/mvp-candidate/ui/cli-success.recording
```

Omit `-d` to preserve timing. BSD recording playback is not portable to Linux's
util-linux player. It re-emits terminal queries and may leave reply characters at
a shell prompt; do not submit those characters. The recording hashes are in the
parent `SHA256SUMS`. No screenshot, screen-reader, alternate locale/font, browser
or mobile coverage is claimed.

Evidence privacy, whitespace and checksum checks passed. Full pinned Go 1.26.4
host CI passed with cached tests after an actual container attempt could not reach
the Docker daemon. Builds are not additional native-platform execution.

This checkpoint covers core installation, same-candidate handoff, actual repair
and partial recovery. The subsequent [switch/refusal checkpoint](switch-and-refusals.md)
adds eleven actual recordings for installer negative/source-selection cases and a
distinct retained-runtime switch; older implementation recordings are not relabeled. Native platform matrix,
independent acceptance and publication prerequisites remain separate work.
