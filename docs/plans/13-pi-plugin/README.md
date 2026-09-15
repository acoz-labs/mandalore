# Solution Design: native Pi memory integration

- **Status:** Draft
- **Issue:** #13
- **Planning PR:** Pending
- **Repository basis:** 594399caf97c7496faf9a3c5ebc0640e8831ed5d
- **Execution envelope:** implementation

## Decision

Ship a native Pi package containing a thin extension and the two Mandalore skills.
Reuse the runtime's typed operation catalog and CLI transport, not another memory
engine. An explicitly installed, retained connection binds the package to one
signet independently of cwd. Ordinary Pi sessions retain their native tools,
resources, authentication and model access.

## Needs Attention

Engineering implementation and synthetic verification are authorized by ADR 0003.
New immutable-candidate acceptance, public release and personal activation are
separate gates. Pi 0.85.1 is the inspected native contract; other versions must
not be represented as tested. Lifecycle synchronization discovery #55 O1/O2 is
still unresolved and is not implemented indirectly by this issue.

## Decision Spotlight

- Use short-lived, shell-free CLI calls with authoritative Go validation. Keep
  the same operation schemas and receipts as MCP, without a second IPC protocol
  or a JavaScript storage implementation.
- Pin a connection's binding bytes and signet identity. A changed binding fails
  closed; restarting must not silently select an unintended bank.
- Supply bounded, fresh local context per turn. Hooks never save, synchronize,
  interpret cue strings as permission, or read native transcripts. Learning
  remains the host agent's semantic use of memory tools and skills.
- Embed the Pi package in the existing CLI payload. Preserve format-1 release
  inventory and the existing strict `version` response; expose Pi package
  metadata separately. Existing installers must remain able to verify upgrades.
- Extend the current setup/Armorer journeys with a harness choice. Apply uses
  Pi's package manager and owns only its proven registration, not the profile.
  No credential copying or replacement native launcher.

## Plan Map

- [Context](01-context.md)
- [Decision](02-decision.md)
- [Design](03-design.md)
- [Verification](04-verification.md)
- [Implementation handoff](05-handoff.md)

## Final Gate

Finalize after a distinct exact-head self-review under
[ADR 0003](../../decisions/0003-post-1-roadmap-self-review.md), recording its
findings in the planning PR. This is engineering self-review, not independent
product acceptance. Required hosted checks still precede merge. No production
code belongs in this planning PR.
