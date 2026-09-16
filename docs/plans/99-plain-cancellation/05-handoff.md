# Handoff

Implement tests before the read-wait correction. Review the race where a newline
and cancellation become ready together, late worker completion, bounded input,
and no accidental apply. Keep code confined to plain-menu input and its tests.

Promote cancellation semantics and reader lifetime into `docs/development.md`
and concise native evidence. Record red/green results and exact-head self-review
in the implementation PR; remove this temporary plan before leaving draft.
CI/merge is not candidate or product acceptance. Keep #99 open for release.

Return to design if the correction requires global input modes, process-wide
signal changes, a new dependency or changes to existing authorization semantics.
