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

## Native Codex conversation

The same runtime/package was connected to the already-authenticated isolated
lab profile through reviewed `connection_plan` / `connection_apply`, with the
lab session stopped. The native result reported installed/verified and required
a fresh session. No credentials were copied, and no personal connection was
selected. Codex 0.155.0 (binary SHA256
`b0b14f9c1901c1ec44671094b2dc18b39e4bd8d36a6dc2302cc9d961a7e2a197`)
ran its configured gpt-6-astra model in the empty synthetic project. The directory
and both candidate hooks were reviewed/trusted under standing test authority.

Observed scenario, September 18, 2026:

1. A read-only request for the fictional route caused skill loading and one
   `memory_recall`. It returned Maple Quay from the correct synthetic signet.
2. A conversational request to stop using that memory, preserve history, and
   avoid journal/sync caused on-demand `visibility.md` loading,
   `memory_visibility_history`, then `memory_withdraw` with the exact observed
   content/visibility heads. The receipt reported withdrawn and durable locally.
3. A read-only follow-up explicitly excluding withdrawn history and prior
   conversation guidance caused a fresh empty-query recall. It returned no
   current records. The agent declined to use any current route name, despite
   Maple Quay remaining visible in its preceding conversation context.
4. Explicit restoration of the latest recorded content caused content and
   visibility history inspection, then `memory_restore` with the exact current
   heads. It restored Maple Quay, not the older Orchid Pier revision, and reported
   local durability without delivery.

The observed calls contained no journal, sync, repair or configuration mutation.
Synthetic Git status afterward showed exactly two new visibility files and no
tracked-file changes. This is a specific, explicitly directed conversational
scenario, not proof of every model, ambiguous user wording, spontaneous history
suppression or fresh-session recall after withdrawal. No public raw native
transcript or workstation path is retained here.

## Pi baseline and discovered restoration gap

Pi 0.85.1 with pinned Node24.1.0 and gpt-6-astra/high used the same synthetic
bank through candidate package
`05ab1aa8ad6c6da027958408769593c55e91e620d35de93abb62a5a397b81b6c`.
A disposable managed connection was installed and explicitly loaded into a native
test process, with ambient extensions/skills/context discovery disabled and the
two candidate skills explicitly supplied. Existing native authentication stayed
in place; this was not a personal-profile installation or credential isolation.

Pi recalled Maple Quay, used the visibility guide and exact-head withdrawal, then
started a fresh native session. The same ordinary read-only route question (no
withdrawal reminder) returned no available current name after two recall queries.
No historical body was surfaced as an answer. Explicit restoration inspected
the current content and visibility heads and restored Maple Quay successfully.

However, that fresh session had no retained record ID. It used `memory_inspect`
and two filesystem listings to discover one before using the historical tools.
This is a discovery gap, not accepted ergonomic behavior. The subsequent
`memory_withheld` operation supplies bounded scoped routing metadata for explicit
inspection/restoration; ordinary recall remains unchanged. Its tests cover
scope, summary-only matching, withheld-state routing and refreshed visibility.
The updated native behavior is recorded below.

## Pi fresh-session restoration after discovery fix

Runtime source `59025e54e726f988e6f9949adee448afb9afd476` was built with the same
pinned toolchain and installed into the disposable Pi connection after the prior
session exited. Binary SHA256:
`c2265b4f1111e5bda577474b01d09d1a7ce2dea879751d1f213fcf6f3178c03b`;
Pi package SHA256:
`ebe44459a70d087f44a84d0ba4142e20eb96b244e672725aa4c4c77a2c35d0b8`.
Native version, model and explicit-resource isolation were unchanged.

A new conversational withdrawal passed. After `/new` confirmed a new native
session, the restoration prompt supplied no ID, name, discovery tool or filesystem
directions. Pi loaded the visibility guide, called `memory_withheld`, inspected
content history and both visibility-history pages, then restored the exact
current heads. Maple Quay became visible; all historical evidence remained.
The only filesystem read in this restoration turn was the packaged guidance.
There were no shell listings, journal, sync, repair or configuration operations.

Together with the baseline's ordinary fresh-session withheld recall, this verifies
the specific tested discovery/restoration flow without relying on retained IDs.
It does not imply that historical queries are current guidance, or that all
ambiguous wording/model combinations are covered. The Codex fresh-session
discovery path is recorded below; the earlier directed withdraw/restore results
remain evidence for their explicitly pinned source.

## Codex fresh-session discovery after the fix

The same `59025e5` runtime was installed into the stopped synthetic Codex profile;
the reviewed generation reported installed/verified. Its embedded Codex package
SHA256 was `3ac5538e4fdf47b8c2baa4d68ec46375ad095aa3e44d601ca1237b6afc0c7e0a`.
Native Codex remained 0.155.0 with its configured gpt-6-astra model. The selected
record was withdrawn through the exact-head CLI before starting a fresh session.

The restoration prompt supplied neither ID nor remembered name nor discovery
sequence. Codex loaded the packaged memory and visibility instructions, used
`memory_withheld` with a summary query in the explicit signet scope, inspected
content/visibility history and restored Maple Quay using the exact observed heads.
It did not inspect signet files, manufacture another record, journal, synchronize
or change configuration. The result accurately separated local durability from
delivery. This closes the demonstrated ID-discovery gap for the tested Codex and
Pi flows, not every possible model or phrasing.

## Remaining evidence

These runs do not establish new-release bootstrap compatibility, native Linux
behavior or product acceptance.
Those boundaries must not be inferred from unit tests or cross-built binaries.
Final documentation reconciliation and exact-head review remain required.
