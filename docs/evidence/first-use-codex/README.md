# First-use Codex profile regression (#93)

Engineering verification, September 16, 2026. This is not owner acceptance,
provider authentication, live-model proof, personal activation or release consent.
The failed retained candidate `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`
and its human acceptance record remain unchanged.

## Failure and correction

Codex 0.154.0 rejects `plugin marketplace list --json` when CODEX_HOME is absent.
Apply previously made installation state but inventoried the native profile before
preparing it. Earlier upgrade fixtures precreated profiles; the fake inventory did
not enforce the real requirement. The regression test failed before the fix with
`selected native home must exist before inventory`. Both diagnostic-context cases
also failed before the fix. Afterward, the complete installer suite passed.

Apply now uses the existing canonical-directory guard after plan revalidation and
locking, before native inventory. Preview remains read-only. New directories use
0700; existing modes and unrelated files remain untouched. A later failure keeps
the prepared profile and reports incomplete preflight. Inventory diagnostics name
the failing command without exposing subprocess output. There is no new schema,
authentication enrollment, profile copying or rollback behavior.

## Reproduction

Build the reviewed source with pinned Go, then create an empty temporary test
directory. In the designated engineering terminal run:

```text
node docs/evidence/first-use-codex/verify.mjs RUNTIME CODEX EMPTY_LAB
node docs/evidence/first-use-codex/menu.mjs RUNTIME CODEX SAME_LAB
```

The driver discovers required operations, creates a synthetic signet/binding,
asserts preview leaves state and profile absent, applies through real Codex and
requires healthy structural inspection, 0700, unchanged binding and no auth file.
The menu driver captures actual plain-mode success with a second absent profile
and a synthetic native failure with bounded diagnostic text. It retains original
output privately and emits path-sanitized output plus hashes; never publish raw
receipts. Neither driver retries into previous partial state.

Verified environment: macOS arm64, Go 1.26.4, Node 24.1.0, Codex 0.154.0 native
SHA-256 `4f85982624b3898c8991cb80c0981b2aa71070e3537046c9a95950318a95afcc`.
The PR review binds final-head native and CI receipts. The initial native run
passed verified install, healthy inspection, preview absence, profile mode and
credential-file absence. Actual menu success and failure also passed.

## UI scope and limits

Classification: in-pattern diagnostic text, unchanged navigation/layout. The
matrix covers preview, default-No confirmation, explicit apply, first-use success
and retained preflight failure. Review actual text captures for labeled phases,
operation-specific explanation and no raw child output. Plain mode is keyboard
driven and does not rely on color. No pixel baseline, screen-reader, alternate
font/locale, color-mode, Linux-native or other-architecture claim is made.

Local CI uses the documented host fallback (container runtime unavailable), pinned
toolchains, all Pi adapter checks, race tests, vet, privacy checks and cross-builds.
Required hosted checks remain mandatory. A new retained candidate must repeat the
fresh-profile check before the affected human acceptance journey resumes.

## Plan reconciliation

Planning PR #94, exact reviewed head
`a7ff1868e7f07f3214f66559ee8e2e8957a9c788`, selected this minimal fix under ADR 0003.
No product-contract drift. Durable behavior is promoted to `docs/setup.md`;
verification/coverage gap lives here and in the regression tests. Temporary plan
removed in implementation PR #95. Original failure, human Pi successes and absence
of a whole-candidate verdict are preserved; no acceptance gate is waived.
