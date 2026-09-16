# Privacy and the lifetime of memory

Mandalore keeps useful knowledge in your own local, Git-backed signet. Ordinary
confirmed learning remains enabled; no special phrase is required. It is not a
credential vault, encryption layer or promise to preserve every conversation.

Three different questions matter: **is it current, was anything saved this turn,
and does any copy still exist?** Correction, read-only use and erasure answer
different questions. This guide describes current source behavior, not new
privacy controls added to the immutable v1.0.0 release.

## What is stored and where it goes

| Surface | Contents and exposure | Limits |
| --- | --- | --- |
| Portable signet | Plain JSON records and every revision/branch; source reasons/citations; semantic journals; signet identity; device labels/IDs; actor/harness and optional model/session provenance; foundling registrations | Structural validation is not encryption, secret detection or recipient authorization |
| Git | Checkpoints include valid portable data, including historical revisions and journals. Commits also contain Git author metadata | Authorized sync delivers to the configured origin; copies can persist in other clones and backups |
| Machine-local state | Ignored `.mandalore/` lock, sync receipt and foundling connections; separate bindings and native installation configuration | Mandalore checkpoints exclude these; external backup tools or ordinary Git use are outside that guarantee |
| Ordinary recall | Current non-conflicting heads in the explicitly selected scope; bounded count/bytes | Sensitivity labels do not filter content. Conflicts and omissions must be inspected, not silently resolved |
| Native context and tools | Prompt integration supplies bounded bank-wide evidence and scope routing; tools can read selected scopes, history, journals and foundlings | Returned content is available to the invoking harness/model, including during read-only tasks |
| Foundlings | Portable registration/pin and promoted citations; original reference files stay outside the signet; local connection holds the machine path | Retrieval does not clone/fetch or import everything. Promotion explicitly saves adapted memory |
| Native sessions/providers | May retain prompts, returned memory and tool results under their own settings | Mandalore cannot retract already-returned text, erase native sessions or control provider retention |

Scope IDs, timestamps, summaries, reasons, citations and device labels may reveal
information even without a record body. Do not assume metadata is safe to publish.
Git checkpoints use `Mandalore <mandalore@localhost>` by default; an explicit
repository-local name/email pair overrides that attribution. Record provenance
is separate. See [synchronization](synchronization.md).

Mandalore creates bank directories/files with restrictive permissions, but the
contents remain plaintext to authorized local processes. Existing directories,
Git clones, OS encryption, other applications and backups have their own controls.
Separate signets isolate memory routing, not unrestricted computer access.

The current search engine reads text lexically. There is no persistent vector or
search-index store to erase. A future index must document its storage, rebuilding
and invalidation before claiming deletion support. Native prompt hooks use prompt
text transiently for bounded local matching; they do not read transcripts or save
that prompt. This is not a claim that the surrounding harness never logs it.
The examined memory components have no telemetry collector; explicit Git transport
and release lookups are separate network operations, not a private-memory telemetry
stream. See [format](signet-format.md) and [interface](interface.md).

## Labels, learning and read-only use

`public`, `private`, `sensitive` and `restricted` are descriptive record metadata,
not access controls, encryption, expiry policies or DLP. Every value is eligible
for recall. Compact recall results do not include the sensitivity field. Journals
have neither a sensitivity label nor an entity-scope field. Do not save a secret
and rely on `restricted` to hide it from the model.

No-save/read-only direction prohibits saves, journals, checkpoints, Git
initialization and synchronization within the task's scope. Explicit no-sync can
allow a local save without delivery. Neither direction means existing content is
hidden from recall or from the native model receiving its result.

An enforced read-only connection rejects mutating API operations before input
decoding. A conversational instruction is interpreted by the agent; it is not a
universal sandbox against independent filesystem tools or another writable
connection. Existing startup hooks perform local reads; this does not make an
otherwise writable session read-only. Normal authorized learning is unchanged.

## Correction, disconnection and retention

Superseding a fact changes which revision is current. It preserves predecessors,
reasons and provenance; journal entries containing the earlier fact remain
independent evidence. Both old and new versions can still be in today's signet
files, Git commits, remotes and backups. Competing heads are not resolved by time.

Disconnecting a foundling withholds source access through its connection while
preserving original files, registrations, local connection metadata and already
promoted memory. It does not recall earlier tool results from a native session.
See [foundlings](foundlings.md).

There is no automatic age-based expiry, memory-forget operation or secure-erasure
command. Memory history is retained indefinitely by current behavior, subject to
external storage actions; that is not an archival durability guarantee. Removing
a required predecessor invalidates the graph. Even otherwise valid deletion of
committed evidence is refused by append-only synchronization. Editing/removing a
file is therefore neither supported retention nor proof that Git history is gone.

Git can retain objects through branches, tags, index entries and reflogs. Immediate
pruning can race writers; garbage collection is not a remote-erasure mechanism.
See [Git's garbage-collection documentation](https://git-scm.com/docs/git-gc).

## Accidental secret or sensitive-content save

1. If it is an active credential, revoke or rotate it through its provider first.
   Do not reproduce the value in chat, issues, cleanup arguments or a new journal.
2. Stop further use/sync of affected material in the explicitly selected scope.
   Identify records/events and reviewed local paths without printing their values.
   Do not shut down unrelated banks or claim a conversational request has stopped
   every native process.
3. Account separately for working files, revisions, sources, journals, Git
   refs/objects/reflogs, remotes/clones/forks, backups, exports, foundlings and
   native session/provider copies. Mandalore cannot enumerate every copy.
4. Review any cleanup's exact targets, backup exposure, recovery and coordination
   requirements before acting. A backup preserves recovery and the unwanted data.
   History rewriting conflicts with normal append-only sync and needs a separate
   design and explicit authority; it is not an Armorer repair step.
5. Report verified local effects separately from unverified remote/third-party
   effects. Never promise “forgotten everywhere.”

GitHub advises credential rotation first and explains that rewritten history may
remain in clones, forks, cached views and pull-request references. Old copies can
reintroduce it; some cleanup requires coordination or provider support. Follow the
provider's current [sensitive-data guidance](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository),
not a generic force-push/prune recipe. This is operational guidance, not legal,
compliance or physical-disk sanitization certification.

## Designed future operations — not available commands

These are deferred contracts from [#16's reviewed discovery](https://github.com/acoz-labs/mandalore/blob/b0e6880bc5f1e6f2085e26a093625ba69e2a0abb/docs/discovery/16-privacy-lifetime/README.md).
They do not authorize action on a user's bank and are not features to invoke now.

### Scoped export and derived redaction — #80

A readable report is different from a restorable backup. Git-backed portability
retains original history; an initial report must not impersonate a signet clone
that could reconnect or merge as its source.

- Preview explicitly selects one bound signet and exact record IDs/scopes.
  Default: current unambiguous heads, with counts for conflicts/omissions.
  History, detailed citations and exact journal event IDs are separate opt-ins;
  do not invent journal scope from keyword matching.
- Preview is read-only: counts, IDs, digests, categories, bounds and destination
  effects, not bodies or a written export. Even this manifest may be sensitive.
- Apply requires unchanged reviewed source/binding and a fresh destination
  outside the bank, operational state, foundling roots and existing files.
  Recheck for drift, use restrictive permissions/no overwrite, and disclose any
  retained partial artifact after failure. Source invariance is not permission
  to write an artifact during a no-save/read-only task.
- No network delivery, Git history, credentials, local connections, transcripts
  or copied foundling contents. Explicitly review metadata dependencies; a
  current-only report must not silently smuggle in historical source content.
- Redaction affects only the derived report. Review bodies, summaries, reasons,
  citations, labels, IDs and journal content; omit dependent metadata or mark
  provenance redacted rather than claiming complete evidence. Disclose retained
  unredacted copies. Never infer source deletion or promise “safe to publish.”

Before implementation, [#80](https://github.com/acoz-labs/mandalore/issues/80)
must settle the consumer/format and provenance-disclosure questions. Verification
must cover selected-only output, history/journal opt-ins, conflicts, source
invariance, every sensitive field, stale inputs, unsafe/overlapping destinations,
partial writes and denied actions. No automatic secret-detection guarantee.

### Recall withdrawal and retention preview — #81

A future withdrawal targets stable record IDs and expected heads; its reason
need not repeat content. It should be portable and append-only, hide withdrawn
content from ordinary recall/context, preserve explicit history and support an
explicit restore decision. It is not factual correction or global deletion.

Older clients must not silently ignore withdrawal and return the content.
Concurrent withdrawal/correction/restore needs an inspectable conflict rule, not
a timestamp winner. Scope counts, history, export, foundling re-promotion and
independent journal/source content all need specified behavior. Do not smuggle
new semantics into an unconstrained extension field.

Retention preview may enumerate affected IDs and relationships under an explicit
user policy. Age alone never grants deletion authority. No default TTL is chosen;
an initial retention action, if later selected, should be semantic withdrawal,
not destructive purge. [#81](https://github.com/acoz-labs/mandalore/issues/81)
requires compatibility/concurrency design and material data-model review first.

## Evidence and limits

[Privacy regression evidence](evidence/privacy/README.md) records synthetic
service/API/Git tests and their limits. They prove specific storage and denial
behavior, not third-party erasure or protection from a filesystem owner.
No user signet, real credential or native provider transcript was used.
