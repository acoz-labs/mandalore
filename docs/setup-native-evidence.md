# Setup native engineering evidence

This records synthetic #7 engineering, not independent acceptance or a released
candidate. The menu/rendered journeys and exact-candidate acceptance remain.

## Identities and observations

The managed connection test used source
`b7834ffc1d7bfe9553d16792639290551e83266e`, binary SHA256
`9eb885fd0d54a4703b23ff6fdf0db41fb1fa40570fc97217e936e915002ee109`,
embedded package SHA256
`b77e0270632825dada12614c2d871dded4c9c6dc02cdfd6846fb8c38b9f8a30f`,
and native Codex 0.153.4 on macOS arm64. Hosted CI run `34794964256` passed.
The generated plugin kept base version `0.0.0-dev` and a content-addressed
`+codex.` suffix. The native package validator passed on the generated source.

- A real apply against the known unmanaged development marketplace refused
  replacement before activation. That registration was then explicitly removed
  for the controlled test; no unrelated marketplace was selected for removal.
- Native apply reported the selected source/version enabled, verified actual
  cache contents and completed in approximately two seconds in this run.
- Doctor reported the inherited test environment's runtime/binding overrides as
  failures. After clearing those process variables, structural health passed;
  login, hook trust, live MCP, remote freshness and active context remained
  explicitly `not-tested` by doctor.
- The native authentication file's before/after fingerprint matched. This is
  not a claim that every native file remains unchanged: registration and native
  session state legitimately change.
- The exact two generated Mandalore hooks were reviewed and trusted in the
  native UI. The existing unrelated SessionStart hook remained enabled.
- A fresh native session, using normal native authentication and explicitly
  authorized YOLO execution, had no Mandalore runtime/binding environment values.
  Ordinary confirmed facts prompted MCP saves into the selected synthetic signet
  without a magic phrase. Short-budget synchronization truthfully reported local
  history only; no remote was configured in this fixture.
- The controller intentionally removed one generated source hook file. Doctor
  identified the incomplete bundle; repair preview left every signet file
  unchanged. Applying that plan produced a fresh verified generation, retained
  the old receipt and returned structural health to passing. Signet fingerprints
  including Git metadata and ignored receipts still matched after repair.
- A fresh session after repair started without another hook-review prompt on
  this native version and recalled the project and preference through MCP with
  explicit no-write instructions. This does not waive native trust review for
  other package changes or versions. The signet fingerprint also matched after
  that session exited.

Review found that adding installation error receipts to the common CLI envelope
had also inflated every memory MCP output schema. A failing regression exposed
the installation-only fields; the corrected memory-specific error schema keeps
those fields out while preserving CLI diagnostics. In the synthetic SDK test,
the checkpoint output schema decreased from 4342 to 1714 encoded bytes. This is
a schema-size measurement, not a claim about exact model-token savings.

## Native cache lifecycle

Actual native marketplace removal deleted the old development plugin cache.
The source remained available for reinstall. Retention guarantees must refer to
managed source/runtime copies, not native-owned cache directories. The installer
checks edited cache files before replacement; native cleanup must not be used to
discard user edits. The automated partial-retry scenario additionally exercises
a native implementation that clears its cache when unregistering.

## Remaining evidence

Runtime update and the interactive menu matrix remain in
progress. This document will be updated from observations,
not intended outcomes. Physical-machine and immutable-candidate acceptance
remain #10/#11. Raw sessions, user identity and workstation paths are not public
evidence artifacts.
