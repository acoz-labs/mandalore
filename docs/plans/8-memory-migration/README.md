# Solution Design: explicit memory-only migration

- **Status:** Draft
- **Issue:** #8
- **Repository basis:** 3cebe0b51a1f28aef91e3df3d793826243265123
- **Execution envelope:** implementation

## Decision

Convert a supported My Friday memory-only bank into a new, separately published
migration bundle. Preserve the source, original portable data and semantic
history. Do not convert assistant repositories, activate a native writer or
copy a Git/native authentication configuration.

## Decision Spotlight

- Support the pinned predecessor's bank.json format 1, not arbitrary legacy
  Markdown or agent.json. Other historical sources belong to foundlings (#12).
- Preserve opaque bank/record/revision/source/device IDs and original dates,
  authorship, evidence and supersession. Only bank-wide scope kind changes from
  assistant to signet; the bank ID need not be renamed to a signet prefix.
- Output is an explicit new bundle containing signet/, original/ and a local
  conversion receipt. The original snapshot preserves portable source bytes;
  operational provenance remains historical evidence, not new current memory.
- No in-place rewrite, Git-history/filter/config copying, automatic binding,
  native plugin replacement, synchronization or source deletion.
- Preflight reads bounded local state and reports observed writer/integration
  risks without claiming every session or other machine has been stopped.
- Applying requires explicit quiescence acknowledgement, revalidates the plan,
  validates the converted bank and publishes without replacing an existing path.

## Needs Attention

Live migration and fresh-session activation remain separate explicit choices.
Retaining the source and a snapshot is not proof of an off-device backup.
Independent candidate acceptance and distribution remain #10/#11.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)
