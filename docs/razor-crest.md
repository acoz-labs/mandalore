# Razor Crest

Razor Crest is the optional service that connects authorized remote MCP clients
to a Mandalore signet. It runs beside a persistent replica and reuses Mandalore's
memory operations. Local agent connections continue to use their own replicas.

**Implementation and acceptance are in progress under [#146](https://github.com/acoz-labs/mandalore/issues/146).**
The instructions below describe the candidate implementation. They are not a
claim that a released build or any particular mobile application supports it.
Ordinary cloud-chat, mobile, OAuth refresh and natural tool-selection acceptance
must be recorded separately before release.

## Ownership and privacy

Mandalore owns the memory format and semantics. Razor Crest owns the remote
transport, authentication adapter, operation authorization and service lifecycle.
The deployment owns ingress, identity-provider configuration, credentials,
persistent storage and recovery. The agent application owns tool selection and
its user interface.

Keep the deployment directory outside the product checkout. Real hostnames,
network topology, account IDs, bindings, signet data and credentials do not belong
in public source, issues, pull requests, screenshots or validation logs. Examples
use `memory.example.com` and synthetic data. Publish sanitized scenario results;
retain private installation evidence separately.

The authenticated cloud agent receives the memory returned by a tool. Its
provider and the ingress provider are part of the deployment's trust boundary.
A phone's VPN connection does not make a cloud provider's servers members of the
owner's private network.

## Service contract

The candidate exposes Streamable HTTP MCP at `/mcp`. It uses stateless HTTP
requests; each request must pass the host allowlist, optional browser Origin
allowlist, private-origin secret and signed identity validation. No inbound home
router port forwarding is required with an outbound tunnel.

One service selects one explicitly guarded binding. Clients cannot select a
repository, binding path, signet or identity through tool arguments. Read and
write grants map verified issuer subjects to that fixed signet. A subject with
read permission can recall the signet's permitted memory; do not treat scopes
or sensitivity labels as separate access-control boundaries.

Only semantic tools are exposed: scope discovery, recall, history, journals,
visibility history, synchronization status, and authorized idempotent saves.
Installation, migration, arbitrary filesystem access, shell execution and
repository administration remain local. Canon tools are available only for
explicitly configured direct foundling references.

Use a new `razor_session_open` result for each conversation that consults canon.
The opaque session identifier is bound to the authenticated subject and expires
after 24 hours. Ordinary canon reads retain its snapshot; `foundling_refresh`
advances that session explicitly. Restarting the service requires opening a new
canon session. Refresh cannot erase information already supplied to a model.

## Authentication and ingress

The first adapter validates Cloudflare Access assertions. Cloudflare handles the
client's OAuth flow; Razor Crest verifies the resulting signed
`Cf-Access-Jwt-Assertion`, including issuer, audience, subject, signature and
expiry. Neither an email header nor possession of the public URL grants access.
An authenticated subject must also appear in the service's grant map.

Configure a dedicated Access application for the endpoint, a restrictive owner
policy, and Managed OAuth. Use the client's actual OAuth redirect URI when
configuring registration; do not allow arbitrary redirect hosts. Leave localhost
and loopback registration disabled unless a selected client needs them.

Cloudflare documents Managed OAuth as beta and requires an RFC 8707-compatible
client. It provides discovery and token refresh and sends a signed assertion to
the origin. A browser-cookie login wall alone does not provide that flow. Verify
the installed client's discovery, authorization, refresh and reconnect behavior;
vendor documentation is not an acceptance result.

Route an outbound Cloudflare Tunnel to the service's private listener. Bind the
host-published port to loopback or use a private container network accessible only
to the tunnel connector. Keep administration on a separate private path. Do not
expose the service port directly on the public network.

The origin also requires `X-Razor-Crest-Origin`. Generate an independent random
secret and store it in the private file selected by `origin_secret_file`. For
the Cloudflare recipe, use a Request Header Transform Rule matching only the
application hostname to **set/overwrite** this header with that value. Do not
append it or accept a caller-provided value. Cloudflare supports literal header
overwrites; validate that the rule is active for the tunnel route and that the
secret is absent from responses and logs. This secret protects the origin route;
it does not replace identity validation or the subject grant.

Select a short OAuth access-token lifetime and a suitable refresh-grant duration.
Cloudflare re-evaluates its policy at refresh. Existing access tokens and requests
already admitted may outlive a policy edit; measure the actual enforcement
window. To revoke a service grant, remove the subject from private configuration
and restart the service. To rotate the origin secret, coordinate the edge rule
and service file and restart; expect failures during a mismatched transition.
Do not claim instantaneous revocation from a successful dashboard edit alone.

See [Managed OAuth](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/managed-oauth/),
[Cloudflare Tunnel](https://developers.cloudflare.com/tunnel/) and
[Request Header Transform Rules](https://developers.cloudflare.com/rules/transform/request-header-modification/).

## Configure the candidate

The runtime entry point is:

```sh
mandalore razor-crest serve --config /config/service.json
```

Start from [the synthetic service configuration](../packaging/razor-crest/service.example.json).

| Setting | Meaning |
| --- | --- |
| `binding` | Machine-local binding file outside the signet, using paths visible inside the container |
| `binding_sha256`, `signet_id` | Exact binding-byte digest and expected signet identity; unexpected replacement fails closed |
| `listen` | Explicit internal host and port |
| `hosts` | Exact HTTP Host values accepted from the ingress |
| `origins` | Browser origins explicitly permitted; an empty list rejects requests carrying Origin |
| `origin_secret_file` | Private file holding the independent ingress secret |
| `auth.issuer`, `auth.audience` | Trusted Access organization origin and application audience |
| `auth.subjects` | Verified immutable subject IDs and explicit read/write grants |
| `synchronization` | Explicit service authorization for bounded Git synchronization |
| `canon_foundlings` | Direct canon registrations permitted through the remote interface |

Provision and enroll a dedicated service replica through the existing
[signet and binding procedures](interface.md). Bind using the final container
paths and keep that writer identity across replacement. Do not reuse a local
agent's binding with host paths that are unavailable inside the container.
Hash the final binding bytes; an edited binding requires a deliberate new pin.

Mount private configuration read-only and the entire service state read-write.
Use a filesystem that supports hard links, file and directory synchronization,
and atomic directory rename without replacement. Some shared or FUSE-backed
paths reject those operations even when ordinary writes succeed. Validate
creation, save, retry and recovery on the actual mounted volume; use supported
local storage rather than weakening atomic-publication guarantees.
The [Compose example](../packaging/razor-crest/compose.yaml) takes private
`RAZOR_CONFIG_DIR` and `RAZOR_STATE_DIR` paths. Its non-root runtime needs access to
those directories with the configured container UID. Git credentials and trusted
SSH host keys must be provisioned separately and scoped to the selected replica's
repository. Never bake them into an image, command example or checked-in file.

Build from a clean fixed checkout and set `RAZOR_SOURCE_COMMIT` to its full commit
ID when using Compose. This is a source-build recipe, not a claim of a published
container-registry image. Retain the resulting image digest alongside that source
identity; a build argument alone does not prove that the checkout was clean or
that the running image matches it. The image embeds the repository version and
source commit. Its `/healthz` endpoint is loopback-only process liveness, not a
storage-readiness or successful-delivery check.

## Saves, retries and freshness

Remote saves require a stable `request_id`. Retry an ambiguous response with the
same key and identical semantic input. Reusing the key with different input is
rejected. A new key is a new operation and can create a duplicate.

The service writes a durable intent before publishing the memory revision or
journal entry. Retain the complete replica, including its machine-local
`.mandalore` directory, to preserve that retry contract. Re-cloning only Git
history loses local retry state and may lose undelivered writes.

Save responses distinguish local durability from Git delivery. A durable save
with pending or disabled synchronization remains local. A delivered Git head does
not establish semantic agreement: conflicting heads still require an explicit
correction. Do not repeat a semantic save with a new key to retry delivery.

With `synchronization: true`, the candidate makes bounded attempts at startup,
approximately every minute, before reads and after saves, serialized with memory
operations. A failed attempt preserves pending local work. These are service
boundaries, independent of native-agent lifecycle hooks. Setting the flag false
disables those signet synchronization attempts; it does not disable authentication
key retrieval or explicitly configured canon-reference access.

## Recovery and operation

Keep the selected image identity, configuration, secret-manager references and
volume recovery material privately. Monitor process/container health and delivery
receipts separately: a responding process is not proof of fresh remote memory.

For a consistent backup, stop the service and any other writers to its replica,
then preserve its full volume, including `.git` and `.mandalore`, and the matching
binding/configuration. Encrypt backups and keep access restricted. Define the
acceptable loss window based on the backup cadence. Git delivery alone does not
protect pending local writes or local idempotency state.

Restore into an isolated environment first. Verify the signet identity and binding
pin, recover required credentials, and replay an identical synthetic request key.
Confirm that its original receipt is returned without creating another revision.
Then verify pending delivery and recall from another replica before returning the
service to its ingress route.

Upgrade by retaining the old image and a consistent volume backup, stopping the
old process, and starting the selected candidate against the preserved state.
Exercise authentication, denied access, recall, save/retry and synchronization.
Rollback selects the prior compatible image with the preserved state. Do not
silently downgrade the signet format or replace the volume from a remote clone;
use the existing explicit format-upgrade/recovery procedure if required.

## Compatibility evidence

Protocol tests, ordinary cloud-chat tests and automatic tool selection are
different evidence. Record the exact client surface, account capability, date,
candidate identity and result for each. A web connector installation does not
prove that the mobile application uses it, and a prompted tool call does not
prove recall will occur for an ordinary question.

The release acceptance record must include two provider clients, mobile use,
cross-client save/recall, denied second-identity access, token expiry/revocation,
origin bypass attempts, cancellation/retry/restart, withdrawal/conflicts and
backup/upgrade recovery. Until those results exist, compatibility remains
unverified. Keep private endpoint and identity details out of that public record.
