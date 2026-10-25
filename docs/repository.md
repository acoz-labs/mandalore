# Repository-specific instructions

Shared delivery rules are in `docs/operations/sdlc.md`. They supersede older
mandatory planning/owner gates and prospective self-review exceptions. Local verification and `docs/delivery.md` supersede Actions-only and mandatory staging instructions. Historical
records remain evidence. Product behavior, privacy boundaries and actual
validation/release requirements remain in force.

## Product boundary

Mandalore is a memory-only product. Follow docs/product.md and docs/migration.md.
Use synthetic examples; preserve privacy, licenses and upstream attribution.
Never import personal memory, credentials, workstation paths, machine inventory
or raw transcripts into this public repository. Assigned product delivery may
include release; unrelated live signet migration or installation is not implied.
Historical ADR0001–0003 exceptions do not authorize new self-reviewed deliveries.

## Validation

Run:

```sh
bin/container bin/ci
```

If container support is not ready yet, run:

```sh
bin/ci
```

Document any required deviation in `docs/development.md`.


## Current delivery

Profile: `artifact`. Follow [the local delivery runbook](delivery.md)
and [portable evidence commands](operations/local-verification.md). All configured
application validation remains required. Persistent staging and Actions are
optional; independent acceptance and exact-artifact verification are mandatory.
`bin/finalize-release artifact` and `bin/promote-candidate` are Actions-only and
are not part of the local release; see [publication](delivery.md#publication-and-recovery).
