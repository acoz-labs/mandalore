# Fresh native session on the distribution candidate

Contributor verification for draft PR #36, not independent acceptance. This uses
the same exact source and candidate as the [installer recovery checkpoint](recovery/README.md):

- Source: `065b481a3dbdf66e731bf1558bc5c5b4302fc0fb`.
- Manifest SHA-256: `e8562e2938ebfefa0aebcad18f2859b804c8b2947d2b4d2887ae99fcab3effd4`.
- Native runtime SHA-256: `63f9ab8200fa36e1e8c184b8c699130ffb8838f82b8d7c9625d824ab16dbb4f4`.
- Embedded plugin SHA-256: `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.
- Codex 0.153.4, macOS ARM64, executable SHA-256
  `b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.
- Native model observed: `gpt-6-astra`, high reasoning. No model configuration changed.

## Setup and native boundaries

The current owned test connection passed doctor before updating. The actual
candidate generated its own connection plan and applied it through native
registration, retaining the previous generation. The new installed cache passed
plugin validation; the retained runtime matched the candidate byte-for-byte.
Before/after inventories confirmed that connection installation changed no signet
file. The selected bank contains synthetic fixtures only.

Both native sessions ran in the designated Herdr pane from separate empty project
directories, with ordinary native authentication and inherited resources. Neither
`MANDALORE_BIN` nor `MANDALORE_BINDING` was supplied: the installed connection's
retained defaults had to work. No auth was copied or enrolled. The update prompt
was skipped to keep the pinned Codex version. The test directories were explicitly
trusted; the two Mandalore hooks were inspected in `/hooks`, already active and
trusted for this installed generation. The existing unrelated native hook was
left unchanged. No blanket hook-trust bypass was used.

The owner-authorized YOLO flag was invocation-local, not a global settings change.
It removes execution approvals and sandboxing; synthetic fixture selection is
not an OS security boundary. See the [native command reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli)
and [native profile/state locations](https://learn.chatgpt.com/docs/config-file/config-advanced).

## Observed conversations and durable checks

| Scenario | Observed result |
| --- | --- |
| Ordinary statement introducing a separate fictional project, without “remember” or a special cue | Saved Cobalt Kestrel and its two-short-sentence status format in a new project scope; appended a short semantic journal. |
| Ordinary correction to Juniper Kite | Recalled the current revision, preserved its record and `cobalt-kestrel` scope, and explicitly superseded the old name; retained the format and appended a journal. |
| Direct “This is the way” plus a decision-note format | Added a separate preference in the existing project scope: decision followed by one brief reason; appended a concise consolidation journal. |
| Fresh native session in the other project directory | One scoped MCP recall returned Juniper Kite, the old name, status format and decision-note format. No resumed thread or runtime/binding exports. |
| Explicit no-save/no-journal/no-checkpoint/no-sync/no-repair request in that fresh session | Only the skill read and one memory recall; all signet file paths and SHA-256 values matched before/after the session and normal exit. Both project directories remained empty. |

Typed CLI recall independently confirmed the current records. Stored revisions
carry the binding's opaque device ID, neutral synthetic actor, `harness: codex`,
user-direction evidence and generated source references. Old revisions remain in
history. Native session/model fields remain unset rather than fabricated; device
provenance does not require publishing workstation names.

Each learning turn requested one three-second sync after saving. This fixture has
no `.git` directory: sync stopped at checkpoint, reported pending/not delivered,
and the agent described the saves as local rather than claiming remote delivery.
The explicit-cue turn also inspected read-only sync status. No Git repository or
remote was silently created. This is a local-only boundary test, not successful
cross-machine synchronization or an offline-remote test.

After both sessions, doctor again passed all 12 structural/inventory checks, the
retained binary matched the candidate, and the previous connection generation was
still present. Doctor continues to label native login, hook trust, live MCP, remote
freshness and active context as not tested by *doctor*; the first three were
separately exercised here. No remote freshness was established.

## Retrieval observations and limits

The first learning turn used recall/save/journal/sync; the correction used the
same four tools. Consolidation used those four plus read-only sync status.
The cold no-save turn used one memory recall and no memory mutation/sync tools;
its textual MCP result was 1,612 UTF-8 bytes. Hook-provided scope metadata allowed
routing to the stable scope without guessing from cwd or current display name.

Native counters for the cold turn reported 62,785 aggregate input tokens, including
51,072 cached input tokens, and 272 output tokens across its model requests. These
include native inherited resources and repeated prompt processing; they are **not**
Mandalore-only overhead or the size of one context window. This one observation is
not a replacement for the [retrieval evaluation](../../retrieval.md) or #10's
exact-candidate measurements.

Scoped contributor verdict: pass. No product source change was needed. Plugin
validation used isolated PyYAML 6.0.2 after the system interpreter reported that
validation dependency missing; it is not a product/runtime dependency. Full source
CI is linked in the recovery checkpoint. This turn did not run a new code build.

Raw native sessions, recordings and machine paths remain in the disposable local
fixture, not public Git. This is semantic native engineering evidence, not a new
owned-UI acceptance recording. Final source reconciliation, retained default-branch
candidate nomination, #10 independent acceptance, physical-machine continuity and
native OS/architecture coverage remain outstanding. Earlier compaction,
interruption, separate-bank, foundling and sync tests must not be relabeled as
tests of this exact candidate. No release or real-memory migration was performed.
