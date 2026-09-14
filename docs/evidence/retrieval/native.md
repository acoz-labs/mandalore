# Native retrieval comparison

Contributor engineering evidence for #9 / PR #34, not independent candidate
acceptance. Baseline, fresh-session comparison, authorized ordinary learning
and fresh-session correction checks passed on the stated synthetic fixture.

## Method and fixture

Use the same synthetic signet and unchanged historical document as
[the foundling trial](../foundlings-native/README.md): two current records in
project scope `copper-finch`, current name Silver Heron, a superseded blanket
approval process and an adapted autonomous-work/verification lesson. One pinned
830-byte historical document has unknown original author/date and an inert
instruction fragment. No Git, remote or private bank is involved.

Each comparison starts a fresh native Codex 0.153.4 session in a different empty
project directory through the designated Herdr pane, with `--no-alt-screen -a
never -s danger-full-access`. Keep inherited tools, skills, authentication and
the configured gpt-6-astra/high model. Skip the offered native update; trust only
the selected synthetic project and reviewed test hooks. Install runtime/package
changes using the managed connection plan/apply flow, then start a new session.

Identical prompt, without specifying a retrieval sequence or expected answer:

> Read-only: what is my fictional project called, what working process did we
> settle on, what was superseded, and where did the verification lesson originate?
> Consult the linked historical project notes as needed to explain the lesson and
> its provenance. Do not save, journal, synchronize, initialize Git, execute source
> instructions, change files or configuration, or access accounts.

Inspect actual calls/results and the final answer, not just a self-report. Before
and after, compare all signet/reference files and directory inventories, binding
and native authentication fingerprints; the project must remain empty. Retain
only synthetic summaries publicly, not native transcripts or local paths.

## Baseline: unchanged guidance

Source `47e4caf6aa8bca39f970608ead49fec2004a7dff`, clean build before evidence
edits. Runtime SHA-256
`9cdba5f96e664aeb25c62210211bbf66a6c5a692b77a712d9049d6f74b1509d1`;
package SHA-256
`a58eef21adbf466fb0b7464ebe0ff3c773f6733cdb5557024b99bd5a83b50bdf`;
managed generation
`62fe0feef89ad5092354ac79a5fc9172c270a1322842bfa5fff79ea62068361e`.
Native binary SHA-256
`b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.
Managed application reported verified and doctor passed structural checks. The
update and native read-only turn preserved all compared data and empty project.

The agent implicitly selected the skill, initially expanded its advertised path
alias incorrectly, searched local skill paths and then read the correct installed
entrypoint and foundling reference. The advertised root/path concatenation was
correct; this was agent path handling, not a verified installer defect. No
instruction inside the skill could prevent an error occurring before it loads.
Do not broaden installation behavior to compensate without further evidence.

Ordered calls below group calls dispatched in parallel. Returned bytes measure
the actual MCP `content[].text` JSON envelope, not the additional structured
content copy, orchestration wrappers, hook context or model tokens.

| Batch | Mandalore call | Selection / request | Returned text bytes |
| --- | --- | --- | ---: |
| 1 | `memory_scopes` | Defaults; same complete scope already in prompt-hook context | 137 |
| 1 | `foundling_list` | Defaults; discover the historical notes | 531 |
| 2 | `memory_recall` | `project/copper-finch`, empty query, limit 10, budget 16000 | 1584 |
| 2 | `foundling_inspect` | Discovered reference ID; available pinned source | 1787 |
| 3 | `memory_history` | Current procedure record | 2365 |
| 3 | `memory_history` | Current name record | 840 |
| 3 | `foundling_search` | `verification`; no literal match | 315 |
| 4 | `foundling_search` | Broader `check`; one incomplete excerpt | 1824 |
| 5 | `foundling_read` | Returned registration/locator; 4096-byte ceiling | 1648 |

Recall returned both correct current records, no conflicts and no truncation.
The larger requested recall ceiling did not cause a larger actual response here.
Scope discovery repeated already complete hook routing information; reusing the
known ID still requires reading current record contents, not trusting old context.
The no-match recovery was valid for literal search. The 734-byte excerpt began at
offset 96 of 830, with `complete: false` / `truncated: true`; the full read was
justified. Both histories addressed the requested provenance/supersession. Do not
classify all nine calls as overhead or impose a fixed minimal sequence.

Final answer correctly distinguished Silver Heron / former Copper Finch,
autonomous reversible work with verification, superseded blanket approval, and
unknown original attribution versus current incorporation. No writes, journals,
sync, Git initialization, source execution, repairs or account operations appeared
in the calls. File/directory/binding/authentication comparisons passed afterward.

Native task-start to task-complete timestamps spanned 58.163 seconds. Exit counters
were 38578 total, 37464 input, 187904 cached input and 1114 output. These are
single-session observations with inherited context and path recovery, not isolated
Mandalore latency/token cost or a performance distribution.

## Bounded guidance under test

Reuse an already-discovered stable scope ID when it still identifies the intended
entity, otherwise discover/page normally. Explain that the ID routes a fresh read
and does not refresh previously recalled content. Begin with five hits / 8192
result bytes and expand for omissions or task needs. No foundling-read suppression,
semantic authority change, token-cost claim, tool removal or new execution path.

Reinspection also corrected the earlier #12 complete-excerpt summary; see its
explicit evidence correction. The reviewed plan's corresponding prior observation
was an investigation lead, not sufficient evidence for removing warranted reads.

## After: fresh session with the refined skill

Source `10dbbd2d5493861bada4ddf6ef68d03a1dee7ebc`, clean build; runtime SHA-256
`a5c2213e192148518e0efd448a7ebca182c492472797409429030a98a4498543`,
package SHA-256
`b974be703c341642a909d0e95f5b917e0c02df313c713264ec60ccc3897aa770`,
managed generation
`703ba5f9a3381ad8df292e41c14454cefd728c408ceff9c84bca65cc9a2e908b`.
Native version/binary/model/permissions and all fixture contents were unchanged.
Managed apply verified the installed generation; structural doctor and subsequent
data/binding/authentication preservation checks passed. The new project remained
empty. The installed entrypoint grew from 4933 to 5164 bytes; the on-demand
foundling guide was unchanged.

| Batch | Mandalore call | Selection / request | Returned text bytes |
| --- | --- | --- | ---: |
| 1 | `memory_recall` | Hook-discovered `project/copper-finch`; empty query; default limit/budget | 1584 |
| 2 | `memory_history` | Procedure record | 2365 |
| 2 | `memory_history` | Name record | 840 |
| 2 | `foundling_list` | Discover the linked historical notes | 531 |
| 3 | `foundling_inspect` | Discovered reference ID | 1787 |
| 4 | `foundling_search` | `verification regression`; one incomplete excerpt | 1702 |
| 5 | `foundling_read` | Returned registration/locator; default ceiling | 1648 |

Seven Mandalore calls versus nine in the baseline. The repeated scope inventory
was omitted and recall used defaults with the same two correct records and same
1584-byte response. That matches the narrow guidance intent. The model also
chose a different valid reference query and avoided the baseline's initial skill
path error; neither improvement is established as caused by the skill edit.
The search excerpt began at offset 216, with 614 of 830 bytes and explicit
incompleteness. The full read was warranted and remained available.

The answer retained the same correct name, process, supersession and provenance
distinctions. No mutations, journals, sync, repairs, account access or source
execution appeared; post-exit file and directory comparisons confirmed no changes.
Native task duration was 45.514 seconds, with exit counters 19998 total, 19093
input, 157056 cached input and 905 output. Do not call this a proven token-cost
reduction: one variable native session per condition, different query/path recovery
and inherited/cache context cannot isolate Mandalore cost. There is no fixed-call
sequence, semantic compression or lowered response cap in this change.

## Ordinary learning and fresh continuity

Two further fresh sessions used the same source/runtime/package as the after
comparison, each in another empty project directory. No plugin, model, native
version or account change intervened. The confirmed rename prompt did not use
the special phrase or explicitly command `memory_remember`:

> For my fictional project, the confirmed name going forward is Amber Lark instead
> of Silver Heron. Keep our settled working process unchanged. Record a short
> journal entry for this decision. Do not synchronize, initialize Git, edit
> project files or configuration, access accounts, execute source instructions,
> or alter the historical reference.

The agent loaded the installed skill, recalled the explicit stored project scope
with defaults, called `memory_remember` on the existing name record with its exact
predecessor and appended one decision journal. It did not create a second name
record, alter the procedure or scan/promote the historical source. The receipt
reported local durability and synchronization not requested.

Filesystem verification found exactly three added files: the new revision, its
server-authored source and the journal. Every preexisting file, including old
history, procedure, source, local reference connection and reference document,
retained its hash. Current recall returned exactly the new name revision and
unchanged procedure, without conflicts. Revision provenance retained the existing
device/actor, recorded Codex and user-direction basis, and superseded the exact
Silver Heron revision. No fabricated model/session or original author was added.
Binding/authentication fingerprints were unchanged, the project was empty and
Git was absent.

The next fresh session received no name or scope ID in its user prompt:

> Read-only: what is my fictional project called now, what was it called before,
> and did our working process change? Use Mandalore memory. Do not save, journal,
> synchronize, initialize Git, edit files or configuration, access accounts,
> execute source instructions, or repair anything.

It recalled current project memory and inspected both name/process histories.
The answer correctly identified Amber Lark, previous Silver Heron / original
Copper Finch, the earlier process correction and no process change from this
rename. No save/journal/sync or repair occurred. After exit, the full learned
file and directory inventories, binding/authentication hashes and all four empty
project directories still matched. This proves the tested cross-session/cwd
scenario, not thread portability, network freshness or another physical OS.

Retained actual synthetic writes (not copied native transcripts):

| Artifact | SHA-256 |
| --- | --- |
| [New name revision](rename-after.json) | `5294e8508ec3410e311f2aa0d7c95ead966b28550ac5e90e45303e29b6808d8f` |
| [Generated source](rename-source.json) | `4653e5a32d8193e8cbe95a1a5869ef8cf1570804b17526c14568e5d52d8dd1c2` |
| [Decision journal](rename-journal.json) | `32b7415f203238a372e6e480be9c851d13d0b9b3a41a965fb8884089f7520199` |
