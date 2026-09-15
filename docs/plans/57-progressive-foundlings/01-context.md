# Context

Issue #57 asks for efficient historical consultation without hiding needed evidence
or overriding a user's deliberate exhaustive scan. Existing search ranks documents
by distinct case-insensitive literal term matches, then path, and returns previews
near the earliest matching term. It verifies source/registration on every request;
no semantic index exists. Keep that authority boundary and ranking in this change.

At the repository basis, `internal/foundlings/retrieval.go` returns up to ten
1024-byte search excerpts under a 32768-byte serialized-result ceiling. The API
default count is five, and reads default to 4096 bytes (explicit maximum 8192).
Search reports total matching documents but has no offset/continuation input.
The shared API/CLI and `cmd/mandalore/foundlings_menu.go` own those entrypoints;
the on-demand foundling skill reference guides agents. Read excerpts already carry
exact source/registration/file identity, byte offsets and completeness flags.

## Baseline evidence

2026-09-15 synthetic local source, 16 Markdown documents: three related project
process notes, a long field log and twelve same-term inventory notes. The correct
answer depends on later qualifications, two different documents and a current
memory decision superseding historical routing. No private source was used.
Retained development runtime is c06fd8d (foundling code unchanged at repository
basis), SHA-256 2bd32e805df3d58bfc83a69f30836be3e8322b4da0cd97146a6b97c0e7acdd5e.

Direct CLI baseline result JSON: search 5010 bytes for three hits; default selected
read 4955 bytes (4096 content bytes, final decision still omitted); expanded read
7160 bytes (6285 content bytes, complete). Twelve same-term matches yield ten
results and truncation with no input that continues the same result ordering.

Fresh native Codex 0.154.0, macOS arm64, gpt-6-astra/high, code mode, in the dedicated
test lab: correctly answered current Juniper Desk over historical Lantern Desk,
no automatic audit deletion after a later negation, and a retired weekly
spreadsheet. Two explicit 8192-byte reads followed search. All foundling envelopes
including list/inspect total 18786 bytes. Final request input 34057 tokens;
aggregate input 192557, cached input 140416, seven model requests/six code calls.
Tool-output text totals 78819 bytes, including discovery and skill reads; it is
not equivalent to foundling payload size. Complete source/signet file hashes
matched before and after. This is one correct observation, not a statistical
benchmark or proof of isolation from native authentication/resources.

## Scope and risks

Preserve supported existing read ranges, promotion inputs, stale-source refusal,
history and no-write operation flags. Defaults intentionally change, and new
optional search inputs/output continuation fields require documentation and
schema tests. Agent clients requesting explicit old sizes retain them. Do not
pretend a smaller snippet alone contains a complete claim. Narrower defaults can
increase round trips; verify actual composed tasks and allow larger reads.
MCP duplicate-result rendering is a separate #56 treatment, not double-counted
here. Source discovery cost/indexing, reference import, global scan policies and
new credential/harness setup are not part of this issue.
