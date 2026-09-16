# Discovery: privacy and the lifetime of remembered information

- **Status:** Proposed decision
- **Discovery issue:** #16
- **Discovery PR:** #78
- **Repository basis:** a8b56b268ed48f47363d015ab6d5f89e847bee6c
- **Recommended decision:** select O1; deliberately defer O2/O3
- **Gate 1:** exact-head contributor review required under ADR 0003
- **Confidence:** High on current storage boundaries; Medium on future operations
- **Private evidence:** none

## Decision sought

Make today's privacy and retention guarantees explicit and testable, then preserve
a bounded design for export/redaction and current-recall withdrawal. This is not
permission to inspect a private signet, migrate its format, delete remembered
information, activate a personal installation or rewrite Git history.

Select one delivery now: a durable privacy/lifetime contract, incident guidance
and regression tests of existing behavior. Keep export and withdrawal as explicit
deferred work, not fictional commands or unannounced retention policy. No privacy
claim should depend on users mistaking a sensitivity label for access control.

## Audience and critical tasks

People using the same bank across tools and machines need to know:

- What a save preserves, what travels through Git and what an agent/provider sees.
- What no-save/read-only prevents, without disabling ordinary useful learning.
- Whether correcting, disconnecting or "forgetting" actually erases information.
- How to review a future export before creating another sensitive copy.
- What to do after an accidental secret save without spreading it further.

## Evidence

[The reproducible synthetic probe](probe.md) and source inspection at the basis
establish these distinctions. The probe is characterization, not new behavior.

| Surface | Stored or exposed today | Boundary |
| --- | --- | --- |
| Portable signet | Plain JSON revisions, all superseded branches, source reasons/citations, semantic journals, device labels/IDs, actor/harness and optional model/session provenance, foundling registrations | Format validation is not encryption, secret detection or recipient authorization |
| Git checkpoint/delivery | Allowed portable files, including old revisions, journals and provenance; Git commits and repository-local author attribution | Append-only checks refuse evidence deletion/rewrite; sensitivity and ignore rules do not omit otherwise valid memory |
| Local operational state | Ignored writer lock, sync receipt and foundling connections; separate bindings and native installation configuration | Not included by Mandalore checkpoints; local backup tools and direct Git use are outside this promise |
| Ordinary recall | Effective non-conflicting heads in the selected scope, within count/byte bounds | All sensitivity levels are eligible; compact hits omit the sensitivity field |
| Native prompt context | Bounded bank-wide evidence plus scope routing; subsequent tools can retrieve explicit project/account/task scopes, journals/history or foundlings | "Read-only" means no Mandalore mutation, not no disclosure to the native agent |
| Foundling | Registration/pin/citations are portable; local source path is clone-local; selected text is read on demand | No source clone/fetch or wholesale import by retrieval; promotion writes an adapted record |
| Native session/provider | Returned context and tool results are available to the invoking harness/model | Mandalore cannot retract already-returned text, control provider retention, or erase native session files |
| Derived search state | Lexical reads; no persistent vector/index store in the examined implementation | Future indexes must declare data/location/lifetime and rebuild/invalidation behavior before claiming deletion |

Primary code: `internal/memory/{service,memory,journal,store}.go`,
`internal/sync/{checkpoint,candidate,remote}.go`,
`internal/memorycontext/context.go`, `internal/codex/hooks.go`,
`plugins/pi/package/extension.js`, `internal/foundlings/{source,connection,git}.go`
and `internal/api/{api,context,save_sync,sync}.go`.
Existing contracts: [format](../../signet-format.md),
[synchronization](../../synchronization.md), [foundlings](../../foundlings.md),
[security policy](../../../SECURITY.md).

Specific observations:

- `public/private/sensitive/restricted` are validated metadata values. There is no
  sensitivity-based retrieval gate. Journals have no sensitivity or entity-scope
  field. Source reasons, labels, paths, citations and summaries can themselves
  disclose information. No field should be presumed harmless.
- Created bank directories/files use restrictive permissions, but plaintext
  remains readable to authorized local processes. Git clones, existing parent
  permissions, filesystem backups and OS encryption are separate concerns.
- Supersession changes current recall, not the predecessor's files or journal
  copies. Removing a predecessor breaks graph validation; removing otherwise
  valid committed evidence is also refused by append-only synchronization.
- A foundling disconnection withholds source access, preserving registrations,
  local connections, original files and already-promoted memory.
- Runtime hooks do not read transcripts or journal the prompt. Prompt text is
  used transiently for bounded local matching. This does not mean the native
  harness never records prompts/tool outputs. No whole-system zero-logging claim.
- No memory telemetry collector, export/redaction/forget operation, automatic
  age-based expiry or secure-erasure mechanism was found in the memory surfaces.
  Release lookups and explicitly authorized Git transport are separate network
  operations, not evidence of a private-memory telemetry stream.
- Enforced connection read-only rejects mutations before input decoding.
  Conversational no-save is agent direction, not a universal sandbox against
  native filesystem tools or a caller choosing a different writable connection.

Git's own [garbage-collection documentation](https://git-scm.com/docs/git-gc)
explains that references including reflogs can keep objects reachable and that
aggressive immediate pruning can race writers. It is not a remote-erasure tool.
GitHub's [sensitive-data guidance](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
prioritizes credential revocation/rotation; rewritten history may remain in
clones, forks, cached views or pull-request references, and old copies can
reintroduce it. Sources checked 2026-09-16; no history cleanup was performed.

## Assumptions and unknowns

Ordinary confirmed learning remains enabled; users do not need a special phrase
or a new daily approval flow. No-save/read-only/no-sync retain their existing
scope semantics. Do not silently turn a descriptive sensitivity label into a
new refusal policy or automatically discard older experiences.

Unknowns for deferred work: which export consumers need restorable versus
readable data; acceptable source/provenance disclosure; journal selection when
entries span entities; old-client behavior after withdrawal; concurrent
withdrawal/correction conflicts; user-chosen retention periods, if any. Native
provider/session retention is externally controlled, not established by this
code survey. No universal legal, compliance or secure-erasure claim is made.

## Competing options

| Option | Decision |
| --- | --- |
| Document and test the actual lifetime now | Select O1; closes misleading expectations without changing learning behavior |
| Call supersession "forget" or Git deletion "erasure" | Reject; the original remains in multiple surfaces |
| Make sensitivity labels an implicit permission/expiry policy | Reject; would change existing memory behavior and leave journal/source gaps |
| Readable, explicitly scoped out-of-place export | Defer as O2 with the contract below; do not call it a sanitized backup |
| New portable recall-withdrawal metadata | Defer as O3 pending compatibility/concurrency design |
| Scheduled age-based pruning or automatic history purge | Not selected; would destroy useful history and requires new authority |
| Persistent vector/index layer for privacy control | Not needed; adds another copy without solving existing boundaries |

## Designed operation boundaries (not implemented commands)

### Export and out-of-place redaction

Separate a **readable report** from a **restorable backup**. An initial export
should be a versioned report, not a signet clone that could reconnect or merge
as the source bank. Restorable portability continues through the original
validated signet/Git workflow; it includes history and is not sanitized.

A future preview explicitly selects one bound signet and exact record IDs/scopes.
Default report content is current, unambiguous heads, with all omissions and
conflicts counted; no silent resolution. History, journals and citation detail
are separate opt-ins. Journals cannot be safely scope-filtered by today's schema:
select exact event IDs, not guessed entity associations.

Preview returns counts, exact selected IDs, content digests, categories included,
omissions/conflicts, size bounds and destination effects without dumping bodies.
It does not write a report, checkpoint or fetch. A manifest can itself identify
sensitive relationships; keep it local and do not label it non-sensitive.

Apply requires the unchanged reviewed selection and an explicit fresh destination
outside the bank, operational state, foundling roots and existing files. Recheck
source/binding and fail on drift rather than expanding scope. Publish a bounded
artifact with restrictive permissions and no overwrite; an interrupted attempt
must report any retained partial artifact, not imply rollback or retry blindly.
No network, Git history, credentials, connection files, transcripts or copied
foundling contents. Writing an export is a write even though its source is
unchanged; no-save/read-only task direction does not silently authorize it.

Redaction operates only on this derived report, with explicit field/record
omissions or replacements in the preview. Review every selected surface, including
summaries, reasons, citations, IDs, labels and journals; a body-only replacement
does not establish sanitization. Omit dependent metadata or mark provenance
redacted rather than claiming complete original evidence. No automatic "safe to
publish", secret-scanning guarantee, regex inference from pasted credentials, or
source-bank mutation. Keeping both unredacted and redacted artifacts creates two
copies; the receipt must say which remain. Deletion of originals is not cleanup
the exporter may infer.

Acceptance design: source file/hash invariance; selected-only output; explicit
history/journal opt-ins; conflict/omission accounting; malicious/overlapping
destinations; changed-source refusal; partial-write receipts; denied no-save;
synthetic sensitive values spread across every exported field; schema/version
and stable machine-readable preview. Native UI is not selected until this API
contract is shaped.

### Recall withdrawal and retention

Do not invent a correction such as "the old fact is secret" and call it erased.
A future withdrawal targets stable record IDs with explicit current heads and
a reason that need not repeat content. It should be append-only and portable,
hide withdrawn content from ordinary recall/context, preserve explicit history,
and distinguish withdrawal from factual correction. Restore is an explicit new
decision, not a timestamp winner. Concurrent corrections and withdrawal must
remain inspectable without inadvertently exposing a withdrawn value as guidance.

This needs a reviewed schema/compatibility strategy: older readers must not
silently ignore a tombstone and continue returning the content. Counts, scope
inventory, history, export and foundling re-promotion must have specified behavior.
Do not implement by smuggling a new meaning into an unrestricted extension field.

Retention preview may select candidate IDs by explicit user policy and report
affected revisions, sources, journals and references. Age alone does not grant
deletion authority. Initial retention action, if selected later, should be
semantic withdrawal rather than destructive purge. No default TTL is chosen.
History-preserving retention is indefinite today, subject to the user's external
storage actions; it is not an archival durability guarantee.

### Incident response and deletion

If a credential is accidentally saved, revoke/rotate it through its provider
first. Do not paste the value into chat, an issue, a cleanup command argument,
a new journal or a public diagnostic. Stop further use/sync of affected material
within the explicitly selected scope; identify copies by opaque record/event
IDs and reviewed local paths. No automatic global shutdown of unrelated banks.

A cleanup plan must separately account for working files, all revisions/sources/
journals, Git refs/objects/reflogs, remotes, clones/forks, backups, exports,
foundlings and native session/provider copies. Mandalore cannot enumerate or
erase every copy. Rewriting/removing evidence conflicts with current append-only
sync; do not use normal synchronization as a redaction transport.

Backups improve recovery but retain the material being removed. Any exceptional
rewrite needs explicit target/backup/recovery/coordination authority and a
separately reviewed design; provider-support steps remain provider-specific.
Do not prescribe force-push, reflog expiry or pruning as a routine repair.
The result must distinguish verified local removal from unverified remote or
third-party removal, never a blanket "forgotten everywhere."

## Success and stop signals

O1 succeeds when durable docs accurately describe the verified boundaries,
synthetic tests catch regressions, and no new runtime policy or native overhead
is introduced. Stop and return to the owner for irreversible cleanup, hidden
classification policy, private-data inspection or a material new trust boundary.
Native-session erasure, physical-disk sanitization and compliance certification
are not tests this work can claim.

## Candidate outcome map

### O1 — Current privacy/lifetime contract and regression evidence

- Disposition: selected.
- Outcome: publish a durable privacy guide linked from normal user docs; concise
  incident guidance; automated characterization of label exposure, supersession,
  journals/Git persistence and enforced no-save behavior. Preserve the operation
  designs above as future contracts, clearly marked unimplemented.
- Acceptance: accurate storage/exposure table; no false encryption/DLP/erasure
  claims; synthetic-only tests; no source/provider/private-bank mutations; no
  runtime changes or new startup context. Existing synchronization and foundling
  boundaries remain tested. Docs distinguish conversational intent from enforced
  connection mode and no-save from nondisclosure.
- Dependencies: this exact-head reviewed discovery; existing API and Git engines.
- Sequence: next, with a proportional planning PR.

### O2 — Reviewable scoped export and derived redaction

- Disposition: deliberately deferred.
- Outcome: implement the preview/fresh-destination report contract after shaping
  the consumer/format and full metadata-closure questions above.
- Acceptance: the export/redaction matrix above, including explicit omissions,
  complete sensitive-surface review and unchanged source. No restorable-signet
  claim, network delivery or source deletion.
- Dependencies: O1; separate solution/product design and explicit artifact writes.
- Sequence: backlog, not silently added ahead of approved #15.

### O3 — Compatible current-recall withdrawal and retention preview

- Disposition: deliberately deferred.
- Outcome: establish a portable withdrawal/restore model with old-reader and
  concurrency behavior; retain explicit history and no automatic destructive TTL.
- Acceptance: withdrawal respected by every ordinary retrieval surface; older
  clients cannot silently expose it; concurrent decisions not resolved by clock;
  readable history/restore; no implied erasure of Git, journals or external copies.
- Dependencies: O1; schema/compatibility design, decision on journal/source scope.
- Sequence: backlog; return material data-model decisions to the owner.

Global erasure/history rewriting is not selected or authorized by any outcome.
Do not create a delivery issue suggesting an automatic purge was approved.

## Privacy and evidence handling

Only disposable synthetic banks and local Git objects were inspected. The marker
is deliberately not a credential. Public evidence contains no user bank,
workstation path, service account, raw session transcript or authentication state.
The probe removes one test-created file only, verifies refusal and Git persistence,
and leaves live installations unchanged. Native process/provider retention is
documented as a boundary, not probed using private sessions.

## Decision Spotlight

"No longer current", "not saved this turn" and "erased from every copy" are three
different promises. Keep useful automatic memory and tell the truth about which
promise each operation can make.

## Gate 1

ADR 0003 permits a distinct exact-head contributor self-review and ordinary
engineering decision; it is not independent product acceptance. After CI and
review, merge this discovery and materialize selected/deferred outcomes with
immutable provenance. Promote the decision during O1 delivery, retire this
temporary pack, and keep deferred issues visible. Publication, personal activation,
destructive cleanup and new-candidate acceptance retain their existing gates.
