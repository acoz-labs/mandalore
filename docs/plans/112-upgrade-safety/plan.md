# Solution Design: deferred native connection replacement

- **Status:** Draft
- **Issue:** #112
- **Planning PR:** #114
- **Repository basis:** c9e216fa2c9aec443a74a1b8696993780a6c63b2
- **Execution envelope:** implementation

## Context and desired outcome

Updating a connection must not silently invalidate dependencies of running
sessions. The issue permits preservation or an explicit safe deferred-update
path. `internal/install/native.go` currently removes the owned marketplace when
changing connection roots. Codex 0.154.0 deletes its old cached plugin, while an
already-open MCP process can survive. This is not observed signet corruption.
The success notice incorrectly promises existing sessions were unchanged.

A fresh isolated native registration probe also confirms that adding a different
local source with the same marketplace name is refused until removal. It cannot
replace the current remove/add sequence without a different registration design.
No personal native profile, credentials or signet was changed by that probe.

## Decision and alternatives

Select an enforced deferred replacement with explicit stopped-session
acknowledgement. Fresh installation remains straightforward. A replacement or
repair that can replace/remove native plugin dependencies is denied by default
before native registration mutation. The caller can defer and continue using the
old connection, or exit affected sessions and explicitly acknowledge that fact
when applying. Acknowledgement is a user assertion, not automatic process proof.

Rejected: warning-only changes leave the silent disruption possible. Retaining
only hook entrypoints would not preserve other old cache resources (including
skills), and cannot retroactively rewrite definitions in running legacy sessions.
Restoring deleted native caches bypasses the native manager's ownership. A stable
mutable marketplace plus retained generations is a broader architecture change
with crash/recovery and trust implications; not needed for the issue's explicit
deferred-update outcome. No active-session scanner, daemon or hot reload promise.

## Product design and critical journeys

Use the existing confirmation/error/receipt patterns; no new screen framework.
CLI and typed callers get the same guard. Before connection replacement the menu
explains that affected Codex sessions must exit, idle is insufficient, and the
new plugin takes effect only after restart and native trust review. Default is
defer; CLI apply exposes an explicit stopped-session acknowledgement. The Armorer
cannot acknowledge on the user's behalf merely because it is between turns; if
it runs in the affected profile, it supplies a standalone terminal handoff.
CLI binary update alone does not imply plugin activation.

Fresh installation requires no stopped-session assertion. Repeat application of
an already verified identical connection must not invoke a reinstall that could
disturb its cache. It can report verified unchanged. If verification detects
missing/drifted managed dependencies, repair must use the guarded path rather
than treating identical coordinates as proof of a no-op.

## Technical contract

Add an explicit apply-time acknowledgement, kept out of persistent connection
identity and package/cache hashes. Preserve exact plan revalidation and existing
source/runtime/native/binding digest checks. Existing callers without the field
retain fresh-install support; unsafe replacements fail with actionable guidance.
Do not persist a reusable acknowledgement in installed receipts or allow a
previous successful apply to authorize a later replacement automatically.

Guard before registration removal/addition and before changing native cache
contents. Read-only native inventory may create native logs; do not claim a
filesystem-wide no-write preflight. Existing lock, ownership and collision checks
still apply. Recheck inventory after staging; changed registration must be refused.
Explicit acknowledgement never overrides foreign ownership or modified files.

Denial retains the existing connection, runtime and signet. Acknowledged partial
failure retains existing phase receipts; no automatic rollback, resume or retry.
Recovery inspects actual registration and retained sources before another explicit
apply. Updated notices distinguish registration, fresh-session activation,
possible old-cache removal and preserved signet/authentication state.

## Verification and acceptance

1. Failing-first unit tests: replacement without acknowledgement makes no native
   mutation and retains old cache bytes; acknowledgement permits replacement;
   valid identical replay is non-mutating; missing/drifted cache cannot silently
   bypass the guard. Foreign ownership and stale-plan denials remain enforced.
2. API/CLI parity: absent/false acknowledgement denies replacement; explicit true
   passes the guard, with strict input validation. Assertions do not persist or
   change deterministic connection identity. Menu decline never dispatches apply.
3. Native Codex disposable-profile test with real old/new runtimes: hold old hook
   and MCP consumers; unacknowledged replacement is denied and both still work.
   Exit consumers, explicitly apply, then verify the new hook/MCP and fresh-session
   identity. Include native partial-registration failure and recovery where feasible;
   simulated failures must be labeled as such.
4. Model-driven acceptance: an active synthetic conversation remains functional
   after deferred update; stopped-session upgrade followed by a fresh native session
   reaches the same synthetic bank with the new runtime. No copied credentials,
   personal signets or private transcripts in public evidence.
5. Rendered classification: in-pattern visual change. Exercise plain/TUI default
   defer, acknowledgement, error and successful handoff at normal/narrow widths;
   retain synthetic recordings and exact-head evidence. No browser surface.
6. Run pinned host/container CI, native plugin and changed-skill validators,
   privacy review, exact-head engineering self-review and hosted checks.

Native evidence must distinguish component consumers from complete agent-session
behavior. Cross-builds do not establish Linux runtime acceptance. Release and
live installation are outside this implementation envelope; #112 stays open
pending its exact-candidate acceptance/release gates.

## Handoff and documentation promotion

Planning-only PR precedes implementation branch. Self-review follows ADR0003's
project-wide extension. Implement the installer guard and idempotent replay first,
then expose the apply acknowledgement through typed/CLI/menu/Armorer surfaces,
then record native and rendered evidence. Promote the contract into setup/runbook,
interface and plugin administration documentation. Reconcile the exact final head,
record plan drift and remove this temporary plan before marking implementation
ready. Required checks and reviewed merge do not equal product acceptance.

## Needs attention / final gate

Review precise apply-input compatibility and all menu/delegated-runtime call paths
before finalizing. No implementation before a recorded exact-head planning review.
No release, live activation, signet migration or subsequent roadmap issue is
authorized by this plan. Stop after #112 and reassess the remaining roadmap.
