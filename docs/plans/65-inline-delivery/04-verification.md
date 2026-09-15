# Verification

TDD shared API behavior: save plus actual local-origin delivery; failed input/save
causes no delivery; legacy tools stay local; read-only refuses new operations;
missing Git/remote, offline, semantic conflict and cancellation retain saved IDs;
writer contention/concurrent changes use existing locks; wrong bank is not used.
Exercise both record and journal operations and full returned success/error types.
Use existing sync regressions for transport invariants, plus combined-call tests
for publication-to-delivery boundaries. Compiled CLI and actual SDK stdio calls
verify schemas, annotations and response serialization. No string-only assertions
are evidence that model guidance works.

Native baseline/candidate ordinary-learning prompts in fresh synthetic bindings
and local bare origins must measure memory operations, exact saved/delivered heads,
code-mode/model requests and catalog/schema context cost. Candidate must select the
combined operation without a cue naming it, avoid redundant sync, choose local-only
under no-sync, perform no mutations under read-only/no-save, and report offline
delivery honestly. Include ordinary journal behavior. Stop and return to design if
operation-count/decision-gap benefit is not established. Preserve failures as evidence.

Run skill validation, complete pinned Go 1.26.4 CI and exact-head hosted CI. Use the
designated Herdr test lab; do not change the owner's installed connection or bank.

## Production Readiness Preflight

Implementation-only envelope. New immutable artifact nomination, exact-candidate
acceptance and release remain separate gates; no live installation or publication.
