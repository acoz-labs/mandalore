# Native two-machine continuity and synchronization recovery

Fresh contributor engineering evidence against retained source
`5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
Both physical macOS ARM64 machines used executable
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
The same synthetic signet and SSH Git transport from the
[earlier CLI checkpoint](../physical.md) were reused deliberately; thread contents
and credentials were not transferred.

## Native installation and trust

Machine B's existing Codex was 0.154.0, SHA-256
`4f85982624b3898c8991cb80c0981b2aa71070e3537046c9a95950318a95afcc`.
Its native login-status command reported credentials present; that was only a
readiness check until actual subsequent model requests succeeded. Its initial
installed-plugin listing contained no known memory integration. Official
[CLI listing/authentication guidance](https://learn.chatgpt.com/docs/developer-commands?surface=cli)
was consulted through the OpenAI Docs skill before operating native commands.

A read-only managed connection plan selected only the existing synthetic binding,
retained candidate, current native binary and a fresh fixture-owned state directory.
The redundant test-driver attempt to use a system `realpath` executable failed
before planning because that binary was absent; the toolkit's own canonical path
resolution handled the explicit native path successfully. No product patch was
required. The reviewed plan then installed generation
`eae20bf3289caf655519fa4f5c2a1201c2ff59fe2c1130b65f3a24ad886507a3`.
[Apply](apply-projection.json) and [doctor](doctor-projection.json) retain content
identities and status while omitting local paths. All signet files and existing
root native instructions/hooks matched their before/after hashes. Native
registration, owned package cache and trust state were changed explicitly; this
is not a claim that the entire native profile was untouched.

Four hooks initially needed review: the two Mandalore hooks and two unrelated
Stop hooks. Only Mandalore's exact SessionStart/UserPromptSubmit entries were
reviewed and trusted. The unrelated entries were left untrusted. A fresh session
showed all three installed SessionStart and both UserPromptSubmit hooks active,
while the two unrelated Stop entries still required review. No blanket hook-trust
bypass was used. YOLO mode and the owned empty project-directory selection were
authorized test choices; no native authentication or global SSH trust was changed.

Machine A retained Codex 0.153.4, SHA-256
`b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.
Its update prompt was skipped to preserve that measured version. For these peer
tests only, an invocation-local binding override selected A's existing clone;
its original managed default and original test bank were not replaced. Native
resources and each machine's configured model were inherited: B used gpt-5.6-sol
high and A used gpt-6-astra high. This is not a controlled model comparison.

## Actual native flow

1. B's first native session received a read-only question without the expected
   answer. It called Mandalore and returned **Shared Horizon**. Every signet file
   still matched after session exit; no memory/journal/sync write occurred.
2. A fresh B session received ordinary confirmed direction that the project was
   now **Quiet Meridian**, without the special cue or a prescribed tool sequence.
   It corrected the existing record, appended one concise decision journal entry
   and delivered the new Git head. [History](history.json) retains four revisions,
   the exact predecessor and B's existing device with `harness: codex`.
   [Journal](journal.json) records the change without a transcript. Unknown model
   and session fields remain null; they are not fabricated from observer knowledge.
3. The test operator explicitly synchronized A's clone and verified delivery.
   A fresh native session then received the same read-only question, with no name
   supplied, and returned **Quiet Meridian** through actual Mandalore recall.
   All A signet file hashes matched before/after that read.
4. A controlled external CLI writer on B corrected the same record to
   **Morning Harbor** and delivered it. The operator's shell command printed no
   replacement name into A's native context, and verified that A's clone was still
   unchanged. The already-running A session was then asked to synchronize and
   discover the update, with semantic writes and journaling forbidden.
5. Fetch failed because that native launch lacked the fixture-local SSH host-trust
   wrapper used in prior CLI checks. The agent explicitly reported the old local
   value as unverified, a failed fetch and a local checkpoint—not successful
   delivery. The [pending status](agent-fetch-pending.json) preserves the actual
   failure and unchanged old head.
6. A fresh A session inherited the same verified test-local SSH wrapper on PATH;
   no global trust setting changed. It performed Mandalore synchronization,
   fetched the actual external head and recalled **Morning Harbor**, with history
   inspection. [Recovered status](agent-fetch-recovered.json) binds success to
   `2ec630b0b9b2281b311c0a83ba2e6368a0c6d852`, matching B's delivered head.
   The model did not receive the answer in its prompt. Recovery required a fresh
   native process for the changed transport environment, not a hot reload.

Final CLI verification found exactly five revisions in the original record and
the same single journal entry; neither receiving session authored semantic data.
The fourth revision is B's native correction and the fifth is the explicit B-side
CLI fixture correction, both with the correct device and exact predecessors.
The original separate primary test bank also retained every file hash. All four
native test sessions are closed; the new B connection and synthetic fixtures are
retained for this ongoing verification effort, not activated as personal memory.

## Wording and evidence limits

The primary prompts were synthetic and contained no expected recall answer:

> Read-only portability test. Use Mandalore memory to recall the current name of my fictional project from the two-machine test. Report the current name and whether memory was actually read. Do not save, journal, synchronize, change files, or repair any hooks or settings.

The ordinary learning prompt was:

> The fictional project from our two-machine test is now called Quiet Meridian. The previous name is obsolete. This is a confirmed change for future conversations.

The cross-machine refresh prompt permitted synchronization but not semantic writes:

> The other machine has updated the fictional two-machine project. Please check Mandalore for its latest name. You may synchronize this signet to retrieve that change; do not create or change semantic memory records or journal entries. Report the name and what was actually fetched.

Recovery added only that process-local test SSH trust was now configured and
global SSH configuration must not change. The replacement name remained absent.

B's successful learning summary described synchronization as "across machines"
before A had fetched. The receipt proved remote delivery only at that moment;
the subsequent operator pull/native read established actual peer continuity.
That wording is an observed presentation rough edge for acceptance review, not
proof that every clone advances automatically. A's failure and recovery summaries
were precise about pending delivery, local evidence and the lack of a per-record
transfer count in the sync receipt.

The first peer read used an operator-initiated pull; the later recovery used an
actual agent-initiated sync. These are distinct observations. Read-only prompts
did not silently synchronize. No global SSH setup, new login, credential copying,
real-memory migration, plugin replacement or public release was involved. Native
Codex coverage on Linux/Intel macOS and simultaneous old/new memory-plugin
coexistence are not established by these tests. Native recordings remain local
because inherited resources contain private context; published receipts use only
synthetic memory, with path-bearing installation reports explicitly projected.
This is contributor verification, not independent acceptance.
