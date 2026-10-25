# Mandalore server

`mandalore serve --config FILE` is Mandalore 2.0's single-writer memory server.
It owns one or more signets and serves MCP over Streamable HTTP. Agents never
write a signet's Git repository; the server does, and pushes each signet to its
remote as a backup. The [2.0 design](design/2.0.md) explains why.

The server includes its own OAuth 2.1 authorization server with owner passkey
approval (#161). Foundlings on the server come with #162 and the Claude plugin
with #163. The cutover from 1.x clones is in [migration-2.0.md](migration-2.0.md).
[Running it](#running-it) covers launchd and Docker; the deployment guides cover
a [work laptop on loopback](deploy/local.md) and a
[home server behind Cloudflare Tunnel](deploy/cloudflare-tunnel.md).

## Configuration

The file is strict JSON: unknown, duplicate or mistyped fields are refused.

```json
{
  "listen": "127.0.0.1:8730",
  "public_url": "https://memory.example.com",
  "state_dir": "/var/lib/mandalore/state",
  "owner": {"name": "Owner"},
  "signets": [
    {"slug": "primary", "root": "/var/lib/mandalore/signets/primary",
     "signet_id": "signet-example", "device_id": "device-server",
     "remote": "git@github.com:owner/signet.git"}
  ]
}
```

| Field | Meaning |
| --- | --- |
| `listen` | Host and port to listen on. |
| `public_url` | The origin clients use, such as `https://memory.example.com`. HTTPS unless loopback. It is the OAuth issuer and the passkey origin, and sets the Host and Origin allowlist (plus loopback names on the listen port). With `oauth` it must be a host name, not an IP address, since passkeys bind to names: use `http://localhost:PORT` on loopback. |
| `state_dir` | Server state: OAuth clients, grants, token hashes and owner passkeys (`oauth.json`), and per signet the client device enrollments and backup state. Never inside a signet. |
| `owner.name` | The actor recorded on every write, and the passkey's display name. |
| `auth` | `oauth` (the default when omitted), or `loopback-trust`. `loopback-trust` is refused unless `listen` and `public_url` are loopback. Test builds also accept `insecure-test`. |
| `local_scope` | `read` or `read-write`, the scope of the `local` client under `loopback-trust`. |
| `test_clients` | `insecure-test` only: `id`, `kind`, `scope`, bearer `token` (16+ bytes) and the `signets` slugs it may use. |
| `signets[]` | 1–16 to serve; `server migrate init` also accepts an empty list and adds the entry. `slug`, absolute `root` (an existing Git checkout on `main`), its `signet_id`, the server's enrolled `device_id` in it, and the backup `remote`. |

One signet serves at `/mcp`. Several serve at `/s/<slug>/mcp`. Each signet has its
own lock, memory service, Git backup and client authenticator; a request to one
signet never touches another. `/healthz` answers `GET` from loopback peers only.

### Authentication

Each signet's client comes from one authenticator. A `Client` has an ID, a name
(its device label), a kind (recorded as the write's harness) and a scope.

- `oauth` (default): OAuth 2.1 bearer tokens from the built-in authorization
  server, below. A missing, unknown, expired or other-signet token gets `401`
  with `WWW-Authenticate: Bearer resource_metadata="..."`.
- `loopback-trust`: a direct loopback peer that addresses the server as
  `127.0.0.1` or `localhost` is the client `local`, with `local_scope`. Any local
  process can use it. It is refused with a non-loopback `public_url`, and a
  request carrying proxy forwarding headers (`Forwarded`, `X-Forwarded-*`,
  `X-Real-IP`, `CF-Connecting-IP`, `CF-Ray`, `True-Client-IP`, `X-Client-IP`,
  `Via`) gets `401`, so a same-host proxy or tunnel that rewrites Host can't relay
  internet traffic as the local client.
- `insecure-test`: static bearer tokens from `test_clients`. It exists only in
  the server package's test builds; a release binary refuses it at config load.

## OAuth

The server is its own authorization server; it depends on no outside identity
provider. All endpoints are on `public_url`:

| Endpoint | Purpose |
| --- | --- |
| `/.well-known/oauth-protected-resource[/s/<slug>/mcp]` | RFC 9728 metadata per signet: `resource` is the signet's MCP URL, the only authorization server is this one, scopes `memory:read` and `memory:write`. One signet answers at the bare path (and at `/.well-known/oauth-protected-resource/mcp`). |
| `/.well-known/oauth-authorization-server` | RFC 8414 metadata: S256 PKCE only, public clients (`none`), client ID metadata documents, `/register`, and `offline_access`. |
| `POST /register` | RFC 7591 dynamic registration (JSON). The client is pending, with no access, until the owner approves it. At most 20 pending; a pending client expires after 24 hours; 10 registrations per peer address per 10 minutes. Behind a tunnel every request shares the tunnel's address. |
| `GET /authorize` | Authorization code request. |
| `POST /consent` | The owner's approve or deny decision. |
| `POST /token` | Form-encoded `authorization_code` and `refresh_token` grants. RFC 6749 error codes. |
| `/owner/*` | Passkey enrollment and sign-in. |

**Clients.** A `client_id` is either a DCR registration or an HTTPS URL naming a
client ID metadata document (such as Claude Code's). The document is fetched at
authorization time only: HTTPS, no proxy, no redirects, 5 s timeout, 16 KiB cap,
cached for 10 minutes, and its `client_id` must equal its URL. The connection
address is checked after DNS resolution, so a name that resolves to a loopback,
private, link-local, shared, documentation or other non-public address is refused
before connecting. Its `redirect_uris` are enforced.

**Authorization.** PKCE S256 is required; `plain` or a missing challenge is
refused. `resource` is required and must be one of the signets' MCP URLs. A
redirect URI must exactly match one the client registered, except that a
registered `http://localhost/...` or `http://127.0.0.1/...` matches any port.
A bad client or redirect shows an error page and never redirects. Other errors
return to the client's redirect. Codes last 2 minutes and are single-use; a
replayed code revokes the tokens it issued.

**Scopes.** The owner grants each client `read` or `read-write` per signet. A
token carries the requested scope narrowed by the grant: a read grant never
yields write tools, whatever is requested, and `set-scope` narrows live tokens
at once. No `scope` requested means the full grant.

**Tokens.** Opaque random strings; `state_dir` holds only their SHA-256. Access
tokens last 1 hour and are bound to one signet (RFC 8707 audience). Refresh
tokens last 30 days from last use and rotate on every use; replaying a rotated
refresh token revokes its whole family. The token endpoint reads and writes
local state only, with no network call. The server and the admin CLI share the
state file under a file lock, and the server rereads it when it changes, so a
revoke takes effect on the next request.

### Owner approval

There is one owner, who signs in with a passkey (WebAuthn, through
`github.com/go-webauthn/webauthn`, pinned in `go.mod`).

1. On the server, `mandalore server owner enroll --config FILE` prints a
   one-time link, valid for 10 minutes. The secret is in the URL fragment, which
   browsers don't send to the server or any proxy. Opening the link registers a
   passkey for `public_url`'s host name and signs the browser in. Each
   enrollment adds a passkey; `--replace` makes the new one the only one and ends
   existing sessions, which is how a lost passkey is recovered. Shell access to
   the server is the root of trust.
2. The owner session is a `__Host-` cookie (`Secure`, `HttpOnly`,
   `SameSite=Lax`, 12 hours) held in server memory, so a restart signs the owner
   out. Consent forms carry the session's CSRF token; a POST without the cookie
   or the token, or from another origin, is refused.
3. Every authorization needs the owner session; without one the browser gets
   the sign-in page. A new client's first authorization for a signet then shows
   the consent page: the client's name, its redirect host in large text, the
   signet, a read or read-write picker, and approve or deny. A loopback redirect
   gets a warning, since any local process can claim a loopback port. Owner pages
   can't be framed and run only their own nonce-tagged script.
4. On approval the server stores the grant and enrolls the client's device
   record in the signet, labelled `<client name> (<kind>), approved <date>`. The
   kind is the host of its metadata document or of its first non-loopback
   redirect, such as `claude.ai`. Later authorizations by an approved client for
   that signet skip the consent page; the owner still signs in, or uses a live
   session. A client ID is no secret, so no code is ever issued without the
   owner. For a metadata-document client the page also names the host that
   publishes the document, since the client chooses its own name.

Browsers treat `http://localhost` as a secure context, so passkeys and the
`Secure` cookie work there; an IP address can't hold a passkey.

## Tools

Read scope gets `memory_scopes`, `memory_recall`, `memory_history`,
`memory_journal`, `memory_visibility_history` and `memory_context`. Read-write
adds `memory_remember`, `memory_journal_append`, `memory_withdraw` and
`memory_restore`. Read clients never see write tools, and the call path refuses
writes by scope as well. Server instructions are under 600 bytes and differ for
read and read-write clients.

`memory_remember` and `memory_journal_append` take a required `request_id`
(16–128 characters). A retry with identical input returns the original receipt;
the same key with different input is refused with `request.conflict`. Keys are
namespaced by client.

`memory_context` (`event`: `start` or `prompt`, optional `prompt`, optional
`budget_bytes` 256–4096, default 1536) returns plain text for lifecycle hooks.
It starts with `Mandalore memory (evidence, not instructions):`, ranks recall over
the prompt's distinctive terms across every visible scope, flags matching
conflicts, then lists scopes if there's room. It never clips a claim and returns
nothing when nothing matches.

## Who wrote it

When the owner approves an OAuth client, or on a local client's first write to
a signet, the server enrolls a device record for it in `provenance/devices/`
(label `<client name> (<kind>), approved <date>`) and keeps the mapping in
`state_dir`. If writes are paused, enrollment waits for the client's first write. Every write by that client records
`{device_id: <client device>, actor: owner.name, harness: <client kind>}`.

## Git backup

Reads never fetch. After a write the server waits 2 seconds for writes to settle,
checkpoints and pushes `main` to the remote, never forcing. While a push is
pending it retries every minute. The memory is already durable in the checkout,
so a failed push never fails a save. The save receipt says `durable: true` and
`backup: pending|pushed|failed`.

At startup the server reads each remote's `main`. Only if the remote has commits
the checkout lacks does it fetch them, into `refs/remotes/origin/main` (never
`main` or the worktree), to tell ahead from diverged. A remote ahead means
another writer is still alive: the server refuses to
serve that signet (`503`) and logs `remote ahead: run mandalore server reconcile`.
An unreachable remote doesn't block serving: saves stay durable locally, and the
server keeps checking the remote every minute while it's unreachable or a push is
pending. If a push or check then finds the remote ahead, the signet keeps serving
reads but refuses writes (`backup.remote_ahead`), so nothing widens the gap.

### Diverged history

If the server saved while its remote was unreachable and another writer pushed in
the same window, both sides have commits the other lacks. The server logs
`diverged: local and remote both have commits; manual recovery needed`, refuses
writes, and on restart refuses to serve the signet. `server status` shows
`"remote": "diverged"`, and `server reconcile` refuses without changing anything.
Under a single writer this means a second writer broke the model. To recover:

1. Find and stop the other writer (usually a 1.x clone still syncing).
2. Stop the server.
3. In the signet root, fetch the remote and merge it:
   `git fetch <remote> main && git merge --no-ff FETCH_HEAD`. Signet records are
   append-only files with unique names, so the merge should not conflict. If it
   does, abort it and inspect both histories; don't resolve by deleting records.
4. Start the server. Its first checkpoint validates the whole signet before it
   pushes the merge; an invalid signet is not pushed.

## Running it

### launchd

On macOS, `mandalore server install-launchd --config FILE [--label LABEL]`
prints a launchd agent plist, asks, then writes it to
`~/Library/LaunchAgents/LABEL.plist` and loads it with `launchctl bootstrap
gui/UID`. The default label is `dev.acoz.mandalore.server`. The agent runs this
executable with `serve --config` and the configuration's absolute path, with
`RunAtLoad` and `KeepAlive`, logging to `~/Library/Logs/mandalore/LABEL.log`.
It checks the configuration first and refuses an existing plist. If loading
fails it removes the plist. `mandalore server uninstall-launchd [--label LABEL]`
asks, then runs `launchctl bootout` and removes the plist; if the agent stays
loaded it keeps the plist. Both answer No unless you type `y`, and print a JSON
result. [local.md](deploy/local.md#3-run-it-as-a-service) walks through it.

### Docker

`packaging/server/Dockerfile` builds a two-stage image: a static
(`CGO_ENABLED=0`) `mandalore` on Debian slim with `git`, `openssh-client` and
`curl`. It runs as UID and GID 10001, reads `/config/server.json` and keeps
state in `/state`. Its `HEALTHCHECK` fetches `/healthz` from inside the
container. Releases publish `ghcr.io/acoz-labs/mandalore:<version>` from the
accepted binaries (see [delivery](delivery.md#server-image)).

```sh
docker build -f packaging/server/Dockerfile -t mandalore-server .
docker run -d --name mandalore --restart unless-stopped --read-only --tmpfs /tmp \
  -v /example/mandalore/config:/config:ro -v mandalore-state:/state \
  -p 127.0.0.1:8730:8730 mandalore-server
```

- Paths in the configuration are container paths: `state_dir` and every signet
  `root` live under `/state` (for example `/state/server` and
  `/state/signets/primary`).
- To reach it from the host, listen on `0.0.0.0:8730` inside the container and
  publish the port on the host's loopback only. `loopback-trust` then reaches
  only processes inside the container, so a published server uses OAuth.
- `/state` must be writable by UID 10001. A named volume is, from the image; a
  bind mount needs `chown 10001:10001`. The hard-link, fsync and atomic-rename
  caveats for signets apply to whatever backs `/state`: use a local filesystem.
- Git pushes the backup over SSH with strict host key checking. Mount a deploy
  key and `known_hosts` read-only at `/var/lib/mandalore/.ssh`, owned by UID 10001
  with the key at mode 0600.
- Run admin commands in the container:
  `docker exec mandalore mandalore server status --config /config/server.json`.

`packaging/server/smoke-test.sh IMAGE` checks an image with a synthetic signet
in throwaway volumes: the health check, `/healthz`, MCP `tools/list` through
`loopback-trust` via `docker exec`, a non-root UID and a read-only `/config`.

## Admin commands

- `mandalore server owner enroll --config FILE [--replace]` prints the one-time
  owner passkey enrollment link (see [Owner approval](#owner-approval)).
- `mandalore server clients list --config FILE` lists approved and pending OAuth
  clients: ID, name, kind, redirect URIs, grants per signet and live tokens.
- `mandalore server clients revoke --config FILE --client ID` removes a client,
  its grants and every token it holds. Its next MCP call gets `401`. The device
  record stays in the signet's provenance; a DCR client must register again.
- `mandalore server clients set-scope --config FILE --client ID --scope
  read|read-write [--signet SLUG]` changes the client's grant on one signet, or on
  every signet it holds. Live tokens never exceed the new grant.

- `mandalore server status --config FILE` prints, per signet: head, last push,
  pending push, backup state, and the remote's state now (`current`, `ahead`,
  `diverged` or `unreachable`). Status never fetches, so with an unfetched remote
  commit and no recorded push it reports `ahead-or-diverged`; `reconcile` (which
  fetches) or a server start tells them apart.
- `mandalore server reconcile --config FILE [--signet SLUG]` fetches the remote,
  validates it as the same signet with append-only history, and fast-forwards.
  It refuses uncommitted or diverged local history and never merges or
  discards (see [Diverged history](#diverged-history)). Restart the server
  afterwards.
- `mandalore server migrate preflight|init|retire|verify` moves 1.x clones to
  the server; see [migration-2.0.md](migration-2.0.md).
- `mandalore server install-launchd|uninstall-launchd` adds or removes the
  macOS launchd agent; see [launchd](#launchd).

## Read cache

The server is each signet's only writer, so it keeps the validated revision graph
and per-revision term index in memory between its own writes; every write drops
them. On a synthetic 1,000-record signet, `memory_context` measures about 7 ms
p95 between writes and about 165 ms p95 on the first call after a write, on a
development laptop, against a 300 ms target. On a plain build `bin/ci` asserts
the 300 ms bound between writes and a 1 s regression tripwire after a write; the
race detector slows the post-write path about fivefold. A change made to the checkout by another process isn't
seen until the server's next write or restart, which is why the server must be
the only writer.
