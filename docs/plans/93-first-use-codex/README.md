# Solution Design: first-use Codex profile

- Status: Ready for exact-head engineering self-review under ADR 0003
- Issue: #93, discovered during #88 human acceptance
- Planning PR: #94
- Repository basis: bb16a57eb16dcd5e3da1386d6eb72c64169be6b5
- Execution envelope: implementation

## Decision

On explicit apply, prepare the selected native home before native inventory.
Preview stays read-only. Reuse the existing canonical-directory guard and 0700
creation policy; preserve existing profile contents and permissions. Give native
inventory failures bounded operation context without exposing child stderr.

## Needs Attention

The previous retained candidate failed first-use human acceptance. Preserve that
evidence. This fix needs real absent-profile validation and a new reviewed,
retained candidate before the affected human walkthrough resumes. No release,
personal activation, credential copying or acceptance verdict is authorized.

## Decision Spotlight

An empty profile can remain after failure, just like retained installation state.
Do not delete it automatically or broaden repair into authentication setup.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Handoff](05-handoff.md)
