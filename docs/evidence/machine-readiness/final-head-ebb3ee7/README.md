# Final implementation-head readiness evidence

Engineering verdict: **pass** for the scoped #85 implementation matrix. This is
contributor self-review under ADR 0003, not independent product acceptance or
release authorization. PR #87 implementation remains pinned to
`ebb3ee77a867b489a786cc91755353feec1e2a9f`; this separate evidence branch changes
no implementation files. The reconciliation at that head is the requirements,
refinement and documentation-promotion map.

## Identity and environment

- Source-stamped development executable: `0.0.0-dev`, source commit above.
- Executable SHA-256: `c3388235e1b8c2c065315d09e2b329dc56bd0d4fbdbb6c1cba903cf5224491b5`;
  17,336,626 bytes. Not the published v1.0.0 or a nominated candidate.
- Embedded Codex package SHA-256:
  `d8000774e493622682d2f65cb18395427c1119f532bcf695282b96b2686d1835`.
- Actual macOS arm64 Herdr test terminal, outer viewport 310×67. Inner BSD
  `script` PTYs are 80×32 except the Pi journey at 32×32. English interface,
  xterm-256color; plain/no-color and interactive color variants below.
- Native versions confirmed during fixture preparation: Codex 0.154.0, Pi
  0.85.1 with pinned Node 24.1.0. Synthetic profiles, signet and owned connection
  roots only. No credentials copied, provider/model request, personal bank or
  synchronization. These native checks are not new login acceptance evidence.

## Actual rendered matrix

| Recording | Journey and observation |
| --- | --- |
| [Configured Codex](final-codex.recording) | 80 columns, color/plain. Select owned root, inspect static report and prompt, explicitly approve disclosed native checks. Binding/runtime verified-static; development evidence historical/unknown. Native structural checks pass, five live boundaries untested. Snapshot unchanged; Back/Exit 0. |
| [Configured Pi](final-pi.recording) | 32 columns, NO_COLOR/plain. Select Pi/owned root; status words remain intact. Static report then explicitly approved native checks pass with four live boundaries untested; Back/Exit 0. |
| [Stale native identity](final-stale.recording) | 80 columns, color/plain. Disposable Codex wrapper changed after setup. Partial/inconsistent static findings; approved native check fails identity verification. No retry, repair or replacement of snapshot. Exit 1; original wrapper restored after the test. |
| [Invalid and malformed binding](final-malformed.recording) | 80 columns, NO_COLOR/plain. Relative path rejected without losing the journey; corrected to malformed synthetic binding. Partial report and sanitized prompt retain other observations, no raw canary. Back to binding input, EOF exits 0 without changes. |
| [Unsupported retained runtime](final-unsupported.recording) | 80 columns, color/plain. Synthetic owned metadata selects published Linux amd64 runtime on macOS. Byte consistency remains verified-static while support is unsupported and evidence historical. Guidance reviews the selection, not automatic upgrading. No execution of the Linux artifact; Exit 0. |
| [Missing setup](final-missing.recording) | 80 columns, color/interactive. Vim navigation to Armorer; default Codex/missing paths, details confirming source and binary identities, explicit prompt view, arrow navigation to native disclosure and default No. Vim Back to Exit 0. No missing paths created. |
| [Cancel](final-cancel.recording) | 80 columns, NO_COLOR/interactive. Vim navigation, Escape from harness choice, reenter, arrow-select Pi, Ctrl+C before assessment. Exit 0 under existing menu convention; no successful assessment or prompt; shell terminal restored. |

The incompatible runtime is the published Linux amd64 artifact with SHA-256
`74817349099318fc557b4db17b0bf4a69413caf833fe218032320c40d333a719`.
Its fixture was prepared separately using the CLI plan and staged metadata, not
installed as a real native registration. No assessment download occurred.

## Effects, rendering and limits

Complete fixture inventories captured paths, bytes, modes, mtimes and empty
directories. All static phases were unchanged: 141 entries before the final
Codex run and 136 before each subsequent configured/adverse-state run. Separately
approved Codex native inspection cleaned temporary helpers within its profile's
`tmp/arg0` subtree; the later stale check changed only that directory metadata.
Pi native inspection changed only its profile directory metadata. Signet,
binding, owned runtime/package and registration bytes remained unchanged.
Malformed/unsupported journeys remained unchanged end-to-end. The PATH-local
Node execution trap remained absent. This is fixture-scoped observation, not a
global network or filesystem sandbox.

Every recording confirms terminal modes/control characters restored, excluding
only Darwin's transient PENDIN bit (0x20000000), which input handling clears.
The runner restores the original terminal dimensions afterward. Its diagnostic
footer may exceed the 32-column width; that footer is not application UI.
Color and no-color labels remain understandable; keyboard, Back, safe default,
plain input, invalid input recovery, EOF and Ctrl+C paths were exercised. No
approved pixel baseline exists. Screen readers, alternate fonts/locales and
non-host native platforms remain unverified; this CLI has no mobile/browser UI.

Actual live output was inspected during these runs. Rendered hierarchy keeps
selection, independent support/setup/evidence, next step and untested boundaries
distinct. The narrow status-word finding from the earlier checkpoint remains
fixed. The intentional native failure is disclosed without a success verdict or
automatic repair. These observations satisfy the scoped engineering design
review; they do not approve their own candidate for release.

## Validation and exact-head self-review

`mise exec -- bin/ci` passed on clean implementation head `ebb3ee7`: all Go race
suites, vet, public-content validation, 22 Pi adapter tests and four target
builds. Docker was unavailable; documented pinned host fallback used Go 1.26.4
and Node 24.1.0. [Hosted CI passed on the same exact head](https://github.com/acoz-labs/mandalore/actions/runs/35055487788).
Both Armorer skill validations and Codex plugin validation passed on the source
changes; subsequent commits did not change those packaged skill files.

Self-review checked the scoped diff from approved planning merge
`3332938212cc73ee05d03d57e5179d13705a0447`, fixed read boundaries and limits,
identity/evidence matching, pure shared parser refactors, administration-only
exposure, explicit native handoff and sanitized prompt, menu failure/cancel
behavior, test scope and permanent documentation. No unresolved in-scope
engineering finding. Process/path replacement is tested through a private seam,
not claimed as a native loader attestation. Configured network traps observe
their endpoints, not every possible socket. Cross-builds are not native testing.

All ten issue criteria map to implementation/tests in the final reconciliation.
The six-file temporary #85 plan and #15 discovery are removed; reviewed history
and promoted setup/interface/architecture/runbook/development/skill guidance
remain linked. No new product authority or repair behavior was introduced.

Keep #85 open after merge for nominated immutable-candidate acceptance/release.
Repeat the hands-on candidate matrix with fresh evidence; preserve v1.0.0 and
personal installations. Pi acceptance remains separate and precedes Claude work.

## Retained recording hashes

| File | SHA-256 |
| --- | --- |
| final-cancel.recording | `397c5844025b1422afc9ac8a71439eb99372e48266a7d7f06301fce93efb4be8` |
| final-codex.recording | `f30238182b493e266feb8c416fdce433170db3def88802443b598800c9593548` |
| final-malformed.recording | `74ccb453dc9bad3e3d9b121d22e0a2877341bbb24bfb22dd78851fdbf1c7fd57` |
| final-missing.recording | `0a7e80303744bfae12ff3865fbf001a2e1fac61e5c2ed699fc80db5caa6bb329` |
| final-pi.recording | `89bbb0855793f68c5f408d78e797da0c14faa34d820caf60ffd4617d7a0e582f` |
| final-stale.recording | `7fed19457e7b233030444276fe0ea0344931520327848b194ba8bc824348dc5c` |
| final-unsupported.recording | `65e00c1ba335b31bdfd5a75c9941c736fbaec00ac1c31b458207e621646d5894` |

Replay with macOS `script -p <file>`; BSD recording portability is not promised.
Only recordings/this manifest are published, not machine-specific fixture
launchers or private paths. Privacy scan found no personal paths, account names,
tokens or malformed-input canary disclosure.
