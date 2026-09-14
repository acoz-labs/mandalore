# Context

The product contract and #12 require learning from prior memory without making
its historical instructions current authority. #10 explicitly includes this path
before real-memory adoption. #8's structural converter is intentionally different:
it supports a specific memory-only bank, not arbitrary assistant repositories.

At the repository basis, internal/memory/foundlings.go already validates immutable
FoundlingRegistration, FoundlingSource, SourcePin and ExternalOrigin. Registration
graphs retain disconnected and concurrent revisions. Source citations match an
existing registration identity/pin, and Service.Remember already preserves original
attribution separately from incorporation authorship. These are data guarantees,
not proof of fetching or source verification.

Missing are a service-level routing view, local source connections, availability
checks, retrieval, verified promotion and CLI/menu/native workflow. Ordinary recall
already excludes registration metadata. The engine cannot depend on process/network
packages; filesystem/Git reference adapters must remain outside that library.

The shared dispatcher separates bound memory tools from CLI-only administration.
The native plugin is thin, inherits native resources and supplies learning guidance.
The console/menu already supports arrows, Vim navigation, cancellation, no-color
and a plain fallback. Use those mechanisms rather than introducing another UI or
assistant capability system.

Verification uses only synthetic alternative layouts and controlled Git repositories.
No personal memory, account policy, credentials or source capability execution is
needed. Local availability cannot prove remote freshness; arbitrary prose cannot
be deterministically classified as true or authoritative by storage code.
