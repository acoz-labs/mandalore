# Save locally and attempt delivery in one call

- **Status:** Final
- **Issue:** #65
- **Planning PR:** #66
- **Repository basis:** 2bb2436487f5e285e61051d4c346862f520bc5ab
- **Execution envelope:** implementation

Implements #55 discovery outcome O4 from reviewed PR #64. No lifecycle cache or hook transport.

## Decision Spotlight

New explicitly network-capable companions remove the separate post-save model
decision without changing existing local-only tools. A failed delivery must never
hide a successful save or encourage repeating it. Native benefit and schema cost
must be measured; current read-only/no-save/no-sync boundaries remain.

## Plan map

[Context](01-context.md), [decision](02-decision.md), [design](03-design.md),
[verification](04-verification.md), [handoff](05-handoff.md).
