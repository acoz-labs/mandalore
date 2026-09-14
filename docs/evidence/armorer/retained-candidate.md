# Retained-candidate owner walkthrough observations

Observed 2026-09-14. These are contributor-recorded observations of prompts
personally submitted by the product owner, not an inferred acceptance verdict.
No public release or live-memory migration was performed.

## Exact artifact

- Source: `f899cf6a2a255f3b6b35dcd778c672f799c65eb2` (merged #47).
- Manifest SHA-256: `c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea`.
- Darwin ARM64 executable SHA-256: `bcc1b7fdf9e72ccb2bbd734d798c16f3aa5b96de757846711a4e5afe6eb00f93`.
- Embedded package SHA-256: `f181df926b28a2114eacd8ea908df71a0c223722d7e53ec9c689b050c341d250`.
- [Successful retained build](https://github.com/acoz-labs/mandalore/actions/runs/34886390526).
- [Successful nomination](https://github.com/acoz-labs/mandalore/actions/runs/34886694068).

The downloaded archive and payloads were verified before executing the retained
binary. The synthetic native profile used Codex 0.153.4 and GPT-6 Astra on macOS
ARM64, with a project working directory outside the bank and no Mandalore
runtime/binding environment overrides. No raw transcripts, account details,
credential contents or workstation paths are included here.

## Observed scenarios

1. A natural read-only health request selected The Armorer without its name in
   the prompt. Twelve structural checks passed. Live MCP reads reached the same
   bank as the CLI. The response distinguished passing checks, pending local Git
   changes and unverified remote/authentication boundaries. No apply or memory
   write operation was observed; the bank digest was independently unchanged.
2. With the session closed, one receipt-owned generated source bridge was moved
   outside the managed tree to a recoverable backup. Its exact bytes were checked
   against the ownership receipt. The native cached copy remained intact. This
   tests missing-source recovery, not an outage of the running MCP process.
3. A fresh session's read-only inspection correctly reported an incomplete source
   bundle, while the runtime, cache, binding and memory remained healthy. It used
   `connection_repair_plan` and described a new generation, preserved resources
   and the need for a fresh session. It did not apply the plan.
4. On explicit owner instruction, the same session applied the exact preview via
   `connection_apply`, then checked the installation. All twelve structural
   checks passed. Existing pending local history remained pending. No memory
   save, journal or synchronization operation was observed.
5. Independent checks after exit verified the missing file's restoration in the
   exact previewed new generation, unchanged binding/runtime bytes, preservation
   of the previous generation and backup, and unchanged full non-Git bank digest
   and Git HEAD. A local verification script initially used the wrong diagnostic
   result key (`connection_root` instead of `marketplace_root`); correcting that
   checker produced a full pass, without a product-code change.
6. A fresh session then received a natural read-only question about the fictional
   project, update preference and defect-report checklist. It loaded only This
   Is the Way and made two recall calls. It correctly returned Meadow Kestrel
   (formerly Harbor Wren), two short sentences, and reproduction steps/expected
   result/observed result. The checklist remained available after disconnection
   of its historical reference. After exit, independent preservation and
   structural checks passed again.

## Limits and remaining authority

The synthetic bank intentionally has no remote and retains an uncommitted
foundling-disconnection record; pending delivery is expected, not remote success.
Fresh live recall is observed separately from the structural diagnostic's
not-tested fields. This does not newly verify credential enrollment, hook-trust
UI changes, all native session context, other platforms, or every repair failure
mode. Credentials were not read to manufacture a preservation comparison.

Earlier owner tests and platform/fault matrices on the predecessor candidate
remain historical evidence, not executions of these replacement bytes. The
development menu recordings likewise remain tied to their stated development
head. This record is a sanitized observation summary, not a substitute for the
exact-candidate acceptance workflow or its evidence requirements.

The existing owner-review exception is pinned to the previous candidate and
issues #2–#12. Extending it to this candidate and #45 needs explicit owner
authorization. Formal acceptance and public-release authority remain separate.
