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

This is **not** a model-driven agent-session test or independent product
acceptance. The isolated native profile reports not logged in; no credential
copying or live-profile activation was used to bypass that boundary. Complete
session and remaining rendered-menu verification remain pending. The new hook is verified;
fresh-session model/MCP attachment is not inferred from that component result.

Pinned full host CI passed at the product source above, as did hosted CI run
35353790603. Plugin and Armorer skill validators passed. Cross-builds are not
native Linux acceptance. The initial planning-format failure and its corrected
six-file planning pack are retained in PR #114; the initial unit red phase
demonstrated both unguarded replacement and unnecessary replay installation.
