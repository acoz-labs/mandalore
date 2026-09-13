# Context

## Problem and outcome

[#2](https://github.com/acoz-labs/mandalore/issues/2) and the coordinated format
contract in [#3](https://github.com/acoz-labs/mandalore/issues/3) start the clean
memory-only successor. Reuse proven behavior without dragging the old assistant
framework into a new namespace.

## Verified source findings

Public source: `f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8` in acoz-labs/my-friday.

- `internal/memorybank` is a service layer over `internal/portable.Store`.
- `portable/store.go` co-locates identity creation, JSON atomic publication,
  locks, devices, evidence, schemas and legacy assistant validation.
- `portable/memory.go` contains records, graph validation, scopes and lexical
  retrieval; `bank.go` adds memory-only validation and journal reads.
- `events.go` starts with journal writes but then includes capability execution.
- `Store.Validate` touches SourceChanges and sync configuration even for banks.
- `changes.go` includes useful observer validation plus source-change tracking;
  `sync.go` calls it and uses cancellation from the capability hook module.
- Existing v1 uses `bank.json`, a `bank-` identity, bank-wide scope kind
  `assistant`, and `.my-friday` local lock/config paths. Those are real migration
  concerns, not merely documentation names.

Thus copying whole files/packages or replacing every name blindly would retain
unwanted behavior or alter persisted contracts without migration.

## Critical journeys

Create a synthetic signet, enroll a device, save sourced knowledge, recall within
scope, correct it with preserved history, inspect concurrent heads and journal
meaningful outcomes. Invalid or read-only operations leave persistent state
unchanged. Another device's provenance is preserved when inspecting records.

## Constraints

Public fixtures only. Preserve old installations and source unchanged. New code
must not depend on native credentials, machine accounts, capability execution or
provider packages. No CLI/TUI changes in this slice, so no rendered product-design
gate is needed; review the schema/data/privacy decisions explicitly.

Prior native evidence is informative only; the renamed product needs fresh
acceptance in #10. Sync transport, CLI/MCP, bindings, plugins and distribution
remain separate delivery slices, not hidden additions to this library port.

## Foundlings dependency

The approved product direction now includes a minimal foundling workflow before
real-memory adoption (#12 and #10). Historical references stay separate from
current memory; selective promotion preserves source identity and attribution.
Define registration/citation data in #3 now, without adding external retrieval
or a capability/reference execution framework to the core extraction.
