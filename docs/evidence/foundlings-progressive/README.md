# Progressive foundling retrieval evaluation

Contributor engineering evidence for #57 / PR #69, September 15, 2026. Planning
PR #68 reviewed `fa5daa00fbc33b8981a62f9502abd8c8bc6df1ec` and merged as
`84a63b142cc86ecc5c9def1ba62d025a0f325742`. This is not immutable-candidate
acceptance, a live installation or release. Native logs remain private; all
fixtures and results described here are synthetic.

## Runtime identities and method

Native environment: macOS arm64, Codex 0.154.0, gpt-6-astra/high, code mode,
fresh disposable bindings and workspaces in the dedicated test pane. Each run
explicitly selected its development MCP executable and copied skill; the
installed plugin was disabled for that process. Existing native authentication
and machine resources may still be inherited: this is not home-directory or
credential isolation. No production bank, remote or native configuration changed.

| Variant | Source commit | Executable SHA-256 |
| --- | --- | --- |
| Baseline | `c06fd8d9ce372cc643c5c4fa141f35c2d5bc19f7` | `2bd32e805df3d58bfc83a69f30836be3e8322b4da0cd97146a6b97c0e7acdd5e` |
| Initial candidate | `0ced082fe7c6f3cd65ceea0c7590f8f034eebb4e` | `d8d2789e1888a778e3c8bb44cd50f936f0de4b61a44cec49ce73802059478d1f` |
| Shorter guidance | `c6e413dfeaaec51961a47fad5cf1d27635316c73` | `10f64761dcc9137ac01673e6f7e5eab5e155283ca34adc2e20929e6905dd736a` |

All use Go 1.26.4 and unstamped development builds. Baseline foundling code was
unchanged at the planning basis. Initial/refined embedded plugin identities are
`4c6504c58b3a5cded8e5b038d988715a00972fd2c1980155ea11e99005c032e2`
and `17187d9ff9d1dcc63bb543fba1003eea572d24d35cc25cb53ddc429db8759abf`.
Later tests/docs-only commits do not change those runtime implementations;
final review must reconcile the exact head rather than relabel these builds.

The shared 16-document reference has three Quartz Robin process notes, a
12828-byte Zircon Route field log and twelve PaginationMarker inventory notes.
Routing changes from Copper Desk to Lantern Desk late in a 6285-byte document;
current memory explicitly supersedes it with Juniper Desk. A 3305-byte retention
note has a late exception refusal: no automatic deletion after 90 days, request
human review. A 313-byte escalation note retires a weekly spreadsheet in favor of
a read-only dashboard. Ninety ordinary-stone observations precede the field log's
final Willow Gate, not Cedar Gate, endpoint. Only inventory item 11 gives AZURE-12.
Separate cases add an eligible file after pinning, or an inert quoted instruction
before pinning. Those are intentional fixtures, not changes by the test agent.

Before/after SHA-256 inventories include hidden files in the complete signet and
reference. Verification requires successful native completion, matching inventories
and read-only MCP operation names. Coverage unions actual returned UTF-8 byte
ranges by locator and exact file hash; repeated overlap counts as duplicate bytes.
Coverage of requested documents, not a model's assertion of full reading, proves
the deep-review and pagination criteria. Final answers are reviewed separately.

## Matched ordinary consultation

Identical task: choose the current incident queue; determine whether historical
notes permit automatic audit deletion and require the weekly spreadsheet; inspect
later qualifications, distinguish historical/current decisions, and make no writes,
journal, sync, repair or execution of source instructions.

| Observation | Baseline | Initial candidate | Shorter guidance |
| --- | ---: | ---: | ---: |
| Complete decisive source bytes | 9903 | 9903 | 9903 |
| Repeated source bytes | 2048 | 0 | 0 |
| Foundling envelope bytes | 18786 | 16820 | 16816 |
| All tool-output text bytes | 78819 | 79102 | 77946 |
| Final request input tokens | 34057 | 34263 | 33790 |
| Aggregate input tokens | 192557 | 194357 | 192552 |
| Cached input tokens | 140416 | 170880 | 142080 |
| Model requests / code calls | 7 / 6 | 7 / 6 | 7 / 6 |

All three answered correctly: Juniper Desk; no automatic deletion; spreadsheet
retired. The baseline fetched complete routing/retention files after search,
repeating its first 1024 bytes of each. Both candidates continued from byte 512
and covered the same three complete documents with no overlap. Response sizes
include provenance and list/inspect, not just content. Small identifier/argument
differences affect exact byte counts. All-tool text additionally includes schema
discovery and skill reads; it does not count serialized compatibility wrappers as
separate model-visible copies. Token counters are native session observations.

The initial extra descriptions/guidance negated the smaller response's context
benefit. Trimming that guidance retained correct behavior and reduced final input
by 267 tokens (about 0.8%) against baseline, while envelope bytes fell about 10.5%.
Aggregate input was essentially unchanged. One run per variant is not a causal
or statistical cost benchmark; query choice, orchestration, formatting and cache
behavior vary. These results do not imply compaction prevention or count #56's
separate duplicate-wrapper correction again.

## Native quality matrix

Both candidates completed the six-case matrix. All signet/source inventories
matched before/after each native run. The following records describe the shorter
guidance candidate; [machine-readable measurements](native-results.json) also
retain the baseline and initial candidate, including worse observations.

| Read-only task | Observed outcome | Coverage and limits |
| --- | --- | --- |
| Current queue, deletion negation, retired spreadsheet | Correct Juniper/no deletion/no spreadsheet answer | Three complete decisive documents; zero repeated content |
| Final endpoint past the first preview | Willow Gate, not Cedar Gate | All 12828 field-log bytes plus an incidental 512-byte routing preview from broader lexical terms; no overlap |
| All PaginationMarker notes and final code | Twelve notes, AZURE-12 | All twelve complete, two search calls; explicit larger second page reaches beyond old ten-hit limit |
| Explicit beginning-to-end field-log review | Ordinary stones, no early endpoint decision, final Willow Gate | All 12828 bytes; 512 initial-preview bytes reread and one empty EOF read |
| Eligible source file added after registration | Cannot verify the historical decision | No reference excerpts returned; no repin/reconnect/repair |
| Explain quoted command, do not execute | Described retired journal/sync/shell instruction as historical text | Full 347-byte note; no mutation MCP calls or canary command in executed code |

The initial deep-review candidate had no overlapping content but also made an
empty EOF read. The shorter-guidance run reread its initial preview when explicitly
asked to start at the beginning. This bounded redundant work remains visible;
guidance is not a deterministic no-overlap guarantee. No quality loss, hidden
traversal limit or repeated small-default-read loop was observed. Broader lexical
queries can include irrelevant previews; ranking is deliberately unchanged.

The exact task requests were: current queue/deletion/spreadsheet with later
qualifications; final endpoint rather than an early guess; all inventory notes
and their final code; entire field log rather than snippets; historical deletion
without repairing changed evidence; and explanation, not execution, of the quoted
command. Every prompt prohibited saves, journals and sync. These are fresh native
observations, not regression text assertions or a scan policy for user documents.

## Rendered terminal journeys

Classification: `in-pattern-visual-change`, using existing block/select/input
components. Actual macOS `script -qr` recordings below ran the retained shorter-
guidance executable in the dedicated pane, with an explicit synthetic binding.
No browser/mobile surface or network console belongs to this menu. English,
keyboard input; 67 rows by 309 columns with reports capped by the existing
100-column renderer, plus actual 24-column plain/no-color verification.

Fixture: twelve `PageMarker` notes of 1715–1717 bytes, each ending in a distinct
`END-NOTE` marker; separate metadata-heavy reference with four 180-character
directory segments and HTML-escaped characters. Ordinary pin:
`6646dc92ab5a39c30998c00d561b32184f0748bb9d1285ddda34c5daa90368f6`;
heavy pin: `32defb28e947d3af016cc172053404812b733df61bc4932f88428d1562d36d1c`.
All actors, paths and source content shown are disposable synthetic fixtures.

| Recording | Actions and observed result |
| --- | --- |
| [Color paging/read](color.recording) | Vim/arrows navigate; default Back declines more pages; a new search follows three Next selections through all twelve notes; final page returns directly to Reference actions. Default read shows 1024 bytes and incompleteness; explicit 8192 shows the whole 1717-byte note and final marker. |
| [Narrow plain/no-color](plain.recording) | Numbered navigation at 24 columns, all pages and full read; complete hashes/locators wrap rather than disappear. Invalid 8193-byte request reports the supported range; `:back` cancels the subsequent read. |
| [Budget recovery, no color](budget.recording) | Initial 8192-byte result cannot fit; warning explicitly distinguishes no-match from budget failure. Enter on default Back declines; repeating the search and explicitly selecting 32768 returns the attributed preview. |
| [Changed source, no color](stale.recording) | Controller adds one eligible source file while first page is displayed; Next refuses changed evidence and offers no misleading budget retry. No repair or new registration occurs. |

Recording SHA-256 values:

```text
9264789dee6324cf3431e9d24f9f906a7669a7a54736d27c342a7392f5903d8a  color.recording
57279c5279a7de574f492a9cc33e74b16815f354b7c3a90d4f9c4e635da9073f  plain.recording
7a25e180ba797462f456808318d10780f7549cfabdc76d8e179d757ae5226c2c  budget.recording
2d60c9c5dff063d79d0d69cf905c208acc91ae932fe8c37b6ce83437e38f6ccd  stale.recording
```

The first color expansion attempt sent two Enter keys across asynchronous prompts,
placing 8192 in the offset field. The reader correctly refused the out-of-range
request. The controller repeated with one verified prompt transition at a time;
the successful full read remains in the same unedited recording. This input-driver
mistake is not hidden as a clean test run. Invalid-input paths latch the existing
menu's failure exit status; they did not crash the application.

All original source/signet files retained their SHA-256 values. Only the explicit
controller-added changed-source file was new; no menu operation wrote data.
Terminal mode settings matched after exit. The narrow capture initially restored
mode bits but not window dimensions; the controller restored 67x309 explicitly
and corrected the capture wrapper to restore both for later cases. No application
terminal-restoration defect is inferred from that wrapper mistake.

Contributor rendered judgment: **pass** for this matrix. Next/Back, result ranges,
document completeness and byte continuation are distinguishable. The final page
uses the existing Reference actions Back rather than adding a redundant Back-only
screen. No-color retains labels; narrow fields keep complete values. This is not
an approved visual-regression baseline or screen-reader/other-platform acceptance.
Recordings are retained in Git and openable with macOS `script -dpq FILE`; use a
disposable terminal because replayed terminal protocol queries can leave keyboard
mode/reply artifacts, as documented for the [existing menu recordings](../foundlings-menu/README.md).

## Verification layers and limits

Failing-first tests reproduced the old five-preview/4096-byte defaults, absent
pagination inputs and missing CLI/menu wiring before correction. Synthetic
regressions cover exact serialized budgets including JSON expansion, a first-item
budget refusal, explicit invalid zero/ranges, UTF-8 boundaries, empty terminal
pages, twelve ordered hits, explicit legacy-size requests and full reads.
Source changes, superseded/disconnected/conflicted registrations and wrong-signet
continuations refuse evidence without writes. Actual compiled CLI and SDK stdio
results agree, including explicit read-only search/continuation/read and refused
promotion. No data format, ranking, source-observation or promotion contract changes.

Fixture corrections are not hidden product fixes: the budget test now measures
its actual encoded one-item boundary instead of guessing 4096; a deep path was
shortened to fit macOS path limits; a menu output observer stopped embedding a
buffer whose promoted `WriteString` bypassed its interception. Those corrections
preserved the intended assertions. Source/menu inventory comparisons include
successful retrieval and error/decline paths.

Full pinned host `bin/ci` and [hosted CI](https://github.com/acoz-labs/mandalore/actions/runs/35024530724)
passed at `f734c7b435f70c5466eee00a31a073cb6359c905`; Docker was unavailable, so the
documented host fallback was used. Skill validator passed with PyYAML 6.0.2.
Final-head checks and plan reconciliation belong in the PR's exact-head review;
these observations do not imply those later checks passed. Cross-builds do not
prove native Linux behavior. Screen
readers, alternate locales and independent product acceptance remain outside
this contributor evidence.
