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
- [#11: Build reproducible artifacts and publish through GitHub Releases](https://github.com/acoz-labs/mandalore/issues/11)
- [#10: Prove the Codex memory MVP on an immutable release candidate](https://github.com/acoz-labs/mandalore/issues/10)

## Cross-harness continuity

- [#13: Add the Pi plugin after Codex MVP acceptance](https://github.com/acoz-labs/mandalore/issues/13)
- [#14: Add the Claude Code plugin after Pi acceptance](https://github.com/acoz-labs/mandalore/issues/14)

## Later improvements

- [#15: Assess memory integration readiness on a new machine](https://github.com/acoz-labs/mandalore/issues/15)
- [#16: Define signet privacy, export, retention and deletion semantics](https://github.com/acoz-labs/mandalore/issues/16)

## Current handoff

Repository foundations, engine/schema, shared interface, synchronization, Codex
integration, managed setup and explicit memory-only migration are implemented.
Foundlings (#12), retrieval (#9), distribution (#11) and The Armorer (#45) are
included in the owner-accepted [v1.0.0 release](releases/1.0.0.md). Publication
preserved the exact retained bytes. The release record and linked issues separate
the owner's verdict, public verification and completed delivery ledger.
Foundlings belongs before real-memory adoption, not merely an optional post-MVP
improvement. Codex acceptance precedes Pi; Pi acceptance precedes Claude Code.
Engineering merges alone are not release acceptance.

The public [Mandalore board](https://github.com/orgs/acoz-labs/projects/35) contains
the original 16 successor issues. Template fields/views and item status were verified;
see the [project receipt](operations/project.md) and issue #1 for live follow-up.
My Friday is archived and its planning board is closed; historical work is
preserved rather than marked successfully implemented.
