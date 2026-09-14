# Native legacy-name coexistence refusal

Contributor-operated verification on 2026-09-14, not independent acceptance.
The unchanged retained candidate is source `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`,
manifest `c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`,
macOS ARM64 binary `eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
The selected native Codex 0.153.4 binary hashes to
`b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.

## Fixture and actual operations

The existing isolated installer-test profile initially contained the verified
`mandalore@mandalore` connection, generation
`cb064c23a068f377bffb2ec995bec1c0ec3d72f307ed841883b1442ac81c7b4a`.
No legacy plugin or personal-named marketplace was registered in that profile.
The plugin-creator scaffold produced a separate local test marketplace and an
inert `my-friday-memory@personal` plugin. Its [manifest](fixture-plugin.json)
explicitly identifies it as a synthetic name-only fixture. It contains no skills,
hooks, scripts, MCP server, credentials or predecessor implementation.

The scaffold validator initially lacked PyYAML and stopped before registration.
An isolated temporary Python environment with PyYAML 6.0.2 supplied validation;
no global Python or shell configuration changed. Validation then passed.

All native commands and candidate checks ran in the designated testing pane.
Using the isolated profile, actual `codex plugin marketplace add` and
`codex plugin add my-friday-memory@personal --json` registered the fixture.
No model session ran while both plugins were enabled.

1. Actual native listing and retained `migration preflight` identified both
   enabled names as two **potential** writers; see [inventory](inventory.json).
2. A read-only `connection plan` pinned the candidate, binding and native binary.
   Applying it returned `connection.failed`, `installed: false`, phase `preflight`,
   with the explicit competing-integration message in [refusal](refusal.json).
3. Sorted SHA-256 inventories of every bank and managed-state file matched the
   pre-test inventory. The native configuration also matched its hash taken
   after explicit fixture registration and before refused application.
4. The driver initially expected exit 2, but documented `connection.failed` is
   exit 1. After inspecting the refusal and proving preserved state, an explicit
   retry recorded exit 1, the identical response bytes and unchanged data.
   This was a corrected test expectation, not a product repair or blind retry.
5. Native `plugin remove my-friday-memory@personal --json` and
   `plugin marketplace remove personal --json` removed only the newly owned
   registration/cache. The synthetic fixture source remains retained locally.
   Installed-plugin and marketplace JSON matched their original values exactly.
6. [Recovered inventory](recovered-inventory.json) returned one potential writer.
   [Doctor](doctor.json) was healthy, with 12 passing and five not-tested checks.
   Every original bank/managed-state file and the binding still matched its
   pre-test hash. No migration output was published or new connection activated.

## Evidence limits

Inventory files are exact projections of native-backed preflight results.
The refusal projection omits machine paths but preserves the actual code,
message, phase, installed flag and conservative ambiguity fields. Doctor retains
check names/statuses, not path-bearing details. These are explicitly projections,
not raw transcripts or a claim that the parent API returned their reduced shape.

This proves real native registration, name-based collision detection, refusal
before connection replacement, preservation and explicit cleanup. It does not
test an actual old implementation running beside the new one, active sessions,
same-bank targeting, every legacy name, remote writer quiescence, or authentication.
An external native install can enable a competing plugin independently; the guard
refuses a subsequent Mandalore connection operation, not every possible native
configuration change. Native logs/caches are not claimed byte-identical.

The conservative `write_may_have_occurred: true` response is preserved despite
this test independently proving no persistent managed/data changes. No source
fix, rebuilt candidate, private-memory adoption or release occurred.

Receipt projections were compared to their local originals, preserved-file
inventories compared across refusal/retry/cleanup, and public-content checks
passed. An actual container-first CI attempt failed because Docker was unavailable;
the documented pinned Go 1.26.4 host fallback passed the full CI suite. Go reused
unchanged test results where applicable; this documentation-only checkpoint does
not relabel those source checks as additional retained-candidate executions.
