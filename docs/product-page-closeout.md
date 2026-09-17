# Product-page release closeout

The release maintainer owns this checklist for each stable Mandalore release.
Product publication and website publication are separate deliveries. This is a
required operational closeout step, not an automatic Actions or deployment gate.

## Prepare and verify

1. Verify the published, non-draft stable release and accepted source/manifest
   identity. Use the existing public-byte verifier and release record; never
   derive public claims from an unaccepted candidate or intended `VERSION` alone.
2. Prepare concise highlights in `docs/releases/<version>.md`, including native
   support, important limits and upgrade guidance. Do not rewrite immutable
   assets or the machine-verifiable GitHub release body as marketing content.
3. Find or create one version-specific website issue in `acoz-labs/acoz`; link
   it from this release's record. Reuse its existing PR when retrying. The initial
   contract is discovery [acoz#130](https://github.com/acoz-labs/acoz/issues/130),
   implemented by [acoz#132](https://github.com/acoz-labs/acoz/issues/132).
4. Follow the website's `docs/mandalore-releases.md` contract and update its
   `src/data/mandalore-releases.ts` catalog. Review changelog, current version,
   installation commands, directory release link and capability wording together.
   Keep historical entries. Do not expand a tested setup-prompt claim merely
   because the native integration is available.

## Deliver and record

5. Follow website design/review/CI, fresh exact-stage acceptance and explicitly
   authorized production promotion. Product-release authority does not waive
   those gates. No new cross-repository token or automatic deployment is implied.
6. Verify the production page at <https://acoz.dev/projects/mandalore/>: version,
   changelog date/highlights, release links, copied download/install commands,
   guide deep links and accurate harness support. Record the website commit,
   accepted artifact, production release and verification evidence.
7. Add the receipt below to the product release record and linked issue. Only
   mark website closeout complete when the live page was actually verified.

```text
Product version / source / manifest:
Website issue / PR:
Website source / accepted artifact / production release:
Live URL:
Verified at / evidence:
Status: pending | verified
Next action / owner (if pending):
```

## Failure and resumption

If the site update is delayed or fails, report **product published; website
update pending**, retaining its issue/PR and last verified version. A published
artifact remains released; do not reopen its acceptance, rebuild it, force-push,
or delete a release because editorial delivery is incomplete. Website rollback
uses the site's existing operator workflow and restores pending closeout status
if its current-release guidance becomes stale again.

Automatic PR creation is deferred. A future implementation needs separately
reviewed identity, deduplication, retry, credential and permission boundaries.
Do not present this manual checklist as deterministic cross-repository automation.
