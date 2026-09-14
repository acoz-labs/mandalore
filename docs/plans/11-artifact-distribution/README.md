# Solution Design: reproducible artifact distribution

- **Status:** Draft
- **Issue:** #11
- **Planning PR:** Pending
- **Repository basis:** 680e352898c8bcaf8b14c16b721ea51441296c0b
- **Execution envelope:** implementation

## Decision

Build a versioned CLI/plugin candidate once, retain its exact manifest and asset
digests, test it, then promote those same bytes through the existing independent
acceptance gate. Add an inspectable release install/update journey without
changing any signet or native connection implicitly.

The contributor product-design contract precedes this technical plan:
https://github.com/acoz-labs/mandalore/issues/11#issuecomment-5659960691.
Engineering self-review follows `docs/decisions/0001-mvp-self-review.md` for this
dependency of the #1–#10 effort; it is not independent candidate acceptance.

## Needs Attention

No implementation decision requires additional product authority. Publication
still requires explicit authority, a deliberately configured independent
acceptor, fresh exact-candidate evidence under #10 and verified lifecycle links.
Those are release prerequisites, not permission to invent an acceptor now.
Container validation currently cannot connect to Docker; use the documented
Go 1.26.4 fallback and retain hosted Linux CI evidence separately.

## Decision Spotlight

- First intended stable version: 1.0.0. A candidate is not a published release;
  acceptance binds its commit and manifest digest, not its version label alone.
- Deliver standalone platform binaries and a platform-neutral plugin archive.
  Bootstrap needs no Go, Python or archive extraction to inspect/install the CLI.
- Verified downloads establish integrity relative to the official GitHub source,
  not an independent publisher signature. Do not oversell checksums as identity.
- Keep immutable versioned local runtimes and replace only an absent or proven
  owned launcher. No sudo, shell startup edits, package manager or credential work.
- Installing the CLI leaves all memory connections unchanged. Updating a chosen
  connection is separately previewed and performed by the new runtime using its
  own embedded plugin. A fresh native session is required afterward.
- Retain old binaries; do not promise that binary rollback makes a newer signet
  schema readable. No automatic memory migration or downgrade is introduced.
- Patch this repository's artifact publication integration explicitly; do not
  change the shared template or weaken independent acceptance requirements.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Record the planning PR and exact-head contributor self-review, resolve findings,
and pass checks before finalization. Implementation may stack after self-review
under the delegated engineering policy. No release is authorized by this plan.
