# Verification And Release Design

## Test Strategy

Failing-first adapter tests cover startup/prompt/unsupported events, missing and
swapped bindings, malformed/oversized/duplicate input, output budgets, Unicode,
scope isolation and byte-identical no-write state. Bridge tests use disposable
fake executables for missing/older runtime, invalid override, stderr suppression,
argument/environment routing and unrelated cwd. Validate manifests, skill YAML,
packaged paths and the actual compiled command together.

Run repository CI with pinned Go in the user-designated Herdr pane. Run plugin
and skill validators there too. Cross-builds and simulated hooks are not native
acceptance. Test fresh Codex against the exact binary/plugin hashes: discovery,
MCP tools, hook trust/warnings, plain-language save and recall, correction history,
no-save hashes, direct versus quoted cue, speculation, cwd and signet isolation,
resume, compaction and interruption. Record unsupported/not-exercised behavior
explicitly. Do not publish raw transcripts or workstation identifiers.

## Red/Green Sequence

First introduce hook/bridge contract tests that fail because the adapter/package
is absent. Implement read-only hook and bridge, validate routing, then add skill
and run forward native scenarios. Inspect actual memory graphs/journals and tool
receipts rather than accepting the model's final statement as proof.

## Rollout And Recovery

Use a disposable synthetic signet and explicitly chosen local runtime. Keep
existing native resources enabled except an explicitly scoped duplicate-memory
exclusion previewed before use. Changed hooks may need fresh trust. A missing
runtime should warn and allow unrelated work. Retain test artifacts for diagnosis;
do not repair user-owned bindings implicitly.

## Production Readiness Preflight

Implementation only: no deployment, public release, credential enrollment or
live-memory migration. Exact immutable artifact and independent product
acceptance remain #10/#11. This slice changes text/native terminal behavior,
not a custom rendered UI; document terminal readability and failure messages.
