# Codex native engineering evidence

These are synthetic integration results, not independent product acceptance or
an accepted release candidate. Issue #6 owns integration; #10/#11 own immutable
candidate acceptance and publication. No live memory was migrated.

## Tested identities

- Native CLI: Codex 0.153.4, macOS arm64, inherited native account/resources.
- Native model displayed: gpt-6-astra, high reasoning; this is an observation,
  not a Mandalore model requirement or new model configuration.
- Runtime source: `ed8651b67365d05b3cbf7f431802e690f568af8c`.
- Tested binary SHA256:
  `3074f295d53a581cbecd8b5b0c3624391f95efa187e7207159e554fec56baac8`.
- Plugin assets from `987f133c1a6a53fb43422c33388d32d70989a107`, unchanged at
  the runtime source above. Native-installed cache compared byte-for-byte with
  source. Namespace: `mandalore@mandalore`; skill: `this-is-the-way`.
- Hosted CI for runtime source: run `34791745700`, passed. Prior package validators
  passed with isolated PyYAML 6.0.2; this is not a runtime dependency.

The native session used explicit absolute runtime/binding environment selection,
unrestricted execution and no per-command approvals. The two authored hooks were
reviewed and trusted individually in the native UI; the existing Herdr hook was
left unchanged. No blanket hook-trust bypass or separate copied native home was
used. Native update prompts were skipped to retain the inspected CLI version.

## Observed defect and correction

The first forward test on `987f133` discovered the skill and all eleven tools,
but did not save ordinary confirmed facts. A read-only diagnostic attributed
this to the orientation sentence beginning “Current read-only/no-save scope”,
which the model interpreted as declaring the session read-only. No write error
had occurred. Structural and compiled-process tests had passed; those were
insufficient evidence for this instruction behavior.

Commit `ed8651b` makes the condition explicit and distinguishes read-only hooks
from the session's permitted memory writes. The same ordinary prompt was rerun
in a fresh session, without the earlier misleading context, and saved correctly.
Do not generalize this fix into ignoring actual user no-save instructions.

## Scenario observations

| Scenario | Observed result |
| --- | --- |
| Ordinary confirmed project name and response preference, without a memory command or special phrase | Two saves: project-scoped entity and signet-wide preference; local receipt and short-budget sync, with no remote-delivery claim |
| Confirmed project rename | Superseded the existing entity revision, preserving record and stable project scope; appended a semantic journal |
| Direct consolidation cue plus a project-specific draft-format decision | Saved a scoped decision and semantic journal; did not generalize the decision into a signet-wide rule |
| New session in a different project directory | Discovered stored scope, recalled current/old names, draft format and preference via MCP, including history |
| Quoted cue in a translation task | Returned a translation, without consolidation or a memory mutation |
| Explicit native `/compact`, then read-only recall | Native UI reported context compaction; subsequent scoped MCP recall returned the current name and format |
| Interrupted bounded sleep, then read-only recall | Native turn interrupted; subsequent MCP recall succeeded without retrying or resuming the command |
| Native resume of the same session, then an unsettled naming brainstorm | Discussed the option without saving it as a decision; subsequent MCP recall confirmed the existing decided name |
| Fresh session bound to a second, empty signet in the same project directory | MCP returned the second signet ID and empty scope/recall results; the agent did not invent or reveal the first signet's facts |
| Missing binding in a fresh native session | Native MCP startup failed and the hook emitted a sanitized nonblocking warning; an unrelated arithmetic task completed without repair or writes |
| Original binding restored before a fresh session | MCP started successfully and recalled the original project name and draft format without a repair action |
| Read-only file comparison | Before/after fingerprints of every signet file, including Git metadata and ignored sync receipts, matched across cold recall, quoted cue, compaction, interruption recovery and normal exit |

CLI inspection after the native saves confirmed healthy structure, current
project records and both journal entries with device/actor/Codex attribution.
The test signet had standalone local Git history and no configured remote:
`local-only` is not tested remote delivery. File fingerprints prove unchanged
file paths/content for this run, not universal filesystem transactionality. A
second comparison after resume, brainstorming and recall also matched.
Both signets' fingerprints also matched after the isolation, missing-binding
and restored-binding recovery scenarios.

The missing-binding session displayed the MCP failure and hook warning twice.
Native plugin enumeration and configuration inspection showed one enabled
Mandalore plugin and one authored hook per event. This rules out an obvious
duplicate registration, not repeated native execution or display. The cause is
unproven; do not count displayed messages as independent invocations. Hooks are
read-only, so this did not duplicate a memory mutation.

## Native interruption limitation

After Escape, Codex briefly still displayed the bounded sleep in a background
terminal. Turn interruption is therefore not evidence that all asynchronous
shell children were killed. Mandalore does not own the native process supervisor
or undo arbitrary side effects. The interrupted command made no file changes,
was not retried, and did not prevent the subsequent memory read. A hook or turn
completion must not be used as a transactional commit/rollback guarantee.

## Still required

Automatic mid-turn compaction,
other Codex versions/surfaces and operating systems are not established by the
manual compaction scenario. Physical-machine continuity, foundling workflow,
migration and exact immutable-candidate acceptance remain downstream work.

Raw native sessions and workstation paths are deliberately not published. The
private disposable fixture is retained for diagnosis; this document records
sanitized observations and exact source/artifact identities, not raw transcripts.
