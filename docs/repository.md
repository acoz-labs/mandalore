# Repository-specific instructions

Shared delivery rules are in `docs/operations/sdlc.md`. They supersede older
mandatory planning/owner gates and prospective self-review exceptions. Historical
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

## Delivery configuration

Delivery profile: `artifact`.
The actual candidate, verification and rollback contract is in `docs/deployment.md`.
