# Live reference adaptation and recovery checkpoint

Contributor-operated engineering verification for #10, not independent acceptance
or a release. These tests used the retained macOS ARM64 candidate identified in
[README](README.md): source `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`, executable
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
There was no product patch or candidate rebuild in this checkpoint.

## Native selective adaptation

A fresh Codex 0.153.4 session used the same installed connection, pinned binary,
gpt-6-astra/high model and YOLO mode described in [native.md](native.md). Before
launch, the test driver explicitly seeded an obsolete approval procedure and
registered the synthetic [historical source](process.md). Preparation was delivered
to the local bare fixture at `70c83e76c853a68e990a5fd883c904c1049b17d6`.

The user prompt confirmed autonomous routine reversible local work, asked to retain
the historical verification lesson and requested a remembered correction plus one
short journal entry. It prohibited project edits, source-command execution,
configuration changes and account access.

The agent recalled current memory/history, inspected the foundling, searched and
read the source, then promoted one adapted procedure revision with the exact prior
revision as predecessor. It appended one journal entry and verified the result.
The current project remained Nimbus Finch; the historical Lantern Otter name and
blanket approval requirement were not adopted. The source's inert command was not
executed and both synthetic project directories remained empty.

[Procedure history](live-procedure-history.json) preserves both revisions.
[Citation](live-incorporation-source.json) retains the foundling/registration IDs,
source identity, pin, relative locator and file digest. Current authorship is the
explicit test actor/device and Codex; unknown original author/date remain unknown.
The high-confidence user-direction basis records the confirmed new instruction,
not independent verification of everything in the historical source.

The first search, `verification`, returned no match. A shorter `verif` query
returned a truncated excerpt; the following read obtained the complete 456-byte
file. That read is not counted as needless duplication. The full turn took 72.213
seconds in this single sample. [Measured operations](live-summary.json) preserve
the actual call sequence and response sizes, not a transcript or benchmark.

## External correction in the same running session

The native agent was instructed to run only a prepared separate-writer fixture,
without reading its script/input or inferring the new value. The actual command
call returned only a generic completion message, not the new project name.

That fixture used the retained CLI with a separate explicit device/actor binding
to supersede Nimbus Finch with Saffron Kite in the same project record and scope.
Attribution records `Example Operator`, a different device ID and the
`external-cli-test` harness. It did not journal or synchronize.

The following read-only prompt did not contain the new name and explicitly asked
the agent not to rely on earlier context. One MCP recall and one history call
returned Saffron Kite, the preserved former names and the adapted procedure. The
recall envelope was 1,694 UTF-8 bytes; the turn took 15.123 seconds. No save,
journal, synchronization or repair followed.

The test driver compared the MCP process PID, parent PID, start time and command
before the external correction, after it and after native recall: all matched.
This was not a restarted server or fresh session. Every bank-file hash, including
Git and local receipt files, matched the post-correction baseline through recall,
inspection, session exit and the separate recovery tests. Source bytes and both
project directories were unchanged.

This proves fresh explicit reads in this running process, not automatic rewriting
of already-read model context or freshness of an uncontacted remote. The native
promotion, journal and external correction remain **local and pending** in
[sync status](live-sync-status.json); the last delivered head is preparation only.
No extra sync was performed merely to make that status look clean.

## Two-clone conflict and offline recovery

A separate disposable signet and local bare remote isolated these checks from the
native session. Two independently bound clones corrected the same baseline
revision differently. After delivery/merge, both proposals survived as a semantic
conflict: [receipt](recovery/right-conflict-sync.json) reports delivered Git history
and one unresolved memory conflict; [recall](recovery/conflict-recall.json) exposes
the conflict without selecting either proposal as current guidance.

An explicit resolution named both competing revisions as predecessors. Delivery
and recall then reported one effective resolved value, no semantic conflicts and
four preserved history entries. No force-push, reset or arbitrary winner was used.

Only the left fixture's remote was temporarily changed to an unavailable local
path. Sync returned a successful operation with `pending` delivery at fetch, a
local checkpoint and `delivered: false`. The first driver incorrectly expected a
command error and stopped; its assertion was corrected to the documented pending
state and the test resumed without repeating earlier writes. Local recall stayed
available and read-only hash comparisons passed. Restoring the original fixture
remote allowed normal delivery to recover.

## Corruption refusal and exact restoration

After retaining an exact copy, the test driver deliberately replaced one resolved
revision file in the left clone with malformed JSON. Inspection and recall both
returned exit 1 and `store.invalid`, without automatic repair or any additional
bank-file changes. The [inspection error](recovery/corrupt-inspect.json) explicitly
reported no write. Restoring the exact retained bytes recovered healthy structure
and the original resolved recall. This is corruption detection/restoration, not
proof of automatic recovery from arbitrary filesystem or power failures.

## Push delivered but confirmation failed

For one sync only, a fixture-local Git wrapper delegated the actual local push,
recorded its success, then deliberately returned failure with a synthetic stderr
canary. The [receipt](recovery/uncertain-push.json) correctly reported pending,
checkpointed, unconfirmed delivery and instructed inspection before retrying. It
did not expose raw stderr or claim that the push had rolled back.

Before any retry, read-only inspection found identical actual local and remote
heads, matching the receipt's checkpoint. Bank hashes were unchanged by inspection.
A normal unwrapped sync then confirmed delivery of that same head. The five-entry
memory history was identical before and after retry: no duplicate revision or
commit was needed. This models a lost success response at the Git process boundary
after a real **local** push, not real network packet loss or exactly-once delivery.

## Evidence and limits

The JSON files retain synthetic store/CLI outputs and sanitized native metrics.
Raw recordings, transcripts, bindings, local process commands and machine paths
remain private. Earlier checkpoint files are preserved, not rewritten to appear
current. Native aggregate token counters include inherited resources and repeated
context; they are not isolated Mandalore cost or a per-request context budget.

Contributor self-review checked exact-candidate identity, predecessor/source
attribution, pending-versus-delivered claims, process continuity, read-only hash
windows and publication privacy. Whitespace/privacy checks and all evidence hashes
passed. An actual container-first CI attempt could not reach the Docker daemon;
the documented pinned Go 1.26.4 host fallback passed the full engine/interface
checks and target builds (cached test results). The observations support these bounded scenarios,
not blanket prompt-injection resistance or full failure-mode coverage. Physical
machine/platform coverage, the exact-candidate rendered install/update/repair
journey and independent acceptance remain separate work.
