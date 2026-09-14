# Technical design

## Shared operations

Add internal/install for plan, stage, native registration, doctor and repair;
embed the marketplace assets from plugins/codex. Add internal/console for the
terminal interaction primitives and a small cmd/mandalore menu adapter. Reuse
existing signet create/bind/Git/sync services rather than hand-editing memory.

Expose `connection plan`, `connection apply`, `connection doctor` and
`connection repair` commands with JSON receipts. Plan accepts selected absolute
runtime, binding, native binary/profile and machine-local state directory.
Apply accepts the emitted plan on stdin; repair previews by default and requires
explicit apply for mutation. Connection update is a new plan with a selected
runtime; it does not rewrite the currently executing CLI or shell PATH.
Advertise equivalent typed CLI-only operations through the operation catalog;
do not attach machine-configuration mutations to the normal bound memory MCP.

Plan reads bounded local inputs, validates binding and destination overlap and
hashes the selected runtime and bundled assets. Applying revalidates the plan
before execution, so changed source bytes, binding identity or paths require a
new preview. The chosen executable is trusted and runs only after apply. Probe
its product/protocol/OS/architecture before changing native registration.

## Local state and publication

Default state is a mandalore installation directory under the platform config
root, separate from the signet and native home. Explicit state/profile inputs
support testing and deliberate alternate native profiles, not auth cloning.
Retain runtime copies by SHA256 and generated marketplace roots by a digest of
package/configuration identity. Keep a versioned connection receipt with signet
ID, binding, runtime digest, native profile and generated file digests. Paths
remain local and never enter Git-backed memory.

Validate existing ancestors, symlinks, regular-file types, bounded sizes and
bank/state overlap before writes. Use no-replace publication and an exclusive
installation lock. A stale lock is reported, not guessed safe to remove.
Generated hooks/MCP share a local wrapper with pinned defaults. Native cache
versions include generated content identity so a fresh session sees the chosen
connection. Source package bytes are not rewritten in the public repository.

## Native registration and recovery

Inspect native marketplace/plugin inventory before replacement. Refuse unknown
ownership, redirected paths, duplicate legacy/new memory integrations and edited
managed contents. No auto-removal of a user-created development registration.
Use native commands only for registration/cache changes, with bounded output,
timeouts and sanitized errors. Preserve other marketplaces/auth/session files.

For a proven managed registration, retain its source/runtime before replacing
the registration. On partial remove/add/install failure, report the completed
phase and previous/target paths; retry revalidates state. Never claim rollback
unless the previous registration has actually been restored and verified.

Repair regenerates missing managed files into a fresh retained generation, not
in-place over unknown edits. Preserve edited bundles/caches and report a manual
ownership-resolution requirement where safe automated recovery is unavailable.
Successful reinstall verifies reported source/version/enabled state and actual
managed cache bytes. Warn that native trust review and a fresh session remain.

## Doctor and menu

Doctor inspects receipt, binding identity, runtime bytes, package/cache integrity,
native inventory and environment overrides. Report pass/fail/not-tested per
boundary; do not equate structural health with login, live MCP, hook trust,
network freshness or active model context. Doctor does not fetch/sync or repair.

Menu create preflights binding before creating memory; bind and Git-init remain
separate durable steps with partial receipts. Connect existing expects a local
clone, rejects predecessor schemas and does not synchronize without consent.
Inspect/sync delegate to existing operations. Native connect/repair present the
same plan the agent-ready command consumes, including execute/trust boundaries.

Traceability: navigation and previews belong to console/menu; machine portability
to binding plus generated defaults; update/repair to retained publication and
native registration; health distinctions to doctor; candidate acceptance to
#10/#11. No background installer, remote credentials or memory schema change.
