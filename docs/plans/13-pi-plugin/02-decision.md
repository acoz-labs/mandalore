# Solution Decision

## Decision Drivers

Preserve one memory contract, native Pi behavior, bank isolation, honest partial
effects, small context cost and existing upgrade compatibility. Installation and
recovery must remain both conversationally and deterministically drivable.

## Competing Approaches And Adversarial Comparison

1. Skills that shell out directly: least code, but discovery/recall would depend
   on the model reading instructions first. No reliable native attachment or
   typed tool exposure. Keep the skills, not this as the whole integration.
2. A Pi MCP client extension: shares MCP, but adds connection initialization,
   framing, long-lived child cleanup and client dependencies that Pi's native
   tools do not require. Credible if a shared native MCP facility is adopted
   later, but not necessary to meet this issue.
3. Native tools over the typed CLI: selected. Schemas and envelope semantics stay
   in Go; Pi contributes presentation and lifecycle only. Costs are child-process
   startup and possible binding changes between calls. Bound and measure the
   former, and guard the latter in Go on the actual decoded bytes.
4. Reimplement memory or RAG in the extension: rejected; creates divergent
   validation, locking, search and migration semantics.

## Selected Approach

Author an ESM JavaScript package using Node builtins and native Pi interfaces,
with no runtime npm dependencies or build-time JavaScript transpiler. Discover
the retained runtime's operation catalog at attachment, register the same bound
operations as MCP, and invoke its CLI with explicit guarded connection arguments.
Keep administration out of memory tool registration.

Confidence is high for native package/schema primitives (observed), conditional
on integration and model tests for conversational behavior (not yet claimed).
Avoid refactoring all Codex installation into a general plugin framework. Share
small byte/path/process helpers only where both adapters have the same invariant.

## Decisions Ledger

| Decision | Rationale and consequence |
| --- | --- |
| One owned package per selected Pi profile | Multiple banks remain separate explicit profiles; prevent duplicate memory tool collisions. |
| Binding digest enforced in runtime | JS-only prechecks leave a check/read gap; compare the same bounded bytes decoded by `binding.Open`. |
| Per-turn read-only context | Fresh files without accumulating durable attachment messages or parsing transcripts. |
| Native tools carry one complete JSON envelope | Preserve partial writes and delivery evidence without duplicate result bodies. |
| Native event mapping is explicit | `agent_end` can precede retries; even `agent_settled` is not write authorization. No automatic checkpoints here. |
| Pi bytes embedded in CLI | Existing manifest/updater compatibility; Pi identity is part of executable digest and separately inspectable metadata. |
| Separate Pi connection operations | Preserve Codex's existing typed contracts rather than changing their meaning. |
| Native install/remove for registration | Avoid hand-editing settings; retain generated files and report failure phases. |
| Pinned test runtime | Record Node 24.1.0/Pi 0.85.1 as initial inspected engineering baselines, not a claim of supported future versions. |

No new stored authorization object, observer daemon, hook subscription framework
or automatic updater is justified. If native tests contradict a selected
mechanism, revise this plan instead of quietly weakening the issue's outcome.
