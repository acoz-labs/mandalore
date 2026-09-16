# Verification

Failing first: assert native home exists before the first fake inventory command,
with restrictive permissions when new. Assert preview leaves it absent. Then cover
existing sentinel files/mode preservation, canceled apply, a destination replaced
by a file or symlink after preview, and bounded operation-specific inventory errors.
Existing collision, edited-cache, upgrade, partial retry and cancellation tests
remain required. Run pinned full CI, including race tests and privacy checks.

In the engineering lab, use real Codex 0.154.0 and a newly absent synthetic home:
record preview absence, apply success, actual registration/cache and structural
checks. No provider authentication or personal profile copying is needed. Record
native binary and source identities, exact commands and bounded sanitized results.

UI classification: existing-pattern diagnostic text, no new rendered flow or
substantive product-design gate. Verify failure copy and success through the
existing command/menu surfaces. Replacement retained-candidate human acceptance
remains separate from engineering checks. Old evidence is never rewritten.

Production preflight: no production execution in this implementation envelope.
Release requires reviewed main, immutable artifact build/nomination, actual owner
acceptance under a valid candidate-specific policy and explicit publication.
