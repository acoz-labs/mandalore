# Discovery: compatible withdrawal and retention preview

- Status: owner-selected direction; not an implemented feature or accepted migration.
- Issue: #81, selected for investigation by the current owner roadmap goal.
- Basis: `d7690a55315a7f7c1ce69c3df9bee895a45b1b79`.
- Authority: engineering self-review under ADR0003; material data-model choice
  returns to the owner before implementation, as required by #16 O3.
- Private evidence: none. All proposed probes use fresh synthetic banks.

## Owner decision

On September 18, 2026 the owner approved the recommended explicit, opt-in upgrade
within the existing signet/repository after reviewing the compatibility cost.
Preserve signet identity, memory, journals and Git history. Older clients must
refuse an upgraded bank; offline old copies and past context cannot be revoked.
This authorizes building and testing the feature, not migration of a personal
bank, live installation, deletion, automatic expiry, history rewriting, product
acceptance or public release. The separate-new-bank alternative is not selected.

## Problem to resolve

Withdraw a stable record from ordinary current recall without rewriting its
history, turning an ordinary correction into a privacy decision, or letting a
concurrent writer restore it accidentally. Restore must be explicit. Retention
preview must be inspectable and non-destructive, with no default TTL.

The original #16 discovery was reviewed at
`b0e6880bc5f1e6f2085e26a093625ba69e2a0abb` (PR #78, O3). Its parked sequencing
is superseded by the present owner goal, not by new authority to migrate a live
bank or rewrite Git history. #80 now provides derived reports; withdrawal must
compose with that contract rather than introduce a second export engine.

## Verified present behavior

`memory.Open` validates `signet.json` before opening a service. Current format is
1; `validateSignetMetadata` rejects unsupported versions. Current heads derive
from the complete supersession graph, not timestamps. Every record sensitivity
label remains eligible; extensions are not behavior-bearing policy.

`sync.validateCandidate` validates fetched portable objects in a disposable
directory before adoption. `appendOnly` protects `signet.json` as well as memory,
provenance and registrations: an ordinary synchronization cannot silently modify
the manifest. A format transition therefore needs a separately reviewed,
explicitly authorized upgrade path, not a relaxed general append-only check.

The [released-reader probe](old-reader-probe.mjs) ran in the designated native
Herdr lab on September 18, 2026. [Retained results](old-reader-result.json) pin
the real 1.1.0 binary and source identity. Direct recall/scopes/history/journal/
inspect/sync refused a disposable manifest-version-2 fixture without disclosing
the canary or changing portable files. A local bare-remote fixture showed that
old sync refuses adoption in `state: conflicted`, `phase: validate`, with
`delivered: false`, preserving the original HEAD and portable tree. Subsequent
old-local recall still returned the old canary, demonstrating the stale-copy
boundary rather than claiming revocation.

The first probe correctly reached remote validation but its assertion expected
an outer protocol error. Inspection confirmed sync deliberately returns an outer
successful envelope containing the conflicted delivery state. The corrected
assertion checks that state, non-delivery and unchanged HEAD/tree; it passed on a
fresh fixture. No validation rule or released binary was changed. Original test
fixtures are retained outside the repository. This is format-gate characterization,
not proof that an actual format-2 migration or withdrawal implementation exists.

## Candidate options

| Option | Assessment |
| --- | --- |
| A flag in unrestricted extensions | Reject: old clients ignore it and keep disclosing the record |
| A special correction/body/label | Reject: changes meaning, preserves old disclosure routes and cannot reliably signal old readers |
| New mandatory bank format with explicit events | Recommend for owner review: old readers reject the bank; semantics are validated and portable |
| Machine-local suppression list | Reject as the product contract: another machine/tool can ignore it |
| Delete revisions or rewrite Git history | Out of scope; destroys evidence and still cannot erase external copies |

A version gate only protects readers that encounter the upgraded bank. An offline
old clone, a historical checkout, already-returned model context or an external
export can still contain the content. No local schema can revoke those copies.
An old client's refused sync is not permission to claim that its old local memory
is up to date. Compatibility documentation and refusal messages must say this.

## Proposed record model — subject to proof and owner choice

Use a new bank format with a separately validated append-only visibility event
graph per stable record ID. Each event identifies its action (`withdraw` or
`restore`), observed current content-head IDs, parent visibility-head IDs,
authorship, time and reason. Reasons need not repeat the withdrawn text. Empty
visibility history means visible legacy behavior, not an implicit restore.

Visibility event parents and content references must exist, name the same record
and form acyclic graphs. Writes compare expected content and visibility heads
under the normal writer lock; stale expectations fail without retry. No timestamp
selects a visibility winner. Invalid or incomplete graphs fail closed.

Proposed conservative resolution:

- A unique withdrawal head withholds all current content for that record.
- Multiple visibility heads withhold content as an explicit visibility conflict,
  even if all branches say restore. Resolving them requires a new decision that
  names every parent head and the current content heads.
- A unique restore permits only content it explicitly observed, or later content
  authored after observing that restore. This prevents a concurrent correction
  from hitchhiking on a restoration that never reviewed it.
- New-format content revisions therefore need validated visibility references.
  Existing immutable revisions are legacy revisions with no such references;
  their bytes must not be rewritten merely to add the field.
- Correcting a withdrawn record may preserve useful history, but does not restore
  it. A correction whose visibility context is stale can be retained after sync
  but remains withheld until explicitly reconciled.
- No evidence is deleted. Content conflicts and visibility conflicts are distinct
  inspectable states, neither resolved by wall-clock order.

The [executable design model](model.mjs) and [nine tests](model.test.mjs) exercise
these rules, including all 120 permutations of a five-object concurrent scenario.
They pass with Node 24.1.0. The model validates combined content/visibility causal
cycles, missing or wrong-kind references, cross-record inputs, stale expected
heads and replay. It demonstrates that a correction concurrent with restore stays
withheld until a new restore explicitly observes it; an aware later correction
can be current. Two concurrent restore heads are a visibility conflict, not a
timestamp tie-break. Existing content conflicts remain conflicts after restore.

This is a single-record state-machine proof, not a production schema validator,
file transaction, scale test or complete formal proof. Incomplete delivery
prefixes are invalid and must never produce guidance. Production content writes
must capture visibility heads under the writer lock; importing missing metadata
as though a writer had observed a restore would invalidate the argument.

Reproduce with `mise exec -- node --test docs/discovery/81-withdrawal-compatibility/model.test.mjs`.

## Surface contract to prove

| Surface | Proposed behavior |
| --- | --- |
| Ordinary recall, search scoring, native context | Never include withheld bodies/summaries or treat a visibility conflict as guidance |
| Scope inventory | Explicit visible/withheld/conflicted counts; scope IDs remain metadata, not a secrecy boundary; native routing must not insert withheld text |
| Explicit history | Preserve revisions/events and their provenance with clear visibility status; history is deliberate disclosure, not ordinary recall |
| Current-only export | Omit withdrawn/visibility-conflicted records, reporting IDs/counts in sensitive preview |
| Explicit historical export | Require an explicit include-withdrawn opt-in in addition to history; disclose status, never silently include withdrawn content |
| Exact journal selection/search | Journals remain independent evidence; record withdrawal cannot infer which journal text to suppress |
| Source provenance | Preserve independent source objects; current recall cannot leak them through a withheld record; explicit history may expose them |
| Foundling search/read | Original external reference remains available unless explicitly disconnected; no implied source erasure |
| Foundling promotion | Refuse a withdrawn target; prevent automatic re-promotion of the same exact known origin as a fresh record without explicit resolution |
| New unrelated manual save | Record withdrawal is not a global phrase ban or semantic DLP; do not pretend to detect every paraphrase/duplicate |
| Synchronization | Merge valid append-only events/content, then compute visibility; never use a merge clock as restore authority |

Withdrawal/restore should be explicit typed operations with read-only denial and
receipts distinguishing local durability, visibility state and remote delivery.
Ordinary automatic learning remains enabled for unaffected records. No new daily
approval mode is proposed.

## Retention preview

Require explicit scope/record selection and an explicit policy. An age criterion
must name the timestamp being compared and an absolute cutoff, not a hidden
default age. Report candidate record IDs, content/visibility heads, state, matched
policy, reachable source IDs and known provenance relationships without bodies.
Journal IDs cannot be inferred from text or scope: any selected journals are a
separate explicit set, with the lack of record relationships stated honestly.

The result is a bounded read-only preview, not deletion authority. It pins the
binding/source/policy and identifies unknown external copies. No scheduled TTL,
purge, automatic withdrawal or retention apply is selected by this discovery.
A later explicit withdrawal command must still validate its expected heads.

## Format-transition question for the owner

Recommend an explicit format upgrade, never on ordinary launch/recall/sync.
Preserve signet identity and every existing evidence blob. A reviewed transition
could append an upgrade receipt and change only the manifest version; upgraded
sync would need a narrowly validated exception for this exact monotonic transition,
not permission to edit old records or downgrade the format. Existing format-1
banks continue normal operation until their owner elects to upgrade.

The alternative is creating a separate new-format bank with explicit import,
leaving the old bank/remote untouched. That avoids a mixed-client transition but
introduces a second bank and a deliberate continuity handoff. Neither path is
authorized against a live bank by this investigation. The choice is material and
must return to the owner with compatibility evidence and operational tradeoffs.

### Recommended upgrade recovery contract

Subject to that decision, prefer the same signet/repository over a second bank:
it preserves established scope IDs, bindings, Git history and remote continuity.
Require explicit preview/apply plus acknowledgement that affected writers are
stopped. Preview pins source/binding/HEAD and the proposed monotonic transition.
No implicit upgrade during launch, recall, export or background sync.

Preserve all existing revision/source/journal bytes. New-format readers support
legacy records within the upgraded bank, while new writes carry the required
visibility references. A portable upgrade record precedes the manifest flip;
the final manifest replacement is atomic under the writer lock and occurs only
after the prepared new state validates. Do not allow withdrawal writes before
that format gate is active. These transaction details require implementation
fault tests, not inference from the state-machine model.

Interrupted pre-flip preparation leaves old semantics plus inspectable pending
metadata; it cannot claim withdrawal succeeded. Post-flip failure leaves a valid
new-format bank or an explicit fail-closed repair condition, never an automatic
downgrade. Git checkpoint/delivery and local format activation are separate
receipts. No history rewrite, force push or automatic cleanup is permitted.

Upgraded synchronization must allow only the reviewed 1-to-2 transition with
unchanged signet identity, exact preserved evidence and a valid upgrade record.
It must not weaken append-only enforcement for ordinary records or permit 2-to-1
downgrade. A format-1 clone needs explicit transition consent before adopting a
format-2 remote; old clients retain their generic conflict/refusal behavior.
Offline-old-clone recall remains outside the withdrawal guarantee until upgraded
and synchronized. No automatic ability to enumerate or stop all machines exists.

## Integration and implementation acceptance map

| Integration point | Required proof before shipping runtime behavior |
| --- | --- |
| `memory.Open`, manifest/schema validation, runtime metadata | Format-1 unchanged; unsupported/new format refused by old readers; no pre-gate visibility mutation |
| `Service.Remember` and `RememberFromFoundling` | Capture visibility context with expected-head CAS; stale write cannot fabricate awareness of restore |
| `Store.Recall` / shared current-head selection | Withdrawn, visibility-conflicted and unreviewed-content states excluded before scoring/output |
| `Store.Scopes` / `Service.ScopePage` | Counts agree with recall; no withheld summaries in routing; legacy scope IDs retained |
| `Service.HistoryPage` plus explicit visibility history | Full content/events retained with distinct status, bounded output and provenance |
| `memorycontext.Build`, Codex hook and Pi startup/turn context | Same exclusion after fresh reads and after sync; no stale cached guidance claim |
| `exportreport` snapshot/projection/preview/apply | Visibility files validated/pinned; default exclusion and explicit withdrawn-history opt-in; stale plan refuses |
| `foundlings.Manager.Promote` | Withdrawn target and same exact known origin cannot silently create fresh guidance |
| Journal and source APIs | Independent evidence stays accessible explicitly; no invented record association or erasure claim |
| Sync path allowlist/candidate validation/append-only checks | Monotonic consented transition only; all concurrent content/events validated before adoption; old-client and no-downgrade tests |
| Typed API/MCP/CLI and native tool discovery | Strict schemas, bounded metadata-only retention preview, read-only denial, partial/delivery receipts; no automatic expiry |

Materializing these implementation tasks follows the recorded owner decision and
exact-head discovery review. Passing the design model does not close runtime
acceptance criteria or authorize a production migration.

## Required evidence and completion gates

1. Run a known old released reader against an unchanged synthetic format-1 bank,
   then a disposable manifest-version fixture. Prove ordinary memory operations
   refuse unsupported format without returning the content or writing repairs.
2. Prove old synchronization refuses a new-format remote and preserves its old
   local evidence. Explicitly demonstrate the stale-clone limitation, not hide it.
3. Executable causal model: correction/withdrawal/restore races, divergent restore
   heads, out-of-order union, missing parents, repeated requests and restoration
   requiring the current content/visibility heads. No time-based winner.
4. Map every surface above to a concrete integration point and acceptance test.
5. Define upgrade/import recovery, interrupted transitions and old-client refusal;
   bring the material data-model/transition choice to the owner before runtime
   implementation. Engineering self-review is not that product choice.

Keep this pack marked investigatory until those proofs and the decision are
recorded. No runtime policy change, live migration, new native plugin, installed
binary replacement, cleanup of personal data or release is part of these probes.
