# Withdrawal engineering evidence

Issue #81, implementation PR #123. This is engineering evidence, not product
acceptance, release authority or permission to upgrade a personal signet.

## Compiled CLI

On September 18, 2026, the [driver](native-driver.mjs) passed in the designated
Herdr lab on macOS arm64 with Node 24.1.0 and the pinned Go 1.26.4 build.
The retained runtime source is `55a34f9e3f1d9f7fda501a3ee4ab26e6f6ae9039`.

- Binary SHA256: `407f8d2ae06e917ad46abc795f6597511b53d8beeb169e534e0788835205bb1c`
- Embedded plugin SHA256: `c6db05a13299f8f387f0e6ead170c7f75e04a11ec58cfb8e4ebb4b41b3e4adce`
- Driver SHA256: `51de1e95bcb280f5e54c5ec30bbcef52be85269c8c2cd27914e148519583d90b`
- Released old source: `51aee17afec015ba2ad44584f8190b4bb6d901a8`
- Released old binary SHA256: `09827de1c1340284dbc5506f2c0db41f8cbd66d9231e3e5dbd9d93ea01fc2429`

The driver takes explicit new and released-old executable paths, creates only
synthetic fixtures and a local bare Git remote, and retains its private fixture
and evidence JSON. No personal connection or external account is selected.
Run with the repository's pinned Node runtime:

```sh
node docs/evidence/withdrawal/native-driver.mjs /absolute/new/mandalore /absolute/old/mandalore
```

Verified checks:

- Read-only upgrade preview, explicit acknowledgement, local activation and
  idempotent recovery preserve existing portable evidence without checkpointing.
- The actual released old reader refuses the upgraded bank. Its offline old
  clone still reads old content: this is not revocation or erasure.
- Withdrawal suppresses ordinary recall/context, remains counted in scopes,
  and refuses stale restoration. Retention preview returns metadata only and
  does not implicitly select the independent journal.
- Default export withholds the record. Both historical opt-ins disclose its
  preserved content. The separate journal remains intact.
- A format1 clone reports upgrade-required without moving its HEAD. Explicit
  independent local upgrade then synchronizes, retaining both upgrade receipts.
- A correction does not restore a withdrawn record. Stale-content restoration
  refuses; explicit fresh restoration makes the correction visible in both clones
  while retaining both content revisions.

The first attempted run refused export because the test placed the destination
inside the binding directory. The corrected fixture uses a separate configuration
directory; the export protection was not relaxed. Both fixtures were retained.

## Remaining evidence

This compiled-CLI run does not establish conversational Codex/Pi behavior,
new-release bootstrap compatibility, native Linux behavior or product acceptance.
Those boundaries must not be inferred from unit tests or cross-built binaries.
Final documentation reconciliation and exact-head review remain required.
