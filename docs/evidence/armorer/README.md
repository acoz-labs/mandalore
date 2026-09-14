# The Armorer: contributor verification

This is engineering self-review for #45/#47, not product acceptance or a release.
Implementation head: `9584be916efe2e97d2070c09ecd934f52996a0d6`.
Native development executable SHA-256:
`1c5ed26425a35ceacdd201f0722cf1d66309cd2f92e8ae1e89a87aaeb4a3f918`.
Embedded package SHA-256:
`92e065e8e3330b86e9033c7d56eaf344d1eefb7b05820d051611c9ba32cead1d`.
This evidence-only branch preserves the reviewed implementation head separately
from generated recordings. It does not replace immutable-candidate verification.

Subsequent owner-driven checks of the replacement immutable artifact are recorded
separately in [retained-candidate observations](retained-candidate.md).

## Automated checks

Alias parity, embedded skill resources and exact machine-local context each
failed before implementation, then passed. Context tests include quoted/spaced
paths, absence from the public package, ownership hashing and refusal after edits.
The full Go 1.26.4 race/vet/four-target-build suite passed on the initial
implementation; hosted CI also passed on the final implementation head
([run](https://github.com/acoz-labs/mandalore/actions/runs/34885694228)).
The existing MCP test still exposes exactly 16 memory tools, with no installation
schemas. Cross-builds are not native platform acceptance. Docker was unavailable;
the documented pinned host fallback was used. Skill/plugin validators passed
using an isolated PyYAML dependency, not a global Python installation change.

## Native behavioral checks

Environment: actual Codex CLI 0.153.4 with GPT-6 Astra on macOS ARM64, existing
isolated synthetic profile, unrelated project cwd, no Mandalore runtime/binding
exports and no required global Mandalore PATH entry. No credentials were copied.
Installed through the managed plan/apply pipeline and its deterministic version
suffix; the public source marketplace was not hand-edited.

At initial implementation `2e0e6a43971042e10efa6a6199b70c7022659c21`:

- Natural read-only health request selected The Armorer, read generated context,
  called typed diagnostics and live read-only MCP, and distinguished structural
  checks, local pending history and untested authentication/remote boundaries.
- An authorized request created/bound a separate synthetic signet and initialized
  local Git using the typed CLI. No menu interaction or additional owner prompt
  was required. The new bank was healthy, clean and had no remote; the existing
  signet and connection were preserved.
- Efficiency findings: it printed the full operation catalog before filtering,
  and performed an excessive retained-state hash inventory. Neither is desirable
  workflow. The skill was narrowed to filter result.operations before output and
  verify only relevant resources, avoiding native profiles/credentials/archives.

At final implementation `9584be9`, a fresh native session performed a natural
health request plus a read-only connection preview for the separate test binding:

- Automatically selected the administrative skill and read its exact local
  context. Both runtime/binding environment overrides were absent.
- Filtered the catalog to the requested operations before output; did not perform
  the broad inventory scan. It also loaded the memory skill for live read checks;
  no isolated token-cost or universal optimal-routing claim is made.
- Current connection reported 12 structural passes; CLI/MCP identified the same
  current bank. Target binding structure passed separately.
- Preview used the selected runtime/native executable and a separate profile/state,
  with the target signet identity explicitly reported. Neither destination existed
  afterward. No apply, save, journal or synchronization was called.
- Full non-Git source-file hash of the existing bank matched before/after both
  development sessions and installation refreshes. The old candidate artifacts
  remain retained; the test profile now uses the development generation.

Native transcripts contain local profile paths and inherited resources and are
not published. These are bounded contributor-operated cases, not owner verdicts,
all-harness support or exhaustive autonomous administration verification.

## Rendered menu evidence

Classification: in-pattern-visual-change. Product design preserves separate
inspection and repair, default-No mutation review, explicit status labels and
the existing keyboard/plain-mode model. Contributor visual self-review only.

| Recording | Environment and actions | Observed result |
| --- | --- | --- |
| [Normal](menu-normal.recording) | Actual terminal, 96 columns by 28 rows, English, NO_COLOR; gg/j/Enter selects inspection of an absent synthetic profile | The Armorer labels readable; structured failure and five not-tested boundaries remain distinct; no implicit setup |
| [Narrow](menu-narrow.recording) | Same executable, 40 columns by 28 rows; select repair, Escape its input, select inspection, exit | Repair cancellation returned without writes; report wrapped at the width; long menu labels/help elided according to existing behavior |

Recordings were captured by macOS script from the real executable, not fabricated
expected output. They include only synthetic administrative UI, no login/model
transcript. The missing-profile error is intentional; it is not a runtime failure
of a configured connection. Missing native/state directories remained absent.
Normal and narrow diagnostic flows returned an operation-failure exit status as
expected. Keyboard/no-color behavior was observed; screen readers, alternate
locales/fonts, other operating systems and browser/mobile surfaces were not tested
in this change. There is no approved pixel baseline or automatic visual verdict.

## Handoff and limits

Preserve compatibility for both connection armorer and connection doctor; typed
connection_doctor remains stable. Native configured-profile CLI reports from both
spellings were compared for byte equivalence separately. No automatic self-healing,
new MCP administration surface or credential provider is introduced.

A new retained candidate must include #45 and be nominated and reviewed on its
own identity. Prior owner experience with the old artifact is not acceptance of
this development executable. Published download/update and formal release remain
separate, explicitly authorized work.
