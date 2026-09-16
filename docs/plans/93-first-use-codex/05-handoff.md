# Handoff

Smallest complete outcome: first-use Codex apply works for a missing selected home
without changing preview or existing profiles. Add failing tests, implement the
directory prerequisite and bounded diagnostic, run targeted/full/native checks.

Use a planning-only PR and distinct exact-head self-review under ADR 0003, then a
draft implementation PR. Reconcile the implementation head, record any drift,
promote shipped behavior to the existing installation documentation and retain
sanitized verification evidence. Delete this temporary pack before ready/merge.
Required checks and exact-head implementation self-review still apply.

Do not add automatic login, credentials, personal migration, profile resets,
unconditional rollback, a new lifecycle hook or a release-policy exception here.
Reopen design for a new data/trust boundary or incompatible native requirements.
