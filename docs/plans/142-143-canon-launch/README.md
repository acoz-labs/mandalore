# Canon foundlings and named signet launching

Delivery contract for [#142](https://github.com/acoz-labs/mandalore/issues/142)
and [#143](https://github.com/acoz-labs/mandalore/issues/143). This is a proposed
implementation and acceptance plan, not evidence of shipped behavior. The owner
assigned both outcomes through independently verified artifact release.

## Product decisions

Use **legacy** for an explicitly pinned historical reference and **canon** for a
Git-hosted Mandalore signet whose tracked branch refreshes at session entry.
Canon does not mean trusted instructions or universally correct knowledge.
Existing registrations retain legacy behavior and transport permissions.

Each native session selects one primary writable signet. It may consult that
signet's explicitly registered direct foundlings read-only. No reciprocal or
transitive access, automatic promotion, or automatic removal of previously
promoted knowledge is implied. Source withdrawals affect current recall at the
selected snapshot, not content already returned to a model.

`mandalore launch NAME [--agent HARNESS] [-- NATIVE_ARGS...]` is an optional thin
entry point. Names, native profiles, executable paths and default harnesses are
machine-local configuration. Shell functions and aliases remain user-owned.
Direct native starts must retain equivalent memory lifecycle behavior.

Launching does not install software, enroll a signet, copy a native profile,
change credentials or authorize a bank migration. Authentication remains owned
by each harness. Memory routing is not an operating-system sandbox.

## Engineering boundaries

The contributor owns code, meaningful regression tests and implementation PRs.
The maintainer owns product documentation, independent review, actual candidate
acceptance, release and verification. One credential switch is not independent
review. Keep implementation and review in isolated worktrees.

Registration identity, refresh permissions and source identity are distinct from
an observed branch commit. Routine refresh must not write a new portable
registration revision. New registration/schema support must fail safely in old
clients instead of silently allowing them to misinterpret canon as legacy.

Hooks, MCP and Pi tool calls must share the correct session selection. A mutable
bank-wide “latest session” pointer is insufficient when two sessions overlap.
Startup/resume selects a verified immutable snapshot; ordinary turns retain it;
explicit refresh can advance only its requesting session. Missing identity must
have a deterministic, documented outcome, not an invented association.

A failed fetch may reuse only a still-eligible verified cache, clearly marked
stale with its commit and observation time. Disconnected/conflicted registration
withholds access. A successful fetch cannot establish semantic agreement, that
unpublished remote work exists, or that already-read context was replaced.

Canon reads use current signet semantics, including scope, supersession,
withdrawal and conflicting heads. Never search raw journals/history as a fallback
for an ordinary failed recall. Citations bind source, registration, snapshot and
record revisions. Promotion remains an explicit separate write.

## Verification matrix

Use synthetic personal/work signets and a synthetic Git remote. Preserve failed
attempts and bind every acceptance result to the exact source and retained
artifact. A green test command alone cannot establish the user experience.

| ID | Scenario | Required observable evidence |
| --- | --- | --- |
| F1 | Existing legacy registration and old-client compatibility | Same reference bytes/history and no new fetch; unsupported new metadata fails explicitly |
| F2 | First canon registration and refresh | Verified source identity/branch/commit before initial context; no source mutation |
| F3 | Branch advances during open session | Original session retains its commit; fresh/resumed session sees new commit; explicit refresh advances the caller |
| F4 | Concurrent sessions | Distinct snapshot selections remain stable across interleaved hooks/tools |
| F5 | Offline, cancellation, absent branch, auth failure | Bounded total work; eligible stale cache or unavailable; distinct truthful receipts |
| F6 | Wrong identity, unsupported format, corrupt cache, disconnected/conflicted registration | Access withheld or accurately scoped stale fallback; no fresh-success claim |
| F7 | Supersession, withdrawal, restoration and conflicting heads | Current semantic recall at selected commit; no raw-history resurrection or winner-by-time |
| F8 | Provenance and promotion | Immutable source/record citation; explicit destination-only write; disconnect preserves prior provenance |
| F9 | Direct-only source access | Work cannot discover personal; personal cannot traverse work's foundlings; no source code/hook execution |
| F10 | Multiple foundlings and startup latency | Aggregate timeout/cancellation and output budgets; per-source receipts |
| L1 | Default and overridden harness | Correct signet, profile, executable, access banner and unchanged local defaults |
| L2 | Missing/bad configuration or changed binding | Actionable refusal; no alternate bank, implicit setup or credential mutation |
| L3 | Native arguments and working directory | Exact literal arguments, caller cwd and no shell evaluation; conflicting routing overrides refused |
| L4 | Simultaneous personal/work launches | Distinct synthetic saves stay in intended banks; no global connection rewrite |
| L5 | Resume and native memory | Same-environment continuation works; foreign history cannot silently enter the chosen environment |
| L6 | Preview, terminal and interruption | Preview has no launch/setup effects; narrow/no-color labels; keyboard, signal and exit-code fidelity |
| I1 | Direct Codex, Pi and Claude Code startup/resume | Actual native conversations plus hook/tool evidence of correct session snapshot |
| I2 | Launcher plus native hook | One shared lifecycle refresh path; no duplicate independent synchronization |
| I3 | Disabled/read-only connections | Existing transport/content restrictions preserved; no implicit policy adoption |
| R1 | Candidate installation/update/rollback | Retained hashes verified; synthetic signets/bindings unchanged by runtime-only operations |
| R2 | Public artifact release | Accepted bytes equal downloaded assets; immutable release/tag and recovery evidence |

Terminal changes are a materially changed experience. Exercise 80-column and
32-column/no-color output, success, validation failure, unavailable references,
and interruption. Browser/mobile screens are outside this CLI surface. Document
screen-reader and platform limits honestly; cross-builds are not native tests.

## Sequence and completion

1. Implement registration/reader/lifecycle and launcher behavior with regression
   tests; keep proposed behavior clearly labeled until verified.
2. Update permanent product, setup, foundling, format, synchronization, runbook,
   native skill and interface documentation to match actual behavior.
3. Review the exact PR head independently and run all configured checks. Resolve
   findings before merge; implementation PRs use `Refs #142` and `Refs #143`.
4. Build one retained versioned artifact from a fixed merged source. Nominate
   both issues with their complete implementation sets and run the matrix against
   those bytes. Publish honest independent receipts and retain failure evidence.
5. Release the accepted bytes under the existing artifact workflow. Verify public
   install/update/recovery, immutable identity, and all public asset hashes.
6. Close both issues only after criterion-by-criterion evidence review. Promote
   durable docs, retire this temporary plan and clean up task-created workspaces
   when their retained evidence/recovery obligations are satisfied.

Personal signets and installed connections are not acceptance fixtures. Actual
work-data transfer or personal activation remains a separately scoped operation.
