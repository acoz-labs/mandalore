# Implementation handoff

This is substantive user-facing integration, not a cosmetic wrapper. Implement
reviewable slices in dependency order: embedded assets/local plan and staging;
native apply/doctor/repair; JSON administration; console/menu; actual native and
rendered evidence. Each slice begins with failing tests for its owned behavior.

Port selected public console and installation primitives from the pinned source,
reviewing both existing flaws and current native contracts. Keep memory logic in
the merged engine. Do not import portable assistant/capability modules to obtain
one helper. Update NOTICE for adapted portions and run public-content checks.

Acceptance traceability is in 03-design.md and the scenario matrix in
04-verification.md. The shared operation functions are the agent-ready surface;
the menu must not become the only way to discover or apply a configuration.

Promote the final behavior into docs/interface.md, docs/runbook.md,
plugins/codex/README.md and a focused setup guide; architecture/development docs
record ownership and validation. Keep sanitized rendered/native engineering
receipts linked from the implementation PR. Record final-head self-review,
drift, documentation promotion and remaining acceptance gaps, then delete all
six temporary plan files before marking the implementation PR ready.

Use design/7-setup-recovery for the planning-only PR, feature/setup-recovery for
implementation. Self-review authorization permits implementation on the reviewed
planning head while CI/merge is pending. Required checks still must pass. Keep
#7 open after merge until immutable-candidate everyday setup/update/repair is
accepted under #10/#11.

Non-goals: automatic OS package installation, provider account/credential setup,
remote repository provisioning, capability scripting, arbitrary lifecycle hooks,
assistant aliases or launchers, auth/profile copying, live migration, release
publication and automatic garbage collection. A missing release artifact is an
honest update limitation, not permission to execute a mutable remote script.

Reopen design for a new data/trust boundary, incompatible native contract, or
unplanned irreversible mutation, not ordinary implementation choices.
