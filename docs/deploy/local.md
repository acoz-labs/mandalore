# Local server on a work laptop

A work laptop runs its own [Mandalore server](../server.md) on loopback. It owns
the work signet as its only writer, and Claude Code on the same laptop connects
to `http://127.0.0.1:8730/mcp`. Nothing listens beyond the laptop and no tunnel
is involved. A home server that needs the work signet reads its remote as a
canon reference; it never writes it ([2.0 design](../design/2.0.md), section 3).

## 1. Prepare the signet checkout

The server needs its own checkout of the signet on `main`, with its own device
enrolled. Create it with `mandalore server migrate init`: coming from 1.x, see
[the cutover](../migration-2.0.md); for a new signet, see
[A new signet](../server.md#a-new-signet). Paths under `/example/` stand for absolute paths of your choice.

## 2. Configure the server

`~/.config/mandalore/server.json`:

```json
{
  "listen": "127.0.0.1:8730",
  "public_url": "http://127.0.0.1:8730",
  "state_dir": "/example/mandalore/state",
  "owner": {"name": "Owner"},
  "auth": "loopback-trust",
  "local_scope": "read-write",
  "signets": [
    {"slug": "work", "root": "/example/mandalore/signets/work",
     "signet_id": "signet-example", "device_id": "device-server",
     "remote": "git@github.com:example-org/work-signet.git"}
  ]
}
```

`loopback-trust` makes every loopback request that addresses the server as
`127.0.0.1` or `localhost` the client `local`, with `local_scope`. Any process on
the laptop can use it; that is the trade-off for skipping OAuth. The server
refuses it unless both `listen` and `public_url` are loopback, and refuses any
request that carries proxy forwarding headers.

To use OAuth instead, drop `auth` and `local_scope`, set `public_url` to
`http://localhost:8730` (a passkey needs a host name, not an IP address) and
enroll the owner with `mandalore server owner enroll --config FILE`. The MCP URL
is then `http://localhost:8730/mcp`.

## 3. Run it as a service

On macOS, install a launchd agent:

```sh
mandalore server install-launchd --config ~/.config/mandalore/server.json
```

It prints the plist, then asks before writing it to
`~/Library/LaunchAgents/dev.acoz.mandalore.server.plist` and loading it with
`launchctl bootstrap`. The agent runs `mandalore serve --config FILE` from the
executable you ran, with `RunAtLoad` and `KeepAlive`, so launchd starts it at
login and restarts it if it exits. Logs go to
`~/Library/Logs/mandalore/dev.acoz.mandalore.server.log`. `--label LABEL` picks
another label, for a second server or a test. If `serve` exits on its first
start, for example because another process holds the port, install reports
`launchd.not_running` instead of success; the log says why.

```sh
launchctl print gui/$(id -u)/dev.acoz.mandalore.server   # state, pid, last exit
curl http://127.0.0.1:8730/healthz                       # alive
mandalore server status --config ~/.config/mandalore/server.json
```

`mandalore server uninstall-launchd [--label LABEL]` asks, then unloads the agent
and removes its plist. Logs stay. To move the executable or change the
configuration path, uninstall and install again. A new version at the same path
takes effect on the next restart:
`launchctl kickstart -k gui/$(id -u)/dev.acoz.mandalore.server`.

On Linux, run `mandalore serve` under your service manager, or use the
[Docker image](../server.md#docker). In a container, `loopback-trust` only
reaches processes inside the container, so a container on a laptop uses OAuth:
`"listen": "0.0.0.0:8730"`, `"public_url": "http://localhost:8730"`, and
`docker run -p 127.0.0.1:8730:8730`.

## 4. Connect Claude Code

Give the Mandalore Claude plugin the server URL `http://127.0.0.1:8730/mcp` when
it asks at install. Without the plugin, add the server directly:

```sh
claude mcp add --transport http mandalore http://127.0.0.1:8730/mcp
```

With OAuth, use `http://localhost:8730/mcp`; Claude Code opens the owner sign-in
and consent page in the browser on first use.
