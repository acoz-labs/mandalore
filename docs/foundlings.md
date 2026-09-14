# Foundlings

Foundlings are explicitly selected historical references, not current memory or
standing instructions. Connecting old notes must not make their instructions
override the current user, automatically install their tooling, or silently
promote their contents into the signet.

## Implementation status

Issue #12 is in progress. The engine has immutable registrations with explicit
active, disconnected and conflicting heads, internal read-only local/Git
source observation, and clone-local connections/inspection. These are implementation
components, not yet a complete user-facing workflow. Retrieval/promotion operations,
CLI/menu integration and native-agent verification remain in the reviewed issue plan.
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
