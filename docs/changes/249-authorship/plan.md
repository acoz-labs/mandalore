# Plan: record which client wrote it, and its MCP connection where one exists

## Issue

Refs #249 (https://github.com/acoz-labs/mandalore/issues/249). Outcome: a
write through an OAuth client records a harness that names the client, so
Claude Code and the claude.ai hosted app no longer both read as `claude.ai`.
A write also records the MCP transport session it came from, when the
transport has one. Existing authorship is untouched. Ships as minor release
2.4.0, server only; the Claude plugin changes only in version-gated README
wording.

The requester's ruling after the session probe
(https://github.com/acoz-labs/mandalore/issues/249#issuecomment-6799835409)
replaces acceptance 2 with:

- (a) on a transport that issues a session ID (the `initialize` path), a
  write carries that ID, non-empty;
- (b) on sessionless MCP `2026-07-28`, a write carries `session_id` null and
  nothing fails;
- (c) the "Who wrote it" docs (`docs/server.md` and the plugin README) say
  `session_id` identifies a transport connection where one exists, not the
  agent session, and may be null.

Acceptance 1, 3, 4 and 5 stand. Session identity on sessionless MCP, which
is what Claude Code uses today, is #258's job, not this change's.

### No signet format bump

The change fills existing fields with new values. `Authorship` has had
`harness string` and `session_id *string` since 2.1.2
(`internal/memory/memory.go:90`). The revision schema
(`internal/memory/schemas/memory-revision.schema.json`) hasn't changed since
2.1.2, and it accepts any non-empty `session_id` string (`minLength: 1`) or
null. The only code check on authorship is that actor and harness aren't
blank and the device exists (`validateAuthorship`,
`internal/memory/sourced.go:130`), identical at `v2.1.2`, `v2.2.1` and
`v2.3.0`. The harness is never used in a path or file name. Every released
reader reads a 2.4.0 write, and 2.4.0 reads every older one.

### Harness: the client's registered name

The source is the client's registered name, not the MCP client info. The
registered name is the `client_name` from the client's metadata document or
DCR registration, which the server already stores as `Client.Name` and shows
the owner on the consent page. MCP client info doesn't work here: on the
`initialize` path the server runs the MCP SDK in stateless mode
(`internal/server/tools.go:149`), which discards it before a tool runs. On
`2026-07-28` it arrives in every request's `_meta`, but either way the client
sends it unauthenticated and can change it on any request.

| Client | Harness | Example |
| --- | --- | --- |
| Metadata-document client | `<name> (document on <host>)` | `Claude Code (document on claude.ai)` |
| DCR client | `<name> (redirect to <host>)` | `Claude (redirect to claude.ai)` |
| Local client | unchanged: `local` | `local` |

The host stays in the value because the name is chosen by the client: any
document host can call itself "Claude Code", but only claude.ai can serve a
document on claude.ai. The `document on` and `redirect to` wording keeps a
document client and a DCR client on the same host apart even when they
choose the same name (acceptance 1). An OAuth client always has a stored
name: `displayName` falls back to the kind (the host) when `client_name` is
missing or blank (`oauth.go:203`, `:310`). A nameless DCR client therefore
records `claude.ai (redirect to claude.ai)`, which still differs from a
document client on the same host.

Bounds and sanitising are applied at write time to whatever name is stored,
so a name saved by an older server gets the same treatment:

- The name passes through `displayName` (`internal/server/oauth_clients.go:114`)
  again. It keeps only runes `unicode.IsPrint` accepts. That drops C0 and C1
  controls (so ESC goes, and an ANSI sequence becomes inert text), format
  characters (`Cf`, including bidi overrides and zero-width characters),
  U+2028, U+2029, U+0085 and every non-ASCII space. It then trims and cuts to
  64 bytes at a UTF-8 boundary. #223's gaps are in a different sanitiser,
  `unprintable` in `oauth_admin.go:144` (installation labels and User-Agent
  hints, which lets U+2028 and U+2029 through). This change doesn't use or
  touch it; #223 stays open for it.
- The host passes through the same filter.
- The whole value is at most 128 bytes. The suffix (` (document on <host>)`
  or ` (redirect to <host>)`) is at most 96 bytes. A host too long for that
  keeps its rightmost bytes behind `…`, so the registrable domain stays
  visible, cut at a rune boundary. The name gets the remaining bytes, at
  least 32, cut at a rune boundary.

### Session: the MCP connection's ID, where the transport has one

`session_id` is the MCP transport session ID, issued by the server. It
identifies one MCP connection, not the agent's session. The stateless server
issues none today. With this change, on the `initialize` path:

1. When a POST to a signet's MCP endpoint is an `initialize` request without
   an `Mcp-Session-Id` header, the server sets `Mcp-Session-Id` on the
   response to a fresh random value (128 bits from `crypto/rand`, base32,
   26 characters).
2. The client sends that header on every later request on the connection, as
   the MCP specification requires (2025-03-26 to 2025-11-25).
3. A write tool reads the header from the tool request (`req.Extra.Header`)
   and records it as `session_id` when it is 1–128 visible ASCII characters
   (0x21–0x7E, the specification's range). Anything else, or no header,
   records null. The write never fails over it.

On sessionless MCP `2026-07-28` there is no `initialize` and no session
header, so `session_id` is null and the write succeeds. The server keeps
accepting `2026-07-28` (the ruling). It keeps no session table and never
refuses or expires a session ID, so a server restart doesn't break a
connected client with a 404 and forced re-initialize. GET and DELETE keep
their current stateless answers (405).

The value holds no secret. It is random, not derived from the token or any
credential, and grants nothing: the bearer token alone authorizes a request,
and the server never looks the ID up. After issue the client supplies it, so
it is a label within an authenticated installation, with the same trust as
the installation's device. A client can make its own writes share or split a
connection ID, but can't change the device, actor or harness.

The connection ID is not the Claude Code session ID that the plugin's hook
already sends to `memory_context` (`"session": "${session_id}"` in
`plugins/claude/hooks/hooks.json`), and the two never match:

- The Claude Code session ID stays the same across `--resume`. The connection
  ID is new for every connection: a resumed session, a new process, or any
  re-initialize (`/mcp` reconnect, or Claude Code's reconnect after a 404 or
  a session error) gets a new one.
- Subagents in one Claude Code process share the process's connection, so
  they write with the same connection ID.
- Today Claude Code 2.1.322 uses `2026-07-28` with this server, so its writes
  record null. Step 0 below shows this.

### Plugin changes, by kind

Kind (a) is wording only and may merge on the reviewer's evidence. Kind (b)
covers hooks, `.mcp.json` or unconditional tool calls, which wait for the
requester's confirmation after their server upgrade.

- **The plugin README** (`plugins/claude/README.md`): kind (a), the lead's
  ruling in plan review round 1. The "Plugin and connector together"
  paragraph and the "Who wrote it" link are worded version-gated: "From
  server 2.4.0, Claude Code's writes record the harness `Claude Code (document
  on claude.ai)`…". That makes them true on a 2.3.0 server as well, and they
  call no tool. It says `session_id` identifies a transport connection where
  one exists, not the Claude Code session, and is null on sessionless MCP
  (#258). No `plugin.json` bump, as in 2.2.1.
- **No kind (b) change.** The hook and `.mcp.json` stay as they are, and the
  server doesn't decline `2026-07-28` to force a session (the ruling).

## Files that change

- Server authorship (`internal/server/signet.go`, `oauth_clients.go`): a
  `harness(Client)` function with the table and bounds above. `author` takes
  the session and returns `{device, owner, harness(c), session}`. Device
  labels and `server clients list` keep the kind, unchanged.
- Server MCP path (`internal/server/server.go` `serve`, `tools.go`):
  - Issue the session ID on `initialize`, peeking the body within the
    existing 64 KiB limit and restoring it for the SDK.
  - Pass the header from the tool request through `call` to `author`.
  - Add `c.Name` to the handler cache key. The cached handler's closure holds
    the client, and a refreshed document name must reach the harness (the
    per-installation cache lesson in `docs/repository.md`).
- Tests (`internal/server/*_test.go`): the Proof scenarios below, against an
  `httptest` server.
- Docs:
  - `docs/server.md` "Who wrote it" (around `:416`): the new harness values,
    `session_id` as a transport connection's ID where one exists, not the
    agent session, null on `2026-07-28`, with a link to #258. The connector
    caveat stays.
  - `docs/server.md:56`: the stale line that says a client's kind is
    "recorded as the write's harness". It should say the harness comes from
    the name and the kind.
  - The authorship sentence in `docs/signet-format.md`: what `session_id`
    holds when it is set.
  - The plugin README, as above.
  - `docs/interface.md`, if it describes authorship values.
- Release: `VERSION` to 2.4.0, `docs/releases/2.4.0.md`, and the version
  strings that 2.3.0 touched (`docs/interface.md`, README). The PR includes a
  repository-wide grep for `2.3.0`, excluding dated evidence records by path.

## Order of work

0. **Done: probe whether Claude Code sends a session header**
   (https://github.com/acoz-labs/mandalore/pull/256#issuecomment-6799807563).
   Setup: Claude Code 2.1.322 in an isolated `HOME` and `CLAUDE_CONFIG_DIR`, a
   synthetic invalid API key, a plugin copy pointed at a throwaway stub on Go
   SDK v1.7.0. With the stub accepting `2026-07-28`, as the server does,
   Claude Code sent `server/discover` and no session header on any request.
   With `server/discover` refused, it fell back to `initialize`, echoed the
   issued ID on every request, including the hook's, and got a new ID on
   `--resume` while the hook's session ID stayed the same. This led to the
   ruling above.
1. Write failing tests for acceptance 1, 2(a) and 4 against the current
   server. They fail because the harness is `claude.ai` and `session_id` is
   null.
2. Add the harness function and use it in `author`; acceptance 1 and 4 pass.
3. Issue the session ID on `initialize` and record it on writes; acceptance
   2(a) and 2(b) pass, with the restart and malformed-header cases.
4. Add the name to the handler cache key, with the refreshed-name test.
5. Add the pre-change read test (acceptance 3).
6. Write the docs and release notes, bump the version, run `bin/ci` and
   govulncheck.

## Risks

- **Readers that match `harness == "claude.ai"`.** Nothing in this
  repository does, because the harness is display data, but an owner's own
  scripts might. The release notes give the old and new values. Old records
  keep `claude.ai`.
- **Reading `session_id` as the agent session.** It is a connection ID,
  null for Claude Code today. The docs and release notes say so and point to
  #258.
- **A client that reacts badly to a session ID from a stateless server.**
  In the probe, Claude Code on the `initialize` path opened a GET stream and
  carried on; a stateless GET gets 405, which the specification allows.
  Tested with the SDK client.
- **A hostile client name** in the owner's terminal or a reader's output.
  The sanitiser and bounds above contain it, and the tests use
  attacker-shaped names.
- **Caching.** A handler built before a document's name changed would keep
  the old name; the cache key change contains this.

## Proof

These tests run on a synthetic signet against a running `httptest` server,
with `GOFLAGS=-count=1`:

1. **Harness (acceptance 1).** A metadata-document client named "Synthetic
   Agent", with its document on a synthetic host, saves a record and gets the
   harness `Synthetic Agent (document on <host>)`. A DCR client with a
   redirect on the same host saves one and gets `… (redirect to <host>)`. The
   values differ, also when both use the same name and when the DCR client
   has no name.
2. **Session (acceptance 2).**
   - (a) Two SDK client connections on the `initialize` path (protocol
     `2025-11-25`), from one installation, each save a record and append a
     journal entry. That is four writes, with two distinct non-empty
     `session_id` values, each matching its connection's issued
     `Mcp-Session-Id`, and one device ID. A scope label and a move record the
     ID too.
   - (b) A client on `2026-07-28` saves a record and appends a journal entry.
     Both succeed with `session_id` null, and the response carries no
     `Mcp-Session-Id`.
   - In both cases, `memory_history`, `memory_journal` and `memory_prime`
     return these values where they return authorship.
   - A `memory_context` call carrying a hook session value records nothing
     from that value. A reconnect, meaning a new `initialize` from the same
     installation, gets a different connection ID.
   - (c) Docs review: the reviewer checks the "Who wrote it" wording in
     `docs/server.md` and the plugin README against the ruling.
3. **Existing authorship (acceptance 3).** A record and a journal entry
   written by the current code (harness `claude.ai`, `session_id` null), read
   after the change, return their authorship byte for byte as stored.
4. **Sanitising and bounds (acceptance 4).**
   - Names with C0 and C1 controls, an ANSI sequence, bidi overrides,
     U+2028, U+2029, U+0085, and 300 bytes of multibyte runes record a
     harness with none of them. It is at most 128 bytes, valid UTF-8, with
     the suffix intact.
   - A 200-byte host records a suffix of at most 96 bytes that ends in the
     host's rightmost bytes behind `…`.
5. **Failure paths.**
   - A session header that is 129 bytes, contains a space or a control, or is
     missing records null, and the write succeeds.
   - A request after a server restart that carries the old ID still writes,
     with that ID.
   - A read-only client gets an ID and can't write.
   - GET and DELETE answer as before.
6. **Checks (acceptance 5).** `bin/ci` (with the TTY fallback in
   `docs/repository.md`), `bin/sdlc check` and
   `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`.
   `bin/check-old-reader` isn't required, since nothing changes in the
   layout, intents or sync paths, but I'll run it anyway and post the result.

For the independent verifier, on the real binary: run it locally with a
synthetic signet, connect real Claude Code with the plugin in an isolated
`HOME` and `CLAUDE_CONFIG_DIR`, save a record, and read its authorship with
`memory_history`. Expect the harness `Claude Code (document on claude.ai)`
and `session_id` null (Claude Code uses `2026-07-28`). Then connect the Go
SDK client on the `initialize` path and save again; expect a non-empty
`session_id` equal to its issued `Mcp-Session-Id`.

## Rollback

Revert the PR, or reinstall 2.3.0. Records written by 2.4.0 keep their
harness and session values and stay valid for 2.3.0 and every older reader,
since the fields and validation are the same. New writes go back to the host
harness and a null session. Nothing changes in the state file, OAuth state,
configuration, signet format or public paths, so nothing needs migrating in
either direction. The Cloudflare path allowlist is unchanged: the change adds
a response header, not a path.
