# Foundlings

Foundlings are explicitly selected historical references, not current memory or
standing instructions. Connecting old notes must not make their instructions
override the current user, automatically install their tooling, or silently
promote their contents into the signet.

## Implementation status

Issue #12 is in progress. The engine has immutable registrations with explicit
active, disconnected and conflicting heads, internal read-only local/Git
source observation, clone-local connections/inspection and verified retrieval and
promotion, exposed through the shared CLI/MCP dispatcher. The complete guided
workflow is still in progress: the guided menu has
[actual rendered engineering evidence](evidence/foundlings-menu/README.md), and
the native skill has an on-demand consultation/promotion workflow. Fresh native
scenarios and final reconciliation remain pending.
No real historical memory has been adopted by these synthetic tests.

## Portable identity versus local content

A registration records a portable source identity and pin. Local directories use
an opaque source ID, not a workstation path, and a deterministic SHA-256 over
sorted eligible relative paths and their bytes. The length-framed digest excludes
timestamps and machine-specific roots. Git sources use a credential-free locator
and a commit ID with its explicit SHA-1 or SHA-256 object format.

The internal observer accepts an explicitly selected absolute directory. It reads
the selection twice and refuses differing observations. This checks observed
content, not a permanent filesystem lock or a guarantee against later changes.
Normal reads do not create local state, copy a source, write its repository,
fetch, execute scripts or load native skills.

For Git, the directory must be an existing standalone checkout with exactly one
matching local `remote.origin.url`. Eligible worktree bytes must match the pinned
local HEAD's tracked blobs; the selected blobs must also be available locally.
Identity and HEAD are checked again after reading. There is no automatic locator
normalization, clone, credential lookup, submodule traversal or linked-worktree
support. A different commit is a different observed pin, not an automatic update
to an existing registration.

Git inspection disables lazy fetch, replacement objects, fsmonitor, hooks and
interactive transport. It does not run checkout, status refresh, filters or
text conversion. Untracked files are not enumerated or counted; neither full
checkout cleanliness nor remote freshness is claimed. An excluded code or binary
file can be modified without becoming reference evidence.

## Clone-local connections

The internal connection manager stores a strict version-1 JSON connection at
`.mandalore/foundlings/<foundling-id>.json`, under the signet's ignored local-state
directory. It binds the signet ID, foundling ID, exact registration revision,
portable source identity/pin and canonical absolute local root. Its connection ID
is a SHA-256 fingerprint of those typed fields (excluding the ID itself), not a
credential or signature. Identical configuration retains the same identity;
editing a field without updating its fingerprint is invalid configuration.

Connecting requires the expected current active registration revision and verified
source pin. Replacing a connection additionally requires its exact prior ID.
Unknown, malformed, redirected or concurrently changed local configuration is
preserved, not automatically repaired. Another signet's connection is invalid.
Sources cannot contain or be contained by the selected signet/local-state tree.
Overlap checks compare filesystem identity through ancestor aliases, not just
path spelling, so an aliased signet root cannot bypass the boundary.
Missing old source paths can be explicitly replaced with a verified new path.

Connection writes share the signet writer lock, recheck metadata/content and use
temporary-file publication. New connections cannot overwrite an existing file;
deliberate replacements use the expected connection identity and recheck it before
publication. This coordinates cooperating toolkit writers, not arbitrary programs
that ignore the lock. Parent directories are synced after publication. A result
can report `connected: true, durable: false` with an error if publication happened
but a subsequent directory sync failed; callers must inspect before retrying.

Inspection reports `unconnected`, `available`, `changed`, `unavailable`,
`invalid_connection`, `disconnected` or `conflicted`. Only `available` means the
selected content matched the connected registration during inspection. Missing
paths or unsupported source reads are unavailable, not proof of absent historical
knowledge. A new registration revision makes an old connection changed, even if
its content pin is the same. Disconnection/conflict withholds source access while
preserving local connections, original files and historical registrations.

Inspection does not acquire a writer lock, create missing local directories,
repair a connection or publish a registration. Local paths never become portable
registration fields. Setting up the same signet on another machine requires its
own local connection; there is no fallback to another bank's configuration.

## Eligibility and bounds

Eligible extensions are `.md`, `.markdown`, `.txt` and `.json`. Selected text must
be valid UTF-8 without NUL bytes; malformed text is refused rather than silently
decoded. Dot-prefixed filenames, code and transcript-specific formats such as
JSONL are excluded. Runtime/cache directories including `.git`, `.mandalore`,
`.my-friday`, `node_modules`, `.venv` and `__pycache__` are excluded. Exclusion
counts refer to visited excluded entries/subtrees, not all descendants.

Symlinks, traversal and special files cannot provide selected evidence. Local
directory enumeration is bounded to 30,000 entries and 64 levels; selected content
is bounded to 10,000 files, 4 MiB per file and 64 MiB total. Git tree enumeration
has a 30,000-entry bound and commands have an 8 MiB stdout/64 KiB stderr bound,
ten-second command deadlines, caller cancellation and process-group cleanup.
Raw subprocess failures are not reflected back to callers.

Format eligibility is not automatic secret or transcript detection. Ordinary
Markdown, JSON or text can still contain sensitive material. Users must choose
appropriate references; the eventual agent workflow must not store secrets or
raw transcripts merely because the source format is eligible.

## Retrieval and selective promotion

The internal retrieval manager searches only an explicitly selected, connected
foundling. It rechecks registration, local connection and content on every request.
Search accepts 1–16 whitespace-separated literal terms (at most 1024 UTF-8 bytes),
matches case-insensitively and ranks documents by the number of distinct matching
terms, then relative path. A match on any term is sufficient. It does not interpret
query text as regular expressions, run source instructions or search ordinary memory.

Search returns at most ten deterministic excerpts, each up to 1024 content bytes,
near the first match, with a 32 KiB serialized result budget. The full matching
document count and truncation flag distinguish omitted results from no matches.
Individual excerpts can themselves omit parts of a document, independently of
whether the result list was truncated. No index or semantic ranking is claimed.

Reading requires an exact registration revision, eligible relative locator and
explicit byte range of 1–8192 bytes. Excerpts never split UTF-8 characters and
include a next offset when more bytes remain. JSON-encoding expansion can shorten
an excerpt to keep its serialized representation within 32 KiB. Every excerpt
includes its actual file SHA-256, portable source/pin, registration revision,
relative locator, byte offset, total document size, completeness/truncation and
an explicit unreviewed-reference notice. A short excerpt is not a complete claim.

Promotion takes that exact registration/locator/hash and an independently authored
normal memory write. It generates the citation from the verified connection; a
supplied `write.external_origin` is refused. Known original author/date may be
provided explicitly, but absent provenance remains absent. The new revision uses
the current binding's device, actor, harness and recording time. The requested
memory body and reason describe what was retained, qualified or changed; they do
not have to copy the reference verbatim. No automatic journal entry is created.

Source verification and the active-registration guard execute under the same
signet writer lock as publication. Cooperating writers cannot disconnect or
replace the registration between that guard and the write. Source bytes are
checked immediately before publication, not locked forever against unrelated
programs. Failed verification leaves no new source evidence or revision; later
filesystem failures retain the sourced writer's existing partial-I/O semantics.

Ordinary `memory_remember` still accepts structurally valid historical citations,
including citations to a subsequently disconnected registration. That is not the
same guarantee as this dedicated verified promotion path. Disconnecting a foundling
does not erase previously incorporated knowledge, provenance or supersession history.

The host agent must compare current knowledge and user direction before promotion,
avoid duplicate memories and use explicit record/predecessor IDs for corrections.
Neither lexical relevance nor a quoted “this is the way” authorizes saving or
executing anything. These semantics still require native-agent scenario validation.

## CLI workflow

Use an explicitly selected signet binding. Paths/IDs below are placeholders.
`mandalore menu` also provides guided foundling management; see [setup](setup.md).

```sh
mandalore foundling preview --binding /example/local/binding.json < reference.json
mandalore foundling register --binding /example/local/binding.json < registration.json
mandalore foundling list --binding /example/local/binding.json
mandalore foundling inspect --binding /example/local/binding.json --foundling-id FOUNDLING_ID
mandalore foundling search --binding /example/local/binding.json --foundling-id FOUNDLING_ID --query 'project naming'
mandalore foundling read --binding /example/local/binding.json --foundling-id FOUNDLING_ID --registration-id REGISTRATION_ID --locator notes.md
```

`reference.json` selects a source, for example:

```json
{"source":{"kind":"local","locator":"source-historical-notes"},"local_root":"/example/historical-notes"}
```

Create `registration.json` with `name`, `description`, `reason`, that exact `source`
and the preview's `pin`. Include `local_root` to connect it on this machine, or omit
it for portable metadata only. Inspection must report available before retrieval.
Use returned IDs rather than inventing them. An unavailable source is not evidence
that its historical knowledge is absent.

For an adapted memory, use `mandalore foundling promote --binding FILE < promotion.json`
with the returned foundling ID, registration revision ID, relative locator and
content SHA, plus `write` containing the confirmed memory, basis and change reason.
Recall current memory first. The same bound operation is available through MCP;
source setup/reconnection/history administration remains CLI-only. Every operation
is also available as `mandalore call OPERATION --binding FILE < input.json`.

## Engineering checks so far

Failing-first synthetic tests cover local deterministic pins, source preservation,
changed content, invalid identities/paths/text, symlinks/FIFOs, size/count limits
and cancellation. Real local Git fixtures cover SHA-1/SHA-256, exact tracked text,
dirty selected files, wrong/ambiguous origins, missing trees/blobs, redirected
files, excluded untracked/code/binary content, inherited Git environment
redirection, source-program canaries and unchanged source/metadata bytes.

An excessive-output test exposed a stream-copy bypass caused by embedding a
buffer with a promoted `ReadFrom` method. The bounded writer now owns a named
buffer so all copied bytes pass through its limit check. Regression tests cover
stdout/stderr flooding, cancellation and sanitized failures. This is contributor
engineering evidence, not native workflow or immutable-candidate acceptance.

Connection tests additionally cover fresh routing lookups, absent local state,
read-only inspection, source disappearance/change, stale/disconnected/conflicting
registrations, explicit path replacement, matching-request identity stability,
tampered/unknown connections, wrong-signet isolation, overlap, symlinked local
state, cancellation and preservation of source/registration bytes. An injected
post-publication directory-sync failure verifies the partial result, retained
inspectable connection, removal of the owned temporary file, refusal of a blind
retry and successful explicit recovery. End-to-end management/API partial outcomes
remain pending with the rest of the workflow.

Retrieval/promotion tests cover attributed read-only results, deterministic ranking,
no-match and truncation reporting, UTF-8 continuation and JSON expansion bounds,
stale/unsafe/missing locators, changed bytes and hashes, cancelled/disconnected
promotion, generated citations, unknown versus known original provenance, explicit
supersession and surviving history after disconnection. A storage-level test proves
the source-verification callback holds the writer lock, rejects a failed check
without learning and preserves ordinary historical-citation semantics.

Shared-interface tests cover strict inputs, binding requirements, CLI-only setup
visibility, read-only mutation rejection and retained registration after a failed
connection. A compiled-process test runs the CLI and real MCP stdio from an
unrelated working directory, compares identical search results, promotes an adapted
memory, rejects promotion through read-only MCP and preserves source bytes. No
model/provider is involved in that test; native-agent behavior remains pending.

The current discovery measurement is 16 MCP tools, 5729 input-schema bytes,
32678 output-schema bytes and 2488 description bytes, excluding transport overhead.
These are serialized byte counts, not token counts, latency or quality evidence.
They are an input to issue #9, not a claim that context efficiency is finished.

Automated menu tests cover register/decline/incomplete confirmation, read-only list
and search, disconnection, a moved local path, explicit superseding pin updates,
failed output/cancellation and a completed registration retained after connection
failure. Actual rendered terminal evidence is a separate gate, not established by
these scripted tests.
