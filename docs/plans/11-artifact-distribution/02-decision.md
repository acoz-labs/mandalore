# Solution Decision

## Decision Drivers

Same-byte provenance, a low-friction first install, honest compatibility and
failure reporting, separate activation authority, deterministic tests, existing
Go/CLI patterns and a small public dependency surface.

## Competing Approaches

1. Publish manually built files and ask users to copy them. Minimal code, but
   cannot establish the issue's repeatability, guided update or partial-publication
   contract. Rebuilding after acceptance destroys the artifact identity.
2. Add a general package manager or arbitrary setup-script system. It could manage
   many tools, but exceeds this memory-only product and expands trust unnecessarily.
3. Product-specific manifest, deterministic builder, bounded release installer
   and candidate promotion integrated with the existing ledger. Selected.

## Adversarial Comparison

Unsigned checksums do not prove publisher identity; the selected path explicitly
trusts official GitHub transport and source, validates metadata and fails closed
on digest disagreement. Candidate workflow origin and exact source must be checked
as well as archive bytes; a user-supplied artifact ID alone is insufficient.

An old installer can combine a new binary with its own old embedded plugin.
Selected-connection activation therefore delegates only known typed operations to
the verified new runtime. It is never a generic remote command hook.

Updating a stable CLI path could erase a user's unrelated program. Retained
versioned binaries and a verified owned symlink/receipt give a narrow activation
boundary; unknown files/symlinks are denied, even if their filename is mandalore.

Candidate expiry and interrupted publication remain possible. Missing transport
cannot be silently replaced by a rebuild; a newly built candidate is nominated
again. Identical existing release assets can be reused, never clobbered.

## Selected Approach

Build four raw standalone binaries plus a deterministic plugin archive, strict
manifest, checksums and reviewed bootstrap script from clean pinned source.
Retain one complete Actions artifact and verify its identity before nomination and
promotion. Build IDs and timestamps do not enter deterministic payloads.

Expose CLI-only release inspect/plan/apply operations and an existing-pattern menu
journey. Default official discovery excludes drafts/prereleases. Explicit version
selection supports a published prerelease but clearly labels it. Local candidate
inspection is explicit and never represented as a GitHub release.

Confidence is high in the architecture; actual reproducibility, native behavior
and acceptance are evidence to earn during implementation and #10, not assertions
made by this design.

## Decisions Ledger

| Decision | Reason |
| --- | --- |
| One `VERSION`, stable target 1.0.0, release tag `vVERSION` | CLI, plugin and manifest agree; candidate identity is still stronger than version |
| Four raw CLI binaries, one plugin archive | No bootstrap archive traversal or extra interpreter dependency |
| Manifest format v1; explicit protocol and signet read/write versions | Compatibility is checked rather than inferred from the product name |
| SHA-256 identity plus verified official provenance | Detect corruption/mismatch without claiming independent cryptographic signing |
| Explicit CLI plan/apply, then separately selected connection | No silent bank switch, old-plugin mixing or all-session update |
| Existing console and API envelope patterns | Preserve keyboard/plain usability and machine-driven access |
| Repository-local artifact ledger adaptation | SemVer asset release, not a second empty calendar release; shared template unchanged |
| Retained assets and phased receipts | Retry is inspection-driven; rollback preserves data and older artifacts |
