# Verification and release design

## Automated sequence

First characterize the actual MCP server using temporary synthetic signets and
the existing SDK test transport. Capture success, invalid input, read-only refusal,
pending delivery, conflicts, truncation and foundling citations. Assert equality
of the two full envelopes and error-state consistency. These tests may pass on
the current implementation; label them characterization, not missing behavior.

Add failing-first checks for the new instruction contract and exercise the exact
documented presentation example in a JavaScript-capable diagnostic environment.
Cases: equal structured/text, absent structured data, already-unwrapped result,
different text, additional content, unknown metadata, annotations, malformed
envelope, inconsistent error flag, Unicode, quotes/newlines and numeric precision
boundaries. Verify fallback does not lose information or call the tool again.
The native code-mode test, not a new production JS dependency, exercises the
example on the real host path.

Run `bin/container bin/ci`, or documented pinned Go 1.26.4 host fallback if Docker
is unavailable. Preserve meaningful failures and final exact-head CI evidence.

## Native evidence

Use an isolated synthetic signet/profile in the designated pane when available.
Never replace the user's active personal connection for an output-cost test.
Run before/after memory recall and foundling queries with the same fixture and
budgets, without prompting the candidate agent with the desired implementation.
Verify correct answers, citations, current/historical separation and no-write
hash windows. Include successful, error and truncated response presentations.

Record candidate SHA, plugin digest, actual harness/model versions, raw wire bytes,
model-visible output bytes and observed native usage separately. Whole-session
token deltas include instruction and model overhead and are not exact per-tool
token counts. Do not equate serialized wrapper savings with total-context savings.
Require the duplicate representation to disappear in the supported tested path;
do not assert all MCP clients change behavior. If it does not, return to design.

No graphical layout changes; evidence is synthetic tool-output/native behavior,
not a screenshot baseline. Retain only synthetic transcripts and compact receipts.

## Rollout and recovery

Planning and implementation do not change installed plugins. New native
instructions require a deliberately refreshed test connection and fresh session.
Keep v1.0.0 immutable; eventual production work needs a new exact artifact and
normal acceptance/release gates. Rollback is retention/reselection of the prior
runtime/plugin; no signet format changed and no memory rollback is needed.

## Production Readiness Preflight

Not applicable to the proposed `implementation` envelope: it cannot publish,
activate or migrate production. The implementation handoff must record the new
candidate identity and unresolved release prerequisites without borrowing the
MVP's exact-candidate acceptance exception.
