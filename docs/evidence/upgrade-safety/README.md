# Upgrade safety engineering evidence (#112)

## Native component run — September 18, 2026

Product source: `18e607a4a423e81274a7a152140bd34c5baa2c67`.
Candidate executable SHA256:
`7b45444c091a6aa3f40d1b05cabd77f2db4d82f228bd840a1f16e81602fb3b43`.
Driver: [probe.py](probe.py), SHA256
`38b3d037bffbb9165af89bc75ede2a6dcf2bad694343f881c657b880db52f8b5`.

Actual native Codex 0.154.0 on macOS arm64, Go 1.26.4 host build, released
Mandalore 1.1.0 as the old installer. A disposable native profile, synthetic
signet/binding and separate installation state were used. No credentials were
copied, no model/provider call was made, and no personal registration changed.

Run with four explicit absolute paths, selecting an **absent** fixture root:

```sh
python3 docs/evidence/upgrade-safety/probe.py --root /example/absent-fixture --old /example/released-mandalore --new /example/candidate-mandalore --codex /example/codex
```

Actual assertion-backed output:

```json
{"check": "deferred_with_live_consumers", "old_cache_unchanged": true, "old_hook_working": true, "old_mcp_working": true}
{"check": "acknowledged_activation_and_replay", "new_hook_working": true, "replay_cache_bytes_and_mtimes_unchanged": true, "bank_and_binding_unchanged": true}
{"check": "COMPLETE", "passed": true, "scope": "native installer and hook/MCP components; no model session"}
```

The first phase holds an actual MCP process and invokes the actual old hook
before/after a denied replacement. After closing the held process, an explicit
acknowledged apply installs the new generation. Identical replay preserves cache
bytes and mtimes. The synthetic bank and binding remain byte-identical.

## Plain terminal deferral

[Actual plain-menu recording](menu-defer.txt), captured through macOS `script`
in the designated terminal lab against the same candidate executable. Workstation
paths were replaced by synthetic labels and trailing prompt whitespace normalized;
text and entered choices are retained.
The alternate source path selected identical executable bytes but produced a new
connection plan, forcing the replacement guard instead of verified replay.

Observed: choose connection/update, select Codex, review the plan, approve the
initial apply, receive the separate stopped-session question, accept its default
No by pressing Enter, return to the main menu, exit. No successful native-update
message is shown on this declined path. This is actual plain-terminal interaction,
not a generated screenshot or a claim of TUI/narrow-width accessibility acceptance.

## Boundaries and remaining verification

The [TUI transcript](menu-tui.txt) records actual arrow/Enter interaction in the
same native terminal: default-No deferral returned to the main menu, then a
second reviewed attempt with explicit stopped-session confirmation installed the
selected generation and displayed the verified phase plus restart/trust notice.
Workstation paths and trailing whitespace were normalized and ANSI cursor/color
controls removed for the public text transcript; repeated redraw text is retained.
It is not a replayable raw recording or pixel screenshot. The raw capture remains
local. This verifies the normal-width task flow, not narrow-width or screen-reader
behavior.

The [48-column TUI transcript](menu-narrow.txt) is a separate actual run in a
macOS script PTY configured and checked with `stty size` (24 rows, 48 columns).
The terminal itself remained in the existing lab pane; this is a constrained
PTY rendering test, not a physical display resize. The warning wrapped fully,
both complete handoff choices remained visible, Enter selected default defer,
and the menu returned to Exit. Existing long menu labels/help were ellipsized;
the new safety question and choices were not. Paths/ANSI/trailing whitespace
were normalized as above. No screen-reader or alternate-locale claim is made.

## Complete native-session verification

After the owner completed native device login in the isolated profile, a separate
model-driven run exercised the same executable and Codex version above. No
credentials were copied from another profile. Both sessions used the native
default model, read-only sandbox and no approval prompts. Invocation-local hook
trust bypass was explicitly selected for these inspected synthetic hooks under
the authorized lab workflow; this does not test normal hook-trust approval UX.

1. Reinstalled the released 1.1.0 plugin with all synthetic consumers stopped.
   Started a native session and called live `memory_inspect`: the synthetic
   signet was healthy.
2. Kept that session open and attempted candidate apply from a separate lab
   shell, without acknowledgement. Receipt: `installed: false`,
   `phase: deferred`, with the old registration retained.
3. Prompted that same session again: live `memory_inspect` succeeded and returned
   the same healthy signet. No hook failure appeared in either turn.
4. Exited the old session, confirmed its pane was back at the shell and no
   fixture runtime consumer remained, then applied with `--sessions-stopped`.
   Receipt: `installed: true`, `phase: verified`.
5. Started a fresh native session. Its actual MCP process used the retained
   candidate executable with SHA256 listed above; its loaded skill path used
   candidate generation
   `711a39b63e28a5657194b8a0768bc803a0b06ebac258e1079c8bc2a0e9ceb24a`.
   Live `memory_inspect` again returned the same healthy synthetic signet.
   Exited the test session afterward.

This completes engineering model-session verification, not independent product
acceptance. All prompts prohibited saves, journals, synchronization and repair;
no such calls were observed. Personal registration and signet were not targeted.
The expected trust-bypass startup warning was visible; no live authentication
material or private transcripts are included here. Partial-registration failure
and recovery coverage remains simulated unit evidence, not an induced native
failure. Other native platforms and screen readers remain unverified.

Pinned full host CI passed at the product source above, as did hosted CI run
35353790603. Plugin and Armorer skill validators passed. Cross-builds are not
native Linux acceptance. The initial planning-format failure and its corrected
six-file planning pack are retained in PR #114; the initial unit red phase
demonstrated both unguarded replacement and unnecessary replay installation.
