# Retained-candidate engineering follow-up

September 16, 2026. **Changes required before product sign-off.** This is
contributor verification, not a human acceptance verdict. The retained source
`b89458617b49d2eb0a8686dcd75c26aa7ea6f6ba` and manifest
`7995f820a7182a95b828851afce12bacea7108f82afac9369d307a43d961f45a`
were not rebuilt or modified. [Manifest](index.json) binds the fresh receipts
and recordings. All banks, projects, references and bare Git origins are synthetic.

## Findings that stop sign-off

1. [#99](https://github.com/acoz-labs/mandalore/issues/99): a plain terminal menu
   remains blocked after Ctrl+C until another input byte arrives. The automated
   [readiness recording](failures/plain-cancel-timeout.recording) timed out after
   120 seconds. A separate [direct terminal reproduction](failures/plain-cancel-direct.recording)
   showed the same behavior; Enter then produced exit130 and restored terminal
   modes. Full readiness fixture inventory remained unchanged. TUI cancellation
   and plain EOF passed, but do not erase this plain-input failure.
2. [#100](https://github.com/acoz-labs/mandalore/issues/100): a fresh Codex run read
   the packaged skill but treated absence of useful new content as a no-save
   instruction. It made **zero delivery attempts** on a direct consolidation cue
   with existing pending work. [Failure receipt](failures/consolidation.json)
   retains the exact prompt, answer, distinct heads and unchanged full fixture.
   The missing-connection case also omitted the expected explanation. These
   observations require clearer intent guidance; successful Pi delivery is not
   a substitute for this failing Codex observation.

Original failures are retained. They were not rerun until green or relabeled
as controller problems. Corrected source will need new immutable nomination;
the owner-acceptance exception is still scoped to the old exact artifact only.

## Fresh completed checks

| Surface | Observed evidence |
| --- | --- |
| Passive continuity | [Initial native](initial-native.json): separate banks give Thursday/Saturday; ordinary Pi learning selects one combined save-and-deliver; fresh Codex recalls Friday and history. |
| Correction and explicit cue | [Continuity](continuity.json): Codex saves Monday locally without journaling/sync; Pi recalls that correction; a direct cue delivers it once without filler or another-bank changes. Quoted cue causes no calls. |
| Live context and lifecycle | [Live refresh](live-refresh.json): one Pi session changes from River to Willow after an external confirmed update, with one bounded attachment and no accumulated custom messages. [Lifecycle](lifecycle.json) covers startup/reload/new/resume/fork, 18 unique tools, two skills, native tools retained and clean shutdown; [control](lifecycle-control.json) has no Mandalore resources. |
| Delivery boundaries | [CLI](delivery.json), actual [Pi callbacks](native-delivery.json) and [contention/ambiguity](native-boundaries.json): durable save versus delivery, invalid/read-only, no Git/origin, offline recovery, cancellation, concurrency, unresolved conflicting heads and inspect-before-retry without duplicate saves. Callback runs request no model. |
| Model-selected delivery | Eight operation-count checks passed: [ordinary](delivery-model/ordinary.json), [baseline](delivery-model/baseline-record.json), [journal](delivery-model/journal.json), [useful cue](delivery-model/cue-learning.json), [no-sync](delivery-model/cue-nosync.json), [read-only](delivery-model/cue-readonly.json), [missing](delivery-model/missing.json), [offline learning](delivery-model/ordinary-offline.json). The ninth direct-cue/offline case failed above. Missing-connection mutation refusal passed, but diagnostic wording did not. |
| Single representation | [Rendering](rendering.json): four actual Codex code-mode runs preserve complete wire envelopes, including two error-first cases, history and historical provenance; no extra wrapper printed. Twenty-one selector fixtures separately cover fallback/error/conflict cases. |
| Historical retrieval | [CLI](retrieval/cli.json) and seven native observations: [current/history/negation](retrieval/ordinary.json), [later passage](retrieval/late.json), [12-document pagination](retrieval/page.json), [full review](retrieval/deep.json), [stale source](retrieval/stale.json), [quoted command](retrieval/quoted.json) and [controlled baseline](retrieval/controlled-baseline.json). Source bytes/pins and full read-only inventories verified. |
| Quota recovery | [Eight recordings and request-count checks](quota/evidence.json): 13 plan/apply requests, zero completed replay requests, 13 verifier requests; known/unknown timing, refusal, partial state, default-No, Back, color/plain/no-color and narrow display. These are test drivers compiled from exact clean source, **not the retained executable**. No live quota exhaustion. |
| Readiness | [Ten initial static cases](readiness/static-initial.json) plus [stale](readiness/static-stale.json) preserve full fixtures and do not execute the dependency trap. Fresh retained-binary recordings cover configured Codex/Pi, details/prompt/default-No/separate native check, missing, malformed EOF, alias refusal and stale native failure. Native checks change only disclosed profile-directory mtimes. Unsupported classification rendered correctly, but plain cancellation failed. |
| Update/recovery | [Actual published1.0 and retained1.1](upgrade/evidence.json): isolated CLI/native update, intentional partial activation, same-plan recovery, no-effect replay, default-No, rollback and re-upgrade. Protected memory, binding, auth/session sentinels and prior native generations stay intact. Published cached assets were reverified, not newly downloaded or executed on other platforms. |

## Efficiency judgment remains bounded

Progressive search returned 1,337 source-text bytes versus 2,361 in the scripted
legacy comparison; envelope bytes were 4,100 versus 5,052. This measures the
initial payload, not total conversation context.

In the fresh native ordinary comparison both versions read the complete decisive
documents without overlap and used seven model requests. Candidate final-request
input was 33,937 tokens versus baseline33,795; foundling envelopes were 16,818
versus16,758 bytes. **This does not establish overall context savings.** The full
review and separate rendering case each repeated a 512-byte preview. No empty
EOF read occurred in this fresh matrix, but earlier counterexamples remain valid.

Combined delivery removed the separate mutation-to-sync decision in the ordinary
comparison: one companion versus separate remember/sync calls. That narrower
operation-count result is not a universal model-round/token/latency guarantee.

## Evidence boundaries and remaining work

- Native environment: macOS arm64, Codex0.154.0, Pi0.85.1, Node24.1.0. Model
  observations use the recorded native provider session in place; no auth copy.
- Recordings are BSD `script -qr` captures; use `script -p FILE` in a disposable
  terminal. Only synthetic temporary paths are retained. No private transcript,
  account, provider credential or personal memory is published.
- The color retrieval-menu controller consumed a collapsed previous selection
  as a new prompt and timed out at page three. Its raw failure stays private;
  it is not a passing recording or a product pagination defect. Remaining
  old-candidate rendered retrieval and Pi installation/display matrices were
  stopped after the product failures were confirmed. They are **not complete**.
- The quota controller's first no-TTY invocation never ran a product test;
  subsequent real-PTY evidence is separate. CLI verifier schema/exit corrections
  are recorded in the index. No failed native model run was replaced.
- No new human acceptance, publication, personal activation, screen-reader,
  alternate locale/font or non-host native certification. Previously accepted
  human setup/recall remains scoped to that observation; no repeat requested.

The source comparison from the previous candidate changes only Codex first-use
preparation/diagnostics plus tests/docs; embedded native package hashes match.
That comparison explains reuse of fixtures, not transfer of candidate verdicts.
