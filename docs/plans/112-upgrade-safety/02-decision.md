# Solution decision

Preserve active consumers by refusing replacement until an explicit handoff.
Warnings alone fail the issue. Moving only future hooks leaves legacy hooks and
other resources exposed. Reconstructing native caches violates ownership. A new
mutable marketplace/retention architecture exceeds the needed deferred-update
outcome. Detailed tradeoffs are in [Technical design](03-design.md).

Confidence is high in the observed removal mechanism and native registration
constraint. Complete model-session behavior still requires verification; direct
hook/MCP tests alone cannot prove it. A stopped-session assertion is deliberately
not represented as verified automatic process detection.
