# Privacy lifetime: engineering evidence

Delivery [#79](https://github.com/acoz-labs/mandalore/issues/79) implements #16 O1.
It changes documentation and tests, not runtime behavior, native instructions,
tool catalogs, data formats or privacy policy. Exact final-head review/checks
belong to the implementation PR; this document does not grant acceptance.

## Provenance and decisions

- [Approved discovery](https://github.com/acoz-labs/mandalore/tree/b0e6880bc5f1e6f2085e26a093625ba69e2a0abb/docs/discovery/16-privacy-lifetime), PR78.
- [Approved plan](https://github.com/acoz-labs/mandalore/tree/e8a4c24fc966153036b8cd0128fc01aa4628b24d/docs/plans/79-privacy-contract), PR82.
- Ordinary confirmed learning remains enabled. Labels describe records, not
  access controls. Supersession is not historical erasure. No automatic expiry.
- Export/redaction [#80](https://github.com/acoz-labs/mandalore/issues/80) and
  withdrawal/retention preview [#81](https://github.com/acoz-labs/mandalore/issues/81)
  remain deferred; no implementation or live-data authority is inferred.

## Maintained tests

| Requirement | Executable evidence |
| --- | --- |
| All four sensitivity labels are recallable | `internal/memory/privacy_test.go`: table-driven label preservation and exact recalled body/ID |
| Correction changes current recall, not old copies | Memory test verifies exact successor plus predecessor in history and independent original journal entry |
| No-save is not nondisclosure | `internal/api/privacy_test.go`: restricted marker appears in native context; recall/journal remain allowed |
| Enforced read-only denies mutations | Valid remember/journal/combined/Git init/checkpoint/sync calls denied in initialized and uninitialized fixtures; directory/file names, modes and hashes, including `.git`/`.mandalore`, remain unchanged |
| Old Git evidence persists | `internal/sync/privacy_test.go`: old and new heads retain original blob; graph-invalid predecessor deletion cannot advance HEAD |
| Append-only protection is independently exercised | Deleting a committed journal entry still passes Store.Validate, then returns ErrHistory and preserves HEAD/original blob |

Existing context, corruption/error-redaction, foundling disconnection and full
sync suites remain part of CI. No test treats metadata labels as secret scanning.

## Assertion sensitivity and verification

New characterization tests passed against unchanged runtime at test commit
`ed5f5cf`. A disposable worktree of `fadf41a` temporarily changed only two guards:
the API ReadOnly condition and the sync append-only changed-path condition were
made false. The targeted API test failed all seven mutation denials and its full
inventory check in both Git modes. The valid-journal-deletion test failed because
the checkpoint succeeded instead of returning ErrHistory. These failures confirm
the tests detect those regressions; they are not production fixes or a general
mutation-coverage claim. Both edits were restored exactly and the same focused
tests rerun. Final restoration/check results are recorded on the PR.

Run maintained cases with:

```sh
mise exec -- go test ./internal/memory ./internal/api ./internal/sync -run '^TestPrivacy' -count=1 -v
```

Use full `bin/container bin/ci`, or the documented pinned host fallback
`mise exec -- bin/ci` when Docker is unavailable. The final PR records host/hosted
checks and exact head. Go race/vet and four-target builds do not prove native
provider or cross-machine erasure. Tests create and clean up only their own
synthetic fixtures; no real bank, credential, remote service or native transcript.
Inventory evidence excludes access times and OS caches. This work has no rendered
impact, so UI recordings and new provider logins are not relevant evidence.

## Reconciliation and documentation promotion

| Temporary material | Durable destination |
| --- | --- |
| Storage/exposure, labels, no-save and retention | [Privacy guide](../../privacy.md) |
| Scoped incident and recovery boundaries | Privacy guide, SECURITY and runbook entry points |
| Future-operation designs and unresolved questions | Privacy guide plus #80/#81; clearly unimplemented |
| Tests and verification limits | Maintained tests, this evidence and exact-head PR review |
| Full discovery/solution-design history | Immutable links above |

The temporary #16 discovery and #79 plan are removed after promotion. No runtime
behavior drift is intended. The full inventory includes modes/empty directories
as additional regression coverage. No separate UI design or release deployment
is required for this docs/test change. Reviewed merge and post-merge checks can
complete #79/#16's selected outcome; deferred features and pending runtime
candidate acceptance/release remain separate.
