# Verification and release

The preserved first-error native run is failing-first evidence. Repeat the same
invalid-input-then-valid-recall prompt twice in fresh isolated native sessions,
with the revised skill and explicitly identified unchanged candidate runtime.
Also repeat ordinary current/history recall and a truncated historical reference
case. Do not add instructions to the user prompt telling the model to use the
selector. Verify actual native code inputs/outputs, complete envelope equivalence,
no duplicate valid-equivalent wrappers, no repeated invalid operation, correct
answers/citations and complete read-only fixture inventories. Retain every run.

Run the skill validator, exact-reference selector cases and full pinned CI.
Docker was checked and its daemon is unavailable; use the documented host
fallback, Go 1.26.4 and Node 24.1.0. Distinguish synthetic selector coverage from
native observations and memory bytes from total context/usage. No new graphical
or terminal layout: `no-rendered-impact`; native model-presentation evidence is
still required. Existing native authentication stays in place, never copied.

## Production Readiness Preflight

Not applicable to this implementation-only authorization: no secrets, deployment,
activation or publication is performed. Preserve v1.0.0 and the retained 1.1.0
candidate. A changed package requires a new candidate and nomination, followed
by existing human acceptance/release gates. Rollback reselects prior owned bytes;
no memory/schema rollback is needed. Retain exact source, package and test receipts.
