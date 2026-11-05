# Plan: memory_remember reports input-limit failures as input.invalid

## Issue

Refs #243 (https://github.com/acoz-labs/mandalore/issues/243). Outcome:
`memory_remember` tells the caller which input it must fix, with
`write_may_have_occurred: false`, instead of a `save.incomplete` that invites
retrying input that can never succeed. Ships as patch 2.1.2.

### Decisions the plan review asked for

- **Characters versus bytes.** JSON Schema `maxLength` counts characters,
  meaning Unicode code points (2020-12 validation, section 6.3.1, which defers
  to RFC 8259). The server counts UTF-8 bytes, and a code point is 1 to 4 bytes.
  The schema sets `maxLength` to the byte limit (256, 8192, 1024). Every string
  within the byte limit is therefore within `maxLength`, so a validating client
  never refuses input the server would accept; the reverse gap (non-ASCII text
  under `maxLength` but over the byte limit) is closed by the server's
  `input.invalid`. Each field description says so, for example the summary:
  "1–256 UTF-8 bytes, not blank. The server counts bytes; maxLength counts
  characters, so non-ASCII text can be refused below it." Body and reason say
  the same with 8192 and 1024. The MCP SDK's low-level `Server.AddTool`, which
  the server uses, does not validate arguments against the schema, so an
  over-limit call still reaches the handler and gets the envelope below; a
  test proves it.
- **Existing clients.** The envelope shape doesn't change: the same `error`
  object with `code`, `message`, `retryable`, `write_may_have_occurred` and
  `inspect_before_retry`. `memory_remember` already returns `input.invalid`
  (malformed arguments, bad `request_id`), so every client already handles the
  code. For the refused inputs, `retryable` stays false and
  `write_may_have_occurred` and `inspect_before_retry` go from true to false. A
  client that inspected before retrying just skips the inspection; a client
  that retried on `save.incomplete` now stops, which is the point. Nothing was
  ever stored for these inputs (validation runs before the request's intent
  file is written), so a client that retried under an older server finds no
  leftover state, and a later corrected save with the same `request_id`
  succeeds. `save.incomplete` keeps its exact text and fields for every
  failure that can follow a write.
- **Claude plugin.** No plugin change. The plugin's skills and README name
  neither the limits nor `save.incomplete`; the limits reach Claude through the
  server's tool schema, so the guidance stays correct for 2.1.1 and 2.1.2
  servers alike. `plugins/claude/.claude-plugin/plugin.json` stays at 2.1.1, so
  merging rolls nothing out to installs and #229's owner-confirmation rule
  doesn't apply. If `bin/ci` turns out to require `plugin.json` to match
  `VERSION`, I stop and tell the lead before bumping it.

## Files that change

- **Input errors in the memory service** (`internal/memory/service.go`, a new
  small `internal/memory/input_error.go`). A sentinel `ErrInvalidInput`, and
  errors that wrap it with a safe message that never echoes input text. The
  remember checks in `prepareRemember` split into one check per field. A field
  over its limit says, for example, "summary is 363 bytes; the limit is 256
  UTF-8 bytes". A blank field or one containing NUL says "summary is required:
  1–256 UTF-8 bytes, not blank, no NUL". Each of these wraps the sentinel:
  `validateScope` when it runs for a write, more than 32 `supersedes`, and the
  `record_id`/`supersedes` pairing rule. The journal's kind (64) and summary
  (4096) checks in `AppendJournalIdempotent` wrap it too, because both tools
  share the same failure mapping.
- **The pre-write graph checks** (`internal/memory/sourced.go`). Before
  validating the whole graph, the new revision alone is checked against the
  revision schema. A failure there is the caller's (bad `kind`, `basis`,
  `sensitivity`, `volatility`, `confidence` or an ID pattern), and the message
  names the field from the schema error's location, not its value. When the
  whole-graph check fails, the existing graph is checked on its own, only on
  that failure path, as the CLI classifier already does. If the existing graph
  is valid, the new revision caused the failure (a correction whose `kind` or
  `scope` differs from the original, or that supersedes a revision that isn't
  there), and it becomes an input error with a fixed message pointing to
  `memory_history`. If the existing graph is invalid, the error stays what it
  is today. Successful writes do no extra work.
- **The MCP failure mapping** (`internal/server/tools.go`, `saveFailure`).
  `ErrInvalidInput` maps to `input.invalid` with the error's message and
  `write_may_have_occurred: false`. `memory.ErrStaleHeads` (a correction's
  `supersedes` no longer matches the record's current heads, checked before any
  write) maps to `memory.stale_heads`, the code the visibility tools already
  return for the same condition, also with `write_may_have_occurred: false`.
  That code isn't literally `input.invalid`, but it tells the caller to reread
  history rather than to change its input, and it matches the visibility
  tools. Everything else, including I/O and store failures, stays
  `save.incomplete`, unchanged.
- **The served tool schema** (`internal/server/tools.go`). After
  `memory_remember`'s schema is generated, `minLength` 1 and `maxLength` are set
  on `record.summary`, `record.body` and `record.reason`, and the descriptions
  above are added. `memory_journal_append`'s `entry.kind` and `entry.summary`
  get the same treatment (64 and 4096), since they have the same error and the
  same gap. The tags in `memory.Write` stay description-only;
  `jsonschema-go` has no tag for limits.
- **The CLI classifier** (`internal/api/api.go`). `ErrInvalidInput` maps to
  `input.invalid` with its message. The CLI already returns `input.invalid` for
  these inputs, but with a generic message; this gives it the field, limit and
  size.
- **Tests** in `internal/memory`, `internal/server` and `internal/api` (see
  Proof).
- **Docs.** `docs/server.md`, next to the `request_id` paragraph, gains the
  remember and journal limits (bytes, with the `maxLength` note) and what each
  save error means: `input.invalid` and `memory.stale_heads` (nothing written;
  fix the input or reread history), `request.conflict`, `store.busy`, and
  `save.incomplete` (a write may have happened; retry only identical input with
  the same `request_id`). `docs/interface.md`'s failure table gains rows for
  `save.incomplete`, `request.conflict` and `memory.stale_heads`, which it
  doesn't list today. `docs/signet-format.md` keeps its limits and adds that
  over-limit service writes are refused as `input.invalid`.
- **Release preparation.** `VERSION` goes to 2.1.2 and
  `docs/releases/2.1.2.md` follows the 2.1.1 notes: what changed, that no
  plugin update is needed, rollout and rollback. README, deployment and install
  pointers move to 2.1.2 in the post-publication contributor PR, as
  `docs/repository.md` requires.

## Order of work

1. This plan, alone, as the first commit; draft PR; plan review.
2. Failing tests first: a 257-byte summary through `memory_remember` expects
   `input.invalid`, and today gets `save.incomplete`.
3. The memory-service input errors and per-field checks, and the
   `saveFailure` and CLI mappings. The tests from step 2 pass.
4. The single-revision schema check and the failure-path graph recheck, with
   their tests, including the corrupt-store case that must not be blamed on
   the caller.
5. Schema limits and descriptions, with a `tools/list` test.
6. Docs, `VERSION` and release notes; `bin/ci`, `bin/sdlc check` and
   `govulncheck`; simplify pass; mark the PR ready.

## Risks

- **Blaming the caller for a broken store.** A whole-graph failure is an input
  error only when the existing graph has just passed on its own, and a test
  with a corrupt stored revision proves a valid save still gets
  `save.incomplete`, not `input.invalid`.
- **Echoing private text.** Input-error messages carry field names, limits and
  sizes, never values; the schema check reports the field location only. A
  test asserts that a refused body's text doesn't appear in the response.
- **Clients that branch on `save.incomplete`.** Covered above: same envelope,
  a code they already handle, and no state left behind.
- **Schema drift.** The limits live in the service checks and in the schema
  patch. A test reads both from the served `tools/list` and from the service
  (accept at the limit, refuse one byte over), so a change to one without the
  other fails `bin/ci`.
- **Mixed versions.** No state-file fields and no plugin change; a 2.1.1
  server keeps its old behaviour for the same inputs.

## Proof

Tests, run with `GOFLAGS=-count=1`:

- For each of summary (256), body (8192) and reason (1024) through
  `memory_remember` over MCP: exactly the limit is accepted, and one byte over
  returns `input.invalid` with `retryable` false, `write_may_have_occurred`
  false, and a message naming the field, the limit and the size received. No
  revision, source or intent file is created, and a corrected save with the
  same `request_id` then succeeds.
- A multibyte summary: 128 two-byte characters (256 bytes) is accepted; 129
  (258 bytes, still under `maxLength` 256) is refused, which shows the bytes
  versus characters gap the description states.
- Blank and NUL-containing fields, more than 32 `supersedes`, `record_id`
  without `supersedes`, a bad scope and a bad `kind` return `input.invalid`
  with no write.
- A correction that changes `kind`, or supersedes a revision that doesn't
  exist, returns `input.invalid`; a correction against stale heads on a
  format 2 signet returns `memory.stale_heads`; neither writes.
- A corrupt stored revision plus a valid save still returns `save.incomplete`.
- A real mid-write failure (the sources directory made read-only after the
  intent is written) still returns `save.incomplete` with
  `write_may_have_occurred` true, and the same `request_id` succeeds after the
  directory is restored.
- The journal's kind and summary limits return `input.invalid`.
- `tools/list` shows `maxLength` and `minLength` on the five fields and the
  byte note in each description.
- The CLI `memory remember` with a 257-byte summary exits 2 with the field
  message.

Repository checks: `bin/container bin/ci` (or `bin/ci`), `bin/sdlc check`, and
`go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`.

For the independent verifier, on the built 2.1.2 binary: start `mandalore
serve` on a scratch signet, call `memory_remember` over MCP with the issue's
363-byte summary and see `input.invalid` naming summary, 256 and 363 with no
new files in the signet; save the same record with a 256-byte summary and see
a durable receipt; check `tools/list` for the limits; and repeat the 363-byte
call against a 2.1.1 binary to confirm the old `save.incomplete` it replaces.

## Rollback

Reinstall 2.1.1 (`sh install.sh 2.1.1`, or the bare-binary atomic swap). No
data, state-file or signet-format change is made, so rolling back needs no
migration, and records saved by 2.1.2 read the same under 2.1.1. Clients go
back to seeing `save.incomplete` for over-limit input. No plugin update is
involved either way.
