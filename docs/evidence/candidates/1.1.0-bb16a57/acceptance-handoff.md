# 1.1.0 acceptance handoff

Engineering reconciliation, September 16, 2026. **Ready for human review, not
accepted or published.** This document does not waive a criterion or supply a
product verdict. It maps the live contracts for #13, #56, #57, #61, #65, #74,
#85 and #88 to fresh evidence. The [candidate guide](../../../releases/1.1.0-candidate.md)
and original issues remain authoritative.

## Exact review target

- Source: `bb16a57eb16dcd5e3da1386d6eb72c64169be6b5`.
- Candidate: `mandalore:bb16a57eb16dcd5e3da1386d6eb72c64169be6b5:sha256:ab4af42cee70495b66f4a64b3d754dc0e83d50c789ac1d8b6c4a419e7ffcb986`.
- Retained [build](https://github.com/acoz-labs/mandalore/actions/runs/35060663575),
  attempt 1, artifact `10432790497`; archive SHA-256
  `417b331ea598bb312744900eccd67e96ca458cd3fe7bf7d27e47c7b1c61b380b`.
- [Nomination](https://github.com/acoz-labs/mandalore/actions/runs/35060900748)
  and [issue marker receipt](nomination.json) identify all eight issues.
- [Payload verification](verified.json) checks all eight assets; [package
  inspection](packages.json) identifies runtime and both native packages.
- Exact-source [CI](https://github.com/acoz-labs/mandalore/actions/runs/35060517617)
  and [main audit](https://github.com/acoz-labs/mandalore/actions/runs/35060517620)
  succeeded. Artifact availability was rechecked September 16: not expired,
  expiry December 15, 2026 at 05:43:33 UTC. Recheck before acceptance/promotion.

The evidence branch contains documentation only. It is not a rebuilt candidate.
Use the retained payload, not a newly compiled evidence-branch executable.

## Original criteria reconciliation

| Contract | Fresh engineering evidence | Human review and remaining limits |
| --- | --- | --- |
| #56: faithful single representation, compatibility, correctness and boundaries | [Rendering receipt](rendering.json), [delivery model](delivery-model/README.md), [retrieval](retrieval/README.md): native error-first, current/history, pins and partial-delivery envelopes match complete wire objects. Selector fixtures cover text-only/distinct-content fallback. Wire, selected text and native token costs are reported separately. | Verify understandable outcomes and preservation of errors/provenance. Fixture fallback coverage is not a native run of every variant. No universal whole-session context reduction is claimed. |
| #61: direct cue, useful learning, inert quotations, restrictions, offline/missing connection | [Continuity](continuity.json) and [nine model scenarios](delivery-model/README.md): pending delivery without filler, useful consolidation, local-only/read-only, quoted cue, offline and absent connection. Actual saved content and Git heads checked. | Repeat the hands-on matrix. Negative-run Git metadata changed before model start through the observer; byte-level and semantic no-effect evidence passes, but whole-inventory immutability does not. |
| #65: ordinary companion selection, local-only compatibility, truthful failures, single bounded attempt | [Delivery boundary receipt](delivery.json), [native lifecycle](lifecycle.md), [model comparison](delivery-model/README.md): exact identity, cancellation, contention, concurrency, ambiguity, recovery and schema checks. Combined save removed one mutation call and the save-to-sync decision gap. | Total model rounds and context did not decrease in the measured comparison. Do not claim an atomic transaction, guaranteed delivery or deterministic lifecycle sync. |
| #57: bounded initial retrieval, provenance, expansion and answer quality | [Retrieval](retrieval/README.md) and [four rendered journeys](retrieval-ui/README.md): later passage, negation, multi-document pagination, full review, current/history distinction, stale refusal and no-write checks. Before/after payload and native context measured independently of #56. | Two inefficiencies remain: repeated 512-byte preview in an earlier candidate run and one empty EOF read. The controlled comparison does not establish overall context savings. Human judgment must assess the efficiency outcome explicitly; do not silently count it as proven or erase counterexamples. |
| #13: thin Pi integration, continuity, lifecycle, isolation and recovery | [Initial native](initial-native.json), [reverse continuity](continuity.json), [lifecycle](lifecycle.md), [eight installation journeys](pi-install/README.md), [three display journeys](pi-display/README.md). Real profile registration, unique tools/skills, fresh context, retained generations, interruption and repair verified. | Agent-driven native checks are not human acceptance. Native cache/directory effects are disclosed; no copied authentication or whole-native-home immutability claim. Claude remains gated on Pi acceptance. |
| #74: request budget, verification safety and truthful quota recovery | [Quota matrix](quota/README.md): plan/apply 13 requests, completed replay zero, verifier 13 with all assets verified; safe explicit retry and header/security regressions; eight rendered quota/refusal/partial journeys. | Quota injection uses test drivers compiled from exact clean source, **not the retained executable**. Review this distinction explicitly. No live public-quota exhaustion, unconditional retry or borrowed credentials. Narrow command copyability #77 is not resolved. |
| #85: non-executing readiness, typed uncertainty, CLI/menu parity and separate native handoff | [Readiness](readiness/README.md): eleven static cases, seven retained-binary recordings, execution/network traps and complete fixture inventories. Both harnesses, malformed/stale/unsupported/partial, prompt, default-No and cancellation covered. | Alias-path diagnostic limitation retained. Support declarations/cross-builds are not native certification. Static assessment and explicitly selected native checks have different effect boundaries. |
| #88: immutable candidate, compatibility, update/recovery and unchanged release gates | [Root identity/evidence index](README.md), [upgrade/recovery](upgrade/README.md), Pi installation and display above. Real isolated published 1.0-to-retained-1.1 updates, partial activation, same-plan recovery, replay, rollback and re-upgrade preserve protected state. | Preparation does not publish, activate personal connections, accept issues or complete the broader roadmap. Keep release-bearing issues open. |

## Practical acceptance sequence

1. Identify an authorized human acceptor under ordinary author-separation rules.
   The MVP-only owner exception does not cover this candidate. If the owner wants
   to perform acceptance despite matching implementation-account authorship,
   request a new explicitly bounded authority decision first; do not change
   workflow flags, actor configuration or policy speculatively.
2. Verify the retained target above and create disposable signets, native profiles
   and local Git origins. Preserve personal banks, sessions and credentials.
   Follow the full seven-part matrix in the candidate guide, not a replacement
   combined smoke test. The engineering recordings identify fixtures, expected
   outcomes, effects and recovery paths for each journey.
3. Have the human exercise and judge the affected native/model-facing journeys:
   passive cross-harness learning; current/history retrieval; explicit cue and
   restrictions; clear durable-save versus delivery status; Pi setup/recovery;
   quota recovery; readiness and update/rollback. Retain fresh openable evidence
   bound to these exact bytes. Review each limitation above rather than assuming
   a passing test gives product approval.
4. Record an actual `approved` or `changes-required` verdict with criteria,
   limitations and durable evidence through **Product acceptance**, for each
   nominated issue under the workflow's real authorization checks. Do not use
   engineering self-review as that verdict. Rejection returns affected work to
   implementation and requires a new candidate if product bytes change.
5. Only after acceptance and explicit publication authority, promote the same
   retained bytes through the release workflow and verify downloadable assets
   and issue ledger. Personal runtime/plugin activation remains a separate choice.

## Unfinished roadmap, not hidden candidate completion

- #55 O1/O2: lifecycle authorization/checkpoints remain unresolved; model-selected
  delivery is not deterministic compaction/exit synchronization.
- #80/#81: export and withdrawal remain deferred; privacy documentation alone
  does not implement them.
- #77: narrow recovery-command copyability remains open.
- #19: CI operates, but the historical trigger cause is not proven.
- #14: Claude Code follows Pi acceptance, not this engineering checkpoint.

Native evidence covers the recorded macOS arm64 environment, Codex 0.154.0,
Pi 0.85.1 and Node 24.1.0. Other architectures/OS targets are cross-built, not
native-accepted. Keyboard, plain/no-color and narrow-terminal checks do not
certify screen readers, alternate fonts or locales. No such verdict is implied.
