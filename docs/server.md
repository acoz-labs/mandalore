# Mandalore server

`mandalore serve --config FILE` is Mandalore 2.0's single-writer memory server.
It owns one or more signets and serves MCP over Streamable HTTP. Agents never
write a signet's Git repository; the server does, and pushes each signet to its
remote as a backup. The [2.0 design](design/2.0.md) explains why.

This is the server core. OAuth and owner passkeys come with #161, foundlings on
the server with #162, the Claude plugin with #163, migration from 1.x with #164
and packaging with #165. Until #161 lands, the server accepts only the two
loopback-only authenticators below.

## Configuration

The file is strict JSON: unknown, duplicate or mistyped fields are refused.

```json
{
  "listen": "127.0.0.1:8730",
  "public_url": "http://127.0.0.1:8730",
  "state_dir": "/var/lib/mandalore/state",
  "owner": {"name": "Owner"},
  "auth": "loopback-trust",
  "local_scope": "read-write",
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
| `public_url` | The origin clients use, such as `https://memory.example.com`. HTTPS unless loopback. It sets the Host and Origin allowlist (plus loopback names on the listen port). |
| `state_dir` | Server state: client device enrollments and backup state, per signet. Never inside a signet. |
| `owner.name` | The actor recorded on every write. |
| `auth` | `loopback-trust` or `insecure-test`. Both are refused unless `listen` is loopback; `loopback-trust` also needs a loopback `public_url`. |
| `local_scope` | `read` or `read-write`, the scope of the `local` client under `loopback-trust`. |
| `test_clients` | `insecure-test` only: `id`, `kind`, `scope`, bearer `token` (16+ bytes) and the `signets` slugs it may use. |
| `signets[]` | Up to 16. `slug`, absolute `root` (an existing Git checkout on `main`), its `signet_id`, the server's enrolled `device_id` in it, and the backup `remote`. |

One signet serves at `/mcp`. Several serve at `/s/<slug>/mcp`. Each signet has its
own lock, memory service, Git backup and client authenticator; a request to one
signet never touches another. `/healthz` answers `GET` from loopback peers only.

### Authentication

Authentication sits behind `Authenticator.Authenticate(r) (Client, error)`, one
per signet. A `Client` has an ID, a kind (recorded as the write's harness) and a
scope. An unknown client gets `401`; a known client without access to that
signet gets `403`.

- `loopback-trust`: a direct loopback peer that addresses the server as
  `127.0.0.1` or `localhost` is the client `local`. Any local process can use it.
  It is refused with a non-loopback `public_url`, and a request carrying proxy
  forwarding headers (`Forwarded`, `X-Forwarded-*`, `X-Real-IP`,
  `CF-Connecting-IP`, `CF-Ray`, `True-Client-IP`, `X-Client-IP`, `Via`) gets
  `401`, so a same-host proxy or tunnel that rewrites Host can't relay internet
  traffic as the local client.
- `insecure-test`: static bearer tokens from `test_clients`. It exists only in
  the server package's test builds; a release binary refuses it at config load.

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

On a client's first write to a signet, the server enrolls a device record for it
in `provenance/devices/` (label `<client id> (<kind>), approved <date>`) and keeps
the mapping in `state_dir`. Every write by that client records
`{device_id: <client device>, actor: owner.name, harness: <client kind>}`.

## Git backup

Reads never fetch. After a write the server waits 2 seconds for writes to settle,
checkpoints and pushes `main` to the remote, never forcing. While a push is
pending it retries every minute. The memory is already durable in the checkout,
so a failed push never fails a save. The save receipt says `durable: true` and
`backup: pending|pushed|failed`.

At startup the server checks each remote without fetching. If the remote has
commits the checkout lacks, another writer is still alive: the server refuses to
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

## Admin commands

- `mandalore server status --config FILE` prints, per signet: head, last push,
  pending push, backup state, and the remote's state now (`current`, `ahead`,
  `diverged` or `unreachable`).
- `mandalore server reconcile --config FILE [--signet SLUG]` fetches the remote,
  validates it as the same signet with append-only history, and fast-forwards.
  It refuses uncommitted or diverged local history and never merges or
  discards (see [Diverged history](#diverged-history)). Restart the server
  afterwards.

## Read cache

The server is each signet's only writer, so it keeps the validated revision graph
and per-revision term index in memory between its own writes; every write drops
them. On a synthetic 1,000-record signet, `memory_context` measures about 7 ms
p95 between writes and about 165 ms p95 on the first call after a write, on a
development laptop, against a 300 ms target. `bin/ci` asserts both on a plain
build; the race detector slows the post-write path about fivefold. A change made to the checkout by another process isn't
seen until the server's next write or restart, which is why the server must be
the only writer.
