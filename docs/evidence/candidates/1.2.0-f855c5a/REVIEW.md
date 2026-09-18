# Mandalore 1.2.0 candidate review packet

Engineering evidence for issues #77, #112, #80, #81 and preparation #124.
This is not a human acceptance verdict, published release or live migration.

## Exact artifact

- Source: `f855c5a7c05ffc9fbdb87bdb564d79c542bb26f6`.
- Manifest SHA256: `38266a4a3656f06ef62331bbcbb5f96c721fa1eb0315b87d5a8600e17ace3804`.
- [Build35381453879](https://github.com/acoz-labs/mandalore/actions/runs/35381453879),
  attempt1; artifact10563155886, expires2026-12-17T18:39:29Z.
- Archive SHA256: `ab2fb1a6edb5ba24c340f06bcd214f9de0f28e144d919da6fdd96abecf5abb72`.
- [Nomination35381902329](https://github.com/acoz-labs/mandalore/actions/runs/35381902329)
  succeeded; matching workflow-authored receipts were verified on all five issues.
- [Transport](transport.json), [byte verification](verification.json) and
  [manifest](manifest.json) retain machine-readable identity and provenance.

All eight archive members were independently verified before extraction/execution.
No product rebuild supplies the native CLI results below. The macOS arm64 payload
SHA256 is `cddf47c21315a3db59d6ac6c7470737a2ec522f910a1231c329de133e2713cdf`.
The build itself ran full pinned CI. The preparation PR passed exact-head hosted
checks and engineering self-review; merged temporary-plan removal was verified.

## Outcome map

| Issue | Actual candidate evidence | Boundaries |
| --- | --- | --- |
| #77 | Six native renderer scenarios and literal-shell-meaning checks; [matrix](#recovery-rendering). | Renderer is an exact-source test binary, not the retained product executable; no simulated error is presented as real provider behavior. |
| #112 | [Native update probe](update-probe.json): live old hook/MCP remain working after deferral; stopped/acknowledged update and replay pass, preserving bank/binding. Fresh candidate native model observations are tracked separately. | Component probe is not proof of universal hot reload or every model-session update pattern. |
| #80 | [Retained executable export](export.json): explicit scoped preview/apply, omission/history/journal controls, private output, no overwrite, source preservation and refusal checks. | Synthetic report, not secure erasure, a restorable bank or safe-publication certification. |
| #81 | [Retained executable withdrawal](withdrawal.json): explicit activation/recovery, old-reader refusal, withheld recall/context/export, stale-head refusal, retention preview, independent-clone opt-in/convergence and correction/restore semantics. [Native observations](native.md). | Offline/previously-read copies persist; native model outcomes cover specific scenarios only. |
| #124 | [Installation transition](installation.json): old updater safely refuses expanded formats; verified new executable installs candidate while preserving old runtime, unrelated files and format1 bank/binding. | Not a public bootstrap download of an unpublished release. |

Source CI additionally exercises upgrade phase faults, tamper/downgrade/source
drift, causal permutations, output/source bounds, cancellation and partial receipts,
export redaction dependencies, independent journals/origins and read-only guards.
See the original issue contracts and [withdrawal reconciliation](../../withdrawal/reconciliation.md);
the compact native scenarios do not replace those full regression suites.

## Recovery rendering

On macOS arm64 in the designated real terminal, the pinned-source renderer test
binary used synthetic API receipts. SHA256:
`194780259e763f395309e498a96f2c8b35fcb756f3a5b8297b908f2c409bd686`.
Each capture records the constrained PTY size. Normal-width TUI used its ordinary
theme; narrow/plain/default/Back cases requested no color.

| Scenario | Result | Actual text capture |
| --- | --- | --- |
| Plain32 | PASS; intact command | [Capture](recovery-plain32.txt) |
| Plain80 | PASS; same command | [Capture](recovery-plain80.txt) |
| TUI80 | PASS; Down/Enter | [Capture](recovery-tui80.txt) |
| TUI32 | PASS; Down/Enter | [Capture](recovery-tui32.txt) |
| Default-No32 | PASS; no apply | [Capture](recovery-default32.txt) |
| Back32 | PASS; Escape/no apply | [Capture](recovery-back32.txt) |

All four command lines are identical155-byte logical lines, including apostrophe
quoting and double spaces. [Results](recovery-results.json) pin raw/text hashes.
[Inert-shell checks](recovery-shell-meaning.txt) cover apostrophes, double spaces,
Unicode and metacharacters without calling real recovery. Public text captures
strip terminal escapes/carriage returns/trailing spaces; private raw captures are
retained. They are not pixel screenshots or proof of every clipboard, terminal,
locale or screen reader. Existing narrow choice/help ellipses remain visible.

## Acceptance boundary

Review the original criteria of all five issues, the [candidate guide](../../../releases/1.2.0-candidate.md),
exact identities above, and the recorded limitations. All product tests used
synthetic banks/profiles and local bare remotes; no personal installation or
signet-format upgrade occurred. Authentication was used in place, never copied.

Format1 remains supported/default; a runtime update does not activate format2.
Format2 requires explicit per-clone consent and compatible readers; no downgrade,
TTL, destructive retention, Git-history rewriting or global revocation is promised.
Native evidence is macOS arm64; other targets are cross-built, not native-tested.

Actual human acceptance, any necessary exact-candidate eligibility authorization,
publication authority, public-download verification and issue delivery closeout
remain pending. Historical candidate-specific owner exceptions do not cover these
bytes. Do not label this engineering packet an independent product approval.
