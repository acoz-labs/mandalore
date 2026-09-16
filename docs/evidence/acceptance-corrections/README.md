# Acceptance-discovered corrections: #99 and #100

Contributor verification, September 16, 2026. This is engineering self-review
under ADR0003, not independent review, human acceptance or a release nomination.
The original retained-candidate failures remain preserved at
[d4e80de](https://github.com/acoz-labs/mandalore/tree/d4e80dedf68e5fd430c7eec5eee7eba413773bb8/docs/evidence/candidates/1.1.0-b894586/engineering).
No personal installation, authentication or bank was changed.

## #99: blocked plain-input cancellation

PR #103, exact source `0c12991713bf15bf4b332b0ed233e9fe1e0c9491`.
Built with the pinned Go toolchain using `go build -trimpath`; the executable hash
is in [the receipt](cancellation/evidence.json). This is a development executable,
not the retained 1.1 candidate. Native environment: macOS arm64, real Herdr PTYs,
80-column plain/NO_COLOR and color; narrow 32-column color TUI.

The failing-first regression deliberately keeps the underlying reader blocked
after context cancellation. Old code failed seven blocked cases. Corrected code
passes choice/text/confirmation with empty and partial input, cancelled setup
approval without effects, and 100 ready-answer/cancellation races. Existing
bounded-input, incomplete-answer, EOF and default-No tests remain in the suite.
Local full CI and hosted CI pass; the local run uses the documented host fallback.

Nine native scenarios are represented by real `script` recordings and capture
receipts in [cancellation](cancellation): idle and partial choice, partial text,
partial setup approval, default-No, EOF, TUI Ctrl+C, partial release approval and
TUI Escape/exit. Plain interruption exits 130 without Enter or EOF; TUI Ctrl+C
retains its existing successful-cancel exit 0. Terminal state is restored.
Setup paths stay absent; release confirmation leaves the complete synthetic
installation inventory unchanged. No release apply was authorized or executed.

The controller initially expected TUI Ctrl+C to return 130; inspection of the
existing behavior corrected that expectation without rerunning the observation.
It also expected Escape at the top-level menu to exit; that menu correctly
redisplayed. The timed-out `tui-back-color` capture is retained, and a distinct
corrected journey sends Escape then Enter on the default Exit choice. These are
controller expectation failures, not discarded product failures.

Replay on macOS with `script -p <recording>` in a disposable terminal. Recordings
contain synthetic temporary paths. This checks keyboard behavior and terminal
restoration, not an approved pixel baseline, screen-reader compatibility or other
native operating systems. Cross-builds are not native acceptance.

## #100: no new content versus an explicit restriction

PR #104, exact guidance source `8a7106c14eb5480cbcd562dad5d151e776a9f144`.
Both native skills have the same bounded clarification. Skills are validated;
full local CI and hosted CI pass. Actual fresh native sessions use Codex 0.154.0
and Pi 0.85.1, model gpt-6-astra, high reasoning. See [codex](codex) and [pi](pi).

The changed skills are loaded separately from source while the unchanged runtime
and Pi transport remain from retained `b894586` (runtime SHA256
`25a980f7c59cfd7b1fb511a4caa0c6fe6ee738006ffbc96a31440e4741dbc93a`).
Receipts bind both identities. This verifies development guidance, not a new
packaged candidate. Actual read calls verify the new skill for direct cues; an
irrelevant quoted cue need not activate or read the skill.

Each fresh synthetic bank begins with committed undelivered knowledge and a
local bare remote. The offline case renames that remote out of reach. No new
facts or useful journal content are supplied. Check actual tools, semantic files,
full negative-case inventories and local/remote heads, not prose alone.

- Codex: available origin, unavailable origin, missing connection, no-sync,
  no-save, read-only, no-journal-only and quoted cue.
- Pi: available origin, unavailable origin, no-sync and quoted cue.
- Permitted direct cues make exactly one bounded sync, with no new records or
  journals. Available origins receive the existing head; unavailable origins
  remain pending without a retry. No-journal alone does not prohibit delivery.
- Explicit prohibitions and quotations cause no delivery or memory changes.
  The missing connection is explained without repair or fallback.

Two Codex analysis checks were initially too restrictive: they did not recognize
the truthful wording “isn't connected” (with typographic apostrophe), and expected
an irrelevant quoted cue to read the skill. Only the analysis was corrected; the
original native conversations were not repeated. The first Pi fixture loaded the
old package skill because the package-directory extension argument also exposed
its skills. It is retained privately and excluded from new-guidance evidence.
A fresh fixture loads only the retained extension entry point plus the explicit
new skill, whose actual read path is checked. No installed package was edited.

These are observed model behaviors, not a statistical reliability claim or a
deterministic hook. Original passing and failing observations are both retained.
The narrower Pi matrix checks shared-guidance alignment; Codex carries the full
restriction matrix for the reported failure.

## Review and handoff

Self-review checked cancellation ordering, one outstanding read, bounded channel
completion, unchanged parsing/approval semantics and blocked-worker cleanup on
CLI exit. It checked skill activation, explicit restrictions, no filler, truthful
delivery and the absence of automatic repair or broader authority.

Plan promotion: #101 context/decision/design/verification/handoff become the
development cancellation contract, regression tests and this evidence; #102
becomes the shared skills, durable consolidation contract and this evidence.
Both issue-local plans are removed in their corresponding implementation PRs.
No product-scope drift, new dependency, new API or migration is introduced.

The original candidate remains changes-required. Corrected code needs a new
immutable build, nomination and actual human verdict. The prior same-account
owner exception is pinned to the old candidate and does not cover corrected
source. No acceptance, publication or personal activation is claimed here.
