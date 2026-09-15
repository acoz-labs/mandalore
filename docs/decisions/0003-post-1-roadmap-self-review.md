# Post-1.0 roadmap engineering self-review

Status: accepted by explicit product-owner direction, September 15, 2026.

The owner extended the [MVP engineering self-review](0001-mvp-self-review.md)
approach to the approved post-1.0 roadmap, after being asked to choose between
that extension and a separate reviewer. Scope: release documentation #52 and CI
triage #19; model-visible MCP rendering #56; synchronization #55; foundling
retrieval #57; Pi #13; anonymous release quota UX #54; privacy/export/retention
#16; new-machine readiness #15; and Claude Code #14 after Pi acceptance.

Contributor self-review of discovery, solution design and implementation replaces
the separate engineering reviewer and ordinary product sign-off gates within
these outcomes. Record a distinct review pass, the exact head, meaningful tests,
findings and their resolution. A self-reviewed planning head permits stacked
implementation while the planning PR awaits CI/merge. Required checks must pass
before merge. Do not impersonate an independent approver or fabricate approvals.

Surface consequential decisions and remaining risks. Return material scope,
trust/data-boundary or irreversible changes to the owner. This repository-specific
extension does not modify the organization's shared project template.

Required checks, runtime verification, privacy, exact-candidate acceptance,
public release and live-installation authority are unchanged. The
[MVP owner-acceptance exception](0002-mvp-owner-acceptance.md) remains limited to
its enumerated candidates; this decision neither accepts a new candidate nor
authorizes publication, private signet migration or personal plugin activation.
