# Pi implementation reconciliation

Issue [#13](https://github.com/acoz-labs/mandalore/issues/13), planning PR
[#70](https://github.com/acoz-labs/mandalore/pull/70), implementation PR
[#72](https://github.com/acoz-labs/mandalore/pull/72). Engineering self-review is
authorized by ADR 0003; this document grants no acceptance or publication.
The PR's current reconciliation comment binds review and validation to an exact
head. The source-specific evidence below is not relabeled final-head evidence.

## Requirements and evidence

| Requirement | Implementation and actual evidence |
| --- | --- |
| One shared engine, native Pi tools | Dependency-free `plugins/pi/package` discovers the Go operation catalog and exposes its 18 bound operations through shell-free guarded CLI calls. Native loader/schema/callback checks retained native read/bash tools; CLI-only administration stayed excluded. See [package evidence](README.md). |
| Passive continuity and learning | Actual fresh Pi sessions recalled an ordinary scoped question, learned a confirmed changed decision without a memory cue, and delivered it. Codex recalled and superseded that record; fresh Pi recalled the correction and all three linked revisions. Direct consolidation journaled once; quoted cue did not write. See [model baseline](lifecycle-model-baseline.md). |
| Bank identity and provenance | Two synthetic banks/profiles remained isolated across working directories. Revision history retained stable device identity and fixture/Pi/Codex authorship. Guarded-binding and compiled adapter tests reject changed binding/runtime/package/root identity before writes. |
| Lifecycle and context bounds | Actual native startup/reload/new/resume/fork with an extension-disabled control, plus repeated-event regression. Per-turn context is fresh and bounded, not a stored message. An already-open actual model saw River superseded by Willow, with one attachment and no accumulated custom messages. See [refresh evidence](delivery-refresh.md). |
| Read-only and cancellation | Conversational read-only model turns preserved complete banks. Enforced read-only compiled calls reject mutations. Transport and compiled tests cover pre-dispatch cancellation, partial saved receipts, phase-observed cancellation and child reaping; native callbacks confirm those outcomes. |
| Offline/concurrent delivery | Native callbacks against two clones/local bare remote retained compatible writes and unresolved conflicting heads; unavailable-remote and interrupted-fetch cases preserved local save IDs. Explicit sync recovered without duplicate saves. Delivery is distinct from semantic agreement. See [delivery evidence](delivery-refresh.md). |
| Installation/update/recovery | Immutable synthetic candidates exercised real Pi install/remove, unrelated relative packages/filters, stale/foreign refusal, missing-asset repair, selected-runtime delegation and interrupted registration recovery. Old generations and memory were preserved. See [owned workflow](README.md#owned-connection-workflow) and [rendered matrix](menu/README.md). |
| Tool and setup UX | Actual terminal recordings cover color/no-color, narrow/plain, arrows/vim, Back/default-No/EOF/Ctrl+C, success/error/partial/recovery, native tools and unavailable-memory warning. Fixes make absent previous generation explicit and preserve bounded typed preview rejection reasons. See [recordings and limitations](menu/README.md). |
| Distribution compatibility | Pi embeds in the executable; the legacy version response and eight-file format-1 inventory stay unchanged. Two clean builds matched byte-for-byte. Published-source strict installer accepted the synthetic candidate. These are engineering artifacts, not replacements for published v1.0.0. See [distribution evidence](README.md#skills-and-distribution-slice). |

## Documentation promotion

| Temporary design material | Durable destination |
| --- | --- |
| Native architecture, lifecycle, guards and boundaries | [architecture](../../architecture.md), [interface](../../interface.md), [Pi integration](../../../plugins/pi/README.md) |
| Harness selection, preview/apply, ownership and recovery | [setup](../../setup.md), [runbook](../../runbook.md), Pi integration |
| Node/Go pins, adapter tests, embedding and format compatibility | [development](../../development.md), package tests and distribution evidence above |
| Memory and administration guidance | Pi's native `this-is-the-way`/`the-armorer` skills; neutral foundling/delivery references are tested byte-identical to Codex. Codex-only rendering guidance is not copied. |
| Verification, findings and platform limitations | This evidence directory, source-bound result files and actual terminal recordings |

The six-file temporary plan is removed after this promotion. Its reviewed
history remains in planning PR #70 and Git; it is not another permanent manual.

## Refinements and honest limitations

Native RPC resume/fork can emit repeated session-start notifications; the control
did too. The implementation retains native behavior and avoids duplicate tools,
rather than inventing event deduplication. Fixture observation errors were fixed
and repeated in fresh fixtures, not counted as successful product runs.

The Codex correction was durable locally, but that model reported a sync approval
restriction without an actual rejected tool call in its event stream. Do not
infer its cause or claim successful Codex delivery. A later explicit Pi sync
delivered the journal and preceding changes; separate two-clone native checks
establish delivery behavior.

Per-turn attachments, combined save/sync and shutdown cleanup are not lifecycle
checkpoint synchronization. #55 O1/O2 remain unresolved. Ordinary learning is
enabled by default; read-only is an explicit user/binding boundary, not inferred
from an ordinary question. No regex interprets cue strings as authorization.

Native results apply to Pi 0.85.1, Node 24.1.0 and macOS arm64. Four-target builds
are not native Linux/Intel acceptance. Screen readers, alternate locales/fonts
and a pixel-regression baseline are unverified. Existing native authentication
was used in place only for authorized synthetic model tests; none was copied.

## Remaining gates

Before PR readiness: complete exact-final-head rendered evidence binding,
current full/hosted validation and a separate exact-head engineering self-review.
Keep the issue open after merge for nominated immutable-candidate product
acceptance and release. Personal activation, publication and accepting a new
candidate remain separate authority gates. Claude #14 waits for Pi acceptance,
not merely this implementation merge. Published v1.0.0 stays immutable.
