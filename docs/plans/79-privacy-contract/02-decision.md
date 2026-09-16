# Solution decision

## Drivers and alternatives

Truthful guarantees, minimal scope and regression protection take priority.
Docs alone are insufficient because exposure/lifetime claims can drift.
New privacy mechanisms would change user behavior and exceed selected O1.
Select docs plus characterization tests; do not manufacture a feature change
to make an existing-behavior test fail first.

## Decisions

| Choice | Rationale |
| --- | --- |
| One durable privacy guide | Keep current and explicitly future contracts together without phase-document duplication |
| Service/API/Git synthetic tests | Exercise actual evidence lifetime and denial paths without private data or provider calls |
| Labels remain descriptive | A regression test documents behavior; it does not endorse saving secrets |
| Full file inventory for read-only | Include operational state and Git files, not just portable records; no access-time claim |
| Independent valid-file deletion test | Separate append-only refusal from graph validation failures |
| No rendered impact | No menus, command presentation, native instructions or runtime code changes |

Confidence is high on this bounded delivery. Return contradictions in observed
behavior to design rather than silently adapting the claim to pass tests.
