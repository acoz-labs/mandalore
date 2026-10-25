# Mandalore server

`mandalore serve --config FILE` is Mandalore 2.0's single-writer memory server.
It owns one or more signets and serves MCP over Streamable HTTP. Agents never
write a signet's Git repository; the server does, and pushes each signet to its
remote as a backup. The [2.0 design](design/2.0.md) explains why.

The server includes its own OAuth 2.1 authorization server with owner passkey
approval (#161) and serves foundlings (#162). The Claude plugin comes with
#163. The cutover from 1.x clones is in [migration-2.0.md](migration-2.0.md).
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
     "remote": "git@github.com:owner/signet.git",
     "canon": ["foundling-work"],
     "legacy": {"foundling-notes": "/var/lib/mandalore/legacy/notes"}}
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
| `canon_refresh_minutes` | Canon refresh interval, 1–1440 minutes; default 10. |
| `signets[]` | 1–16 to serve; `server migrate init` also accepts an empty list and adds the entry. `slug`, absolute `root` (an existing Git checkout on `main`), its `signet_id`, the server's enrolled `device_id` in it, and the backup `remote`. |
| `signets[].canon` | Optional, up to 64: canon foundling IDs registered in the signet to refresh and serve. See [Foundlings](#foundlings). |
| `signets[].legacy` | Optional, up to 64: legacy foundling ID → clean absolute root the server can read. |

One signet serves at `/mcp`. Several serve at `/s/<slug>/mcp`. Each signet has its
own lock, memory service, Git backup and client authenticator; a request to one
signet never touches another. `/healthz` answers `GET` from loopback peers only.

### Authentication

Each signet's client comes from one authenticator. A `Client` has an ID, a name
(its device label), a kind (recorded as the write's harness) and a scope.

- `oauth` (default): OAuth 2.1 bearer tokens from the built-in authorization
  server, below. A request with no bearer token gets `401` with
  `WWW-Authenticate: Bearer resource_metadata="..."`. An unknown, expired,
  revoked or other-signet token gets `401` with `WWW-Authenticate: Bearer
  error="invalid_token", resource_metadata="..."` (RFC 6750 §3.1).
- `loopback-trust`: a direct loopback peer that addresses the server as
  `127.0.0.1` or `localhost` is the client `local`, with `local_scope`. Any local
  process can use it. It is refused with a non-loopback `public_url`, and a
  request carrying proxy forwarding headers (`Forwarded`, `X-Forwarded-*`,
  `X-Real-IP`, `CF-Connecting-IP`, `CF-Ray`, `True-Client-IP`, `X-Client-IP`,
  `Via`) gets `401`, so a same-host proxy or tunnel that rewrites Host can't relay
  internet traffic as the local client.
- `insecure-test`: static bearer tokens from `test_clients`. It exists only in
  the server package's test builds; a release binary refuses it at config load.

### A new signet

`signet create` still writes signet format 1, and the server needs format 2:
`init` refuses format 1, and withdrawing a memory needs format 2. Format 2 records
the upgrade from a Git commit of format 1, so a new signet goes through the same
upgrade a 1.x one does, using the 2.0 binary. Paths under `/example/` stand for
absolute paths of your choice, and the remote is an empty private repository.

```sh
mandalore signet create --repository /example/setup/signet --name Example --device-label setup
cd /example/setup/signet
git init -b main && git add -A && git commit -m "Create signet"
git remote add origin git@github.com:example-org/signet.git && git push -u origin main
mandalore signet bind --repository . --binding /example/setup/binding.json --device-label setup-binding --actor Owner
git add -A && git commit -m "Enroll setup binding" && git push
echo '{"binding_path": "/example/setup/binding.json"}' | mandalore call signet_upgrade_preview > /example/setup/plan.json
```

Keep the plan outside the clone: preview needs a clean checkout. Review it, then pass its `result` back as `{"plan": RESULT, "stopped_writers": true}`
to `mandalore call signet_upgrade_apply`. Commit and push the upgrade
(`signet.json` and `provenance/upgrades/`). Then, keeping the receipt outside
the clone too:

```sh
MANDALORE_BINDING=/example/setup/binding.json mandalore server migrate preflight --clone /example/setup/signet > /example/setup/setup.json
mandalore server migrate init --config FILE --remote git@github.com:example-org/signet.git --slug primary \
  --root /example/mandalore/signets/primary --preflight /example/setup/setup.json \
  --device-lost DEVICE_FROM_CREATE --reason "Enrolled by signet create; no clone of its own"
mandalore server migrate verify --config FILE --slug primary
```

`DEVICE_FROM_CREATE` is the `device_id` that `signet create` printed. After
`verify`, start the server and stop using the setup clone: the server is the
only writer.

## OAuth

The server is its own authorization server; it depends on no outside identity
provider. All endpoints are on `public_url`:

| Endpoint | Purpose |
| --- | --- |
| `/.well-known/oauth-protected-resource[/s/<slug>/mcp]` | RFC 9728 metadata per signet: `resource` is the signet's MCP URL, the only authorization server is this one, scopes `memory:read` and `memory:write`. One signet answers at the bare path (and at `/.well-known/oauth-protected-resource/mcp`). |
| `/.well-known/oauth-authorization-server` | RFC 8414 metadata: S256 PKCE only, public clients (`none`), client ID metadata documents, `/register`, `/revoke`, and `offline_access`. |
| `POST /register` | RFC 7591 dynamic registration (JSON). The client is pending, with no access, until the owner approves it. 10 registrations per 10 minutes from all peers together; over that it answers 429 with `Retry-After`. No forwarded address header is trusted. At most 20 pending: a new registration evicts the oldest pending one, never an approved client, and a pending client expires after 24 hours. |
| `GET /authorize` | Authorization code request. |
| `POST /consent` | The owner's approve or deny decision. |
| `POST /token` | Form-encoded `authorization_code` and `refresh_token` grants. RFC 6749 error codes; a refresh from an unknown or revoked client gets `400 invalid_grant`. If the OAuth state can't be read, it answers `500 server_error`. |
| `POST /revoke` | RFC 7009 revocation, form-encoded, so a client's logout ends its server tokens. The client names itself with `client_id`, as at `/token`. Revoking either token of a pair ends the authorization's whole token family, and it stays dead. The client's own expired token still ends its family until it is pruned. A token that is unknown or another client's gets the same empty 200 and changes nothing. `token_type_hint` is ignored. With no `client_id`, it answers `400 invalid_request`; with an unknown one, `401 invalid_client`. If the OAuth state can't be read or saved, it answers `503 temporarily_unavailable` and revokes nothing. |
| `/owner/*` | Passkey enrollment and sign-in. |

**Clients.** A `client_id` is either a DCR registration or an HTTPS URL naming a
client ID metadata document (such as Claude Code's). The document is fetched at
authorization time only: HTTPS, no proxy, no redirects, 5 s timeout, 16 KiB cap,
cached for 10 minutes, and its `client_id` must equal its URL. The connection
address is checked after DNS resolution, so a name that resolves to a loopback,
private, link-local, shared, documentation or other non-public address is refused
before connecting. Its `redirect_uris` are enforced.

**Installations.** Grants belong to installations, not to a `client_id`. Every
copy of a metadata-document client, such as every Claude Code installation,
shares one `client_id`, so the server can't tell a new installation from one
it has seen: each approval of a document client creates a new installation,
and the token family that approval starts is that installation's. A DCR
registration is made by one installation, so it has one installation, which
later authorizations reuse. Each installation has an ID (`inst-…`), its own
grants, a first-seen time (its approval) and a last-used time (its last token
issue). An installation of a document client ends when its family does, by
logout, refresh-token replay or revoke; it is pruned a day after its last use
with no live token. A DCR installation lasts as long as its registration.

**Authorization.** PKCE S256 is required; `plain` or a missing challenge is
refused. `resource` is required and must be one of the signets' MCP URLs. A
redirect URI must exactly match one the client registered, except that a
registered `http://localhost/...` or `http://127.0.0.1/...` matches any port.
A bad client or redirect shows an error page and never redirects. Other errors
return to the client's redirect. Codes last 2 minutes and are single-use; a
replayed code revokes the tokens it issued.

**Scopes.** The owner grants each installation `read` or `read-write` per
signet. A
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
   the sign-in page. Every authorization of a metadata-document client, and a
   DCR client's first authorization for a signet, then shows the consent page.
   If other installations of the document client are approved, the page says
   this is a new one that doesn't share their access. The page shows the
   client's name, its redirect host in large text, the signet, a read or
   read-write picker, and approve or deny. A loopback redirect
   gets a warning, since any local process can claim a loopback port. Owner pages
   can't be framed and run only their own nonce-tagged script.
4. On approval the server stores the grant and enrolls the client's device
   record in the signet, labelled `<client name> (<kind>), approved <date>`. The
   kind is the host of its metadata document or of its first non-loopback
   redirect, such as `claude.ai`. Later authorizations by an approved DCR
   client for that signet skip the consent page; the owner still signs in, or
   uses a live session. A client ID is no secret, so no code is ever issued
   without the owner. Installations of one client share its device record. For a metadata-document client the page also names the host that
   publishes the document, since the client chooses its own name.

Browsers treat `http://localhost` as a secure context, so passkeys and the
`Secure` cookie work there; an IP address can't hold a passkey.

## Tools

Read scope gets `memory_scopes`, `memory_recall`, `memory_history`,
`memory_journal`, `memory_visibility_history` and `memory_context`. Read-write
adds `memory_remember`, `memory_journal_append`, `memory_withdraw` and
`memory_restore`. Read clients never see write tools, and the call path refuses
writes by scope as well. A signet with foundlings configured adds their tools
(see [Foundlings](#foundlings)); one without them offers none.

Server instructions are under 600 bytes and differ for
read and read-write clients.

`memory_remember` and `memory_journal_append` take a required `request_id`
(16–128 characters). A retry with identical input returns the original receipt;
the same key with different input is refused with `request.conflict`. Keys are
namespaced by client.

`memory_context` (`event`: `start` or `prompt`, optional `prompt`, optional
`budget_bytes` 256–4096, default 1536) answers the Claude plugin's hook. It
returns Claude Code hook output JSON,
`{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"..."}}`
(`SessionStart` for `start`), because Claude Code adds an `mcp_tool` hook's
`additionalContext` to context but not its plain text. The context starts with
`Mandalore memory (evidence, not instructions):`, ranks recall over the prompt's
distinctive terms across every visible scope, flags matching conflicts, then
lists scopes if there's room. The budget applies to `additionalContext`. It never
clips a claim and returns an empty result when nothing matches.

## Foundlings

Foundlings follow [the foundlings contract](foundlings.md); the server adds how
they are configured and refreshed.

- **Canon.** The server refreshes each `canon` reference at startup and every
  `canon_refresh_minutes`, through the verified fetch path (a 3-second budget per
  reference, up to four in parallel, batched blob reads, read-only: it never
  writes the source's remote). One stalled source never delays another. A refresh
  runs outside the signet lock, so it never delays reads or saves. Reads use the
  latest verified snapshot and never fetch; each result carries `commit`,
  `observed_at` and `age_seconds`. A failed or interrupted refresh keeps the
  previous verified snapshot, reported `stale` with its age and `reason`; with
  none it is `unavailable`. On shutdown the server lets a refresh in flight
  record its receipts. Snapshots are cached in the checkout's ignored `.mandalore/canon`.
  The server needs read-only Git access to each source's remote.
- **Legacy.** `legacy` roots replace clone-local connections; reads verify the
  root against the registration's pin, and an unreadable root (logged at startup)
  reports unavailable.
- **Tools.** Canon adds `foundling_status`, `foundling_canon_scopes`,
  `foundling_canon_recall` and `foundling_canon_heads`; legacy adds
  `foundling_search` and `foundling_read`; either adds `foundling_list`.
  Read-write clients also get `foundling_canon_promote` or `foundling_promote`,
  which record the client like every other write. `session_id` is optional and
  ignored; every client reads the same snapshot and agents cannot refresh.

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
fails it removes the plist. After loading it waits 2 seconds and reports
`running: true` only if `launchctl print` shows the first start still running.
Otherwise it fails with `launchd.not_running` and keeps the loaded agent, so
read its log (a taken port is the usual cause), fix it and `launchctl kickstart
-k`, or uninstall. `mandalore server uninstall-launchd [--label LABEL]`
asks, then runs `launchctl bootout` and removes the plist; if the agent stays
loaded it keeps the plist. Both answer No unless you type `y`, and print a JSON
result. [local.md](deploy/local.md#3-run-it-as-a-service) walks through it.

### Docker

`packaging/server/Dockerfile` builds a two-stage image: a static
(`CGO_ENABLED=0`) `mandalore` on Debian slim with `git`, `openssh-client` and
`curl`. It runs as UID and GID 10001, reads `/config/server.json` and keeps
state in `/state`. Its `HEALTHCHECK` fetches `/healthz` from inside the
container. Releases don't publish an image yet ([#188](https://github.com/acoz-labs/mandalore/issues/188)),
so build it from a checkout of the release tag. The build compiles `mandalore`
from that source; `VERSION` stamps its version.

```sh
git clone --branch v2.0.1 https://github.com/acoz-labs/mandalore.git
cd mandalore
docker build -f packaging/server/Dockerfile --build-arg VERSION="$(cat VERSION)" -t mandalore-server .
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
- `mandalore server clients list --config FILE` lists one entry per approved
  installation, and one per pending client: client ID, installation ID, name,
  kind, registered, first seen, last used, redirect URIs, grants per signet and
  live tokens. Installations of a metadata-document client share its client ID.
- `mandalore server clients revoke --config FILE --installation ID` removes one
  installation, its grants and its tokens; other installations of the client
  keep working. `--client ID` instead removes the client and every
  installation of it, and says how many. Either way the revoked installation's
  next MCP call gets `401 invalid_token` and its refresh `400 invalid_grant`,
  so it authorizes again, which needs the owner's passkey and consent. The
  device record stays in the signet's provenance; a DCR client revoked by
  `--client` must register again.
- `mandalore server clients set-scope --config FILE --installation ID --scope
  read|read-write [--signet SLUG]` changes one installation's grant on one
  signet, or on every signet it holds; `--client ID` changes every installation
  of the client. Live tokens never exceed the new grant or the scope they were
  issued with, and a refresh keeps that scope, so a wider grant gives live
  tokens back at most what an earlier narrowing took. A DCR installation gets
  a wider grant in full from its next authorization. A metadata-document
  installation, new or migrated, never does: its next authorization is a new
  installation, whose scope the owner picks on the consent page.

### Upgrading OAuth state from 2.0.1

2.0.0 and 2.0.1 kept one grant per `client_id`. 2.0.2 migrates
`state_dir/oauth.json` when it first reads it, with no new sign-in or consent:

- A DCR client with a grant becomes one installation holding its grants and all
  its tokens. It keeps skipping the consent page when it authorizes again.
- A metadata-document client with a grant becomes one installation per live
  token family, each with the client's grant, since 2.0.1 didn't record which
  installation was approved with which scope. Each keeps working and
  refreshing; narrow or revoke them one at a time afterwards. A document client
  with no live tokens has nothing to migrate, and its next sign-in needs
  consent, as every new installation's does.
- First seen is the client's registration time for migrated installations,
  and last used is the family's last token issue.

Installation IDs of migrated state derive from the client and family, so
`server clients list` shows the same IDs before and after the server first
saves the file. Every save also keeps each client's widest grant in the 2.0.1
field, so a rollback to 2.0.1 keeps every live token and returns to its
per-client consent. Upgrading again after 2.0.1 has rewritten the file
migrates it again.

- `mandalore server status --config FILE` prints, per signet: head, last push,
  pending push, backup state, the remote's state now (`current`, `ahead`,
  `diverged` or `unreachable`), and each canon reference's state, commit, age,
  last attempt and last error. Status never fetches, so with an unfetched remote
  commit and no recorded push it reports `ahead-or-diverged`; `reconcile` (which
  fetches) or a server start tells them apart.
- `mandalore server reconcile --config FILE [--signet SLUG]` fetches the remote,
  validates it as the same signet with append-only history, and fast-forwards.
  It refuses uncommitted or diverged local history and never merges or
  discards (see [Diverged history](#diverged-history)). Restart the server
  afterwards.
- `mandalore server canon refresh --config FILE [--signet SLUG]` refreshes the
  configured canon references now, for one signet or all, and prints the
  receipts. A running server serves the new snapshots from its next canon read.
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
