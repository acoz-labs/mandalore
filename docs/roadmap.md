# Roadmap

The issue tracker owns detailed acceptance and status. This document maps the
sequence and scope; issue creation is not implementation or acceptance.

## Memory MVP

- [#1: Establish Mandalore and preserve My Friday as the predecessor archive](https://github.com/acoz-labs/mandalore/issues/1)
- [#2: Extract the memory-only engine and its regression tests](https://github.com/acoz-labs/mandalore/issues/2)
- [#3: Define the versioned signet structure and compatibility contract](https://github.com/acoz-labs/mandalore/issues/3)
- [#4: Port the agent-ready CLI and shared stdio MCP interface](https://github.com/acoz-labs/mandalore/issues/4)
- [#5: Finalize local-first synchronization and honest lifecycle freshness](https://github.com/acoz-labs/mandalore/issues/5)
- [#6: Deliver the Mandalore Codex plugin and This Is the Way memory skill](https://github.com/acoz-labs/mandalore/issues/6)
- [#7: Finish setup, connection, update, doctor and repair UX](https://github.com/acoz-labs/mandalore/issues/7)
- [#8: Provide explicit migration and safe coexistence with prior memory installations](https://github.com/acoz-labs/mandalore/issues/8)
- [#12: Foundlings — linked historical references and provenance-preserving promotion](https://github.com/acoz-labs/mandalore/issues/12)
- [#9: Measure retrieval quality, context cost and growing-corpus performance](https://github.com/acoz-labs/mandalore/issues/9)
- [#10: Prove the Codex memory MVP on an immutable release candidate](https://github.com/acoz-labs/mandalore/issues/10)
- [#11: Build reproducible artifacts and publish through GitHub Releases](https://github.com/acoz-labs/mandalore/issues/11)

## Cross-harness continuity

- [#13: Add the Pi plugin after Codex MVP acceptance](https://github.com/acoz-labs/mandalore/issues/13)
- [#14: Add the Claude Code plugin after Pi acceptance](https://github.com/acoz-labs/mandalore/issues/14)

## Later improvements

- [#15: Assess memory integration readiness on a new machine](https://github.com/acoz-labs/mandalore/issues/15)
- [#16: Define signet privacy, export, retention and deletion semantics](https://github.com/acoz-labs/mandalore/issues/16)

## Current handoff

Repository foundations, engine/schema, shared interface, synchronization, Codex
integration, managed setup and explicit memory-only migration are implemented.
Foundling consultation/promotion (#12) is engineering-merged with synthetic menu/native
evidence. Retrieval measurements and bounded fixes (#9) have synthetic scale,
freshness and native learning/continuity evidence. Next: final #9 engineering
reconciliation, distribution (#11) and immutable-candidate acceptance (#10), then foundation
reconciliation (#1). Foundlings belongs before real-memory adoption, not merely
an optional post-MVP improvement. Engineering merges are not release acceptance.
Codex acceptance precedes Pi; Pi acceptance precedes Claude Code. Release-bearing
issues stay open through independent acceptance and verified GitHub publication.

The public [Mandalore board](https://github.com/orgs/acoz-labs/projects/35) contains
the original 16 successor issues. Template fields/views and item status were verified;
see the [project receipt](operations/project.md) and issue #1 for live follow-up.
My Friday is archived and its planning board is closed; historical work is
preserved rather than marked successfully implemented.
