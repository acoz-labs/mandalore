# Razor Crest: self-hosted remote memory MVP

Status: proposed implementation; hosting direction selected; not deployed.
Tracking issue: [#146](https://github.com/acoz-labs/mandalore/issues/146).

## Outcome and selected direction

Razor Crest is Mandalore's optional bridge between a signet and authorized agents
running elsewhere. Owners should be able to recall and save knowledge in ordinary
web/mobile chats, then continue with the same knowledge in local coding agents.
A custom GPT-only workflow, remote control of a local agent, and uploaded memory
snapshots do not satisfy this outcome.

Start with a portable container on an existing always-on host. Evaluate Cloudflare
Tunnel plus Access Managed OAuth as the first ingress/authentication integration.
Keep a conventional cloud VM as a later deployment option using the same package.
No specific AI provider, hosting vendor, or home network is a core dependency.

Proposed product wording: "Razor Crest carries your knowledge wherever you work."
Compatibility must name supported clients, protocols and tested surfaces; this
wording must not promise universal or automatic recall in every application.

## Product and deployment boundary

| Layer | Responsibility |
| --- | --- |
| Mandalore core | Signet format, recall/save semantics, history/provenance, conflicts, withdrawal and synchronization |
| Razor Crest | Remote MCP boundary, validated identity integration, signet/operation authorization, lifecycle and delivery receipts, portable service packaging |
| Deployment integration | HTTPS ingress, OAuth provider adapter, secret injection, persistent storage, process supervision, monitoring and recovery |
| Private installation | Host identity, account IDs, domain, signet binding, credentials, access policy and backup destination |
| Agent application | Connection setup, user authorization, tool invocation, approvals and conversation behavior |

The public repository contains synthetic examples and reusable contracts. It must
not contain personal machine names, signet IDs, domains, credentials or policies.
Cloudflare may authenticate identities; Razor Crest remains responsible for
validating trusted assertions and authorizing each requested operation. Keep the
provider-specific validation behind a defined integration boundary.

## Proposed architecture

A remote MCP client reaches an authenticated HTTPS endpoint. In the first hosting
example, Cloudflare routes accepted requests through an outbound tunnel to the
Razor Crest service. The service calls existing Mandalore semantics against a
persistent signet replica with its own writer identity and permitted Git access.
Local agents retain their current replicas and connections.

Only the chosen service is routed. Administration stays private. A public ingress
hostname is reachable by outsiders, but memory access requires authorization.
Cloud providers do not inherit the user's phone VPN connection. The cloud agent
receives returned memory; the ingress provider is also a trusted intermediary.

Prefer a small HTTP adapter around existing services over reimplementing memory
operations. Inspect the current Go MCP SDK and runtime before selecting exact
transport APIs; this plan does not assert that the local stdio server is already
safe for remote multi-client use. Explicitly define supported protocol versions,
origin validation, cancellation and session isolation.

## Authentication and authorization contract

- Deny access by default. Bind a validated issuer/subject to permitted signets and
  read/write operations on the server. Never trust an agent-supplied owner name,
  signet ID, binding path or identity header as authorization.
- For the Cloudflare integration, validate the signed Access assertion including
  signature, issuer, audience and expiry. Reject missing or spoofed assertions.
  Restrict the private origin so callers cannot bypass the authenticated ingress.
- Use an MCP-compatible OAuth flow with controlled client registration/redirects,
  explicit owner authentication and appropriate MFA. Do not substitute a browser
  cookie login wall or require arbitrary custom headers from consumer clients.
- Separate client authorization from repository credentials. Tokens/passwords are
  not prompt content; inject runtime secrets without baking them into images.
- Specify read-only and writable behavior, grant revocation and refresh/expiry
  bounds. Do not promise instantaneous revocation until measured.
- Expose only required semantic operations. Administrative setup, arbitrary shell,
  filesystem editing and repository selection are not remotely callable by default.
- Minimize logs; redact tokens and memory bodies. Preserve enough operation IDs
  and receipts for diagnosing durability, delivery and denial without disclosure.

## Durable writes and synchronization

The replica and pending delivery state live outside the replaceable container.
An acknowledged local save and a successful remote Git delivery are different
outcomes. Return both honestly, including offline/pending/conflict states.

Define service-specific refresh/retry boundaries explicitly: existing local-agent
hooks are not a hosted scheduler. Serialize incompatible operations per replica;
keep requests and client identities isolated. Reuse conflict handling and never
select a winner by timestamp. Specify idempotency and ambiguous-response recovery
before enabling writes; clients must not repeat semantic saves to retry delivery.

Cancellation or loss of a response may follow a durable save. Retain and inspect
operation receipts before retrying. Exercise process termination between local
save and delivery. A successful restart must find undelivered changes and resume
safe delivery. Re-cloning the remote alone is not sufficient recovery for them.

Canon references remain read-only and session-scoped; design how authenticated
remote sessions map to the existing snapshot/freshness contract. Do not expose
transitive signets just because the service has credentials for them.

## Delivery sequence

1. **Feasibility and contracts.** Inspect runtime boundaries; test transport and
   OAuth discovery/login/refresh from two provider clients with a synthetic
   read-only tool. Record exact mobile/web surfaces, denied second-identity access
   and account/plan limitations. Choose the smallest compatible auth integration.
2. **Read path.** Add bounded remote scope/recall/history/freshness capabilities,
   identity/signets/read permissions, origin protection and request isolation.
3. **Write path.** Add authorized saves and journals with durable receipts,
   concurrency handling, explicit refresh/delivery and safe retry semantics.
4. **Packaging and operations.** Deliver container and deployment examples,
   persistent storage/secrets contract, health checks, upgrade/rollback,
   recovery procedure and a Cloudflare ingress recipe. Validate portability.
5. **Acceptance and release.** Independently verify an immutable candidate and
   publish the artifact/documentation through the repository's delivery process.
   Private installation is tracked separately from reusable product release.

Convert these slices into bounded linked delivery issues when their contracts
are sufficiently established. This planning task does not implement or deploy
any slice or create cloud resources.

## Verification and acceptance

Use synthetic signets and isolated credentials for acceptance. Run the repository
checks for implementation, plus focused tests of the service boundaries.

| Scenario | Required evidence |
| --- | --- |
| Two providers, including mobile | Same endpoint authenticates and returns expected current recorded values |
| Natural unprompted recall | Measure whether the host selects Mandalore; separate host behavior from protocol correctness |
| Cross-client save | Save in one cloud chat, successful delivery/refresh, recall in another cloud client and local harness |
| Identity and permission denial | Unauthenticated/second identity/forged/expired token, wrong signet and disallowed writes return no memory |
| Revocation | Previously authorized connection loses access within the documented enforcement window |
| Concurrent corrections | Conflicting heads retained and surfaced with original provenance |
| Withdrawal and references | Hidden records do not reappear; canon permissions and snapshot semantics preserved |
| Offline/cancel/retry | Honest partial receipt; no lost acknowledged local write or duplicate semantic save |
| Restart/replacement | Persistent pending state survives; safe recovery after interruption |
| Upgrade/rollback | Existing signet preserved; incompatible formats fail safely; secrets remain external |
| Existing local clients | Native integrations retain their behavior and authorization boundaries |

A tool connection is not proof of automatic memory usage. Record explicit recall
and natural tool-selection outcomes separately. If a client cannot support the
required flow, document the limitation and investigate a standards-compatible
alternative rather than weakening authentication.

## Operations and cost assumptions

The reference deployment uses existing compute, storage, domain and secret
management. No paid VM or provider snapshot service is inherently required.
Cloudflare free-tier/account eligibility and actual host capacity still require
verification. Home availability depends on power, internet and host uptime.

Recover software from versioned images and configuration, and re-provision
credentials from the owner's secret manager. Define independent protection for
signet data and pending writes; Git synchronization alone is not a complete
backup policy. Document the tolerated loss window and rehearse recovery.

## Open engineering decisions

- Supported MCP protocol versions and client-specific OAuth registration needs.
- Whether Managed OAuth meets the tested client's requirements; keep alternatives
  possible without coupling the core to Cloudflare.
- Minimum remotely exposed tools, per-operation permissions and identity mapping.
- Persistent request/idempotency receipts and synchronization scheduling/locking.
- Remote session mapping for canon snapshots and explicit refresh.
- Packaging resource requirements and restore/upgrade behavior under failure.

These are implementation investigations, not new mandatory owner approval gates.
No implementation or security acceptance is claimed by this document.

## Reference evidence

Vendor documentation was consulted on 2026-09-29; verify again during implementation.

- [Cloudflare Tunnel](https://developers.cloudflare.com/tunnel/): outbound connectivity to a private origin.
- [Cloudflare Managed OAuth](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/managed-oauth/): MCP authorization flow and signed origin assertions; currently beta.
- [Claude remote connectors](https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp): cloud-originating connectivity.
- [ChatGPT plugins](https://learn.chatgpt.com/docs/plugins): supported surfaces and host capability boundaries.
- [Docker volumes](https://docs.docker.com/engine/storage/volumes/): persistent container data.

Authoritative product criteria remain in issue #146. Vendor docs establish
available mechanisms, not tested compatibility or security of this proposed build.
