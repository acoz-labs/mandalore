# Plan: record which client and which session wrote it

## Issue

Refs #249 (https://github.com/acoz-labs/mandalore/issues/249). Outcome: a
write through an OAuth client records a harness that names the client, so
Claude Code and the claude.ai hosted app no longer both read as `claude.ai`,
and a write records the MCP transport session it came from. Existing
authorship is untouched. Ships as minor release 2.4.0, server only; the Claude
plugin's code and version don't change.

### The three answers the brief asks for

**No signet format bump.** The change fills existing fields with new values.
`Authorship` has had `harness string` and `session_id *string` since 2.1.2
(`internal/memory/memory.go:90`), and the only check on authorship at 2.1.2,
2.2.1 and main is that actor and harness aren't blank and the device exists
(`validateAuthorship`, `internal/memory/sourced.go:130`, identical at
`v2.1.2` and `v2.2.1`). No schema, file, directory or manifest changes, so
every released reader reads a 2.4.0 write, and a 2.4.0 reader reads every
older one.

**Harness source: the client's registered name, not MCP `initialize`
`clientInfo`.** The server runs the MCP SDK in stateless mode
(`internal/server/tools.go:149`, `Stateless: true`), so each request gets a
temporary session with default initialization parameters: the `initialize`
`clientInfo` is gone by the time a tool is called. It is also unauthenticated
and sent fresh by any process holding the token. The registered name is the
`client_name` from the client's metadata document or DCR registration, which
the server already stores as `Client.Name` and shows the owner on the consent
page. The harness becomes:

| Client | Harness | Example |
| --- | --- | --- |
| Metadata-document client | `<name> (document on <host>)` | `Claude Code (document on claude.ai)` |
| DCR client | `<name> (redirect to <host>)` | `Claude (redirect to claude.ai)` |
| Local client, or no stored name | unchanged: the client kind | `local` |

The host stays in the value because the name is client-chosen: any document
host can call itself "Claude Code", but only claude.ai can serve a document
on claude.ai. The `document on` / `redirect to` wording keeps a document
client and a DCR client on the same host apart even if they pick the same
name (acceptance 1).

Bounding and sanitising, applied at write time to whatever is stored, so a
name saved by an older server gets the same treatment:

- The name passes through `displayName` (`internal/server/oauth_clients.go:114`)
  again: it keeps only runes `unicode.IsPrint` accepts, which drops C0 and C1
  controls (so ESC, which makes ANSI sequences inert), format characters
  (`Cf`, including bidi overrides and zero-width joiners), U+2028, U+2029,
  U+0085 and every non-ASCII space, then trims and cuts to 64 bytes at a UTF-8
  boundary. #223's gaps are in a different sanitiser, `unprintable` in
  `oauth_admin.go:144` (installation labels and User-Agent hints, which allows
  U+2028 and U+2029). This change doesn't use or touch that one; #223 stays
  open for it.
- The host passes through the same filter, so an internationalised host can't
  carry controls either.
- The whole value is capped at 128 bytes, cutting the name, never the host
  suffix, and never mid-rune. A blank name after sanitising falls back to the
  kind, as today.

**`session_id` is the MCP transport session ID, issued by the server.** The
stateless server issues none today, so there is nothing to record. The change
makes the server issue one, staying stateless:

1. When a POST to a signet's MCP endpoint is an `initialize` request without
   an `Mcp-Session-Id` header, the server sets `Mcp-Session-Id` on the
   response to a fresh random value (128 bits from `crypto/rand`, base32,
   26 characters).
2. A streamable HTTP client includes that header on every later request on
   that connection, as the MCP specification requires (2025-03-26 and later).
3. A write tool reads the header from the request (`req.Extra.Header` in the
   SDK's tool request) and records it as `session_id` when it is 1–128
   visible ASCII characters (0x21–0x7E, the specification's range). Anything
   else, or no header, records `null`, as today; the write never fails over
   it.

The server keeps no session table and never refuses or expires a session ID,
so a server restart doesn't break a connected client (no 404 and forced
re-initialize), and GET and DELETE keep their current stateless answers.

The value holds no secret. It is random, isn't derived from the token or any
credential, and grants nothing: the bearer token alone authorizes a request,
and the server never looks the ID up. It is client-supplied after issue, so
it is a label within an authenticated installation, the same trust as the
installation's device: a client can make its own writes share or split a
session, but can't change the device, actor or harness.

What one session is: one MCP connection. In Claude Code that is one Claude
Code process, so two sessions are two processes and get two IDs, and a
subagent writes with its parent's ID. A client on the coming sessionless
protocol (`2026-07-28`, which drops `initialize`) gets no ID and records
`null`; the requirement covers only a transport that gives a session.

### Plugin changes, by kind

The brief's two kinds: (a) wording-only, may merge on the reviewer's
evidence; (b) hooks, `.mcp.json` or unconditional tool calls, which wait for
the requester's confirmation after their server upgrade.

- **The plugin README's "Plugin and connector together" paragraph**
  (`plugins/claude/README.md`): wording only. It says Claude Code's writes
  carry their own harness and session. Kind (a), with one difference to
  name: the rule's example is wording gated on a tool the server lists, and
  this paragraph is gated on nothing. It is inert on any server: Claude Code
  doesn't load a plugin's README into a session, and it calls no tool. No
  `plugin.json` bump, as in 2.2.1, so installs don't update for it.
- **No kind (b) change.** Requirement 2 is met in the server: Claude Code
  speaks streamable HTTP, whose clients must echo the session header, so the hook
  and `.mcp.json` stay as they are. A plugin-side alternative was rejected:
  Claude Code's `.mcp.json` has no session variable to put in a header, and a
  tool argument would be model-supplied.

If the live check (Proof, last item) finds Claude Code doesn't echo the
header, requirement 2 would need a kind (b) change. I'll stop and report
before building one.

## Files that change

- Server authorship (`internal/server/signet.go`, `oauth_clients.go`): a
  `harness(Client)` function with the table and bounds above; `author` takes
  the session and returns `{device, owner, harness(c), session}`. Device
  labels and `server clients list` keep the kind, unchanged.
- Server MCP path (`internal/server/server.go` `serve`, `tools.go`): issue the
  session ID on `initialize` (peeking the body within the existing 64 KiB
  limit and restoring it for the SDK); pass the header from the tool request
  through `call` to `author`; add `c.Name` to the handler cache key, since the
  cached handler's closure holds the client and a refreshed document name
  must reach the harness (the per-installation cache lesson in
  `docs/repository.md`).
- Tests (`internal/server/*_test.go`): the Proof scenarios below, against an
  `httptest` server with the SDK's streamable client.
- Docs: "Who wrote it" in `docs/server.md` (new harness values, session,
  what the session is and isn't, connector caveat kept); the authorship
  sentence in `docs/signet-format.md` (what `session_id` holds when set);
  the plugin README paragraph above; `docs/interface.md` if it describes
  authorship values.
- Release: `VERSION` to 2.4.0, `docs/releases/2.4.0.md`, and the version
  strings 2.3.0 touched (`docs/interface.md`, README), with the
  repository-wide grep for `2.3.0` in the PR, excluding dated evidence
  records by path.

## Order of work

1. Failing tests for acceptance 1, 2 and 4 against the current server (they
   fail: harness is `claude.ai`, `session_id` is null).
2. Harness function and its use in `author`; acceptance 1 and 4 pass.
3. Session issue on `initialize` and recording on writes; acceptance 2
   passes, with the restart and malformed-header cases.
4. Handler cache key with the name; the refreshed-name test.
5. Pre-change read test (acceptance 3).
6. Docs and release notes, version bump, `bin/ci`, govulncheck.

## Risks

- **Readers that match `harness == "claude.ai"`.** Nothing in this repository
  does (the harness is display data), but an owner's own scripts might. The
  release notes give the old and new values. Old records keep `claude.ai`.
- **A client that reacts badly to a session ID from a stateless server**,
  for example by opening a GET stream (405, which the specification allows
  and clients handle) or sending DELETE on exit (the SDK's stateless DELETE
  answer, unchanged). Tested with the SDK client; the live check covers
  Claude Code.
- **A hostile client name** in an owner's terminal or a reader's output:
  contained by the sanitiser and bounds above, tested with attacker-shaped
  names.
- **Session IDs grow signets slightly**: 26 bytes per write.
- **Caching**: a handler built before a document's name changed would keep
  the old name; the cache key change contains it.

## Proof

Tests, on a synthetic signet against a running `httptest` server, with
`GOFLAGS=-count=1`:

1. A metadata-document client named "Synthetic Agent" with its document on a
   synthetic host saves a record: harness `Synthetic Agent (document on
   <host>)`. A DCR client with a redirect on the same host saves one:
   `… (redirect to <host>)`. The values differ, also when both use the same
   name.
2. Two SDK client connections of one installation each save a record and
   append a journal entry: four writes, two distinct non-empty `session_id`
   values, one per connection, one device ID. `memory_history`,
   `memory_journal` and `memory_prime` return them where they return
   authorship. A scope label and a move record them too.
3. A record and journal entry written by the current code (harness
   `claude.ai`, `session_id` null), read after the change, return authorship
   byte for byte as stored.
4. Names with C0 and C1 controls, an ANSI sequence, bidi overrides, U+2028,
   U+2029, U+0085 and 300 bytes record a harness with none of them, at most
   128 bytes, valid UTF-8, host suffix intact.
5. Failure paths: a session header of 129 bytes, with a space or a control,
   or absent, records `null` and the write succeeds; a request after a server
   restart with the old ID still writes, with that ID; a read-only client gets
   an ID and no write; GET and DELETE answer as before.
6. `bin/ci` (TTY fallback as `docs/repository.md` says), `bin/sdlc check`
   and `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`.
   `bin/check-old-reader` isn't required (no layout, intent or sync change);
   I'll run it anyway and post the result.

For the independent verifier, on the real binary: run it locally with a
synthetic signet, connect real Claude Code with the plugin in an isolated
`HOME` and `CLAUDE_CONFIG_DIR`, save from two Claude Code sessions, and read
both writes' authorship with `memory_history`: harness `Claude Code
(document on claude.ai)` and two different session IDs.

## Rollback

Revert the PR, or reinstall 2.3.0. Records written by 2.4.0 keep their
harness and session values and stay valid for 2.3.0 and every older reader
(same fields, same validation); new writes go back to the host harness and a
null session. No state file, OAuth state, configuration, signet format or
public path changes, so nothing needs migrating either way, and the
Cloudflare path allowlist is unchanged (the change adds a response header,
not a path).
