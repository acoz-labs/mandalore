# Cloudflare Tunnel in front of a home server

A home [Mandalore server](../server.md) listens on loopback, and `cloudflared` on
the same host carries `https://memory.example.com` to it. Nothing is exposed
through the router, and the server's own OAuth and owner passkeys protect it
([2.0 design](../design/2.0.md), sections 2 and 7). Cloudflare is a deployment
choice, not a code dependency. Replace `memory.example.com` with your own host
name throughout.

## 1. Configure the server

```json
{
  "listen": "127.0.0.1:8730",
  "public_url": "https://memory.example.com",
  "state_dir": "/example/mandalore/state",
  "owner": {"name": "Owner"},
  "signets": [
    {"slug": "primary", "root": "/example/mandalore/signets/primary",
     "signet_id": "signet-example", "device_id": "device-server",
     "remote": "git@github.com:example-org/signet.git"}
  ]
}
```

- `public_url` is the origin clients use. It is the OAuth issuer and the
  passkey origin, and the server answers only requests whose `Host` is its host
  name (or a loopback name on the listen port), so `cloudflared` must pass the
  original `Host` through. It does by default; don't set `httpHostHeader`.
- `auth` stays at its default, `oauth`. `loopback-trust` is refused with a public
  URL, because every tunnelled request arrives from loopback.
- Run the server as a service: `mandalore server install-launchd` on macOS (see
  [local.md](local.md#3-run-it-as-a-service)), or the
  [Docker image](../server.md#docker) with `"listen": "0.0.0.0:8730"` and
  `-p 127.0.0.1:8730:8730`.

## 2. Create the tunnel

With a locally managed tunnel:

```sh
cloudflared tunnel login
cloudflared tunnel create mandalore
cloudflared tunnel route dns mandalore memory.example.com
```

`~/.cloudflared/config.yml`:

```yaml
tunnel: TUNNEL-UUID
credentials-file: /example/cloudflared/TUNNEL-UUID.json
ingress:
  - hostname: memory.example.com
    service: http://127.0.0.1:8730
  - service: http_status:404
```

Run it with `cloudflared tunnel run mandalore`, or install it as a service with
`cloudflared service install`. A dashboard-managed tunnel works the same way:
create a tunnel under Zero Trust, Networks, Tunnels, then add a public hostname
`memory.example.com` with service type `HTTP` and URL `127.0.0.1:8730`.

## 3. Enroll the owner and check it

```sh
mandalore server owner enroll --config /example/mandalore/server.json
curl https://memory.example.com/.well-known/oauth-authorization-server
```

Open the enrollment link on the device that will hold the owner passkey (it
works once, for 10 minutes). The metadata request should return JSON whose
`issuer` is `https://memory.example.com`. Then add `https://memory.example.com/mcp`
as a connector in Claude, or as the Claude plugin's server URL; with several
signets each is at `/s/<slug>/mcp`.

## 4. Edge rules

The server's paths fall into these groups:

| Paths | Who calls them | Protection |
| --- | --- | --- |
| `/authorize`, `/consent`, `/owner/*`, `/.well-known/*` | The owner's browser, and clients reading metadata | Owner passkey and session |
| `/mcp`, `/s/<slug>/mcp`, `/token` | Claude's servers and Claude Code | OAuth bearer tokens and codes |
| `/register` | Clients using dynamic registration, such as claude.ai | Nothing grants access before the owner approves, but anyone can fill the registration limits: rate-limit it at the edge (below) |
| Anything else, including `/healthz` | Nobody from outside | None needed: block it |

`/healthz` answers loopback peers, and through the tunnel every request comes
from loopback, so block it at the edge.

Set up these rules:

1. **Block unknown paths.** A WAF custom rule (Security, WAF, Custom rules)
   with action Block:

   ```
   http.host eq "memory.example.com" and not (
     http.request.uri.path in {"/authorize" "/consent" "/token" "/register" "/mcp"}
     or starts_with(http.request.uri.path, "/owner/")
     or starts_with(http.request.uri.path, "/.well-known/")
     or (starts_with(http.request.uri.path, "/s/") and ends_with(http.request.uri.path, "/mcp"))
   )
   ```

2. **Rate-limit `/register` per client address.** A rate limiting rule
   (Security, WAF, Rate limiting rules) matching

   ```
   http.host eq "memory.example.com" and http.request.uri.path eq "/register"
   ```

   counted by IP (`ip.src`), with action Block. Allow a few requests per period,
   such as 5 per 10 minutes, or the longest period your plan offers.

   The rule is needed because the server limits registrations per peer address
   (10 per 10 minutes) and holds at most 20 pending registrations for everyone.
   Behind the tunnel every request comes from `cloudflared` on loopback, so the
   server sees a single peer. Ten anonymous registrations then refuse every new
   registration, including yours, until the window passes. The edge rule
   restores a per-client limit. A lockout doesn't affect approved clients, or
   clients such as Claude Code that use a client ID metadata document instead
   of registering. To clear abandoned pending registrations, list them with
   `mandalore server clients list` and remove them with
   `mandalore server clients revoke`. If only claude.ai registers with your
   server, you can instead allow `/register` from Anthropic's egress range
   alone (see below).

3. **Leave the rest open.** Allow everyone on `/authorize`, `/consent`,
   `/owner/*` and `/.well-known/*`; the passkey protects them. Put no IP rule on
   `/mcp` and `/token`; OAuth protects them.

Don't put a managed challenge, JavaScript challenge or Bot Fight Mode in front
of `/mcp`, `/token` or `/register`: those callers are programs and can't solve
one. Caching doesn't apply, since every response is `Cache-Control: no-store`.

### Restricting by IP, and roaming devices

It's possible to allow only Anthropic's egress range (`160.79.104.0/21`) plus
your own devices. Three paths need more than that:

- **The consent page.** You approve clients from your browser, often a phone on
  cellular, which has no fixed IP. Keep `/authorize`, `/consent`, `/owner/*` and
  `/.well-known/*` open to everyone.
- **Claude Code on a Mac that moves.** It calls `/mcp` and `/token` from the
  Mac's own address, which changes with the network. Leave
  those paths without an IP rule, or run Cloudflare WARP on the Macs and allow
  your WARP egress.
- **The client metadata fetch.** The server fetches Claude Code's client ID
  metadata document itself, outbound. Edge rules don't affect it.

If you still want an allowlist on these paths, add it as a second WAF custom
rule with action Block, after rule 1:

```
http.host eq "memory.example.com"
and (http.request.uri.path in {"/mcp" "/token" "/register"}
     or (starts_with(http.request.uri.path, "/s/") and ends_with(http.request.uri.path, "/mcp")))
and not ip.src in {160.79.104.0/21 192.0.2.10}
```

Replace `192.0.2.10` with your devices' fixed or WARP egress addresses.

Cloudflare's dashboard labels change over time; check the menu paths above
against the current dashboard when you set this up.
